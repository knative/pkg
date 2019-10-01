// Copyright 2018, OpenCensus Authors
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
	"net"
	"sync"
	"testing"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"

	"github.com/golang/protobuf/ptypes/empty"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	googlemetricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

func TestStatsAndMetricsEquivalence(t *testing.T) {
	_, _, stop := createFakeServer(t)
	defer stop()

	startTime := time.Unix(1000, 0)
	startTimePb := &timestamp.Timestamp{Seconds: 1000}
	mLatencyMs := stats.Float64("latency", "The latency for various methods", "ms")
	v := &view.View{
		Name:        "ocagent.io/latency",
		Description: "The latency of the various methods",
		Aggregation: view.Count(),
		Measure:     mLatencyMs,
	}
	metricDescriptor := &metricspb.MetricDescriptor{
		Name:        "ocagent.io/latency",
		Description: "The latency of the various methods",
		Unit:        "ms",
		Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
	}

	// Generate some view.Data and metrics.
	var vdl []*view.Data
	var metricPbs []*metricspb.Metric
	for i := 0; i < 100; i++ {
		vd := &view.Data{
			Start: startTime,
			End:   startTime.Add(time.Duration(1+i) * time.Second),
			View:  v,
			Rows: []*view.Row{
				{
					Data: &view.CountData{Value: int64(4 * (i + 2))},
				},
			},
		}
		metricPb := &metricspb.Metric{
			MetricDescriptor: metricDescriptor,
			Timeseries: []*metricspb.TimeSeries{
				{
					StartTimestamp: startTimePb,
					Points: []*metricspb.Point{
						{
							Timestamp: &timestamp.Timestamp{Seconds: int64(1001 + i)},
							Value:     &metricspb.Point_Int64Value{Int64Value: int64(4 * (i + 2))},
						},
					},
				},
			},
		}
		vdl = append(vdl, vd)
		metricPbs = append(metricPbs, metricPb)
	}

	// Now perform some exporting.
	for i, vd := range vdl {
		se := &statsExporter{
			o: Options{ProjectID: "equivalence", MapResource: defaultMapResource},
		}

		ctx := context.Background()
		sMD, err := se.viewToCreateMetricDescriptorRequest(ctx, vd.View)
		if err != nil {
			t.Errorf("#%d: Stats.viewToMetricDescriptor: %v", i, err)
		}
		pMD, err := se.protoMetricDescriptorToCreateMetricDescriptorRequest(ctx, metricPbs[i], nil)
		if err != nil {
			t.Errorf("#%d: Stats.protoMetricDescriptorToMetricDescriptor: %v", i, err)
		}
		if diff := cmpMDReq(pMD, sMD); diff != "" {
			t.Fatalf("MetricDescriptor Mismatch -FromMetrics +FromStats: %s", diff)
		}

		vdl := []*view.Data{vd}
		sctreql := se.makeReq(vdl, maxTimeSeriesPerUpload)
		tsl, _ := se.protoMetricToTimeSeries(ctx, nil, nil, metricPbs[i], nil)
		pctreql := se.combineTimeSeriesToCreateTimeSeriesRequest(tsl)
		if diff := cmpTSReqs(pctreql, sctreql); diff != "" {
			t.Fatalf("TimeSeries Mismatch -FromMetrics +FromStats: %s", diff)
		}
	}
}

