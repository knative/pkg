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
	"net/http"
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
)

var (
	exporter   view.Exporter
	mConfig    *metricsConfig
	promSrv    *http.Server
	promSrvMux sync.Mutex
)

// newMetricsExporter gets a metrics exporter based on the config.
func newMetricsExporter(config *metricsConfig, logger *zap.SugaredLogger) error {
	// If there is a Prometheus Exporter server running, stop it.
	promSrvMux.Lock()
	if promSrv != nil {
		promSrv.Close()
		promSrv = nil
	}
	promSrvMux.Unlock()

	mux.Lock()
	defer mux.Unlock()
	if exporter != nil {
		view.UnregisterExporter(exporter)
	}
	var err error
	switch config.backendDestination {
	case Stackdriver:
		exporter, err = newStackdriverExporter(config, logger)
	case Prometheus:
		exporter, err = newPrometheusExporter(config, logger)
	default:
		err = fmt.Errorf("Unsupported metrics backend %v", config.backendDestination)
	}
	if err != nil {
		return err
	}
	view.RegisterExporter(exporter)
	view.SetReportingPeriod(60 * time.Second)
	logger.Infof("Successfully updated the metrics exporter; old config: %v; new config %v", mConfig, config)
	mConfig = config
	return nil
}

// newStackdriverExporter function is not thread safe. It is client's responsibility to synchronize any
// multithreaded access.
func newStackdriverExporter(config *metricsConfig, logger *zap.SugaredLogger) (view.Exporter, error) {
	e, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:    config.stackdriverProjectID,
		MetricPrefix: config.domain + "/" + config.component,
		Resource: &monitoredrespb.MonitoredResource{
			Type: "global",
		},
		DefaultMonitoringLabels: &stackdriver.Labels{},
	})
	if err != nil {
		logger.Error("Failed to create the Stackdriver exporter.", zap.Error(err))
		return nil, err
	}
	logger.Infof("Created Opencensus Stackdriver exporter with config %v", config)
	return e, nil
}

// newPrometheusExporter function is not thread safe. It is client's responsibility to synchronize any
// multithreaded access.
func newPrometheusExporter(config *metricsConfig, logger *zap.SugaredLogger) (view.Exporter, error) {
	e, err := prometheus.NewExporter(prometheus.Options{Namespace: config.component})
	if err != nil {
		logger.Error("Failed to create the Prometheus exporter.", zap.Error(err))
		return nil, err
	}
	logger.Infof("Created Opencensus Prometheus exporter with config: %v. Start the server for Prometheus exporter.", config)
	// Start the server for Prometheus scraping
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", e)
		promSrvMux.Lock()
		promSrv = &http.Server{
			Addr:    ":9090",
			Handler: sm,
		}
		srv := promSrv
		promSrvMux.Unlock()
		srv.ListenAndServe()
	}()
	return e, nil
}

func getPromSrv() *http.Server {
	promSrvMux.Lock()
	defer promSrvMux.Unlock()
	return promSrv
}
