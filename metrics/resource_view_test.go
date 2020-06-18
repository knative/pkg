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

package metrics

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
	"testing"

	"contrib.go.opencensus.io/exporter/stackdriver"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"google.golang.org/api/option"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	stackdriverpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/metrics/metricskey"
	//_ "knative.dev/pkg/metrics/testing"
)

var (
	r               = resource.Resource{Labels: map[string]string{"foo": "bar"}}
	NamespaceTagKey = tag.MustNewKey(metricskey.LabelNamespaceName)
	ServiceTagKey   = tag.MustNewKey(metricskey.LabelServiceName)
	ConfigTagKey    = tag.MustNewKey(metricskey.LabelConfigurationName)
	RevisionTagKey  = tag.MustNewKey(metricskey.LabelRevisionName)
)

func TestRegisterResourceView(t *testing.T) {
	meter := meterExporterForResource(&r).m

	m := stats.Int64("testView_sum", "", stats.UnitDimensionless)
	view := view.View{Name: "testView", Measure: m, Aggregation: view.Sum()}
	err := RegisterResourceView(&view)

	if err != nil {
		t.Errorf("RegisterResourceView with error %v.", err)
	}

	viewToFind := defaultMeter.m.Find("testView")
	if viewToFind == nil || viewToFind.Name != "testView" {
		t.Errorf("Registered view should be found in default meter, instead got %v", viewToFind)
	}

	viewToFind = meter.Find("testView")
	if viewToFind == nil || viewToFind.Name != "testView" {
		t.Errorf("Registered view should be found in new meter, instead got %v", viewToFind)
	}
}

func TestOptionForResource(t *testing.T) {
	option, err1 := optionForResource(&r)
	if err1 != nil {
		t.Errorf("Should succeed getting option, instead got error %v", err1)
	}
	optionAgain, err2 := optionForResource(&r)
	if err2 != nil {
		t.Errorf("Should succeed getting option, instead got error %v", err2)
	}

	if fmt.Sprintf("%v", optionAgain) != fmt.Sprintf("%v", option) {
		t.Errorf("Option for the same resource should not be recreated, instead got %v and %v", optionAgain, option)
	}
}

type testExporter struct {
	id string
}

func (fe *testExporter) ExportView(vd *view.Data) {}
func (fe *testExporter) Flush()                   {}
func TestSetFactor(t *testing.T) {
	fakeFactory := func(rr *resource.Resource) (view.Exporter, error) {
		if rr == nil {
			return &testExporter{}, nil
		}

		return &testExporter{id: rr.Labels["id"]}, nil
	}

	resource123 := r
	resource123.Labels["id"] = "123"

	setFactory(fakeFactory)
	// Create the new meter and apply the factory
	_, err := optionForResource(&resource123)
	if err != nil {
		t.Errorf("Should succeed getting option, instead got error %v", err)
	}

	// Now get the exporter and verify the id
	me := meterExporterForResource(&resource123)
	e := me.e.(*testExporter)
	if e.id != "123" {
		t.Errorf("Expect id to be 123, instead got %v", e.id)
	}

	resource456 := r
	resource456.Labels["id"] = "456"
	// Create the new meter and apply the factory
	_, err = optionForResource(&resource456)
	if err != nil {
		t.Errorf("Should succeed getting option, instead got error %v", err)
	}

	me = meterExporterForResource(&resource456)
	e = me.e.(*testExporter)
	if e.id != "456" {
		t.Errorf("Expect id to be 456, instead got %v", e.id)
	}
}

type metricExtract struct {
	Name   string
	Labels map[string]string
	Value  int64
}

func (m metricExtract) Key() string {
	return fmt.Sprintf("%s<%s>", m.Name, resource.EncodeLabels(m.Labels))
}

func (m metricExtract) String() string {
	return fmt.Sprintf("%s:%d", m.Key(), m.Value)
}