// This test creates and uses a "Stackdriver backend" which receives
// CreateTimeSeriesRequest and CreateMetricDescriptor requests
// that the Stackdriver Metrics Proto client then sends to, as it would
// send to Google Stackdriver backends.
//
// This test ensures that the final responses sent by direct stats(view.Data) exporting
// are exactly equal to those from view.Data-->OpenCensus-Proto.Metrics exporting.
func TestEquivalenceStatsVsMetricsUploads(t *testing.T) {
	server, addr, doneFn := createFakeServer(t)
	defer doneFn()

	// Now create a gRPC connection to the agent.
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to make a gRPC connection to the agent: %v", err)
	}
	defer conn.Close()

	// Finally create the OpenCensus stats exporter
	exporterOptions := Options{
		ProjectID:               "equivalence",
		MonitoringClientOptions: []option.ClientOption{option.WithGRPCConn(conn)},

		// Setting this time delay threshold to a very large value
		// so that batching is performed deterministically and flushing is
		// fully controlled by us.
		BundleDelayThreshold: 2 * time.Hour,
		MapResource:          defaultMapResource,
	}
	se, err := newStatsExporter(exporterOptions)
	if err != nil {
		t.Fatalf("Failed to create the statsExporter: %v", err)
	}

	startTime := time.Unix(1000, 0)
	startTimePb := &timestamp.Timestamp{Seconds: 1000}
	mLatencyMs := stats.Float64("latency", "The latency for various methods", "ms")
	mConnections := stats.Int64("connections", "The count of various connections at a point in time", "1")
	mTimeMs := stats.Float64("time", "Counts time in milliseconds", "ms")

	// Generate the view.Data.
	var vdl []*view.Data
	for i := 0; i < 10; i++ {
		vdl = append(vdl,
			&view.Data{
				Start: startTime,
				End:   startTime.Add(time.Duration(1+i) * time.Second),
				View: &view.View{
					Name:        "ocagent.io/calls",
					Description: "The number of the various calls",
					Aggregation: view.Count(),
					Measure:     mLatencyMs,
				},
				Rows: []*view.Row{
					{
						Data: &view.CountData{Value: int64(4 * (i + 2))},
					},
				},
			},
			&view.Data{
				Start: startTime,
				End:   startTime.Add(time.Duration(2+i) * time.Second),
				View: &view.View{
					Name:        "ocagent.io/latency",
					Description: "The latency of the various methods",
					Aggregation: view.Distribution(100, 500, 1000, 2000, 4000, 8000, 16000),
					Measure:     mLatencyMs,
				},
				Rows: []*view.Row{
					{
						Data: &view.DistributionData{
							Count:          1,
							Min:            100,
							Max:            500,
							Mean:           125.9,
							CountPerBucket: []int64{0, 1, 0, 0, 0, 0, 0},
						},
					},
				},
			},
			&view.Data{
				Start: startTime,
				End:   startTime.Add(time.Duration(3+i) * time.Second),
				View: &view.View{
					Name:        "ocagent.io/connections",
					Description: "The count of various connections instantaneously",
					Aggregation: view.LastValue(),
					Measure:     mConnections,
				},
				Rows: []*view.Row{
					{Data: &view.LastValueData{Value: 99}},
				},
			},
			&view.Data{
				Start: startTime,
				End:   startTime.Add(time.Duration(1+i) * time.Second),
				View: &view.View{
					Name:        "ocagent.io/uptime",
					Description: "The total uptime at any instance",
					Aggregation: view.Sum(),
					Measure:     mTimeMs,
				},
				Rows: []*view.Row{
					{Data: &view.SumData{Value: 199903.97}},
				},
			})
	}

	for _, vd := range vdl {
		// Export the view.Data to the Stackdriver backend.
		se.ExportView(vd)
	}
	se.Flush()

	// Examining the stackdriver metrics that are available.
	var stackdriverTimeSeriesFromStats []*monitoringpb.CreateTimeSeriesRequest
	server.forEachStackdriverTimeSeries(func(sdt *monitoringpb.CreateTimeSeriesRequest) {
		stackdriverTimeSeriesFromStats = append(stackdriverTimeSeriesFromStats, sdt)
	})
	var stackdriverMetricDescriptorsFromStats []*monitoringpb.CreateMetricDescriptorRequest
	server.forEachStackdriverMetricDescriptor(func(sdmd *monitoringpb.CreateMetricDescriptorRequest) {
		stackdriverMetricDescriptorsFromStats = append(stackdriverMetricDescriptorsFromStats, sdmd)
	})

	// Reset the stackdriverTimeSeries to enable fresh collection
	// and then comparison with the results from metrics uploads.
	server.resetStackdriverTimeSeries()
	server.resetStackdriverMetricDescriptors()

	// Generate the proto Metrics.
	var metricPbs []*metricspb.Metric
	for i := 0; i < 10; i++ {
		metricPbs = append(metricPbs,
			&metricspb.Metric{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "ocagent.io/calls",
					Description: "The number of the various calls",
					Unit:        "1",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_INT64,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: startTimePb,
						Points: []*metricspb.Point{
							{
								Timestamp: &timestamp.Timestamp{Seconds: int64(1001 + i)},
								Value:     &metricspb.Point_Int64Value{Int64Value: int64(4 * (i + 2))},
							},
						},
					},
				},
			},
			&metricspb.Metric{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "ocagent.io/latency",
					Description: "The latency of the various methods",
					Unit:        "ms",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: startTimePb,
						Points: []*metricspb.Point{
							{
								Timestamp: &timestamp.Timestamp{Seconds: int64(1002 + i)},
								Value: &metricspb.Point_DistributionValue{
									DistributionValue: &metricspb.DistributionValue{
										Count: 1,
										Sum:   125.9,
										BucketOptions: &metricspb.DistributionValue_BucketOptions{
											Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
												Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{Bounds: []float64{100, 500, 1000, 2000, 4000, 8000, 16000}},
											},
										},
										Buckets: []*metricspb.DistributionValue_Bucket{{Count: 0}, {Count: 1}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}},
									},
								},
							},
						},
					},
				},
			},
			&metricspb.Metric{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "ocagent.io/connections",
					Description: "The count of various connections instantaneously",
					Unit:        "1",
					Type:        metricspb.MetricDescriptor_GAUGE_INT64,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: startTimePb,
						Points: []*metricspb.Point{
							{
								Timestamp: &timestamp.Timestamp{Seconds: int64(1003 + i)},
								Value:     &metricspb.Point_Int64Value{Int64Value: 99},
							},
						},
					},
				},
			},
			&metricspb.Metric{
				MetricDescriptor: &metricspb.MetricDescriptor{
					Name:        "ocagent.io/uptime",
					Description: "The total uptime at any instance",
					Unit:        "ms",
					Type:        metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
				},
				Timeseries: []*metricspb.TimeSeries{
					{
						StartTimestamp: startTimePb,
						Points: []*metricspb.Point{
							{
								Timestamp: &timestamp.Timestamp{Seconds: int64(1001 + i)},
								Value:     &metricspb.Point_DoubleValue{DoubleValue: 199903.97},
							},
						},
					},
				},
			})
	}

	// Export the proto Metrics to the Stackdriver backend.
	se.ExportMetricsProto(context.Background(), nil, nil, metricPbs)
	se.Flush()

	var stackdriverTimeSeriesFromMetrics []*monitoringpb.CreateTimeSeriesRequest
	server.forEachStackdriverTimeSeries(func(sdt *monitoringpb.CreateTimeSeriesRequest) {
		stackdriverTimeSeriesFromMetrics = append(stackdriverTimeSeriesFromMetrics, sdt)
	})
	var stackdriverMetricDescriptorsFromMetrics []*monitoringpb.CreateMetricDescriptorRequest
	server.forEachStackdriverMetricDescriptor(func(sdmd *monitoringpb.CreateMetricDescriptorRequest) {
		stackdriverMetricDescriptorsFromMetrics = append(stackdriverMetricDescriptorsFromMetrics, sdmd)
	})

	// The results should be equal now
	if diff := cmpTSReqs(stackdriverTimeSeriesFromMetrics, stackdriverTimeSeriesFromStats); diff != "" {
		t.Fatalf("Unexpected CreateTimeSeriesRequests -FromMetrics +FromStats: %s", diff)
	}

	// Examining the metric descriptors too.
	if diff := cmpMDReqs(stackdriverMetricDescriptorsFromMetrics, stackdriverMetricDescriptorsFromStats); diff != "" {
		t.Fatalf("Unexpected CreateMetricDescriptorRequests -FromMetrics +FromStats: %s", diff)
	}
}

