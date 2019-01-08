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
	"testing"
	"time"

	. "github.com/knative/pkg/logging/testing"
	"github.com/knative/pkg/metrics/metricskey"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	testNS            = "test"
	testService       = "test-service"
	testRoute         = "test-route"
	testConfiguration = "test-configuration"
	testRevision      = "test-revision"
)

var (
	testView = &view.View{
		Description: "Test View",
		Measure:     stats.Int64("test", "Test Measure", stats.UnitNone),
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{},
	}

	nsKey       = tag.Tag{Key: mustNewTagKey(metricskey.LabelNamespaceName), Value: testNS}
	serviceKey  = tag.Tag{Key: mustNewTagKey(metricskey.LabelServiceName), Value: testService}
	routeKey    = tag.Tag{Key: mustNewTagKey(metricskey.LabelRouteName), Value: testRoute}
	revisionKey = tag.Tag{Key: mustNewTagKey(metricskey.LabelRevisionName), Value: testRevision}

	testTags = []tag.Tag{nsKey, serviceKey, routeKey, revisionKey}
)

func mustNewTagKey(s string) tag.Key {
	tagKey, err := tag.NewKey(s)
	if err != nil {
		panic(err)
	}
	return tagKey
}

func getResourceLabelValue(key string, tags []tag.Tag) string {
	for _, t := range tags {
		if t.Key.Name() == key {
			return t.Value
		}
	}
	return ""
}

func TestMain(m *testing.M) {
	resetCurPromSrv()
	os.Exit(m.Run())
}

func TestNewStackdriverExporterForGlobal(t *testing.T) {
	resetMonitoredResourceFunc()
	// The stackdriver project ID is required for stackdriver exporter.
	e, err := newStackdriverExporter(&metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	if e == nil {
		t.Error("expected a non-nil metrics exporter")
	}
	if getMonitoredResourceFunc == nil {
		t.Error("expected a non-nil getMonitoredResourceFunc")
	}
	newTags, monitoredResource := getMonitoredResourceFunc(testView, testTags)
	gotResType, labels := monitoredResource.MonitoredResource()
	wantedResType := "global"
	if gotResType != wantedResType {
		t.Errorf("MonitoredResource=%v, got: %v", wantedResType, gotResType)
	}
	got := getResourceLabelValue(metricskey.LabelNamespaceName, newTags)
	if got != testNS {
		t.Errorf("expected new tag %v with value %v, got: %v", routeKey, testNS, newTags)
	}
	if len(labels) != 0 {
		t.Errorf("expected no label, got: %v", labels)
	}
}

func TestNewStackdriverExporterForKnativeRevision(t *testing.T) {
	resetMonitoredResourceFunc()
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
	if getMonitoredResourceFunc == nil {
		t.Error("expected a non-nil getMonitoredResourceFunc")
	}
	newTags, monitoredResource := getMonitoredResourceFunc(testView, testTags)
	gotResType, labels := monitoredResource.MonitoredResource()
	wantedResType := "knative_revision"
	if gotResType != wantedResType {
		t.Errorf("MonitoredResource=%v, got %v", wantedResType, gotResType)
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

func TestNewPrometheusExporter(t *testing.T) {
	// The stackdriver project ID is not required for prometheus exporter.
	e, err := newPrometheusExporter(&metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   Prometheus,
		stackdriverProjectID: ""}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	if e == nil {
		t.Error("expected a non-nil metrics exporter")
	}
	expectPromSrv(t)
}

func TestMetricsExporter(t *testing.T) {
	err := newMetricsExporter(&metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   "unsupported",
		stackdriverProjectID: ""}, TestLogger(t))
	if err == nil {
		t.Errorf("Expected an error for unsupported backend %v", err)
	}

	err = newMetricsExporter(&metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
}

func TestInterlevedExporters(t *testing.T) {
	// First create a stackdriver exporter
	err := newMetricsExporter(&metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	expectNoPromSrv(t)
	// Then switch to prometheus exporter
	err = newMetricsExporter(&metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   Prometheus,
		stackdriverProjectID: ""}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	expectPromSrv(t)
	// Finally switch to stackdriver exporter
	err = newMetricsExporter(&metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
}

func expectPromSrv(t *testing.T) {
	time.Sleep(200 * time.Millisecond)
	srv := getCurPromSrv()
	if srv == nil {
		t.Error("expected a server for prometheus exporter")
	}
}

func expectNoPromSrv(t *testing.T) {
	time.Sleep(200 * time.Millisecond)
	srv := getCurPromSrv()
	if srv != nil {
		t.Error("expected no server for stackdriver exporter")
	}
}
