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
	"testing"

	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/metrics/metricskey"
	//_ "knative.dev/pkg/metrics/testing"
)

var (
	r = resource.Resource{Labels: map[string]string{"foo": "bar"}}
)

func TestRegisterResourceView(t *testing.T) {
	meter := meterForResource(&r)

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

func TestMeterForResource(t *testing.T) {
	meter := meterForResource(&r)
	if meter == nil {
		t.Error("Should succeed getting meter, instead got nil")
	}
	meterAgain := meterForResource(&r)
	if meterAgain == nil {
		t.Error("Should succeed getting meter, instead got nil")
	}

	if meterAgain != meter {
		t.Error("Meter for the same resource should not be recreated")
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

// Begin table tests for exporters
func TestMetricsExport(t *testing.T) {
	ocFake := openCensusFake{address: "localhost:12345"}
	configForBackend := func(backend metricsBackend) ExporterOptions {
		return ExporterOptions{
			Domain:         servingDomain,
			Component:      testComponent,
			PrometheusPort: 9090,
			ConfigMap: map[string]string{
				BackendDestinationKey: string(backend),
				CollectorAddressKey:   ocFake.address,
			},
		}
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
			records := []ocmetrics.ExportMetricsServiceRequest{}
			for record := range ocFake.published {
				if len(record.Metrics) > 0 {
					ocFake.srv.Stop()
				}
				records = append(records, record)
			}
			expected := []struct {
				name   string
				value  int
				labels map[string]string
			}{
				{"testing/value", 1, map[string]string{"project": "p1", "revision": "r1"}},
				{"testing/value", 2, map[string]string{"project": "p1", "revision": "r2"}},
			}
			// TODO(evankanderson): Finish the comparison and remove the flake!
			if len(records) <= len(expected) {
				t.Errorf("Expected %d records, got %d:\n%+v", len(expected), len(records), records)
			}
		},
	}}
	resources := []*resource.Resource{
		&resource.Resource{
			Type: "revision",
			Labels: map[string]string{
				"project":  "p1",
				"revision": "r1",
			},
		},
		&resource.Resource{
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

	for _, c := range harnesses {
		t.Run(c.name, func(t *testing.T) {
			err := c.init()
			if err != nil {
				t.Fatalf("unable to init: %+v", err)
			}

			view.Register(globalCounter)
			err = RegisterResourceView(gaugeView, resourceCounter)
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
			FlushExporter()
			c.validate(t)

			UnregisterResourceView(gaugeView)
		})
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
		if in.Resource == nil {
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
