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
	"sort"
	"sync"
	"testing"
	"time"

	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	proto "google.golang.org/protobuf/proto"

	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/metrics/metricskey"
	"knative.dev/pkg/metrics/metricstest"
)

var (
	NamespaceTagKey = tag.MustNewKey(metricskey.LabelNamespaceName)
	ServiceTagKey   = tag.MustNewKey(metricskey.LabelServiceName)
	ConfigTagKey    = tag.MustNewKey(metricskey.LabelConfigurationName)
	RevisionTagKey  = tag.MustNewKey(metricskey.LabelRevisionName)
)

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
	prometheusPort := 19090
	configForBackend := func(backend metricsBackend) ExporterOptions {
		return ExporterOptions{
			Domain:         servingDomain,
			Component:      testComponent,
			PrometheusPort: prometheusPort,
			ConfigMap: map[string]string{
				BackendDestinationKey: string(backend),
				collectorAddressKey:   ocFake.address,
				reportingPeriodKey:    "1",
			},
		}
	}

	resources := []*resource.Resource{{
		Type: "revision",
		Labels: map[string]string{
			"project":  "p1",
			"revision": "r1",
		},
	}, {
		Type: "revision",
		Labels: map[string]string{
			"project":  "p1",
			"revision": "r2",
		},
	}}
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
		init     func(t *testing.T) error
		validate func(t *testing.T)
	}{{
		name: "Prometheus",
		init: func(t *testing.T) error {
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
			const want = `# HELP testComponent_global_export_counts Count of exports via standard OpenCensus view.
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
		init: func(t *testing.T) error {
			if err := ocFake.start(len(resources) + 1); err != nil {
				return err
			}
			t.Log("Created exporter at", ocFake.address)
			return UpdateExporter(context.Background(), configForBackend(openCensus), logtesting.TestLogger(t))
		},
		validate: func(t *testing.T) {
			metricstest.EnsureRecorded()
			records := []metricExtract{}
			// Each Resource has an independent thread invoking reportView; this
			// set avoids the race condition where we get two reports for the
			// same metric on the channel before we get any reports for another
			// metric.
			keys := map[string]struct{}{}
			timeout := time.After(5 * time.Second)
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
							metric := metricExtract{
								Name:   m.MetricDescriptor.Name,
								Labels: labels,
								Value:  m.Timeseries[0].Points[0].GetInt64Value(),
							}
							records = append(records, metric)
							keys[metric.Key()] = struct{}{}
						}
					}
					if len(keys) >= len(expected) {
						break loop
					}
				case <-timeout:
					t.Error("Timeout reading input")
					break loop
				}
			}

			if diff := cmp.Diff(expected, records, sortMetrics()); diff != "" {
				t.Errorf("Unexpected OpenCensus exports (-want +got):\n%s", diff)
			}
		},
	}}

	for _, c := range harnesses {
		t.Run(c.name, func(t *testing.T) {
			ClearMetersForTest()
			if err := c.init(t); err != nil {
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
