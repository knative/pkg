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
	"context"
	"net/url"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"knative.dev/pkg/observability/metrics/metricstest"
)

func TestNewClientMetricsProvider(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	p, err := NewClientMetricProvider(WithMeterProvider(provider))
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if p == nil {
		t.Fatal("provider returned was nil")
	}
}

func TestNewClientMetricsProviderGlobal(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	otel.SetMeterProvider(provider)
	p, err := NewClientMetricProvider()
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if p == nil {
		t.Fatal("provider returned was nil")
	}
}

func TestLatencyMetric(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	p, err := NewClientMetricProvider(WithMeterProvider(provider))
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	m := p.RequestLatencyMetric()
	if m == nil {
		t.Fatal("unexpected nil latency metric")
	}

	u, err := url.Parse("https://example.com/path")
	if err != nil {
		t.Fatal(err)
	}

	m.Observe(context.Background(), "GET", *u, time.Second)

	bucketCounts := [15]uint64{}
	bucketCounts[9] = 1

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual(
			scopeName,
			metricdata.Metrics{
				Name:        "http.client.request.duration",
				Unit:        "s",
				Description: "Duration of HTTP client requests.",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{{
						Bounds:       latencyBounds,
						Count:        1,
						Sum:          1,
						BucketCounts: bucketCounts[:],
						Min:          metricdata.NewExtrema[float64](1),
						Max:          metricdata.NewExtrema[float64](1),
						Attributes: attribute.NewSet(
							semconv.ServerAddressKey.String("example.com"),
							semconv.ServerPortKey.Int(443),
							semconv.HTTPRequestMethodKey.String("GET"),
							semconv.URLSchemeKey.String("https"),
							semconv.URLTemplateKey.String("/path"),
						),
					}},
				},
			},
		),
	)
}

func TestResultMetric(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	p, err := NewClientMetricProvider(WithMeterProvider(provider))
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	m := p.RequestResultMetric()
	if m == nil {
		t.Fatal("unexpected nil result metric")
	}

	m.Increment(context.Background(), "200", "GET", "example.com")

	metricstest.AssertMetrics(t, r,
		metricstest.MetricsEqual(
			scopeName,
			metricdata.Metrics{
				Name:        resultMetricName,
				Unit:        "{item}",
				Description: resultMetricDescription,
				Data: metricdata.Sum[int64]{
					IsMonotonic: true,
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.DataPoint[int64]{{
						Attributes: attribute.NewSet(
							semconv.ServerAddressKey.String("example.com"),
							semconv.HTTPRequestMethodKey.String("GET"),
							semconv.HTTPResponseStatusCodeKey.Int(200),
						),
						Value: 1,
					}},
				},
			},
		),
	)
}
