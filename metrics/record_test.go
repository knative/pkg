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
	"context"
	"fmt"
	"path"
	"testing"

	"go.opencensus.io/resource"
	"go.opencensus.io/tag"

	"knative.dev/pkg/metrics/metricskey"
	"knative.dev/pkg/metrics/metricstest"

	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

type cases struct {
	name          string
	metricsConfig *metricsConfig
	measurement   stats.Measurement
}

func TestRecordServing(t *testing.T) {
	measure := stats.Int64("request_count", "Number of reconcile operations", stats.UnitNone)
	shouldReportCases := []cases{
		// Increase the measurement value for each test case so that checking
		// the last value ensures the measurement has been recorded.
		{
			name:          "none stackdriver backend",
			metricsConfig: &metricsConfig{},
			measurement:   measure.M(1),
		}, {
			name: "stackdriver backend with supported metric",
			metricsConfig: &metricsConfig{
				isStackdriverBackend:        true,
				stackdriverMetricTypePrefix: "knative.dev/internal/serving/activator",
			},
			measurement: measure.M(2),
		}, {
			name: "stackdriver backend with unsupported metric and allow custom metric",
			metricsConfig: &metricsConfig{
				isStackdriverBackend:        true,
				stackdriverMetricTypePrefix: "knative.dev/unsupported",
			},
			measurement: measure.M(3),
		}, {
			name:        "empty metricsConfig",
			measurement: measure.M(4),
		},
	}
	testRecord(t, measure, shouldReportCases)
}

func TestRecordEventing(t *testing.T) {
	measure := stats.Int64("event_count", "Number of event received", stats.UnitNone)
	shouldReportCases := []cases{
		// Increase the measurement value for each test case so that checking
		// the last value ensures the measurement has been recorded.
		{
			name:          "none stackdriver backend",
			metricsConfig: &metricsConfig{},
			measurement:   measure.M(1),
		}, {
			name: "stackdriver backend with supported metric",
			metricsConfig: &metricsConfig{
				isStackdriverBackend:        true,
				stackdriverMetricTypePrefix: "knative.dev/eventing/broker",
			},
			measurement: measure.M(5),
		}, {
			name: "stackdriver backend with unsupported metric and allow custom metric",
			metricsConfig: &metricsConfig{
				isStackdriverBackend:        true,
				stackdriverMetricTypePrefix: "knative.dev/unsupported",
			},
			measurement: measure.M(3),
		}, {
			name:        "empty metricsConfig",
			measurement: measure.M(4),
		},
	}
	testRecord(t, measure, shouldReportCases)
}

func TestRecordResource(t *testing.T) {
	measure := stats.Int64("number", "Count of times called", stats.UnitNone)
	var err error
	v := &view.View{
		Measure:     measure,
		Aggregation: view.LastValue(),
		TagKeys: []tag.Key{
			tag.MustNewKey("foo"),
			tag.MustNewKey("hero"),
			tag.MustNewKey("sandwitch"),
		},
	}

	testCases := []struct {
		name string
		rsrc *resource.Resource
		tags map[string]string
		want map[string]string
	}{
		{
			name: "no resource",
			want: map[string]string{},
		},
		{
			name: "with resource",
			rsrc: &resource.Resource{Type: "test", Labels: map[string]string{"foo": "bar"}},
			want: map[string]string{"foo": "bar"},
		},
		{
			name: "resource with tags",
			rsrc: &resource.Resource{Type: "test", Labels: map[string]string{"foo": "bar"}},
			tags: map[string]string{"foo": "baz", "hero": "aquaman"},
			want: map[string]string{"foo": "bar", "hero": "aquaman"},
		},
	}

	for _, backend := range []metricsBackend{Prometheus, Stackdriver, OpenCensus} {

		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				view.Register(v)
				defer view.Unregister(v)

				ctx := metricskey.NewContext(context.Background(), test.rsrc)
				setCurMetricsConfig(&metricsConfig{
					backendDestination: backend,
				})
				for k, v := range test.tags {
					if ctx, err = tag.New(ctx, tag.Insert(tag.MustNewKey(k), v)); err != nil {
						t.Errorf("Failed to set tag %q: %+v", k, err)
					}
				}
				Record(ctx, measure.M(1))
				metricstest.CheckLastValueData(t, measure.Name(), test.want, 1)
			})
		}
	}
}

func testRecord(t *testing.T, measure *stats.Int64Measure, shouldReportCases []cases) {
	ctx := context.TODO()
	v := &view.View{
		Measure:     measure,
		Aggregation: view.LastValue(),
	}
	view.Register(v)
	defer view.Unregister(v)

	for _, test := range shouldReportCases {
		setCurMetricsConfig(test.metricsConfig)
		Record(ctx, test.measurement)
		metricstest.CheckLastValueData(t, test.measurement.Measure().Name(), map[string]string{}, test.measurement.Value())
	}

	shouldNotReportCases := []struct {
		name          string
		metricsConfig *metricsConfig
		measurement   stats.Measurement
	}{
		// Use a different value for the measurement other than the last one of shouldReportCases
		{
			name: "stackdriver backend with unsupported metric but not allow custom metric",
			metricsConfig: &metricsConfig{
				isStackdriverBackend:        true,
				stackdriverMetricTypePrefix: "knative.dev/unsupported",
				recorder: func(ctx context.Context, ms stats.Measurement, ros ...stats.Options) error {
					metricType := path.Join("knative.dev/unsupported", ms.Measure().Name())
					if metricskey.KnativeRevisionMetrics.Has(metricType) || metricskey.KnativeTriggerMetrics.Has(metricType) {
						ros = append(ros, stats.WithMeasurements(ms))
						return stats.RecordWithOptions(ctx, ros...)
					}
					return nil
				},
			},
			measurement: measure.M(5),
		},
	}

	for _, test := range shouldNotReportCases {
		setCurMetricsConfig(test.metricsConfig)
		Record(ctx, test.measurement)
		metricstest.CheckLastValueData(t, test.measurement.Measure().Name(), map[string]string{}, 4) // The value is still the last one of shouldReportCases
	}
}

func TestBucketsNBy10(t *testing.T) {
	tests := []struct {
		base float64
		n    int
		want []float64
	}{{
		base: 0.001,
		n:    5,
		want: []float64{0.001, 0.01, 0.1, 1, 10},
	}, {
		base: 1,
		n:    2,
		want: []float64{1, 10},
	}, {
		base: 0.5,
		n:    4,
		want: []float64{0.5, 5, 50, 500},
	}}

	for _, test := range tests {
		t.Run(fmt.Sprintf("base=%f,n=%d", test.base, test.n), func(t *testing.T) {
			got := BucketsNBy10(test.base, test.n)
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Errorf("BucketsNBy10 (-want, +got) = %s", diff)
			}
		})
	}
}