func initSdFake(sdFake *stackDriverFake) error {
	if err := sdFake.start(); err != nil {
		return err
	}
	conn, err := grpc.Dial(sdFake.address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	newStackdriverExporterFunc = func(o stackdriver.Options) (view.Exporter, error) {
		o.MonitoringClientOptions = append(o.MonitoringClientOptions, option.WithGRPCConn(conn))
		return newOpencensusSDExporter(o)
	}
	// File: must exist, be json of credentialsFile, and type must be a jwtConfig or oauth2Config
	tmp, err := ioutil.TempFile("", "metrics-sd-test")
	if err != nil {
		return err
	}
	credentialsContent := []byte(`{"type": "service_account"}`)
	if _, err := tmp.Write(credentialsContent); err != nil {
		return err
	}
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tmp.Name())
	return nil
}

func sortMetrics() cmp.Option {
	return cmp.Transformer("Sort", func(in []metricExtract) []string {
		out := make([]string, 0, len(in))
		seen := map[string]int{}
		for _, m := range in {
			// Keep only the newest report for a key
			key := m.Key()
			if seen[key] == 0 {
				out = append(out, m.String())
				seen[key] = len(out) // Store address+1 to avoid doubling first item.
			} else {
				out[seen[key]-1] = m.String()
			}
		}
		sort.Strings(out)
		return out
	})
}

