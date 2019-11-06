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

package metrics

import (
	"path"
	"testing"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	. "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/metrics/metricskey"
)

// TODO UTs should move to eventing and serving, as appropriate.
// 	See https://github.com/knative/pkg/issues/608

var (
	testGcpMetadata = gcpMetadata{
		project:  "test-project",
		location: "test-location",
		cluster:  "test-cluster",
	}

	supportedServingMetricsTestCases = []struct {
		name       string
		domain     string
		component  string
		metricName string
	}{{
		name:       "activator metric",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
	}, {
		name:       "autoscaler metric",
		domain:     servingDomain,
		component:  "autoscaler",
		metricName: "desired_pods",
	}}

	supportedEventingBrokerMetricsTestCases = []struct {
		name       string
		domain     string
		component  string
		metricName string
	}{{
		name:       "broker metric",
		domain:     internalEventingDomain,
		component:  "broker",
		metricName: "event_count",
	}}

	supportedEventingTriggerMetricsTestCases = []struct {
		name       string
		domain     string
		component  string
		metricName string
	}{{
		name:       "trigger metric",
		domain:     internalEventingDomain,
		component:  "trigger",
		metricName: "event_count",
	}, {
		name:       "trigger metric",
		domain:     internalEventingDomain,
		component:  "trigger",
		metricName: "event_processing_latencies",
	}, {
		name:       "trigger metric",
		domain:     internalEventingDomain,
		component:  "trigger",
		metricName: "event_dispatch_latencies",
	}}

	supportedEventingSourceMetricsTestCases = []struct {
		name       string
		domain     string
		component  string
		metricName string
	}{{
		name:       "source metric",
		domain:     eventingDomain,
		component:  "source",
		metricName: "event_count",
	}}

	unsupportedMetricsTestCases = []struct {
		name       string
		domain     string
		component  string
		metricName string
	}{{
		name:       "unsupported domain",
		domain:     "unsupported",
		component:  "activator",
		metricName: "request_count",
	}, {
		name:       "unsupported component",
		domain:     servingDomain,
		component:  "unsupported",
		metricName: "request_count",
	}, {
		name:       "unsupported metric",
		domain:     servingDomain,
		component:  "activator",
		metricName: "unsupported",
	}, {
		name:       "unsupported component",
		domain:     internalEventingDomain,
		component:  "unsupported",
		metricName: "event_count",
	}, {
		name:       "unsupported metric",
		domain:     internalEventingDomain,
		component:  "broker",
		metricName: "unsupported",
	}}
)

func TestGetMonitoredResourceFunc_UseKnativeRevision(t *testing.T) {
	for _, testCase := range supportedServingMetricsTestCases {
		testView = &view.View{
			Description: "Test View",
			Measure:     stats.Int64(testCase.metricName, "Test Measure", stats.UnitNone),
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{},
		}
		mrf := getMonitoredResourceFunc(path.Join(testCase.domain, testCase.component), &testGcpMetadata)

		newTags, monitoredResource := mrf(testView, revisionTestTags)
		gotResType, labels := monitoredResource.MonitoredResource()
		wantedResType := "knative_revision"
		if gotResType != wantedResType {
			t.Fatalf("MonitoredResource=%v, want %v", gotResType, wantedResType)
		}
		got := getResourceLabelValue(metricskey.LabelRouteName, newTags)
		if got != testRoute {
			t.Errorf("expected new tag: %v, got: %v", routeKey, newTags)
		}
		got, ok := labels[metricskey.LabelNamespaceName]
		if !ok || got != testNS {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelNamespaceName, testNS, got)
		}
		got, ok = labels[metricskey.LabelConfigurationName]
		if !ok || got != metricskey.ValueUnknown {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelConfigurationName, metricskey.ValueUnknown, got)
		}
	}
}

func TestGetMonitoredResourceFunc_UseKnativeBroker(t *testing.T) {
	for _, testCase := range supportedEventingBrokerMetricsTestCases {
		testView = &view.View{
			Description: "Test View",
			Measure:     stats.Int64(testCase.metricName, "Test Measure", stats.UnitDimensionless),
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{},
		}
		mrf := getMonitoredResourceFunc(path.Join(testCase.domain, testCase.component), &testGcpMetadata)

		newTags, monitoredResource := mrf(testView, brokerTestTags)
		gotResType, labels := monitoredResource.MonitoredResource()
		wantedResType := "knative_broker"
		if gotResType != wantedResType {
			t.Fatalf("MonitoredResource=%v, want %v", gotResType, wantedResType)
		}
		got := getResourceLabelValue(metricskey.LabelEventType, newTags)
		if got != testEventType {
			t.Errorf("expected new tag: %v, got: %v", eventTypeKey, newTags)
		}
		got, ok := labels[metricskey.LabelNamespaceName]
		if !ok || got != testNS {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelNamespaceName, testNS, got)
		}
		got, ok = labels[metricskey.LabelBrokerName]
		if !ok || got != testBroker {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelBrokerName, testBroker, got)
		}
	}
}

