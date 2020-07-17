/*
Copyright 2018 The Knative Authors.
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
	"path"
	"testing"
	"time"

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
	// Set gcpMetadataFunc and newStackdriverExporterFunc for testing
	gcpMetadataFunc = fakeGcpMetadataFunc
	newStackdriverExporterFunc = newFakeExporter
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
			backendDestination: stackdriver,
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID: "testProj",
			},
		},
		expectSuccess: true,
	}, {
		name: "validConfigWithDashInName",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          "test-component",
			backendDestination: stackdriver,
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID: "testProj",
			},
		},
		expectSuccess: true,
	}, {
		name: "stackdriverConfigOnly",
		config: &metricsConfig{
			backendDestination: stackdriver,
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID:   "project",
				GCPLocation: "us-west1",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		name: "fullValidConfig",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                stackdriver,
			reportingPeriod:                   60 * time.Second,
			isStackdriverBackend:              true,
			stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
			stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID:   "project",
				GCPLocation: "us-west1",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		name: "prometheusBackendWithStackdriverConfig",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    5 * time.Second,
			prometheusPort:     defaultPrometheusPort,
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID:   "project",
				GCPLocation: "us-west1",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		// GCP specifies a list of valid locations, check the exporter can be created
		// even if an invalid location is passed in.
		name: "invalidStackdriverGcpLocation",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                stackdriver,
			reportingPeriod:                   60 * time.Second,
			isStackdriverBackend:              true,
			stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
			stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID:   "project",
				GCPLocation: "narnia",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		name: "missingProjectID",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                stackdriver,
			reportingPeriod:                   60 * time.Second,
			isStackdriverBackend:              true,
			stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
			stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
			stackdriverClientConfig: StackdriverClientConfig{
				GCPLocation: "narnia",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		name: "partialStackdriverConfig",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                stackdriver,
			reportingPeriod:                   60 * time.Second,
			isStackdriverBackend:              true,
			stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
			stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID: "project",
			},
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

func TestInterlevedExporters(t *testing.T) {
	// Disabling this test as it fails intermittently.
	// Refer to https://github.com/knative/pkg/issues/406
	t.Skip()

	// First create a stackdriver exporter
	_, _, err := newMetricsExporter(&metricsConfig{
		domain:             servingDomain,
		component:          testComponent,
		backendDestination: stackdriver,
		stackdriverClientConfig: StackdriverClientConfig{
			ProjectID: testProj,
		},
	}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	expectNoPromSrv(t)
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
	// Finally switch to stackdriver exporter
	_, _, err = newMetricsExporter(&metricsConfig{
		domain:             servingDomain,
		component:          testComponent,
		backendDestination: stackdriver,
		stackdriverClientConfig: StackdriverClientConfig{
			ProjectID: testProj,
		},
	}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
}

func TestFlushExporter(t *testing.T) {
	// No exporter - no action should be taken
	setCurMetricsConfig(nil)
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
		t.Errorf("Expected no error. got %v", err)
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
		domain:                            servingDomain,
		component:                         testComponent,
		backendDestination:                stackdriver,
		isStackdriverBackend:              true,
		reportingPeriod:                   1 * time.Minute,
		stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
		stackdriverCustomMetricTypePrefix: path.Join(defaultCustomMetricSubDomain, testComponent),
		stackdriverClientConfig: StackdriverClientConfig{
			ProjectID: testProj,
		},
	}

	e, f, err = newMetricsExporter(c, TestLogger(t))
	if err != nil {
		t.Errorf("Expected no error. got %v", err)
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
