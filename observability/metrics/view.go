/*
Copyright 2026 The Knative Authors

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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
)

// MetricAttributesDenyFilter returns a View that strips the given attribute
// keys from every instrument.
func MetricAttributesDenyFilter(denyList []string) metric.View {
	keys := make([]attribute.Key, len(denyList))
	for i, k := range denyList {
		keys[i] = attribute.Key(k)
	}
	return metric.NewView(
		metric.Instrument{Name: "*"},
		metric.Stream{
			AttributeFilter: attribute.NewDenyKeysFilter(keys...),
		},
	)
}