func TestGetMonitoredResourceFunc_UseKnativeTrigger(t *testing.T) {
	for _, testCase := range supportedEventingTriggerMetricsTestCases {
		testView = &view.View{
			Description: "Test View",
			Measure:     stats.Int64(testCase.metricName, "Test Measure", stats.UnitDimensionless),
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{},
		}
		mrf := getMonitoredResourceFunc(path.Join(testCase.domain, testCase.component), &testGcpMetadata)

		newTags, monitoredResource := mrf(testView, triggerTestTags)
		gotResType, labels := monitoredResource.MonitoredResource()
		wantedResType := "knative_trigger"
		if gotResType != wantedResType {
			t.Fatalf("MonitoredResource=%v, want %v", gotResType, wantedResType)
		}
		got := getResourceLabelValue(metricskey.LabelFilterType, newTags)
		if got != testFilterType {
			t.Errorf("expected new tag: %v, got: %v", filterTypeKey, newTags)
		}
		got, ok := labels[metricskey.LabelNamespaceName]
		if !ok || got != testNS {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelNamespaceName, testNS, got)
		}
		got, ok = labels[metricskey.LabelBrokerName]
		if !ok || got != testBroker {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelBrokerName, testBroker, got)
		}
	}
}

func TestGetMonitoredResourceFunc_UseKnativeSource(t *testing.T) {
	for _, testCase := range supportedEventingSourceMetricsTestCases {
		testView = &view.View{
			Description: "Test View",
			Measure:     stats.Int64(testCase.metricName, "Test Measure", stats.UnitDimensionless),
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{},
		}
		mrf := getMonitoredResourceFunc(path.Join(testCase.domain, testCase.component), &testGcpMetadata)

		newTags, monitoredResource := mrf(testView, sourceTestTags)
		gotResType, labels := monitoredResource.MonitoredResource()
		wantedResType := "knative_source"
		if gotResType != wantedResType {
			t.Fatalf("MonitoredResource=%v, want %v", gotResType, wantedResType)
		}
		got := getResourceLabelValue(metricskey.LabelEventType, newTags)
		if got != testEventType {
			t.Errorf("expected new tag: %v, got: %v", eventTypeKey, newTags)
		}
		got, ok := labels[metricskey.LabelNamespaceName]
		if !ok || got != testNS {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelNamespaceName, testNS, got)
		}
		got, ok = labels[metricskey.LabelName]
		if !ok || got != testSource {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelName, testSource, got)
		}
		got, ok = labels[metricskey.LabelResourceGroup]
		if !ok || got != testSourceResourceGroup {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelResourceGroup, testSourceResourceGroup, got)
		}
	}
}

func TestGetMonitoredResourceFunc_UseGlobal(t *testing.T) {
	for _, testCase := range unsupportedMetricsTestCases {
		testView = &view.View{
			Description: "Test View",
			Measure:     stats.Int64(testCase.metricName, "Test Measure", stats.UnitNone),
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{},
		}
		mrf := getMonitoredResourceFunc(path.Join(testCase.domain, testCase.component), &testGcpMetadata)

		newTags, monitoredResource := mrf(testView, revisionTestTags)
		gotResType, labels := monitoredResource.MonitoredResource()
		wantedResType := "global"
		if gotResType != wantedResType {
			t.Fatalf("MonitoredResource=%v, want: %v", gotResType, wantedResType)
		}
		got := getResourceLabelValue(metricskey.LabelNamespaceName, newTags)
		if got != testNS {
			t.Errorf("expected new tag %v with value %v, got: %v", routeKey, testNS, newTags)
		}
		if len(labels) != 0 {
			t.Errorf("expected no label, got: %v", labels)
		}
	}
}

func TestGetMetricTypeFunc_UseKnativeDomain(t *testing.T) {
	for _, testCase := range supportedServingMetricsTestCases {
		testView = &view.View{
			Description: "Test View",
			Measure:     stats.Int64(testCase.metricName, "Test Measure", stats.UnitNone),
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{},
		}
		mtf := getMetricTypeFunc(
			path.Join(testCase.domain, testCase.component),
			path.Join(defaultCustomMetricSubDomain, testCase.component))

		gotMetricType := mtf(testView)
		wantedMetricType := path.Join(testCase.domain, testCase.component, testView.Measure.Name())
		if gotMetricType != wantedMetricType {
			t.Fatalf("getMetricType=%v, want %v", gotMetricType, wantedMetricType)
		}
	}
}

