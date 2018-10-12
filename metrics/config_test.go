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
	"testing"
	"time"

	. "github.com/knative/pkg/logging/testing"
)

const (
	testProj      = "test-project"
	testDomain    = "test.domain"
	testComponent = "testComponent"
)

func TestNewStackdriverExporter(t *testing.T) {
	// The stackdriver project ID is required for stackdriver exporter.
	err := newMetricsExporter(metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: ""}, TestLogger(t))
	if err == nil {
		t.Error("expected an error if the project id is empty for stackdriver exporter")
	}

	err = newMetricsExporter(metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	expectNoPromSrv(t)
}

func TestNewPrometheusExporter(t *testing.T) {
	// The stackdriver project ID is not required for prometheus exporter.
	err := newMetricsExporter(metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Prometheus,
		stackdriverProjectID: ""}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	expectPromSrv(t)
}

func TestInterlevedExporters(t *testing.T) {
	// First create a stackdriver exporter
	err := newMetricsExporter(metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	err = newMetricsExporter(metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Prometheus,
		stackdriverProjectID: "",
	}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
	expectPromSrv(t)
	// Finally switch to stackdriver exporter
	err = newMetricsExporter(metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj}, TestLogger(t))
	if err != nil {
		t.Error(err)
	}
}

func TestGetMetricsConfig(t *testing.T) {
	_, err := getMetricsConfig(map[string]string{
		"": "",
	}, testDomain, testComponent, TestLogger(t))
	if err == nil {
		t.Error("expected an error")
	}
	if err.Error() != "metrics.backend-destination key is missing" {
		t.Errorf("expected error: metrics.backend-destination key is missing. Got %v", err)
	}
	c, err := getMetricsConfig(map[string]string{
		"metrics.backend-destination": "prometheus",
	}, testDomain, testComponent, TestLogger(t))
	if err != nil {
		t.Error("failed to get config for prometheus")
	}
	expected := metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Prometheus,
		stackdriverProjectID: "",
	}
	if c != expected {
		t.Errorf("expected config: %v; got config: %v", expected, c)
	}
	_, err = getMetricsConfig(map[string]string{
		"metrics.backend-destination": "stackdriver",
	}, testDomain, testComponent, TestLogger(t))
	if err.Error() != "metrics.stackdriver-project-id key is missing when the backend-destination is set to stackdriver." {
		t.Errorf("expected error: metrics.stackdriver-project-id key is missing when the backend-destination is set to stackdriver. Got %v", err)
	}
	c, err = getMetricsConfig(map[string]string{
		"metrics.backend-destination":    "stackdriver",
		"metrics.stackdriver-project-id": testProj,
	}, testDomain, testComponent, TestLogger(t))
	if err != nil {
		t.Error("failed to get config for stackdriver")
	}
	expected = metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Stackdriver,
		stackdriverProjectID: testProj,
	}
	if c != expected {
		t.Errorf("expected config: %v; got config: %v", expected, c)
	}
	c, err = getMetricsConfig(map[string]string{
		"metrics.backend-destination": "unsupported",
	}, testDomain, testComponent, TestLogger(t))
	if err != nil {
		t.Error("failed to get config for stackdriver")
	}
	expected = metricsConfig{
		domain:               testDomain,
		component:            testComponent,
		backendDestination:   Prometheus,
		stackdriverProjectID: "",
	}
	if c != expected {
		t.Errorf("expected config: %v; got config: %v", expected, c)
	}
	c, err = getMetricsConfig(map[string]string{
		"metrics.backend-destination": "prometheus",
	}, "", "", TestLogger(t))
	if err != nil {
		t.Error("failed to get config for stackdriver")
	}
	expected = metricsConfig{
		domain:               "domain",
		component:            "component",
		backendDestination:   Prometheus,
		stackdriverProjectID: "",
	}
	if c != expected {
		t.Errorf("expected config: %v; got config: %v", expected, c)
	}
}

func expectPromSrv(t *testing.T) {
	select {
	case <-promSrvChan:
		t.Log("A server found for prometheus.")
	case <-time.After(200 * time.Millisecond):
		t.Error("expected a server for prometheus exporter")
	}
}

func expectNoPromSrv(t *testing.T) {
	time.Sleep(200 * time.Millisecond)
	select {
	case <-promSrvChan:
		t.Error("expected no server for stackdriver exporter")
	default:
	}
}
