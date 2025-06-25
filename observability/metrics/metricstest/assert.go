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

package metricstest

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"k8s.io/apimachinery/pkg/util/sets"
)

type metricReader interface {
	Collect(ctx context.Context, rm *metricdata.ResourceMetrics) error
}

type testingT interface {
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	Helper()
	Failed() bool
}

type AssertFunc func(testingT, *metricdata.ResourceMetrics)

func AssertMetrics(t testingT, r metricReader, assertFns ...AssertFunc) {
	t.Helper()

	var rm metricdata.ResourceMetrics
	r.Collect(context.Background(), &rm)

	for _, assertFn := range assertFns {
		assertFn(t, &rm)
	}
}

func HasAttributes(
	scopePrefix string,
	metricPrefix string,
	want ...attribute.KeyValue,
) AssertFunc {
	return func(t testingT, rm *metricdata.ResourceMetrics) {
		t.Helper()

		assertCalled := false

		if len(want) == 0 {
			return
		}

		for _, sm := range rm.ScopeMetrics {
			if !strings.HasPrefix(sm.Scope.Name, scopePrefix) {
				continue
			}

			for _, metric := range sm.Metrics {
				if !strings.HasPrefix(metric.Name, metricPrefix) {
					continue
				}

				assertCalled = true

				mt := t.(metricdatatest.TestingT)
				switch data := metric.Data.(type) {
				case metricdata.Sum[int64]:
					metricdatatest.AssertHasAttributes(mt, data, want...)
				case metricdata.Sum[float64]:
					metricdatatest.AssertHasAttributes(mt, data, want...)
				case metricdata.Histogram[int64]:
					metricdatatest.AssertHasAttributes(mt, data, want...)
				case metricdata.Histogram[float64]:
					metricdatatest.AssertHasAttributes(mt, data, want...)
				case metricdata.ExponentialHistogram[int64]:
					metricdatatest.AssertHasAttributes(mt, data, want...)
				case metricdata.ExponentialHistogram[float64]:
					metricdatatest.AssertHasAttributes(mt, data, want...)
				case metricdata.Gauge[int64]:
					metricdatatest.AssertHasAttributes(mt, data, want...)
				case metricdata.Gauge[float64]:
					metricdatatest.AssertHasAttributes(mt, data, want...)
				default:
					t.Fatalf("unsupported metric data type for metric %q: %T", metric.Name, data)
				}
			}
		}
		if !assertCalled {
			t.Error("expected attributes but scope and metric prefix didn't match any results")
		}
	}
}

func MetricsPresent(scopeName string, names ...string) AssertFunc {
	return func(t testingT, rm *metricdata.ResourceMetrics) {
		t.Helper()

		want := sets.New(names...)
		got := sets.New[string]()

		for _, sm := range rm.ScopeMetrics {
			if sm.Scope.Name != scopeName {
				continue
			}
			for _, metric := range sm.Metrics {
				if !want.Has(metric.Name) {
					t.Fatal("unexpected metric", metric.Name)
				}

				got.Insert(metric.Name)
			}
		}

		diff := want.Difference(got)
		if len(diff) > 0 {
			t.Fatal("expected metrics didn't appear", diff.UnsortedList())
		}
	}
}
