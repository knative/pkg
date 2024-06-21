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

// Package metricstest simplifies some of the common boilerplate around testing
// metrics exports. It should work with or without the code in metrics, but this
// code particularly knows how to deal with metrics which are exported for
// multiple Resources in the same process.
package metricstest

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

func TestMetricEqual(t *testing.T) {
	pointValues := []int64{127, -1}
	baseValue := &metricdata.Metric{
		Descriptor: metricdata.Descriptor{
			Name:      "testing/metric",
			Unit:      metricdata.UnitMilliseconds,
			Type:      metricdata.TypeGaugeInt64,
			LabelKeys: []metricdata.LabelKey{{Key: "key1"}, {Key: "key2"}},
		},
		Resource: &resource.Resource{
			Type:   "testObject",
			Labels: map[string]string{"rl1": "aaa", "rl2": "bbb"},
		},
		TimeSeries: []*metricdata.TimeSeries{
			{
				LabelValues: []metricdata.LabelValue{{Value: "val1", Present: true}, {Value: "val2", Present: true}},
				Points:      []metricdata.Point{metricdata.NewInt64Point(time.Now(), pointValues[0])},
			},
			{
				LabelValues: []metricdata.LabelValue{{Value: "val1", Present: true}, {Value: "val3", Present: true}},
				Points:      []metricdata.Point{metricdata.NewInt64Point(time.Now(), pointValues[1])},
			},
		},
	}

	tests := []struct {
		name     string
		want     *Metric
		notEqual bool
	}{{
		name: "Minimal test",
		want: &Metric{
			Name: "testing/metric",
		},
	}, {
		name: "Test resource not equal",
		want: &Metric{
			Name:           "testing/metric",
			VerifyResource: true,
		},
		notEqual: true,
	}, {
		name: "Test unit not equal",
		want: &Metric{
			Name: "testing/metric",
			Unit: metricdata.UnitBytes,
		},
		notEqual: true,
	}, {
		name: "Test unit missing",
		want: &Metric{
			Name:           "testing/metric",
			Type:           metricdata.TypeGaugeInt64,
			VerifyMetadata: true,
		},
		notEqual: true,
	}, {
		name: "Test type not equal",
		want: &Metric{
			Name:           "testing/metric",
			Unit:           metricdata.UnitMilliseconds,
			Type:           metricdata.TypeCumulativeInt64,
			VerifyMetadata: true,
		},
		notEqual: true,
	}, {
		name: "Check resource",
		want: &Metric{
			Name: "testing/metric",
			Resource: &resource.Resource{
				Type:   "testObject",
				Labels: map[string]string{"rl2": "bbb", "rl1": "aaa"},
			},
		},
	}, {
		name: "Mismatched resource",
		want: &Metric{
			Name: "testing/metric",
			Resource: &resource.Resource{
				Type:   "testObject",
				Labels: map[string]string{"rl1": "aaa"},
			},
		},
		notEqual: true,
	}, {
		name: "Test value match",
		want: &Metric{
			Name: "testing/metric",
			Values: []Value{
				{
					Tags:  map[string]string{"key1": "val1", "key2": "val2"},
					Int64: &pointValues[0],
				}, {
					Tags:  map[string]string{"key1": "val1", "key2": "val3"},
					Int64: &pointValues[1],
				},
			},
		},
	}, {
		name: "Too few values",
		want: &Metric{
			Name: "testing/metric",
			Values: []Value{
				{
					Tags:  map[string]string{"key1": "val1", "key2": "val2"},
					Int64: &pointValues[0],
				},
			},
		},
		notEqual: true,
	}, {
		name: "Wrong tags",
		want: &Metric{
			Name: "testing/metric",
			Values: []Value{
				{
					Tags:  map[string]string{"key1": "val1", "key2": "val2"},
					Int64: &pointValues[0],
				},
				{
					Tags:  map[string]string{"key1": "val1", "key2": "bbb"},
					Int64: &pointValues[1],
				},
			},
		},
		notEqual: true,
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMetric(baseValue) // cmp needs to match types to find the Equal method.
			if diff := cmp.Diff(tc.want, &m); (diff == "") == tc.notEqual {
				t.Errorf("Metric.Equal() = %t failed (-want +base):\n%s", !tc.notEqual, diff)
			}
		})
	}
}

func buckets(v ...int64) []metricdata.Bucket {
	ret := []metricdata.Bucket{}
	for _, c := range v {
		ret = append(ret, metricdata.Bucket{Count: c})
	}
	return ret
}

func bucketOpts(bounds ...float64) *metricdata.BucketOptions {
	return &metricdata.BucketOptions{Bounds: bounds}
}

