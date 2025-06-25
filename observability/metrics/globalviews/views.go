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

package globalviews

import (
	"maps"
	"slices"

	"go.opentelemetry.io/otel/sdk/metric"
)

var globalViews map[string][]metric.View = make(map[string][]metric.View)

func Register(pkg string, v ...metric.View) {
	views, ok := globalViews[pkg]
	if !ok {
		// I would expect a single registration call per view
		views = make([]metric.View, 0, 1)
	}

	globalViews[pkg] = append(views, v...)
}

func GetPackageViews(pkg string) []metric.View {
	return globalViews[pkg]
}

func GetAllViews() []metric.View {
	list := slices.Collect(maps.Values(globalViews))
	return slices.Concat(list...)
}
