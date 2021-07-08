/*
Copyright 2018 The Knative Authors

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
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"
	. "knative.dev/pkg/logging/testing"
)

// TODO UTs should move to eventing and serving, as appropriate.
// 	See https://github.com/knative/pkg/issues/608

const (
	testNS            = "test"
	testService       = "test-service"
	testRevision      = "test-revision"
	testConfiguration = "test-configuration"
	testContainer     = "test-container"
	testPod           = "test-pod"
)

func TestMain(m *testing.M) {
	resetCurPromSrv()
	os.Exit(m.Run())
}

func TestMetricsExporter(t *testing.T) {
	tests := []struct {
		name          string
		config        *metricsConfig
		expectSuccess bool
	}{{
		name: "unsupportedBackend",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          testComponent,
			backendDestination: "unsupported",
		},
		expectSuccess: false,
	}, {
		name: "noneBackend",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          testComponent,
			backendDestination: none,
		},
		expectSuccess: true,
	}, {
		name: "validConfig",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          testComponent,
			backendDestination: prometheus,
		},
		expectSuccess: true,
	}, {
		name: "validConfigWithDashInName",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          "test-component",
			backendDestination: prometheus,
		},
		expectSuccess: true,
	}, {
		name: "fullValidConfig",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    60 * time.Second,
		},
		expectSuccess: true,
	}}

	// getStackdriverSecretFunc = fakeGetStackdriverSecret
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := newMetricsExporter(test.config, TestLogger(t))

			succeeded := err == nil
			if test.expectSuccess != succeeded {
				t.Errorf("Unexpected test result. Expected success? [%v]. Error: [%v]", test.expectSuccess, err)
			}
		})
	}
}

func TestInterleavedExporters(t *testing.T) {
	// Disabling this test as it fails intermittently.
	// Refer to https://github.com/knative/pkg/issues/406
	t.Skip()

	// First create a opencensus exporter
	_, _, err := newMetricsExporter(&metricsConfig{
		domain:             servingDomain,
		component:          testComponent,
		backendDestination: openCensus,
	}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}

	// Expect no prometheus server
	time.Sleep(200 * time.Millisecond)
	srv := getCurPromSrv()
	if srv != nil {
		t.Error("expected no server for opencensus exporter")
	}

	// Then switch to prometheus exporter
	_, _, err = newMetricsExporter(&metricsConfig{
		domain:             servingDomain,
		component:          testComponent,
		backendDestination: prometheus,
		prometheusPort:     9090}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	expectPromSrv(t, ":9090")
}

func TestFlushExporter(t *testing.T) {
	// No exporter - no action should be taken
	UpdateExporter(context.Background(), ExporterOptions{
		Domain:    "test",
		Component: "test",
		ConfigMap: map[string]string{
			BackendDestinationKey: string(none),
		},
	}, TestLogger(t))

	if want, got := false, FlushExporter(); got != want {
		t.Errorf("Expected %v, got %v.", want, got)
	}

	// Prometheus exporter shouldn't do anything because
	// it doesn't implement Flush()
	c := &metricsConfig{
		domain:             servingDomain,
		component:          testComponent,
		reportingPeriod:    1 * time.Minute,
		backendDestination: prometheus,
	}
	e, f, err := newMetricsExporter(c, TestLogger(t))
	if err != nil {
		t.Error("Expected no error. got", err)
	} else {
		setCurMetricsExporter(e)
		if want, got := false, FlushExporter(); got != want {
			t.Errorf("Expected %v, got %v.", want, got)
		}
		if f == nil { // This is tested more extensively in resource_view_test.go
			t.Error("Expected non-nil factory, got nil.")
		}
	}

	c = &metricsConfig{
		domain:             servingDomain,
		component:          testComponent,
		backendDestination: openCensus,
		reportingPeriod:    1 * time.Minute,
	}

	e, f, err = newMetricsExporter(c, TestLogger(t))
	if err != nil {
		t.Error("Expected no error. got", err)
	} else {
		setCurMetricsExporter(e)
		if want, got := true, FlushExporter(); got != want {
			t.Errorf("Expected %v, got %v when calling FlushExporter().", want, got)
		}
		if f == nil { // This is tested more extensively in resource_view_test.go
			t.Error("Expected non-nil factory, got nil.")
		}
	}
}