func TestDistributionEqual(t *testing.T) {
	baseValue := NewMetric(&metricdata.Metric{
		Descriptor: metricdata.Descriptor{
			Name:      "testing/distribution",
			LabelKeys: []metricdata.LabelKey{{Key: "key1"}},
		},
		TimeSeries: []*metricdata.TimeSeries{
			{
				LabelValues: []metricdata.LabelValue{{Value: "val1", Present: true}},
				Points: []metricdata.Point{metricdata.NewDistributionPoint(
					time.Now(),
					&metricdata.Distribution{
						Count:                 5, // Values: 0.5, 5, 5, 5, 10
						Sum:                   25.5,
						SumOfSquaredDeviation: 45.2,
						BucketOptions:         bucketOpts(2, 4, 8),
						Buckets:               buckets(1, 0, 3, 1),
					},
				)},
			},
		},
	})
	floatVal := -3.22

	tests := []struct {
		name     string
		want     Value
		notEqual bool
	}{{
		name: "Equal",
		want: Value{
			Tags: map[string]string{"key1": "val1"},
			Distribution: &metricdata.Distribution{
				Count:                 5,
				Sum:                   25.5,
				SumOfSquaredDeviation: 45.2,
				BucketOptions:         bucketOpts(2, 4, 8),
				Buckets:               buckets(1, 0, 3, 1),
			},
		},
	}, {
		name: "Equal when count only is set",
		want: Value{
			Tags: map[string]string{"key1": "val1"},
			Distribution: &metricdata.Distribution{
				Count: 5,
			},
			VerifyDistributionCountOnly: true,
		},
	}, {
		name: "Not equal when count only is not set",
		want: Value{
			Tags: map[string]string{"key1": "val1"},
			Distribution: &metricdata.Distribution{
				Count: 5,
			},
		},
		notEqual: true,
	}, {
		name: "Missing summary",
		want: Value{
			Tags: map[string]string{"key1": "val1"},
			Distribution: &metricdata.Distribution{
				BucketOptions: bucketOpts(2, 4, 8),
				Buckets:       buckets(1, 0, 3, 1),
			},
		},
		notEqual: true,
	}, {
		name: "Wrong bucket splits",
		want: Value{
			Tags: map[string]string{"key1": "val1"},
			Distribution: &metricdata.Distribution{
				Count:                 5,
				Sum:                   25.5,
				SumOfSquaredDeviation: 45.2,
				BucketOptions:         bucketOpts(1, 2, 3, 5, 8),
				Buckets:               buckets(1, 0, 3, 1),
			},
		},
		notEqual: true,
	}, {
		name: "Wrong bucket values",
		want: Value{
			Tags: map[string]string{"key1": "val1"},
			Distribution: &metricdata.Distribution{
				Count:                 5,
				Sum:                   25.5,
				SumOfSquaredDeviation: 45.2,
				BucketOptions:         bucketOpts(2, 4, 8),
				Buckets:               buckets(1, 3, 1, 0),
			},
		},
		notEqual: true,
	}, {
		name: "Wrong type",
		want: Value{
			Tags:    map[string]string{"key1": "val1"},
			Float64: &floatVal,
		},
		notEqual: true,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantMetric := Metric{
				Name:   "testing/distribution",
				Values: []Value{tc.want},
			}
			if diff := cmp.Diff(&wantMetric, &baseValue); (diff == "") == tc.notEqual {
				t.Errorf("Value.Equal() = %t failed (-want +base):\n%s", !tc.notEqual, diff)
			}
		})
	}
}

