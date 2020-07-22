/*
Copyright 2020 The Knative Authors

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

package reconciler

import (
	"os"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"knative.dev/pkg/metrics"
)

const podNameEnvKey = "POD_NAME"

var (
	controllerOwnedBucketCountM = stats.Int64(
		"controller_owned_bucket_count",
		"Number of buckets a controller Pod owns for a reconciler",
		stats.UnitDimensionless)

	podNameKey        = tag.MustNewKey("pod_name")
	reconcilerNameKey = tag.MustNewKey("reconciler_name")

	metricKeys = []tag.Key{podNameKey, reconcilerNameKey}

	podName = os.Getenv(podNameEnvKey)
)

func init() {
	// Create views to see our measurements. This can return an error if
	// a previously-registered view has the same name with a different value.
	// View name defaults to the measure name if unspecified.
	if err := metrics.RegisterResourceView(
		&view.View{
			Description: "Number of pods autoscaler requested from Kubernetes",
			Measure:     controllerOwnedBucketCountM,
			Aggregation: view.LastValue(),
			TagKeys:     metricKeys,
		},
	); err != nil {
		panic(err)
	}
}