// Begin table tests for exporters
func TestMetricsExport(t *testing.T) {
	ocFake := openCensusFake{address: "localhost:12345"}
	sdFake := stackDriverFake{address: "localhost:12346"}
	configForBackend := func(backend metricsBackend) ExporterOptions {
		return ExporterOptions{
			Domain:         servingDomain,
			Component:      testComponent,
			PrometheusPort: 9090,
			ConfigMap: map[string]string{
				BackendDestinationKey:            string(backend),
				CollectorAddressKey:              ocFake.address,
				AllowStackdriverCustomMetricsKey: "true",
				ReportingPeriodKey:               "1",
			},
		}
	}

	resources := []*resource.Resource{
		{
			Type: "revision",
			Labels: map[string]string{
				"project":  "p1",
				"revision": "r1",
			},
		},
		{
			Type: "revision",
			Labels: map[string]string{
				"project":  "p1",
				"revision": "r2",
			},
		},
	}
	gauge := stats.Int64("testing/value", "Stored value", stats.UnitDimensionless)
	counter := stats.Int64("export counts", "Times through the export", stats.UnitDimensionless)
	gaugeView := &view.View{
		Name:        "testing/value",
		Description: "Test value",
		Measure:     gauge,
		Aggregation: view.LastValue(),
	}
	resourceCounter := &view.View{
		Name:        "resource_global_export_count",
		Description: "Count of exports via RegisterResourceView.",
		Measure:     counter,
		Aggregation: view.Count(),
	}
	globalCounter := &view.View{
		Name:        "global_export_counts",
		Description: "Count of exports via standard OpenCensus view.",
		Measure:     counter,
		Aggregation: view.Count(),
	}

	expected := []metricExtract{
		{"global_export_counts", map[string]string{}, 2},
		{"resource_global_export_count", map[string]string{}, 2},
		{"testing/value", map[string]string{"project": "p1", "revision": "r1"}, 0},
		{"testing/value", map[string]string{"project": "p1", "revision": "r2"}, 1},
	}

	harnesses := []struct {
		name     string
		init     func() error
		validate func(t *testing.T)
	}{{
		name: "Prometheus",
		init: func() error {
			return UpdateExporter(configForBackend(Prometheus), logtesting.TestLogger(t))
		},
		validate: func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", 9090))
			if err != nil {
				t.Fatalf("failed to fetch prometheus metrics: %+v", err)
			}
			defer resp.Body.Close()
			t.Logf("TODO: Validate Prometheus")
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read prometheus response: %+v", err)
			}
			want := `# HELP testComponent_global_export_counts Count of exports via standard OpenCensus view.
# TYPE testComponent_global_export_counts counter
testComponent_global_export_counts 2
# HELP testComponent_resource_global_export_count Count of exports via RegisterResourceView.
# TYPE testComponent_resource_global_export_count counter
testComponent_resource_global_export_count 2
# HELP testComponent_testing_value Test value
# TYPE testComponent_testing_value gauge
testComponent_testing_value{project="p1",revision="r1"} 0
testComponent_testing_value{project="p1",revision="r2"} 1
`
			if diff := cmp.Diff(want, string(body)); diff != "" {
				t.Errorf("Unexpected prometheus output (-want +got):\n%s", diff)
			}
		},
	}, {
		name: "OpenCensus",
		init: func() error {
			if err := ocFake.start(); err != nil {
				return err
			}
			t.Logf("Created exporter at %s", ocFake.address)
			return UpdateExporter(configForBackend(OpenCensus), logtesting.TestLogger(t))
		},
		validate: func(t *testing.T) {
			// We unregister the views because this is one of two ways to flush
			// the internal aggregation buffers; the other is to have the
			// internal reporting period duration tick, which is at least
			// [new duration] in the future.
			view.Unregister(globalCounter)
			UnregisterResourceView(gaugeView, resourceCounter)
			FlushExporter()

			ocFake.srv.Stop() // Force close connections
			ocFake.srv.GracefulStop()
			records := []metricExtract{}
			for record := range ocFake.published {
				for _, m := range record.Metrics {
					if len(m.Timeseries) > 0 {
						labels := map[string]string{}
						if record.Resource != nil {
							labels = record.Resource.Labels
						}
						records = append(records, metricExtract{
							Name:   m.MetricDescriptor.Name,
							Labels: labels,
							Value:  m.Timeseries[0].Points[0].GetInt64Value(),
						})
					}
				}
			}

			if diff := cmp.Diff(expected, records, sortMetrics()); diff != "" {
				t.Errorf("Unexpected OpenCensus exports (-want +got):\n%s", diff)
			}
		},
	}, {
		name: "Stackdriver",
		init: func() error {
			var err error
			err = initSdFake(&sdFake)
			if err != nil {
				return err
			}
			return UpdateExporter(configForBackend(Stackdriver), logtesting.TestLogger(t))
		},
		validate: func(t *testing.T) {
			records := []metricExtract{}
			for record := range sdFake.published {
				for _, ts := range record.TimeSeries {
					name := ts.Metric.Type[len("custom.googleapis.com/knative.dev/testComponent/"):]
					records = append(records, metricExtract{
						Name:   name,
						Labels: ts.Resource.Labels,
						Value:  ts.Points[0].Value.GetInt64Value(),
					})
				}
				if len(records) >= 4 {
					// There's no way to synchronize on the internal timer used
					// by metricsexport.IntervalReader, so shut down the
					// exporter after the first report cycle.
					FlushExporter()
					sdFake.srv.GracefulStop()
				}
			}
			if diff := cmp.Diff(expected, records, sortMetrics()); diff != "" {
				t.Errorf("Unexpected Stackdriver exports (-want +got):\n%s", diff)
			}
		},
	}}

	for _, c := range harnesses {
		t.Run(c.name, func(t *testing.T) {
			sdFake.t = t
			err := c.init()
			if err != nil {
				t.Fatalf("unable to init: %+v", err)
			}

			view.Register(globalCounter)
			err = RegisterResourceView(gaugeView, resourceCounter)
			defer func() {
				view.Unregister(globalCounter)
				UnregisterResourceView(gaugeView, resourceCounter)
			}()
			if err != nil {
				t.Fatalf("unable to register view: %+v", err)
			}

			for i, r := range resources {
				ctx := context.Background()
				Record(ctx, counter.M(int64(1)))
				if r != nil {
					ctx = metricskey.WithResource(ctx, *r)
				}
				Record(ctx, gauge.M(int64(i)))
			}
			c.validate(t)
		})
	}
}

