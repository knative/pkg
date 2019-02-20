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
	"reflect"
	"testing"
	"time"

	. "github.com/knative/pkg/logging/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	servingDomain = "knative.dev/serving"
	badDomain     = "test.domain"
	testComponent = "testComponent"
	testProj      = "test-project"
	anotherProj   = "another-project"
)

var (
	errorTests = []struct {
		name        string
		cm          map[string]string
		domain      string
		component   string
		expectedErr string
	}{{
		name: "unsupportedBackend",
		cm: map[string]string{
			"metrics.backend-destination":    "unsupported",
			"metrics.stackdriver-project-id": testProj,
		},
		domain:      servingDomain,
		component:   testComponent,
		expectedErr: "Unsupported metrics backend value \"unsupported\"",
	}, {
		name: "emptyDomain",
		cm: map[string]string{
			"metrics.backend-destination": "prometheus",
		},
		domain:      "",
		component:   testComponent,
		expectedErr: "Metrics domain cannot be empty",
	}, {
		name: "invalidComponent",
		cm: map[string]string{
			"metrics.backend-destination": "prometheus",
		},
		domain:      servingDomain,
		component:   "",
		expectedErr: "Metrics component name cannot be empty",
	}, {
		name: "invalidReportingPeriod",
		cm: map[string]string{
			"metrics.backend-destination":      "prometheus",
			"metrics.reporting-period-seconds": "test",
		},
		domain:      servingDomain,
		component:   "",
		expectedErr: "Invalid reporting-period-seconds value \"test\"",
	}}
	successTests = []struct {
		name           string
		cm             map[string]string
		domain         string
		component      string
		expectedConfig metricsConfig
	}{
		// Note the first unit test is skipped in TestUpdateExporterFromConfigMap since
		// unit test does not have application default credentials.
		{
			name:      "stackdriverProjectIDMissing",
			cm:        map[string]string{"metrics.backend-destination": "stackdriver"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Stackdriver,
				reportingPeriod:    60 * time.Second},
		}, {
			name:      "backendKeyMissing",
			cm:        map[string]string{"": ""},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second},
		}, {
			name: "validStackdriver",
			cm: map[string]string{"metrics.backend-destination": "stackdriver",
				"metrics.stackdriver-project-id": anotherProj},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:               servingDomain,
				component:            testComponent,
				backendDestination:   Stackdriver,
				stackdriverProjectID: anotherProj,
				reportingPeriod:      60 * time.Second},
		}, {
			name:      "validPrometheus",
			cm:        map[string]string{"metrics.backend-destination": "prometheus"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second},
		}, {
			name: "validCapitalStackdriver",
			cm: map[string]string{"metrics.backend-destination": "Stackdriver",
				"metrics.stackdriver-project-id": testProj},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:               servingDomain,
				component:            testComponent,
				backendDestination:   Stackdriver,
				stackdriverProjectID: testProj,
				reportingPeriod:      60 * time.Second},
		}, {
			name:      "overriddenReportingPeriodPrometheus",
			cm:        map[string]string{"metrics.backend-destination": "prometheus", "metrics.reporting-period-seconds": "12"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    12 * time.Second},
		}, {
			name: "overriddenReportingPeriodStackdriver",
			cm: map[string]string{"metrics.backend-destination": "stackdriver",
				"metrics.stackdriver-project-id": "test2", "metrics.reporting-period-seconds": "7"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:               servingDomain,
				component:            testComponent,
				backendDestination:   Stackdriver,
				stackdriverProjectID: "test2",
				reportingPeriod:      7 * time.Second},
		}, {
			name: "overriddenReportingPeriodStackdriver2",
			cm: map[string]string{"metrics.backend-destination": "stackdriver",
				"metrics.stackdriver-project-id": "test2", "metrics.reporting-period-seconds": "3"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:               servingDomain,
				component:            testComponent,
				backendDestination:   Stackdriver,
				stackdriverProjectID: "test2",
				reportingPeriod:      3 * time.Second},
		}, {
			name:      "emptyReportingPeriodPrometheus",
			cm:        map[string]string{"metrics.backend-destination": "prometheus", "metrics.reporting-period-seconds": ""},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second},
		}, {
			name: "emptyReportingPeriodStackdriver",
			cm: map[string]string{"metrics.backend-destination": "stackdriver",
				"metrics.stackdriver-project-id": "test2", "metrics.reporting-period-seconds": ""},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:               servingDomain,
				component:            testComponent,
				backendDestination:   Stackdriver,
				stackdriverProjectID: "test2",
				reportingPeriod:      60 * time.Second},
		}}
	envTests = []struct {
		name           string
		cm             map[string]string
		domain         string
		component      string
		expectedConfig metricsConfig
	}{
		{
			name:      "stackdriverFromEnv",
			cm:        map[string]string{"": ""},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Stackdriver,
				reportingPeriod:    60 * time.Second},
		}, {
			name:      "validPrometheus",
			cm:        map[string]string{"metrics.backend-destination": "prometheus"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second},
		}}
)

