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

package globalviews_test

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"knative.dev/pkg/observability/metrics/globalviews"
)

var testView = metric.NewView(
	metric.Instrument{
		Name:  "latency",
		Scope: instrumentation.Scope{Name: "http"},
	},
	metric.Stream{
		Aggregation: metric.AggregationBase2ExponentialHistogram{
			MaxSize:  160,
			MaxScale: 20,
		},
	},
)

var secondView = metric.NewView(
	metric.Instrument{
		Name:  "latency",
		Scope: instrumentation.Scope{Name: "http"},
	},
	metric.Stream{
		Aggregation: metric.AggregationBase2ExponentialHistogram{
			MaxSize:  160,
			MaxScale: 20,
		},
	},
)

func TestRegistration(t *testing.T) {
	testPackage := "com.example.package"

	if len(globalviews.GetAllViews()) != 0 {
		t.Fatal("expected zero views")
	}

	globalviews.Register(testPackage, testView)

	if count := len(globalviews.GetAllViews()); count != 1 {
		t.Fatalf("expected global view count to be 1 got %d", count)
	}

	if count := len(globalviews.GetPackageViews(testPackage)); count != 1 {
		t.Fatalf("expected a single view for %q got %d", testPackage, count)
	}

	if count := len(globalviews.GetPackageViews("com.example.second.package")); count != 0 {
		t.Fatalf("expected no views for 'second' package got %d", count)
	}

	globalviews.Register(testPackage, secondView)

	if count := len(globalviews.GetAllViews()); count != 2 {
		t.Fatalf("expected global view count to be 2 got %d", count)
	}

	if count := len(globalviews.GetPackageViews(testPackage)); count != 2 {
		t.Fatalf("expected a single view for %q got %d", testPackage, count)
	}

	if count := len(globalviews.GetPackageViews("com.example.second.package")); count != 0 {
		t.Fatalf("expected no views for 'second' package got %d", count)
	}
}
