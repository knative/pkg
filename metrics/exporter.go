/*
Copyright 2018 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"fmt"
	"sync"

	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	sdconfig "knative.dev/pkg/stackdriver/config"
)

var (
	curMetricsExporter   view.Exporter
	curMetricsConfig     *metricsConfig
	curStackdriverConfig *sdconfig.Config

	metricsMux sync.RWMutex
)

type flushable interface {
	// Flush waits for metrics to be uploaded.
	Flush()
}

// ExporterOptions contains options for configuring the exporter.
type ExporterOptions struct {
	// Domain is the metrics domain. e.g. "knative.dev". Must be present.
	//
	// Stackdriver uses the following format to construct full metric name:
	//    <domain>/<component>/<metric name from View>
	// Prometheus uses the following format to construct full metric name:
	//    <component>_<metric name from View>
	// Domain is actually not used if metrics backend is Prometheus.
	Domain string

	// Component is the name of the component that emits the metrics. e.g.
	// "activator", "queue_proxy". Should only contains alphabets and underscore.
	// Must be present.
	Component string

	// PrometheusPort is the port to expose metrics if metrics backend is Prometheus.
	// It should be between maxPrometheusPort and maxPrometheusPort. 0 value means
	// using the default 9090 value. If is ignored if metrics backend is not
	// Prometheus.
	PrometheusPort int

	// ConfigMap is the data from config map config-observability. Must be present.
	// See https://github.com/knative/serving/blob/master/config/config-observability.yaml
	// for details. This contains configuration specific to metrics.
	ConfigMap map[string]string

	// StackdriverConfigMap is the data from a config map which configures where metrics are sent
	// when using Stackdriver. Providing values for this config map is optional.
	// When absent, Google Application Default Credentials will be used.
	// The map fields are defined in "knative.dev/pkg/stackdriver/config/config.go".
	StackdriverConfigMap map[string]string
}

func init() {
	curStackdriverConfig = &sdconfig.Config{}
}

// UpdateExporterFromConfigMap returns a helper func that can be used to update the exporter
// when the config map that populates ExporterOptions.ConfigMap is updated.
func UpdateExporterFromConfigMap(component string, logger *zap.SugaredLogger) func(configMap *corev1.ConfigMap) {
	domain := Domain()
	return func(configMap *corev1.ConfigMap) {
		UpdateExporter(ExporterOptions{
			Domain:    domain,
			Component: component,
			ConfigMap: configMap.Data,
		}, logger)
	}
}

// UpdateExporterFromStackdriverConfigMap returns a helper func that can be used to update the exporter
// when the config map that populates ExporterOptions.StackdriverConfigMap is updated.
func UpdateExporterFromStackdriverConfigMap(logger *zap.SugaredLogger) func(configMap *corev1.ConfigMap) {
	return func(configMap *corev1.ConfigMap) {
		UpdateExporter(ExporterOptions{
			StackdriverConfigMap: configMap.Data,
		}, logger)
	}
}

// UpdateExporter updates the exporter based on the given ExporterOptions.
// This is a thread-safe function. The entire series of operations is locked
// to prevent a race condition between reading the current configuration
// and updating the current exporter.
func UpdateExporter(ops ExporterOptions, logger *zap.SugaredLogger) error {
	metricsMux.Lock()
	defer metricsMux.Unlock()

	if ops.StackdriverConfigMap != nil {
		sc, sErr := sdconfig.NewStackdriverConfigFromMap(ops.StackdriverConfigMap)
		if sErr != nil {
			logger.Warnw("Failed to get a valid Stackdriver config", zap.Error(sErr))
		}

		// Always store updates from Stackdriver config map, even if exporter
		// itself cannot be updated yet.
		curStackdriverConfig = sc
	}

	// The stackdriver config alone is not sufficient to create a metrics exporter.
	// Wait for the first update to ExporterOptions.ConfigMap to update the exporter.
	if ops.ConfigMap == nil && curMetricsConfig == nil {
		return nil
	}

	var newConfig *metricsConfig
	if ops.ConfigMap != nil {
		// If there's an updated config map, build a new config from scratch.
		nc, err := createMetricsConfig(ops, logger)
		if err != nil {
			if curMetricsExporter == nil {
				// Fail the process if there doesn't exist an exporter.
				logger.Errorw("Failed to get a valid metrics config", zap.Error(err))
			} else {
				logger.Errorw("Failed to get a valid metrics config; Skip updating the metrics exporter", zap.Error(err))
			}
			return err
		}

		newConfig = nc
	} else {
		// Otherwise, build off copy of the current config.
		// It is always true that curMetricsConfig is non-nil to reach here.
		newConfig = new(metricsConfig)
		*newConfig = *curMetricsConfig
	}

	// Add the Stackdriver config from any previous config map update.
	newConfig.stackdriverConfig = *curStackdriverConfig

	if isNewExporterRequired(newConfig) {
		logger.Info("Flushing the existing exporter before setting up the new exporter.")
		flushExporterUnlocked(curMetricsExporter)
		e, err := newMetricsExporter(newConfig, logger)
		if err != nil {
			logger.Errorf("Failed to update a new metrics exporter based on metric config %v. error: %v", newConfig, err)
			return err
		}
		existingConfig := curMetricsConfig
		setCurMetricsExporterUnlocked(e)
		logger.Infof("Successfully updated the metrics exporter; old config: %v; new config %v", existingConfig, newConfig)
	}

	setCurMetricsConfigUnlocked(newConfig)
	return nil
}

// isNewExporterRequired compares the non-nil newConfig against curMetricsConfig. When backend changes,
// or stackdriver project ID changes for stackdriver backend, we need to update the metrics exporter.
// This function is not implicitly thread-safe.
func isNewExporterRequired(newConfig *metricsConfig) bool {
	cc := curMetricsConfig
	if cc == nil || newConfig.backendDestination != cc.backendDestination {
		return true
	}

	if newConfig.backendDestination == Stackdriver {
		return newConfig.stackdriverProjectID != cc.stackdriverProjectID || newConfig.stackdriverConfig != cc.stackdriverConfig
	}

	return false
}

// newMetricsExporter gets a metrics exporter based on the config.
// This function is not implicitly thread-safe.
func newMetricsExporter(config *metricsConfig, logger *zap.SugaredLogger) (view.Exporter, error) {
	ce := curMetricsExporter
	// If there is a Prometheus Exporter server running, stop it.
	resetCurPromSrv()

	if ce != nil {
		// UnregisterExporter is idempotent and it can be called multiple times for the same exporter
		// without side effects.
		view.UnregisterExporter(ce)
	}
	var err error
	var e view.Exporter
	switch config.backendDestination {
	case Stackdriver:
		e, err = newStackdriverExporter(config, logger)
	case Prometheus:
		e, err = newPrometheusExporter(config, logger)
	default:
		err = fmt.Errorf("unsupported metrics backend %v", config.backendDestination)
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func getCurMetricsExporter() view.Exporter {
	metricsMux.RLock()
	defer metricsMux.RUnlock()
	return curMetricsExporter
}

func setCurMetricsExporter(e view.Exporter) {
	metricsMux.Lock()
	defer metricsMux.Unlock()
	setCurMetricsExporterUnlocked(e)
}

func setCurMetricsExporterUnlocked(e view.Exporter) {
	view.RegisterExporter(e)
	curMetricsExporter = e
}

func getCurMetricsConfig() *metricsConfig {
	metricsMux.RLock()
	defer metricsMux.RUnlock()
	return curMetricsConfig
}

func setCurMetricsConfig(c *metricsConfig) {
	metricsMux.Lock()
	defer metricsMux.Unlock()
	setCurMetricsConfigUnlocked(c)
}

func setCurMetricsConfigUnlocked(c *metricsConfig) {
	if c != nil {
		view.SetReportingPeriod(c.reportingPeriod)
	} else {
		// Setting to 0 enables the default behavior.
		view.SetReportingPeriod(0)
	}
	curMetricsConfig = c
}

// FlushExporter waits for exported data to be uploaded.
// This should be called before the process shuts down or exporter is replaced.
// Return value indicates whether the exporter is flushable or not.
func FlushExporter() bool {
	e := getCurMetricsExporter()
	return flushExporterUnlocked(e)
}

func flushExporterUnlocked(e view.Exporter) bool {
	if e == nil {
		return false
	}

	if f, ok := e.(flushable); ok {
		f.Flush()
		return true
	}
	return false
}