func TestGetMetricsConfig(t *testing.T) {
	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			_, err := getMetricsConfig(test.cm, test.domain, test.component, TestLogger(t))
			if err.Error() != test.expectedErr {
				t.Errorf("Wanted err: %v, got: %v", test.expectedErr, err)
			}
		})
	}

	for _, test := range successTests {
		t.Run(test.name, func(t *testing.T) {
			mc, err := getMetricsConfig(test.cm, test.domain, test.component, TestLogger(t))
			if err != nil {
				t.Errorf("Wanted valid config %v, got error %v", test.expectedConfig, err)
			}
			if !reflect.DeepEqual(*mc, test.expectedConfig) {
				t.Errorf("Wanted config %v, got config %v", test.expectedConfig, *mc)
			}
		})
	}
}

func TestGetMetricsConfig_fromEnv(t *testing.T) {
	os.Setenv(defaultBackendEnvName, "stackdriver")
	for _, test := range envTests {
		t.Run(test.name, func(t *testing.T) {
			mc, err := getMetricsConfig(test.cm, test.domain, test.component, TestLogger(t))
			if err != nil {
				t.Errorf("Wanted valid config %v, got error %v", test.expectedConfig, err)
			}
			if !reflect.DeepEqual(*mc, test.expectedConfig) {
				t.Errorf("Wanted config %v, got config %v", test.expectedConfig, *mc)
			}
		})
	}
	os.Unsetenv(defaultBackendEnvName)
}

func TestIsMetricsConfigChanged(t *testing.T) {
	setCurMetricsExporterAndConfig(nil, nil)
	for _, test := range successTests {
		t.Run(test.name, func(t *testing.T) {
			mc, err := getMetricsConfig(test.cm, test.domain, test.component, TestLogger(t))
			if err != nil {
				t.Errorf("Wanted valid config %v, got error %v", test.expectedConfig, err)
			}
			changed := isMetricsConfigChanged(mc)
			if !changed {
				t.Error("isMetricsConfigChanged should be true")
			}
			setCurMetricsExporterAndConfig(nil, mc)
		})
	}

	setCurMetricsExporterAndConfig(nil, &metricsConfig{
		domain:             servingDomain,
		component:          testComponent,
		backendDestination: Prometheus})
	newConfig := &metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   Prometheus,
		stackdriverProjectID: testProj,
	}
	changed := isMetricsConfigChanged(newConfig)
	if changed {
		t.Error("isMetricsConfigChanged should be false if stackdriver project ID changes for prometheus backend")
	}
}

func TestUpdateExporterFromConfigMap(t *testing.T) {
	setCurMetricsExporterAndConfig(nil, nil)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config-observability",
		},
		Data: map[string]string{},
	}
	oldConfig := getCurMetricsConfig()
	for _, test := range successTests[1:] {
		t.Run(test.name, func(t *testing.T) {
			cm.Data = test.cm
			u := UpdateExporterFromConfigMap(test.domain, test.component, TestLogger(t))
			u(cm)
			mConfig := getCurMetricsConfig()
			if mConfig == oldConfig {
				t.Error("Expected metrics config change")
			}
			if !reflect.DeepEqual(*mConfig, test.expectedConfig) {
				t.Errorf("Expected config: %v; got config %v", test.expectedConfig, mConfig)
			}
			oldConfig = mConfig
		})
	}

	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			cm.Data = test.cm
			u := UpdateExporterFromConfigMap(test.domain, test.component, TestLogger(t))
			u(cm)
			mConfig := getCurMetricsConfig()
			if mConfig != oldConfig {
				t.Error("mConfig should not change")
			}
		})
	}
}

func TestUpdateExporterFromConfigMap_doesNotCreateExporter(t *testing.T) {
	setCurMetricsExporterAndConfig(nil, nil)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config-observability",
		},
		Data: map[string]string{},
	}
	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			cm.Data = test.cm
			u := UpdateExporterFromConfigMap(test.domain, test.component, TestLogger(t))
			u(cm)
			mConfig := getCurMetricsConfig()
			if mConfig != nil {
				t.Error("mConfig should not be created")
			}
		})
	}
}