func TestGetgetMetricTypeFunc_UseCustomDomain(t *testing.T) {
	for _, testCase := range unsupportedMetricsTestCases {
		testView = &view.View{
			Description: "Test View",
			Measure:     stats.Int64(testCase.metricName, "Test Measure", stats.UnitNone),
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{},
		}
		mtf := getMetricTypeFunc(
			path.Join(testCase.domain, testCase.component),
			path.Join(defaultCustomMetricSubDomain, testCase.component))

		gotMetricType := mtf(testView)
		wantedMetricType := path.Join(defaultCustomMetricSubDomain, testCase.component, testView.Measure.Name())
		if gotMetricType != wantedMetricType {
			t.Fatalf("getMetricType=%v, want %v", gotMetricType, wantedMetricType)
		}
	}
}

func TestNewStackdriverExporterWithMetadata(t *testing.T) {
	tests := []struct {
		name          string
		config        *metricsConfig
		expectSuccess bool
	}{{
		name: "standardCase",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          "autoscaler",
			backendDestination: Stackdriver,
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID: testProj,
			},
		},
		expectSuccess: true,
	}, {
		name: "stackdriverClientConfigOnly",
		config: &metricsConfig{
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
			backendDestination:                Stackdriver,
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
		name: "invalidStackdriverGcpLocation",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                Stackdriver,
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
			backendDestination:                Stackdriver,
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
			backendDestination:                Stackdriver,
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e, err := newStackdriverExporter(test.config, TestLogger(t))

			succeeded := e != nil && err == nil
			if test.expectSuccess != succeeded {
				t.Errorf("Unexpected test result. Expected success? [%v]. Error: [%v]", test.expectSuccess, err)
			}
		})
	}
}

func TestEnsureKubeClient(t *testing.T) {
	// Even though ensureKubeclient uses sync.Once, make sure if the first run failed, it returns an error on subsequent calls.
	for i := 0; i < 3; i++ {
		err := ensureKubeclient()
		if err == nil {
			t.Error("Expected ensureKubeclient to fail due to not being in a Kubernetes cluster. Did the function run?")
		}
	}
}

func assertStringsEqual(t *testing.T, description string, expected string, actual string) {
	if expected != actual {
		t.Errorf("Expected %v to be set correctly. Want [%v], Got [%v]", description, expected, actual)
	}
}

func TestSetStackdriverSecretLocation(t *testing.T) {
	// Reset global state after test
	defer func() {
		secretName = StackdriverSecretNameDefault
		secretNamespace = StackdriverSecretNamespaceDefault
	}()

	sdConfig := &StackdriverClientConfig{
		ProjectID:   "project",
		GCPLocation: "us-west2",
		ClusterName: "cluster",
		UseSecret:   false,
	}

	// Sanity checks
	assertStringsEqual(t, "DefaultSecretName", secretName, StackdriverSecretNameDefault)
	assertStringsEqual(t, "DefaultSecretNamespace", secretNamespace, StackdriverSecretNamespaceDefault)
	if _, err := getStackdriverExporterClientOptions(sdConfig); err != nil {
		t.Errorf("Got unexpected error when creating exporter client options: [%v]", err)
	}

	// Check configuration's UseSecret value is ignored until the consuming package calls SetStackdriverSecretLocation
	// If an attempt to read a Secret was made, there should be an error because there's no valid in-cluster kubeclient.
	sdConfig.UseSecret = true
	if _, e1 := getStackdriverExporterClientOptions(sdConfig); e1 != nil {
		t.Errorf("Got unexpected error when creating exporter client options: [%v]", e1)
	}

	testName, testNamespace := "test-name", "test-namespace"
	// SetStackdriverSecretLocation has been called & config's UseSecret value is set
	// There should be an attempt to read the Secret, and an error because there's no valid in-cluster kubeclient.
	SetStackdriverSecretLocation("test-name", "test-namespace")
	if _, e1 := getStackdriverExporterClientOptions(sdConfig); e1 == nil {
		t.Errorf("Expected a known error when getting exporter options with Secrets enabled (cannot create valid kubeclient in tests). Did the function run as expected?")
	}
	assertStringsEqual(t, "secretName", secretName, testName)
	assertStringsEqual(t, "secretNamespace", secretNamespace, testNamespace)

	randomName, randomNamespace := "random-name", "random-namespace"
	SetStackdriverSecretLocation(randomName, randomNamespace)
	if _, e1 := getStackdriverExporterClientOptions(sdConfig); e1 == nil {
		t.Errorf("Expected a known error when getting exporter options with Secrets enabled (cannot create valid kubeclient in tests). Did the function run as expected?")
	}
	assertStringsEqual(t, "secretName", secretName, randomName)
	assertStringsEqual(t, "secretNamespace", secretNamespace, randomNamespace)
}
