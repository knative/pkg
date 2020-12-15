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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	sd "contrib.go.opencensus.io/exporter/stackdriver"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	stackdriverpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
	proto "google.golang.org/protobuf/proto"

	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/metrics/metricskey"
	"knative.dev/pkg/metrics/metricstest"
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
		t.Fatal("RegisterResourceView =", err)
	}
	t.Cleanup(func() { UnregisterResourceView(&view) })

	viewToFind := defaultMeter.m.Find("testView")
	if viewToFind == nil || viewToFind.Name != "testView" {
		t.Error("Registered view should be found in default meter, instead got", viewToFind)
	}

	viewToFind = meter.Find("testView")
	if viewToFind == nil || viewToFind.Name != "testView" {
		t.Error("Registered view should be found in new meter, instead got", viewToFind)
	}
}

func TestOptionForResource(t *testing.T) {
	option, err1 := optionForResource(&r)
	if err1 != nil {
		t.Error("Should succeed getting option, instead got error", err1)
	}
	optionAgain, err2 := optionForResource(&r)
	if err2 != nil {
		t.Error("Should succeed getting option, instead got error", err2)
	}

	if fmt.Sprintf("%v", optionAgain) != fmt.Sprintf("%v", option) {
		t.Errorf("Option for the same resource should not be recreated, instead got %v and %v", optionAgain, option)
	}
}

type testExporter struct {
	view.Exporter
	id string
}

func TestSetFactory(t *testing.T) {
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
		t.Error("Should succeed getting option, instead got error", err)
	}

	// Now get the exporter and verify the id
	me := meterExporterForResource(&resource123)
	e := me.e.(*testExporter)
	if e.id != "123" {
		t.Error("Expect id to be 123, instead got", e.id)
	}

	resource456 := r
	resource456.Labels["id"] = "456"
	// Create the new meter and apply the factory
	_, err = optionForResource(&resource456)
	if err != nil {
		t.Error("Should succeed getting option, instead got error", err)
	}

	me = meterExporterForResource(&resource456)
	e = me.e.(*testExporter)
	if e.id != "456" {
		t.Error("Expect id to be 456, instead got", e.id)
	}
}

func TestAllMetersExpiration(t *testing.T) {
	allMeters.clock = clock.Clock(clock.NewFakeClock(time.Now()))
	var fakeClock *clock.FakeClock = allMeters.clock.(*clock.FakeClock)
	ClearMetersForTest() // t+0m

	// Add resource123
	resource123 := r
	resource123.Labels["id"] = "123"
	_, err := optionForResource(&resource123)
	if err != nil {
		t.Error("Should succeed getting option, instead got error ", err)
	}
	// (123=0m, 456=Inf)

	// Bump time to make resource123's expiry offset from resource456
	fakeClock.Step(90 * time.Second) // t+1.5m
	// (123=0m, 456=Inf)

	// Add 456
	resource456 := r
	resource456.Labels["id"] = "456"
	_, err = optionForResource(&resource456)
	if err != nil {
		t.Error("Should succeed getting option, instead got error ", err)
	}
	allMeters.lock.Lock()
	if len(allMeters.meters) != 3 {
		t.Errorf("len(allMeters)=%d, want: 3", len(allMeters.meters))
	}
	allMeters.lock.Unlock()
	// (123=1.5m, 456=0m)

	// Warm up the older entry
	fakeClock.Step(90 * time.Second) //t+3m
	// (123=4.5m, 456=3m)

	// Refresh the first entry
	_, err = optionForResource(&resource123)
	if err != nil {
		t.Error("Should succeed getting option, instead got error ", err)
	}
	// (123=0, 456=1.5m)

	// Expire the second entry
	fakeClock.Step(9 * time.Minute) // t+12m
	time.Sleep(time.Second)         // Wait a second on the wallclock, so that the cleanup thread has time to finish a loop
	allMeters.lock.Lock()
	if len(allMeters.meters) != 2 {
		t.Errorf("len(allMeters)=%d, want: 2", len(allMeters.meters))
	}
	allMeters.lock.Unlock()
	// (123=9m, 456=10.5m)
	// non-expiring defaultMeter was just tested

	// Add resource789
	resource789 := r
	resource789.Labels["id"] = "789"
	_, err = optionForResource(&resource789)
	if err != nil {
		t.Error("Should succeed getting option, instead got error ", err)
	}
	// (123=9m, 456=evicted, 789=0m)
}

