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
	"testing"

	"knative.dev/pkg/metrics/metricstest"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

func TestRecord(t *testing.T) {
	ctx := context.TODO()
	measure := stats.Int64("request_count", "Number of reconcile operations", stats.UnitNone)
	v := &view.View{
		Measure:     measure,
		Aggregation: view.LastValue(),
	}
	view.Register(v)
	defer view.Unregister(v)

	shouldReportCases := []struct {
		name          string
		metricsConfig *metricsConfig
		measurement   stats.Measurement
	}{
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
				stackdriverMetricTypePrefix: "knative.dev/serving/activator",
			},
			measurement: measure.M(2),
		}, {
			name: "stackdriver backend with unsupported metric and allow custom metric",
			metricsConfig: &metricsConfig{
				isStackdriverBackend:          true,
				stackdriverMetricTypePrefix:   "knative.dev/unsupported",
				allowStackdriverCustomMetrics: true,
			},
			measurement: measure.M(3),
		}, {
			name:        "empty metricsConfig",
			measurement: measure.M(4),
		},
	}

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
