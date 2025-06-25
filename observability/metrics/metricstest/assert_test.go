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
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestAssertMetricsPresent(t *testing.T) {
	ft := &fakeT{}

	m := &metrics{
		ScopeMetrics: []metricdata.ScopeMetrics{{
			Scope: instrumentation.Scope{
				Name: "first",
			},
			Metrics: []metricdata.Metrics{{
				Name: "a",
			}},
		}},
	}

	AssertMetrics(ft, m, MetricsPresent("first", "a"))

	if ft.Failed() {
		t.Error("unexpected failure")
	}

	ft = &fakeT{}

	AssertMetrics(ft, m, MetricsPresent("second", "a"))

	if !ft.Failed() {
		t.Error("should have failed")
	}

	AssertMetrics(ft, m, MetricsPresent("first", "b"))

	if !ft.Failed() {
		t.Error("should have failed")
	}
}

func TestHasAttributes(t *testing.T) {
	m := &metrics{
		ScopeMetrics: []metricdata.ScopeMetrics{{
			Scope: instrumentation.Scope{Name: "first-scope"},
			Metrics: []metricdata.Metrics{{
				Name: "a-metric",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{{
						Attributes: attribute.NewSet(
							attribute.String("k", "v"),
						),
					}},
				},
			}},
		}, {
			Scope: instrumentation.Scope{Name: "second-scope"},
			Metrics: []metricdata.Metrics{{
				Name: "b-metric",
				Data: metricdata.Sum[int64]{
					DataPoints: []metricdata.DataPoint[int64]{{
						Attributes: attribute.NewSet(
							attribute.String("k", "v"),
						),
					}},
				},
			}, {
				Name: "c-metric",
				Data: metricdata.Histogram[int64]{
					DataPoints: []metricdata.HistogramDataPoint[int64]{{
						Attributes: attribute.NewSet(
							attribute.String("k", "v"),
						),
					}},
				},
			}, {
				Name: "d-metric",
				Data: metricdata.Histogram[float64]{
					DataPoints: []metricdata.HistogramDataPoint[float64]{{
						Attributes: attribute.NewSet(
							attribute.String("k", "v"),
						),
					}},
				},
			}},
		}},
	}

	t.Run("no attributes", func(t *testing.T) {
		ft := &fakeT{}

		AssertMetrics(ft, m, HasAttributes("first", "a"))

		// no metric prefix
		AssertMetrics(ft, m, HasAttributes("first", ""))

		// no scope prefix
		AssertMetrics(ft, m, HasAttributes("", "a",
			attribute.String("k", "v"),
		))

		// no scope and metric prefix
		AssertMetrics(ft, m, HasAttributes("", ""))

		if ft.Failed() {
			t.Error("unexpected failure")
		}
	})

	t.Run("has attributes", func(t *testing.T) {
		ft := &fakeT{}

		AssertMetrics(ft, m, HasAttributes("first", "a",
			attribute.String("k", "v"),
		))

		// no metric prefix
		AssertMetrics(ft, m, HasAttributes("first", "",
			attribute.String("k", "v"),
		))

		// no scope prefix
		AssertMetrics(ft, m, HasAttributes("", "a",
			attribute.String("k", "v"),
		))

		// no scope and metric prefix
		AssertMetrics(ft, m, HasAttributes("", "",
			attribute.String("k", "v"),
		))

		if ft.Failed() {
			t.Error("unexpected failure")
		}
	})

	t.Run("no match", func(t *testing.T) {
		ft := &fakeT{}

		AssertMetrics(ft, m,
			HasAttributes("", "", attribute.String("k", "a")))

		if !ft.Failed() {
			t.Error("expected failure")
		}
	})

	t.Run("no metrics", func(t *testing.T) {
		ft := &fakeT{}
		m := &metrics{}

		AssertMetrics(ft, m,
			HasAttributes("first", "a", attribute.String("k", "a")))

		if !ft.Failed() {
			t.Error("expected failure")
		}
	})
}

type metrics metricdata.ResourceMetrics

func (f *metrics) Collect(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	*rm = (metricdata.ResourceMetrics)(*f)
	return nil
}

type fakeT struct {
	failed bool
}

func (t *fakeT) Failed() bool {
	return t.failed
}
func (t *fakeT) Error(args ...any)                 { t.failed = true }
func (t *fakeT) Errorf(format string, args ...any) { t.failed = true }
func (t *fakeT) Fatal(args ...any)                 { t.failed = true }
func (t *fakeT) Fatalf(format string, args ...any) { t.failed = true }
func (t *fakeT) Helper()                           {}
