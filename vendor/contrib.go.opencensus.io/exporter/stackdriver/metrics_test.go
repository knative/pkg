// Copyright 2019, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stackdriver

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/timestamp"

	distributionpb "google.golang.org/genproto/googleapis/api/distribution"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	googlemetricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/resource"
	"go.opencensus.io/trace"
)

var se = &statsExporter{
	o: Options{ProjectID: "foo"},
}

func TestMetricResourceToMonitoringResource(t *testing.T) {
	tests := []struct {
		in   *resource.Resource
		want *monitoredrespb.MonitoredResource
	}{
		{in: nil, want: &monitoredrespb.MonitoredResource{Type: "global"}},
		{in: &resource.Resource{}, want: &monitoredrespb.MonitoredResource{Type: "global"}},
		{
			in: &resource.Resource{
				Type: "foo",
			},
			want: &monitoredrespb.MonitoredResource{
				Type: "foo",
			},
		},
		{
			in: &resource.Resource{
				Type:   "foo",
				Labels: map[string]string{},
			},
			want: &monitoredrespb.MonitoredResource{
				Type:   "foo",
				Labels: map[string]string{},
			},
		},
		{
			in: &resource.Resource{
				Type:   "foo",
				Labels: map[string]string{"a": "A"},
			},
			want: &monitoredrespb.MonitoredResource{
				Type:   "foo",
				Labels: map[string]string{"a": "A"},
			},
		},
	}

	for i, tt := range tests {
		got := se.metricRscToMpbRsc(tt.in)
		if diff := cmpResource(got, tt.want); diff != "" {
			t.Fatalf("Test %d failed. Unexpected Resource -got +want: %s", i, diff)
		}
	}
}

