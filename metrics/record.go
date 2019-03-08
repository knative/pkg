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
	"path"

	"github.com/knative/pkg/metrics/metricskey"
	"go.opencensus.io/stats"
)

// Record decides whether to record one measurement via OpenCensus based on the
// following conditions:
//   1) No package level metrics config. Users must ensure metrics config are set
//      to get expected behavior. Otherwise it just proxies to OpenCensus based
//      on the assumption that users intend to record metric when they call this
//      function.
//   2) The backend is not Stackdriver.
//   3) The backend is Stackdriver and it is allowed to use custom metrics.
//   4) The backend is Stackdriver and the metric is "knative_revison" built-in metric.
func Record(ctx context.Context, ms stats.Measurement) {
	mc := getCurMetricsConfig()

	// Condition 1)
	if mc == nil {
		stats.Record(ctx, ms)
		return
	}

	if !mc.isStackdriverBackend || mc.allowStackdriverCustomMetrics {
		// Condition 2) and 3)
		stats.Record(ctx, ms)
	} else {
		metricType := path.Join(mc.stackdriverMetricTypePrefix, ms.Measure().Name())
		// Condition 4)
		if metricskey.KnativeRevisionMetrics.Has(metricType) {
			stats.Record(ctx, ms)
		}
	}
}