type fakeMetricsServer struct {
	mu                           sync.RWMutex
	stackdriverTimeSeries        []*monitoringpb.CreateTimeSeriesRequest
	stackdriverMetricDescriptors []*monitoringpb.CreateMetricDescriptorRequest
}

func createFakeServer(t *testing.T) (*fakeMetricsServer, string, func()) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to bind to an available address: %v", err)
	}
	server := new(fakeMetricsServer)
	srv := grpc.NewServer()
	monitoringpb.RegisterMetricServiceServer(srv, server)
	go func() {
		_ = srv.Serve(ln)
	}()
	stop := func() {
		srv.Stop()
		_ = ln.Close()
	}
	_, agentPortStr, _ := net.SplitHostPort(ln.Addr().String())
	return server, ":" + agentPortStr, stop
}

func (server *fakeMetricsServer) forEachStackdriverTimeSeries(fn func(sdt *monitoringpb.CreateTimeSeriesRequest)) {
	server.mu.RLock()
	defer server.mu.RUnlock()

	for _, sdt := range server.stackdriverTimeSeries {
		fn(sdt)
	}
}

func (server *fakeMetricsServer) forEachStackdriverMetricDescriptor(fn func(sdmd *monitoringpb.CreateMetricDescriptorRequest)) {
	server.mu.RLock()
	defer server.mu.RUnlock()

	for _, sdmd := range server.stackdriverMetricDescriptors {
		fn(sdmd)
	}
}

