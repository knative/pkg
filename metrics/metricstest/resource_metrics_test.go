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

// Package metricstest simplifies some of the common boilerplate around testing
// metrics exports. It should work with or without the code in metrics, but this
// code particularly knows how to deal with metrics which are exported for
// multiple Resources in the same process.
package metricstest

import (
	"testing"
	"time"

	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/resource"
)

func TestMetric_Equal(t *testing.T) {
	type fields struct {
		Name           string
		Unit           metricdata.Unit
		Type           metricdata.Type
		Resource       *resource.Resource
		Values         []Value
		VerifyMetadata bool
		VerifyResource bool
	}
	pointValues := []int64{127, -1}
	baseValue := &metricdata.Metric{
		Descriptor: metricdata.Descriptor{
			Name:      "testing/metric",
			Unit:      metricdata.UnitMilliseconds,
			Type:      metricdata.TypeGaugeInt64,
			LabelKeys: []metricdata.LabelKey{{"key1", ""}, {"key2", ""}},
		},
		Resource: &resource.Resource{
			Type:   "testObject",
			Labels: map[string]string{"rl1": "aaa", "rl2": "bbb"},
		},
		TimeSeries: []*metricdata.TimeSeries{
			{
				LabelValues: []metricdata.LabelValue{{"val1", true}, {"val2", true}},
				Points:      []metricdata.Point{metricdata.NewInt64Point(time.Now(), pointValues[0])},
			},
			{
				LabelValues: []metricdata.LabelValue{{"val1", true}, {"val3", true}},
				Points:      []metricdata.Point{metricdata.NewInt64Point(time.Now(), pointValues[1])},
			},
		},
	}

	tests := []struct {
		name     string
		want     Metric
		notEqual bool
	}{{
		name: "Minimal test",
		want: Metric{
			Name: "testing/metric",
		},
	}, {
		name: "Test resource not equal",
		want: Metric{
			Name:           "testing/metric",
			VerifyResource: true,
		},
		notEqual: true,
	}, {
		name: "Check resource",
		want: Metric{
			Name: "testing/metric",
			Resource: &resource.Resource{
				Type:   "testObject",
				Labels: map[string]string{"rl2": "bbb", "rl1": "aaa"},
			},
		},
	}, {
		name: "Mismatched resource",
		want: Metric{
			Name: "testing/metric",
			Resource: &resource.Resource{
				Type:   "testObject",
				Labels: map[string]string{"rl1": "aaa"},
			},
		},
		notEqual: true,
	}, {
		name: "Test value match",
		want: Metric{
			Name: "testing/metric",
			Values: []Value{
				{
					Tags:  map[string]string{"key1": "val1", "key2": "val2"},
					Int64: &pointValues[0]},
				{
					Tags:  map[string]string{"key1": "val1", "key2": "val3"},
					Int64: &pointValues[1],
				},
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.want.Equal(baseValue) == tt.notEqual {
				t.Errorf("Metric.Equal() = %v, want %v", baseValue, tt.want)
			}
		})
	}
}