func TestMetricShortcuts(t *testing.T) {
	tags := map[string]string{
		"foo": "bar",
	}
	r := &resource.Resource{
		Type:   "test-resource",
		Labels: map[string]string{"foo1": "bar1"},
	}
	tests := []struct {
		name string
		want Metric
		got  metricdata.Metric
	}{{
		name: "IntMetric",
		want: IntMetric("test/int", 17, map[string]string{"key1": "val1", "key2": "val2"}),
		got: metricdata.Metric{
			Descriptor: metricdata.Descriptor{
				Name:      "test/int",
				LabelKeys: []metricdata.LabelKey{{Key: "key1"}, {Key: "key2"}},
			},
			TimeSeries: []*metricdata.TimeSeries{{
				LabelValues: []metricdata.LabelValue{{Value: "val1", Present: true}, {Value: "val2", Present: true}},
				Points:      []metricdata.Point{metricdata.NewInt64Point(time.Now(), 17)},
			}},
		},
	}, {
		name: "FloatMetric",
		want: FloatMetric("test/float", 0.17, map[string]string{"key1": "val1", "key2": "val2"}),
		got: metricdata.Metric{
			Descriptor: metricdata.Descriptor{
				Name:      "test/float",
				LabelKeys: []metricdata.LabelKey{{Key: "key1"}, {Key: "key2"}},
			},
			TimeSeries: []*metricdata.TimeSeries{{
				LabelValues: []metricdata.LabelValue{{Value: "val1", Present: true}, {Value: "val2", Present: true}},
				Points:      []metricdata.Point{metricdata.NewFloat64Point(time.Now(), 0.17)},
			}},
		},
	}, {
		name: "IntMetricWithResource",
		want: IntMetric("test/int", 18, tags).WithResource(r),
		got: metricdata.Metric{
			Descriptor: metricdata.Descriptor{
				Name:      "test/int",
				LabelKeys: []metricdata.LabelKey{{Key: "foo"}},
			},
			Resource: r,
			TimeSeries: []*metricdata.TimeSeries{{
				LabelValues: []metricdata.LabelValue{{Value: "bar", Present: true}},
				Points:      []metricdata.Point{metricdata.NewInt64Point(time.Now(), 18)},
			}},
		},
	}, {
		name: "FloatMetricWithResource",
		want: FloatMetric("test/float", 0.18, tags).WithResource(r),
		got: metricdata.Metric{
			Descriptor: metricdata.Descriptor{
				Name:      "test/float",
				LabelKeys: []metricdata.LabelKey{{Key: "foo"}},
			},
			Resource: r,
			TimeSeries: []*metricdata.TimeSeries{{
				LabelValues: []metricdata.LabelValue{{Value: "bar", Present: true}},
				Points:      []metricdata.Point{metricdata.NewFloat64Point(time.Now(), 0.18)},
			}},
		},
	}, {
		name: "DistributionCountOnlyMetricWithResource",
		want: DistributionCountOnlyMetric("test/distribution", 19, tags).WithResource(r),
		got: metricdata.Metric{
			Descriptor: metricdata.Descriptor{
				Name:      "test/distribution",
				LabelKeys: []metricdata.LabelKey{{Key: "foo"}},
			},
			Resource: r,
			TimeSeries: []*metricdata.TimeSeries{{
				LabelValues: []metricdata.LabelValue{{Value: "bar", Present: true}},
				Points: []metricdata.Point{metricdata.NewDistributionPoint(time.Now(), &metricdata.Distribution{
					Count:                 19,
					Sum:                   25.5,
					SumOfSquaredDeviation: 45.2,
					BucketOptions:         bucketOpts(2, 4, 8),
					Buckets:               buckets(1, 3, 1, 0),
				})},
			}},
		},
	}}

	for _, tc := range tests {
		if diff := cmp.Diff(tc.want, NewMetric(&tc.got)); diff != "" {
			t.Error("Metric.Equal() failed (-want +got):", diff)
		}
	}
}

func TestMetricFetch(t *testing.T) {
	count := stats.Int64("count", "Test count metric", stats.UnitBytes)
	tagKey := tag.MustNewKey("tag")
	countView := &view.View{
		Measure:     count,
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{tagKey},
	}

	m := GetMetric("count")
	if len(m) != 0 {
		t.Errorf("Unexpected number of found metrics (%d): %+v", len(m), m)
	}

	view.Register(countView)
	t.Cleanup(func() { view.Unregister(countView) })

	ctx, err := tag.New(context.Background(), tag.Upsert(tagKey, "alpha"))
	if err != nil {
		t.Error("Unable to create context:", err)
	}
	stats.Record(ctx, count.M(5))
	stats.Record(ctx, count.M(3))

	AssertMetricExists(t, "count")
	AssertNoMetric(t, "other")

	ctx, err = tag.New(ctx, tag.Upsert(tagKey, "beta"))
	if err != nil {
		t.Error("Unable to create context:", err)
	}
	stats.Record(ctx, count.M(20))
	EnsureRecorded()

	m = GetMetric("count")
	if len(m) != 1 {
		t.Errorf("Unexpected number of found metrics (%d): %+v", len(m), m)
	}

	alphaValue, betaValue := int64(8), int64(20)

	want := Metric{
		Name: "count",
		Type: metricdata.TypeCumulativeInt64,
		Unit: metricdata.UnitBytes,
		Values: []Value{
			{Tags: map[string]string{"tag": "alpha"}, Int64: &alphaValue},
			{Tags: map[string]string{"tag": "beta"}, Int64: &betaValue},
		},
	}
	if diff := cmp.Diff(want, m[0]); diff != "" {
		t.Error("Incorrect received metrics (-want +got):", diff)
	}
}