func TestResourceAsString(t *testing.T) {
	r1 := &resource.Resource{Type: "foobar", Labels: map[string]string{"k1": "v1", "k3": "v3", "k2": "v2"}}
	r2 := &resource.Resource{Type: "foobar", Labels: map[string]string{"k2": "v2", "k3": "v3", "k1": "v1"}}
	r3 := &resource.Resource{Type: "foobar", Labels: map[string]string{"k1": "v1", "k2": "v2", "k4": "v4"}}

	// Test 5 time since the iteration could be random.
	for i := 0; i < 5; i++ {

		if s1, s2 := resourceToKey(r1), resourceToKey(r2); s1 != s2 {
			t.Errorf("Expect same resources, but got %q and %q", s1, s2)
		}
	}

	if s1, s3 := resourceToKey(r1), resourceToKey(r3); s1 == s3 {
		t.Error("Expect different resources, but got the same", s1)
	}
}

func BenchmarkResourceToKey(b *testing.B) {
	for _, count := range []int{0, 1, 5, 10} {
		labels := make(map[string]string, count)
		for i := 0; i < count; i++ {
			labels[fmt.Sprint("key", i)] = fmt.Sprint("value", i)
		}
		r := &resource.Resource{Type: "foobar", Labels: labels}

		b.Run(fmt.Sprintf("%d-labels", count), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				resourceToKey(r)
			}
		})
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
	newStackdriverExporterFunc = func(o sd.Options) (view.Exporter, error) {
		o.MonitoringClientOptions = append(o.MonitoringClientOptions, option.WithGRPCConn(conn))
		return newOpencensusSDExporter(o)
	}
	// File: must exist, be json of credentialsFile, and type must be a jwtConfig or oauth2Config
	tmp, err := ioutil.TempFile("", "metrics-sd-test")
	if err != nil {
		return err
	}
	defer tmp.Close()
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
	TestOverrideBundleCount = 1
	t.Cleanup(func() { TestOverrideBundleCount = 0 })
	ocFake := openCensusFake{address: "localhost:12345"}
	sdFake := stackDriverFake{}
	prometheusPort := 19090
	configForBackend := func(backend metricsBackend) ExporterOptions {
		return ExporterOptions{
			Domain:         servingDomain,
			Component:      testComponent,
			PrometheusPort: prometheusPort,
			ConfigMap: map[string]string{
				BackendDestinationKey:               string(backend),
				collectorAddressKey:                 ocFake.address,
				allowStackdriverCustomMetricsKey:    "true",
				stackdriverCustomMetricSubDomainKey: servingDomain,
				reportingPeriodKey:                  "1",
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
		{"knative.dev/serving/testComponent/global_export_counts", map[string]string{}, 2},
		{"knative.dev/serving/testComponent/resource_global_export_count", map[string]string{}, 2},
		{"knative.dev/serving/testComponent/testing/value", map[string]string{"project": "p1", "revision": "r1"}, 0},
		{"knative.dev/serving/testComponent/testing/value", map[string]string{"project": "p1", "revision": "r2"}, 1},
	}

	harnesses := []struct {
		name     string
		init     func() error
		validate func(t *testing.T)
	}{{
		name: "Prometheus",
		init: func() error {
			if err := UpdateExporter(context.Background(), configForBackend(prometheus), logtesting.TestLogger(t)); err != nil {
				return err
			}
			// Wait for the webserver to actually start serving metrics
			return wait.PollImmediate(10*time.Millisecond, 10*time.Second, func() (bool, error) {
				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", prometheusPort))
				return err == nil && resp.StatusCode == http.StatusOK, nil
			})
		},
		validate: func(t *testing.T) {
			metricstest.EnsureRecorded()
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", prometheusPort))
			if err != nil {
				t.Fatalf("failed to fetch prometheus metrics: %+v", err)
			}
			defer resp.Body.Close()
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
			if err := ocFake.start(len(resources) + 1); err != nil {
				return err
			}
			t.Log("Created exporter at", ocFake.address)
			return UpdateExporter(context.Background(), configForBackend(openCensus), logtesting.TestLogger(t))
		},
		validate: func(t *testing.T) {
			t.Skip("Skipped because of excessive flakiness, see: https://github.com/knative/pkg/issues/1672")

			// We unregister the views because this is one of two ways to flush
			// the internal aggregation buffers; the other is to have the
			// internal reporting period duration tick, which is at least
			// [new duration] in the future.
			view.Unregister(globalCounter)
			UnregisterResourceView(gaugeView, resourceCounter)

			records := []metricExtract{}
		loop:
			for {
				select {
				case record := <-ocFake.published:
					if record == nil {
						continue loop
					}
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
					if len(records) >= len(expected) {
						break loop
					}
				case <-time.After(4 * time.Second):
					t.Error("Timeout reading input")
					break loop
				}
			}

			if diff := cmp.Diff(expected, records, sortMetrics()); diff != "" {
				t.Errorf("Unexpected OpenCensus exports (-want +got):\n%s", diff)
			}
		},
	}, {
		name: "Stackdriver",
		init: func() error {
			if err := initSdFake(&sdFake); err != nil {
				return err
			}
			return UpdateExporter(context.Background(), configForBackend(stackdriver), logtesting.TestLogger(t))
		},
		validate: func(t *testing.T) {
			records := []metricExtract{}
			for record := range sdFake.published {
				for _, ts := range record.TimeSeries {
					name := ts.Metric.Type[len("custom.googleapis.com/"):]
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
			ClearMetersForTest()
			sdFake.t = t
			if err := c.init(); err != nil {
				t.Fatalf("unable to init: %+v", err)
			}

			view.Register(globalCounter)
			if err := RegisterResourceView(gaugeView, resourceCounter); err != nil {
				t.Fatal("Unable to register views:", err)
			}
			t.Cleanup(func() {
				view.Unregister(globalCounter)
				UnregisterResourceView(gaugeView, resourceCounter)
			})

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
	TestOverrideBundleCount = 1
	t.Cleanup(func() { TestOverrideBundleCount = 0 })
	eo := ExporterOptions{
		Domain:    servingDomain,
		Component: "autoscaler",
		ConfigMap: map[string]string{
			BackendDestinationKey:   string(stackdriver),
			reportingPeriodKey:      "1",
			stackdriverProjectIDKey: "foobar",
		},
	}

	label1 := map[string]string{
		"cluster_name":       "test-cluster",
		"configuration_name": "config",
		"location":           "test-location",
		"namespace_name":     "ns",
		"project_id":         "foobar",
		"revision_name":      "revision",
		"service_name":       "service",
	}
	label2 := map[string]string{
		"cluster_name":       "test-cluster",
		"configuration_name": "config2",
		"location":           "test-location",
		"namespace_name":     "ns2",
		"project_id":         "foobar",
		"revision_name":      "revision2",
		"service_name":       "service2",
	}
	batchLabels := map[string]string{
		"namespace_name":     "ns2",
		"configuration_name": "config2",
		"revision_name":      "revision2",
		"service_name":       "service2",
	}
	harness := []struct {
		name               string
		allowCustomMetrics string
		expected           []metricExtract
	}{{
		name:               "Allow custom metrics",
		allowCustomMetrics: "true",
		expected: []metricExtract{
			{
				"knative.dev/serving/autoscaler/actual_pods",
				label1,
				1,
			},
			{
				"knative.dev/serving/autoscaler/desired_pods",
				label2,
				2,
			},
			{
				"custom.googleapis.com/knative.dev/autoscaler/not_ready_pods",
				batchLabels,
				3,
			},
		},
	}, {
		name:               "Don't allow custom metrics",
		allowCustomMetrics: "false",
		expected: []metricExtract{
			{
				"knative.dev/serving/autoscaler/actual_pods",
				label1,
				1,
			},
			{
				"knative.dev/serving/autoscaler/desired_pods",
				label2,
				2,
			},
		},
	}}

	for _, tc := range harness {
		t.Run(tc.name, func(t *testing.T) {
			eo.ConfigMap[allowStackdriverCustomMetricsKey] = tc.allowCustomMetrics
			// Change the cluster name to reinitialize the exporter and pick up a new port.
			eo.ConfigMap[stackdriverClusterNameKey] = tc.name
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
			notReadyPodCountM := stats.Int64(
				"not_ready_pods",
				"Number of pods that are not ready",
				stats.UnitDimensionless)
			customView := &view.View{
				Description: "non-knative-revision metric per KnativeRevisionMetrics",
				Measure:     notReadyPodCountM,
				Aggregation: view.LastValue(),
			}

			sdFake := stackDriverFake{t: t}
			if err := initSdFake(&sdFake); err != nil {
				t.Error("Init stackdriver failed", err)
			}
			if err := UpdateExporter(context.Background(), eo, logtesting.TestLogger(t)); err != nil {
				t.Error("UpdateExporter failed", err)
			}

			if err := RegisterResourceView(desiredPodsCountView, actualPodsCountView, customView); err != nil {
				t.Fatalf("unable to register view: %+v", err)
			}
			t.Cleanup(func() {
				UnregisterResourceView(desiredPodsCountView, actualPodsCountView, customView)
			})

			ctx, err := tag.New(context.Background(), tag.Upsert(NamespaceTagKey, "ns"),
				tag.Upsert(ServiceTagKey, "service"),
				tag.Upsert(ConfigTagKey, "config"),
				tag.Upsert(RevisionTagKey, "revision"))
			if err != nil {
				t.Fatal("Unable to create tags", err)
			}
			Record(ctx, actualPodCountM.M(int64(1)))

			r := resource.Resource{
				Type:   "testing",
				Labels: batchLabels,
			}
			RecordBatch(
				metricskey.WithResource(context.Background(), r),
				desiredPodCountM.M(int64(2)),
				notReadyPodCountM.M(int64(3)))

			records := []metricExtract{}
		loop:
			for {
				select {
				case record := <-sdFake.published:
					for _, ts := range record.TimeSeries {
						extracted := metricExtract{
							Name:   ts.Metric.Type,
							Labels: ts.Resource.Labels,
							Value:  ts.Points[0].Value.GetInt64Value(),
						}
						// Override 'cluster-name' label to reset to a fixed value
						if extracted.Labels["cluster_name"] != "" {
							extracted.Labels["cluster_name"] = "test-cluster"
						}
						records = append(records, extracted)
						if strings.HasPrefix(ts.Metric.Type, "knative.dev/") {
							if diff := cmp.Diff(ts.Resource.Type, metricskey.ResourceTypeKnativeRevision); diff != "" {
								t.Errorf("Incorrect resource type for %q: (-want +got):\n%s", ts.Metric.Type, diff)
							}
						}
					}
					if len(records) >= len(tc.expected) {
						// There's no way to synchronize on the internal timer used
						// by metricsexport.IntervalReader, so shut down the
						// exporter after the first report cycle.
						FlushExporter()
						sdFake.srv.GracefulStop()
						break loop
					}
				case <-time.After(4 * time.Second):
					t.Error("Timeout reading records from Stackdriver")
					break loop
				}
			}
			if diff := cmp.Diff(tc.expected, records, sortMetrics()); diff != "" {
				t.Errorf("Unexpected stackdriver knative exports (-want +got):\n%s", diff)
			}
		})
	}
}

type openCensusFake struct {
	ocmetrics.UnimplementedMetricsServiceServer
	address   string
	srv       *grpc.Server
	exports   sync.WaitGroup
	wg        sync.WaitGroup
	published chan *ocmetrics.ExportMetricsServiceRequest
}

func (oc *openCensusFake) start(expectedStreams int) error {
	ln, err := net.Listen("tcp", oc.address)
	if err != nil {
		return err
	}
	oc.published = make(chan *ocmetrics.ExportMetricsServiceRequest, 100)
	oc.srv = grpc.NewServer()
	ocmetrics.RegisterMetricsServiceServer(oc.srv, oc)
	// Run the server in the background.
	oc.wg.Add(1)
	go func() {
		oc.srv.Serve(ln)
		oc.wg.Done()
		oc.wg.Wait()
		close(oc.published)
	}()
	oc.exports.Add(expectedStreams)
	go oc.stop()
	return nil
}

func (oc *openCensusFake) stop() {
	oc.exports.Wait()
	oc.srv.Stop()
}

func (oc *openCensusFake) Export(stream ocmetrics.MetricsService_ExportServer) error {
	var streamResource *ocresource.Resource
	oc.wg.Add(1)
	defer oc.wg.Done()
	metricSeen := false
	for {
		in, err := stream.Recv()
		if errors.Is(err, io.EOF) {
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
			oc.published <- proto.Clone(in).(*ocmetrics.ExportMetricsServiceRequest)
			if !metricSeen {
				oc.exports.Done()
				metricSeen = true
			}
		}
	}
}

type stackDriverFake struct {
	stackdriverpb.UnimplementedMetricServiceServer
	address   string
	srv       *grpc.Server
	t         *testing.T
	published chan *stackdriverpb.CreateTimeSeriesRequest
}

func (sd *stackDriverFake) start() error {
	sd.published = make(chan *stackdriverpb.CreateTimeSeriesRequest, 100)
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	sd.address = ln.Addr().String()
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
	sd.published <- req
	return &emptypb.Empty{}, nil
}

func (sd *stackDriverFake) CreateMetricDescriptor(ctx context.Context, req *stackdriverpb.CreateMetricDescriptorRequest) (*metricpb.MetricDescriptor, error) {
	return req.MetricDescriptor, nil
}