func TestMetricToCreateTimeSeriesRequest(t *testing.T) {
	startTimestamp := &timestamp.Timestamp{
		Seconds: 1543160298,
		Nanos:   100000090,
	}
	startTime := time.Unix(1543160298, 100000090)
	endTimestamp := &timestamp.Timestamp{
		Seconds: 1543160298,
		Nanos:   100000997,
	}

	// TODO:[rghetia] add test for built-in metrics.
	tests := []struct {
		in      *metricdata.Metric
		want    []*monitoringpb.CreateTimeSeriesRequest
		wantErr string
	}{
		{
			in: &metricdata.Metric{
				Descriptor: metricdata.Descriptor{
					Name:        "with_metric_descriptor",
					Description: "This is a test",
					Unit:        metricdata.UnitBytes,
					Type:        metricdata.TypeGaugeDistribution,
				},
				Resource: nil,
				TimeSeries: []*metricdata.TimeSeries{
					{
						StartTime: timestampToTime(startTimestamp),
						Points: []metricdata.Point{
							{
								Time: timestampToTime(endTimestamp),
								Value: &metricdata.Distribution{
									Count:                 1,
									Sum:                   11.9,
									SumOfSquaredDeviation: 0,
									Buckets: []metricdata.Bucket{
										{
											Count:    1,
											Exemplar: &metricdata.Exemplar{Value: 11.9, Timestamp: startTime, Attachments: map[string]interface{}{"key": "value"}},
										},
										{}, {}, {},
									},
									BucketOptions: &metricdata.BucketOptions{
										Bounds: []float64{10, 20, 30, 40},
									},
								},
							},
						},
					},
				},
			},
			want: []*monitoringpb.CreateTimeSeriesRequest{
				{
					Name: "projects/foo",
					TimeSeries: []*monitoringpb.TimeSeries{
						{
							Metric: &googlemetricpb.Metric{
								Type:   "custom.googleapis.com/opencensus/with_metric_descriptor",
								Labels: map[string]string{},
							},
							Resource: &monitoredrespb.MonitoredResource{
								Type: "global",
							},
							Points: []*monitoringpb.Point{
								{
									Interval: &monitoringpb.TimeInterval{
										EndTime: endTimestamp,
									},
									Value: &monitoringpb.TypedValue{
										Value: &monitoringpb.TypedValue_DistributionValue{
											DistributionValue: &distributionpb.Distribution{
												Count:                 1,
												Mean:                  11.9,
												SumOfSquaredDeviation: 0,
												BucketCounts:          []int64{0, 1, 0, 0, 0},
												BucketOptions: &distributionpb.Distribution_BucketOptions{
													Options: &distributionpb.Distribution_BucketOptions_ExplicitBuckets{
														ExplicitBuckets: &distributionpb.Distribution_BucketOptions_Explicit{
															Bounds: []float64{0, 10, 20, 30, 40},
														},
													},
												},
												Exemplars: []*distributionpb.Distribution_Exemplar{
													{
														Value:     11.9,
														Timestamp: startTimestamp,
														Attachments: []*any.Any{
															{
																TypeUrl: exemplarAttachmentTypeString,
																Value:   []byte("value"),
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			in: &metricdata.Metric{
				Descriptor: metricdata.Descriptor{
					Name:        "with_metric_descriptor",
					Description: "This is a test",
					Unit:        metricdata.UnitBytes,
					Type:        metricdata.TypeCumulativeDistribution,
				},
				Resource: nil,
				TimeSeries: []*metricdata.TimeSeries{
					{
						StartTime: timestampToTime(startTimestamp),
						Points: []metricdata.Point{
							{
								Time: timestampToTime(endTimestamp),
								Value: &metricdata.Distribution{
									Count:                 1,
									Sum:                   11.9,
									SumOfSquaredDeviation: 0,
									Buckets: []metricdata.Bucket{
										{Count: 1}, {}, {}, {},
									},
									BucketOptions: &metricdata.BucketOptions{
										Bounds: []float64{10, 20, 30, 40},
									},
								},
							},
						},
					},
				},
			},
			want: []*monitoringpb.CreateTimeSeriesRequest{
				{
					Name: "projects/foo",
					TimeSeries: []*monitoringpb.TimeSeries{
						{
							Metric: &googlemetricpb.Metric{
								Type:   "custom.googleapis.com/opencensus/with_metric_descriptor",
								Labels: map[string]string{},
							},
							Resource: &monitoredrespb.MonitoredResource{
								Type: "global",
							},
							Points: []*monitoringpb.Point{
								{
									Interval: &monitoringpb.TimeInterval{
										StartTime: startTimestamp,
										EndTime:   endTimestamp,
									},
									Value: &monitoringpb.TypedValue{
										Value: &monitoringpb.TypedValue_DistributionValue{
											DistributionValue: &distributionpb.Distribution{
												Count:                 1,
												Mean:                  11.9,
												SumOfSquaredDeviation: 0,
												BucketCounts:          []int64{0, 1, 0, 0, 0},
												BucketOptions: &distributionpb.Distribution_BucketOptions{
													Options: &distributionpb.Distribution_BucketOptions_ExplicitBuckets{
														ExplicitBuckets: &distributionpb.Distribution_BucketOptions_Explicit{
															Bounds: []float64{0, 10, 20, 30, 40},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		tsl, err := se.metricToMpbTs(context.Background(), tt.in)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("#%d: unmatched error. Got\n\t%v\nWant\n\t%v", i, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		got := se.combineTimeSeriesToCreateTimeSeriesRequest(tsl)
		// Our saving grace is serialization equality since some
		// unexported fields could be present in the various values.
		if diff := cmpTSReqs(got, tt.want); diff != "" {
			t.Fatalf("Test %d failed. Unexpected CreateTimeSeriesRequests -got +want: %s", i, diff)
		}
	}
}

func TestMetricDescriptorToMonitoringMetricDescriptor(t *testing.T) {
	tests := []struct {
		in      *metricdata.Metric
		want    *googlemetricpb.MetricDescriptor
		wantErr string
	}{
		{in: nil, wantErr: "non-nil metric"},
		{
			in: &metricdata.Metric{},
			want: &googlemetricpb.MetricDescriptor{
				Name:        "projects/foo/metricDescriptors/custom.googleapis.com/opencensus",
				Type:        "custom.googleapis.com/opencensus",
				Labels:      []*labelpb.LabelDescriptor{},
				DisplayName: "OpenCensus",
				MetricKind:  googlemetricpb.MetricDescriptor_GAUGE,
				ValueType:   googlemetricpb.MetricDescriptor_INT64,
			},
		},
		{
			in: &metricdata.Metric{
				Descriptor: metricdata.Descriptor{
					Name:        "with_metric_descriptor",
					Description: "This is with metric descriptor",
					Unit:        metricdata.UnitBytes,
					Type:        metricdata.TypeCumulativeInt64,
				},
			},
			want: &googlemetricpb.MetricDescriptor{
				Name:        "projects/foo/metricDescriptors/custom.googleapis.com/opencensus/with_metric_descriptor",
				Type:        "custom.googleapis.com/opencensus/with_metric_descriptor",
				Labels:      []*labelpb.LabelDescriptor{},
				DisplayName: "OpenCensus/with_metric_descriptor",
				Description: "This is with metric descriptor",
				Unit:        "By",
				MetricKind:  googlemetricpb.MetricDescriptor_CUMULATIVE,
				ValueType:   googlemetricpb.MetricDescriptor_INT64,
			},
		},
	}

	for i, tt := range tests {
		got, err := se.metricToMpbMetricDescriptor(tt.in)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("#%d: \nGot %v\nWanted error substring %q", i, err, tt.wantErr)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}

		// Our saving grace is serialization equality since some
		// unexported fields could be present in the various values.
		if diff := cmpMD(got, tt.want); diff != "" {
			t.Fatalf("Test %d failed. Unexpected MetricDescriptor -got +want: %s", i, diff)
		}
	}
}

func TestMetricTypeToMonitoringMetricKind(t *testing.T) {
	tests := []struct {
		in            metricdata.Type
		wantKind      googlemetricpb.MetricDescriptor_MetricKind
		wantValueType googlemetricpb.MetricDescriptor_ValueType
		wantErr       string
	}{
		{
			in:            metricdata.TypeCumulativeInt64,
			wantKind:      googlemetricpb.MetricDescriptor_CUMULATIVE,
			wantValueType: googlemetricpb.MetricDescriptor_INT64,
		},
		{
			in:            metricdata.TypeCumulativeFloat64,
			wantKind:      googlemetricpb.MetricDescriptor_CUMULATIVE,
			wantValueType: googlemetricpb.MetricDescriptor_DOUBLE,
		},
		{
			in:            metricdata.TypeGaugeInt64,
			wantKind:      googlemetricpb.MetricDescriptor_GAUGE,
			wantValueType: googlemetricpb.MetricDescriptor_INT64,
		},
		{
			in:            metricdata.TypeGaugeFloat64,
			wantKind:      googlemetricpb.MetricDescriptor_GAUGE,
			wantValueType: googlemetricpb.MetricDescriptor_DOUBLE,
		},
		{
			in:            metricdata.TypeCumulativeDistribution,
			wantKind:      googlemetricpb.MetricDescriptor_CUMULATIVE,
			wantValueType: googlemetricpb.MetricDescriptor_DISTRIBUTION,
		},
		{
			in:            metricdata.TypeGaugeDistribution,
			wantKind:      googlemetricpb.MetricDescriptor_GAUGE,
			wantValueType: googlemetricpb.MetricDescriptor_DISTRIBUTION,
		},
		{
			in:            metricdata.TypeSummary,
			wantKind:      googlemetricpb.MetricDescriptor_METRIC_KIND_UNSPECIFIED,
			wantValueType: googlemetricpb.MetricDescriptor_VALUE_TYPE_UNSPECIFIED,
		},
	}

	for i, tt := range tests {
		md := &metricdata.Metric{
			Descriptor: metricdata.Descriptor{
				Name:        "with_metric_descriptor",
				Description: "This is with metric descriptor",
				Unit:        metricdata.UnitBytes,
				Type:        tt.in,
			},
		}

		got, err := se.metricToMpbMetricDescriptor(md)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("#%d: \nGot %v\nWanted error substring %q", i, err, tt.wantErr)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: Unexpected error: %v", i, err)
			continue
		}

		if got.MetricKind != tt.wantKind {
			t.Errorf("got %d, want %d\n", got.MetricKind, tt.wantKind)
		}
		if got.ValueType != tt.wantValueType {
			t.Errorf("got %d, want %d\n", got.ValueType, tt.wantValueType)
		}
	}
}

func TestMetricsToMonitoringMetrics_fromProtoPoint(t *testing.T) {
	startTimestamp := &timestamp.Timestamp{
		Seconds: 1543160298,
		Nanos:   100000090,
	}
	startTime := time.Unix(1543160298, 100000090)
	endTimestamp := &timestamp.Timestamp{
		Seconds: 1543160298,
		Nanos:   100000997,
	}

	traceID := trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 4, 8, 16, 32, 64, 128}
	spanID := trace.SpanID{1, 2, 4, 8, 16, 32, 64, 128}
	spanCtx := trace.SpanContext{
		TraceID:      traceID,
		SpanID:       spanID,
		TraceOptions: 1,
	}
	wantSpanCtxBytes, _ := proto.Marshal(&monitoringpb.SpanContext{SpanName: fmt.Sprintf("projects/foo/traces/%s/spans/%s", traceID.String(), spanID.String())})

	tests := []struct {
		in      *metricdata.Point
		want    *monitoringpb.Point
		wantErr string
	}{
		{
			in: &metricdata.Point{
				Time: timestampToTime(endTimestamp),
				Value: &metricdata.Distribution{
					Count:                 1,
					Sum:                   11.9,
					SumOfSquaredDeviation: 0,
					Buckets: []metricdata.Bucket{
						{},
						{
							Count:    1,
							Exemplar: &metricdata.Exemplar{Value: 11.9, Timestamp: startTime, Attachments: map[string]interface{}{"SpanContext": spanCtx}}},
						{},
						{},
						{},
					},
					BucketOptions: &metricdata.BucketOptions{
						Bounds: []float64{0, 10, 20, 30, 40},
					},
				},
			},
			want: &monitoringpb.Point{
				Interval: &monitoringpb.TimeInterval{
					StartTime: startTimestamp,
					EndTime:   endTimestamp,
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_DistributionValue{
						DistributionValue: &distributionpb.Distribution{
							Count:                 1,
							Mean:                  11.9,
							SumOfSquaredDeviation: 0,
							BucketCounts:          []int64{0, 1, 0, 0, 0},
							BucketOptions: &distributionpb.Distribution_BucketOptions{
								Options: &distributionpb.Distribution_BucketOptions_ExplicitBuckets{
									ExplicitBuckets: &distributionpb.Distribution_BucketOptions_Explicit{
										Bounds: []float64{0, 10, 20, 30, 40},
									},
								},
							},
							Exemplars: []*distributionpb.Distribution_Exemplar{
								{
									Value:     11.9,
									Timestamp: startTimestamp,
									Attachments: []*any.Any{
										{
											TypeUrl: exemplarAttachmentTypeSpanCtx,
											Value:   wantSpanCtxBytes,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			in: &metricdata.Point{
				Time:  timestampToTime(endTimestamp),
				Value: float64(50.0),
			},
			want: &monitoringpb.Point{
				Interval: &monitoringpb.TimeInterval{
					StartTime: startTimestamp,
					EndTime:   endTimestamp,
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_DoubleValue{DoubleValue: 50},
				},
			},
		},
		{
			in: &metricdata.Point{
				Time:  timestampToTime(endTimestamp),
				Value: int64(17),
			},
			want: &monitoringpb.Point{
				Interval: &monitoringpb.TimeInterval{
					StartTime: startTimestamp,
					EndTime:   endTimestamp,
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_Int64Value{Int64Value: 17},
				},
			},
		},
	}

	for i, tt := range tests {
		mpt, err := metricPointToMpbPoint(startTimestamp, tt.in, "foo")
		if tt.wantErr != "" {
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		// Our saving grace is serialization equality since some
		// unexported fields could be present in the various values.
		if diff := cmpPoint(mpt, tt.want); diff != "" {
			t.Fatalf("Test %d failed. Unexpected Point -got +want: %s", i, diff)
		}
	}
}
