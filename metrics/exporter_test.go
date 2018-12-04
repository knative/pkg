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

func TestMain(m *testing.M) {
	resetCurPromSrv()
	m.Run()
}

func TestNewStackdriverExporter(t *testing.T) {
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
		stackdriverProjectID: "",
	}, TestLogger(t))
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
