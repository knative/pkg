/*
Copyright 2019 The Knative Authors

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

	"contrib.go.opencensus.io/exporter/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
)

var (
	curPromSrv       *http.Server
	curPromSrvMux    sync.Mutex
	curExporter      resourceExporter
	metricTypeToProm = map[metricdata.Type]prom.ValueType{
		metricdata.TypeGaugeInt64:        prom.GaugeValue,
		metricdata.TypeGaugeFloat64:      prom.GaugeValue,
		metricdata.TypeCumulativeInt64:   prom.CounterValue,
		metricdata.TypeCumulativeFloat64: prom.CounterValue,
	}
)

type resourceExporter struct {
	opts   prometheus.Options
	logger *zap.SugaredLogger
}

func newPrometheusExporter(config *metricsConfig, logger *zap.SugaredLogger) (view.Exporter, ResourceExporterFactory, error) {
	curExporter = resourceExporter{
		opts: prometheus.Options{
			Namespace: config.component,
			Registry:  prom.NewRegistry(),
		},
		logger: logger,
	}
	e, err := prometheus.NewExporter(curExporter.opts)
	if err != nil {
		logger.Errorw("Failed to create the Prometheus exporter.", zap.Error(err))
		return nil, nil, err
	}
	logger.Infof("Created Opencensus Prometheus exporter with config: %v. Start the server for Prometheus exporter.", config)
	// Start the server for Prometheus scraping
	go func() {
		srv := startNewPromSrv(e, config.prometheusPort)
		srv.ListenAndServe()
	}()
	return e, curExporter.collectorForResource, nil
}

func (re *resourceExporter) collectorForResource(r *resource.Resource) (view.Exporter, error) {
	c := &miniCollector{
		namespace: re.opts.Namespace,
		meter:     meterForResource(r),
		logger:    re.logger,
	}

	err := prom.WrapRegistererWith(r.Labels, re.opts.Registry).Register(c)

	return c, err
}

func getCurPromSrv() *http.Server {
	curPromSrvMux.Lock()
	defer curPromSrvMux.Unlock()
	return curPromSrv
}

func resetCurPromSrv() {
	curPromSrvMux.Lock()
	defer curPromSrvMux.Unlock()
	if curPromSrv != nil {
		curPromSrv.Close()
		curPromSrv = nil
	}
}

func startNewPromSrv(e *prometheus.Exporter, port int) *http.Server {
	sm := http.NewServeMux()
	sm.Handle("/metrics", e)
	curPromSrvMux.Lock()
	defer curPromSrvMux.Unlock()
	if curPromSrv != nil {
		curPromSrv.Close()
	}
	curPromSrv = &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: sm,
	}
	return curPromSrv
}

type miniCollector struct {
	namespace string
	meter     view.Meter
	logger    *zap.SugaredLogger
}

func (mc *miniCollector) ExportView(vd *view.Data) {
	// pass
	// Maybe at some point we register this?
}

func (mc *miniCollector) Describe(d chan<- *prom.Desc) {
	reader, ok := mc.meter.(metricproducer.Producer)
	if !ok {
		mc.logger.Warn("Unable to convert Meter to a metric producer")
		return
	}
	ocMetrics := reader.Read()
	for _, m := range ocMetrics {
		d <- mc.toPromDesc(m.Descriptor)
	}
}

func (mc *miniCollector) Collect(metrics chan<- prom.Metric) {
	reader, ok := mc.meter.(metricproducer.Producer)
	if !ok {
		mc.logger.Warn("Unable to convert Meter to a metric producer")
		return
	}
	ocMetrics := reader.Read()
	for _, m := range ocMetrics {
		desc := mc.toPromDesc(m.Descriptor)
		for _, ts := range m.TimeSeries {
			labels := make([]string, 0, len(ts.LabelValues))
			for _, lv := range ts.LabelValues {
				labels = append(labels, lv.Value)
			}
			pt := ts.Points[len(ts.Points)-1] // TODO: see which order these are in.

			metric, err := MetricFromPoint(desc, m.Descriptor.Type, pt, labels)
			if err != nil {
				mc.logger.Warn("Failed to convert %q to Prometheus Metric: %v", desc, err)
				continue
			}
			metrics <- metric
		}
	}
}

func (mc *miniCollector) toPromDesc(m metricdata.Descriptor) *prom.Desc {
	labels := make([]string, 0, len(m.LabelKeys))
	for _, l := range m.LabelKeys {
		labels = append(labels, sanitizedName(l.Key))
	}
	name := sanitizedName(m.Name)
	if mc.namespace != "" {
		name = mc.namespace + "_" + name
	}
	return prom.NewDesc(name, m.Description, labels, prom.Labels{})
}

func sanitizedName(s string) string {
	// TODO: copy from prometheus/sanitize.go
	// NOTE: unicode.IsLetter covers more than ASCII allowed by Prometheus!
	return s
}

func MetricFromPoint(desc *prom.Desc, t metricdata.Type, pt metricdata.Point, labels []string) (prom.Metric, error) {
	if t == metricdata.TypeCumulativeDistribution {
		data, ok := pt.Value.(*metricdata.Distribution)
		if !ok {
			return nil, fmt.Errorf("bad value for %q", desc)
		}
		var sum uint64
		buckets := make(map[float64]uint64)
		for i, b := range data.BucketOptions.Bounds {
			sum += uint64(data.Buckets[i].Count)
			buckets[b] = sum
		}
		return prom.NewConstHistogram(desc, uint64(data.Count), data.Sum, buckets, labels...)
	}

	if promType, ok := metricTypeToProm[t]; ok {
		var data float64
		switch v := pt.Value.(type) {
		case float64:
			data = v
		case int64:
			data = float64(v)
		default:
			return nil, fmt.Errorf("bad value type for %q", desc)
		}
		return prom.NewConstMetric(desc, promType, data, labels...)
	}
	return nil, fmt.Errorf("unsupported metric type %d", t)
}