func (server *fakeMetricsServer) resetStackdriverTimeSeries() {
	server.mu.Lock()
	server.stackdriverTimeSeries = server.stackdriverTimeSeries[:0]
	server.mu.Unlock()
}

func (server *fakeMetricsServer) resetStackdriverMetricDescriptors() {
	server.mu.Lock()
	server.stackdriverMetricDescriptors = server.stackdriverMetricDescriptors[:0]
	server.mu.Unlock()
}

var _ monitoringpb.MetricServiceServer = (*fakeMetricsServer)(nil)

func (server *fakeMetricsServer) GetMetricDescriptor(ctx context.Context, req *monitoringpb.GetMetricDescriptorRequest) (*googlemetricpb.MetricDescriptor, error) {
	return new(googlemetricpb.MetricDescriptor), nil
}

func (server *fakeMetricsServer) CreateMetricDescriptor(ctx context.Context, req *monitoringpb.CreateMetricDescriptorRequest) (*googlemetricpb.MetricDescriptor, error) {
	server.mu.Lock()
	server.stackdriverMetricDescriptors = append(server.stackdriverMetricDescriptors, req)
	server.mu.Unlock()
	return req.MetricDescriptor, nil
}

func (server *fakeMetricsServer) CreateTimeSeries(ctx context.Context, req *monitoringpb.CreateTimeSeriesRequest) (*empty.Empty, error) {
	server.mu.Lock()
	server.stackdriverTimeSeries = append(server.stackdriverTimeSeries, req)
	server.mu.Unlock()
	return new(empty.Empty), nil
}

func (server *fakeMetricsServer) ListTimeSeries(ctx context.Context, req *monitoringpb.ListTimeSeriesRequest) (*monitoringpb.ListTimeSeriesResponse, error) {
	return new(monitoringpb.ListTimeSeriesResponse), nil
}

func (server *fakeMetricsServer) DeleteMetricDescriptor(ctx context.Context, req *monitoringpb.DeleteMetricDescriptorRequest) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

func (server *fakeMetricsServer) ListMetricDescriptors(ctx context.Context, req *monitoringpb.ListMetricDescriptorsRequest) (*monitoringpb.ListMetricDescriptorsResponse, error) {
	return new(monitoringpb.ListMetricDescriptorsResponse), nil
}

func (server *fakeMetricsServer) GetMonitoredResourceDescriptor(ctx context.Context, req *monitoringpb.GetMonitoredResourceDescriptorRequest) (*monitoredrespb.MonitoredResourceDescriptor, error) {
	return new(monitoredrespb.MonitoredResourceDescriptor), nil
}

func (server *fakeMetricsServer) ListMonitoredResourceDescriptors(ctx context.Context, req *monitoringpb.ListMonitoredResourceDescriptorsRequest) (*monitoringpb.ListMonitoredResourceDescriptorsResponse, error) {
	return new(monitoringpb.ListMonitoredResourceDescriptorsResponse), nil
}