func TestStackDriverExports(t *testing.T) {
	sdFake := stackDriverFake{address: "localhost:12346"}
	eo := ExporterOptions{
		Domain:         servingDomain,
		Component:      "autoscaler",
		PrometheusPort: 9090,
		ConfigMap: map[string]string{
			BackendDestinationKey:            string(Stackdriver),
			AllowStackdriverCustomMetricsKey: "false",
			ReportingPeriodKey:               "1",
			StackdriverProjectIDKey:          "foobar",
		},
	}

	actualPodCountM := stats.Int64(
		"actual_pods",
		"Number of pods that are allocated currently",
		stats.UnitDimensionless)
	actualPodsCountView := &view.View{
		Description: "Number of pods that are allocated currently",
		Measure:     actualPodCountM,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{NamespaceTagKey, ServiceTagKey, ConfigTagKey, RevisionTagKey},
	}
	desiredPodCountM := stats.Int64(
		"desired_pods",
		"Number of pods that are desired",
		stats.UnitDimensionless)
	desiredPodsCountView := &view.View{
		Description: "Number of pods that are desired",
		Measure:     desiredPodCountM,
		Aggregation: view.LastValue(),
	}
	err := initSdFake(&sdFake)
	if err != nil {
		t.Errorf("Init stackdriver failed %s", err)
	}
	err = UpdateExporter(eo, logtesting.TestLogger(t))
	if err != nil {
		t.Errorf("UpdateExporter failed %s", err)
	}
	sdFake.t = t

	err = RegisterResourceView(desiredPodsCountView, actualPodsCountView)
	defer UnregisterResourceView(desiredPodsCountView, actualPodsCountView)

	ctx, err := tag.New(context.Background(), tag.Upsert(NamespaceTagKey, "ns"),
		tag.Upsert(ServiceTagKey, "service"),
		tag.Upsert(ConfigTagKey, "config"),
		tag.Upsert(RevisionTagKey, "revision"))
	if err != nil {
		t.Fatalf("Unable to create tags %s", err)
	}
	Record(ctx, actualPodCountM.M(int64(1)))

	r := resource.Resource{
		Type: "knative_revision",
		Labels: map[string]string{
			metricskey.LabelNamespaceName:     "ns2",
			metricskey.LabelServiceName:       "service2",
			metricskey.LabelConfigurationName: "config2",
			metricskey.LabelRevisionName:      "revision2",
		},
	}
	ctx = metricskey.WithResource(context.Background(), r)
	Record(ctx, desiredPodCountM.M(int64(2)))

	records := []metricExtract{}
	for record := range sdFake.published {
		for _, ts := range record.TimeSeries {
			records = append(records, metricExtract{
				Name:   ts.Metric.Type,
				Labels: ts.Resource.Labels,
				Value:  ts.Points[0].Value.GetInt64Value(),
			})
		}
		if len(records) >= 2 {
			// There's no way to synchronize on the internal timer used
			// by metricsexport.IntervalReader, so shut down the
			// exporter after the first report cycle.
			FlushExporter()
			sdFake.srv.GracefulStop()
		}
	}
	expectedRevisionResults := []metricExtract{
		{"knative.dev/serving/autoscaler/actual_pods", map[string]string{
			"cluster_name":       "test-cluster",
			"configuration_name": "config",
			"location":           "test-location",
			"namespace_name":     "ns",
			"project_id":         "foobar",
			"revision_name":      "revision",
			"service_name":       "service",
		},
			1,
		},
		{"knative.dev/serving/autoscaler/desired_pods", map[string]string{
			"cluster_name":       "test-cluster",
			"configuration_name": "config2",
			"location":           "test-location",
			"namespace_name":     "ns2",
			"project_id":         "foobar",
			"revision_name":      "revision2",
			"service_name":       "service2",
		},
			2,
		},
	}
	if diff := cmp.Diff(expectedRevisionResults, records, sortMetrics()); diff != "" {
		t.Errorf("Unexpected stackdriver knative exports (-want +got):\n%s", diff)
	}
}

type openCensusFake struct {
	address   string
	srv       *grpc.Server
	published chan ocmetrics.ExportMetricsServiceRequest
}

