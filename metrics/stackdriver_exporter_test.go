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

	"contrib.go.opencensus.io/exporter/stackdriver"
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
		domain:     servingDomain,
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
		domain:     eventingDomain,
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
		domain:     eventingDomain,
		component:  "trigger",
		metricName: "event_count",
	}, {
		name:       "trigger metric",
		domain:     eventingDomain,
		component:  "trigger",
		metricName: "event_processing_latencies",
	}, {
		name:       "trigger metric",
		domain:     eventingDomain,
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
		domain:     eventingDomain,
		component:  "unsupported",
		metricName: "event_count",
	}, {
		name:       "unsupported metric",
		domain:     eventingDomain,
		component:  "broker",
		metricName: "unsupported",
	}}
)

func fakeGcpMetadataFun() *gcpMetadata {
	return &testGcpMetadata
}

type fakeExporter struct{}

func (fe *fakeExporter) ExportView(vd *view.Data) {}
func (fe *fakeExporter) Flush()                   {}

func newFakeExporter(o stackdriver.Options) (view.Exporter, error) {
	return &fakeExporter{}, nil
}

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
		got = getResourceLabelValue(metricskey.LabelEventSource, newTags)
		if got != testEventSource {
			t.Errorf("expected new tag: %v, got: %v", eventSourceKey, newTags)
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
		got = getResourceLabelValue(metricskey.LabelFilterSource, newTags)
		if got != testFilterSource {
			t.Errorf("expected new tag: %v, got: %v", filterSourceKey, newTags)
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
		got = getResourceLabelValue(metricskey.LabelEventSource, newTags)
		if got != testEventSource {
			t.Errorf("expected new tag: %v, got: %v", eventSourceKey, newTags)
		}
		got, ok := labels[metricskey.LabelNamespaceName]
		if !ok || got != testNS {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelNamespaceName, testNS, got)
		}
		got, ok = labels[metricskey.LabelSourceName]
		if !ok || got != testSource {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelSourceName, testSource, got)
		}
		got, ok = labels[metricskey.LabelSourceResourceGroup]
		if !ok || got != testSourceResourceGroup {
			t.Errorf("expected label %v with value %v, got: %v", metricskey.LabelSourceResourceGroup, testSourceResourceGroup, got)
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

func TestGetgetMetricTypeFunc_UseKnativeDomain(t *testing.T) {
	for _, testCase := range supportedServingMetricsTestCases {
		testView = &view.View{
			Description: "Test View",
			Measure:     stats.Int64(testCase.metricName, "Test Measure", stats.UnitNone),
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{},
		}
		mtf := getMetricTypeFunc(
			path.Join(testCase.domain, testCase.component),
			path.Join(customMetricTypePrefix, testCase.component))

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
			path.Join(customMetricTypePrefix, testCase.component))

		gotMetricType := mtf(testView)
		wantedMetricType := path.Join(customMetricTypePrefix, testCase.component, testView.Measure.Name())
		if gotMetricType != wantedMetricType {
			t.Fatalf("getMetricType=%v, want %v", gotMetricType, wantedMetricType)
		}
	}
}

func TestNewStackdriverExporterWithMetadata(t *testing.T) {
	e, err := newStackdriverExporter(&metricsConfig{
		domain:               servingDomain,
		component:            "autoscaler",
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	if e == nil {
		t.Error("expected a non-nil metrics exporter")
	}
}
