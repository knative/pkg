/*
Copyright 2025 The Knative Authors

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

package k8s

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"knative.dev/pkg/observability/metrics/metricstest"
)

func TestGauge(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	m := must(provider.Meter("meter").Int64UpDownCounter("instrument"))
	g := gauge{m, attribute.NewSet(attribute.String("key", "val"))}

	g.Inc()

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual("meter", metricdata.Metrics{
			Name: "instrument",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.DataPoint[int64]{{
					Attributes: attribute.NewSet(attribute.String("key", "val")),
					Value:      1,
				}},
			},
		}),
	)

	g.Dec()

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual("meter", metricdata.Metrics{
			Name: "instrument",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.DataPoint[int64]{{
					Attributes: attribute.NewSet(attribute.String("key", "val")),
					Value:      0,
				}},
			},
		}),
	)
}

func TestCounter(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	m := must(provider.Meter("meter").Int64Counter("instrument"))
	c := counter{m, attribute.NewSet(attribute.String("key", "val"))}

	c.Inc()

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual("meter", metricdata.Metrics{
			Name: "instrument",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{{
					Attributes: attribute.NewSet(attribute.String("key", "val")),
					Value:      1,
				}},
			},
		}),
	)

	c.Inc()

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual("meter", metricdata.Metrics{
			Name: "instrument",
			Data: metricdata.Sum[int64]{
				Temporality: metricdata.CumulativeTemporality,
				IsMonotonic: true,
				DataPoints: []metricdata.DataPoint[int64]{{
					Attributes: attribute.NewSet(attribute.String("key", "val")),
					Value:      2,
				}},
			},
		}),
	)
}

func TestHistogram(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	m := must(provider.Meter("meter").Float64Histogram("instrument",
		metric.WithExplicitBucketBoundaries(0.01, 1, 2),
	))
	c := histogram{m, attribute.NewSet(attribute.String("key", "val"))}

	c.Observe(1)

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual("meter", metricdata.Metrics{
			Name: "instrument",
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{{
					Attributes:   attribute.NewSet(attribute.String("key", "val")),
					Bounds:       []float64{0.01, 1, 2},
					BucketCounts: []uint64{0, 1, 0, 0},
					Count:        1,
					Max:          metricdata.NewExtrema[float64](1),
					Min:          metricdata.NewExtrema[float64](1),
					Sum:          1,
				}},
			},
		}),
	)

	c.Observe(2)

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual("meter", metricdata.Metrics{
			Name: "instrument",
			Data: metricdata.Histogram[float64]{
				Temporality: metricdata.CumulativeTemporality,
				DataPoints: []metricdata.HistogramDataPoint[float64]{{
					Attributes:   attribute.NewSet(attribute.String("key", "val")),
					Bounds:       []float64{0.01, 1, 2},
					BucketCounts: []uint64{0, 1, 1, 0},
					Count:        2,
					Max:          metricdata.NewExtrema[float64](2),
					Min:          metricdata.NewExtrema[float64](1),
					Sum:          3,
				}},
			},
		}),
	)
}

func TestSettableGauge(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	m := must(provider.Meter("meter").Float64Gauge("instrument"))
	g := settableGauge{m, attribute.NewSet(attribute.String("key", "val"))}

	g.Set(1)

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual("meter", metricdata.Metrics{
			Name: "instrument",
			Data: metricdata.Gauge[float64]{
				DataPoints: []metricdata.DataPoint[float64]{{
					Attributes: attribute.NewSet(attribute.String("key", "val")),
					Value:      1,
				}},
			},
		}),
	)

	g.Set(2)

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual("meter", metricdata.Metrics{
			Name: "instrument",
			Data: metricdata.Gauge[float64]{
				DataPoints: []metricdata.DataPoint[float64]{{
					Attributes: attribute.NewSet(attribute.String("key", "val")),
					Value:      2,
				}},
			},
		}),
	)
}

func must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