func (oc *openCensusFake) start() error {
	oc.published = make(chan ocmetrics.ExportMetricsServiceRequest, 100)
	ln, err := net.Listen("tcp", oc.address)
	if err != nil {
		return err
	}
	oc.srv = grpc.NewServer()
	ocmetrics.RegisterMetricsServiceServer(oc.srv, oc)
	// Run the server in the background.
	go func() {
		oc.srv.Serve(ln)
		close(oc.published)
	}()
	return nil
}

func (oc *openCensusFake) Export(stream ocmetrics.MetricsService_ExportServer) error {
	var streamResource *ocresource.Resource
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if in.Resource != nil {
			// The stream is stateful, keep track of the last Resource seen.
			streamResource = in.Resource
		}
		if len(in.Metrics) > 0 {
			if in.Resource == nil {
				in.Resource = streamResource
			}
			oc.published <- *in
		}
	}
}

type stackDriverFake struct {
	address   string
	srv       *grpc.Server
	t         *testing.T
	published chan stackdriverpb.CreateTimeSeriesRequest
}

func (sd *stackDriverFake) start() error {
	sd.published = make(chan stackdriverpb.CreateTimeSeriesRequest, 100)
	ln, err := net.Listen("tcp", sd.address)
	if err != nil {
		return err
	}
	sd.srv = grpc.NewServer()
	stackdriverpb.RegisterMetricServiceServer(sd.srv, sd)
	// Run the server in the background.
	go func() {
		sd.srv.Serve(ln)
		close(sd.published)
	}()
	return nil
}

func (sd *stackDriverFake) CreateTimeSeries(ctx context.Context, req *stackdriverpb.CreateTimeSeriesRequest) (*emptypb.Empty, error) {
	sd.published <- *req
	return nil, nil
}

func (sd *stackDriverFake) ListMonitoredResourceDescriptors(ctx context.Context, req *stackdriverpb.ListMonitoredResourceDescriptorsRequest) (*stackdriverpb.ListMonitoredResourceDescriptorsResponse, error) {
	sd.t.Fatalf("ListMonitoredResourceDescriptors")
	return nil, fmt.Errorf("Unimplemented")
}

func (sd *stackDriverFake) GetMonitoredResourceDescriptor(context.Context, *stackdriverpb.GetMonitoredResourceDescriptorRequest) (*monitoredrespb.MonitoredResourceDescriptor, error) {
	sd.t.Fatalf("GetMonitoredResourceDescriptor")
	return nil, fmt.Errorf("Unimplemented")
}
func (sd *stackDriverFake) ListMetricDescriptors(context.Context, *stackdriverpb.ListMetricDescriptorsRequest) (*stackdriverpb.ListMetricDescriptorsResponse, error) {
	sd.t.Fatalf("ListMetricDescriptors")
	return nil, fmt.Errorf("Unimplemented")
}
func (sd *stackDriverFake) GetMetricDescriptor(context.Context, *stackdriverpb.GetMetricDescriptorRequest) (*metricpb.MetricDescriptor, error) {
	sd.t.Fatalf("GetMetricDescriptor")
	return nil, fmt.Errorf("Unimplemented")
}
func (sd *stackDriverFake) CreateMetricDescriptor(ctx context.Context, req *stackdriverpb.CreateMetricDescriptorRequest) (*metricpb.MetricDescriptor, error) {
	resp := *req.MetricDescriptor
	return &resp, nil
}
func (sd *stackDriverFake) DeleteMetricDescriptor(context.Context, *stackdriverpb.DeleteMetricDescriptorRequest) (*emptypb.Empty, error) {
	sd.t.Fatalf("DeleteMetricDescriptor")
	return nil, fmt.Errorf("Unimplemented")
}
func (sd *stackDriverFake) ListTimeSeries(context.Context, *stackdriverpb.ListTimeSeriesRequest) (*stackdriverpb.ListTimeSeriesResponse, error) {
	sd.t.Fatalf("ListTimeSeries")
	return nil, fmt.Errorf("Unimplemented")
}
