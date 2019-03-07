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
		expectedErr: "unsupported metrics backend value \"unsupported\"",
	}, {
		name: "emptyDomain",
		cm: map[string]string{
			"metrics.backend-destination": "prometheus",
		},
		domain:      "",
		component:   testComponent,
		expectedErr: "metrics domain cannot be empty",
	}, {
		name: "invalidComponent",
		cm: map[string]string{
			"metrics.backend-destination": "prometheus",
		},
		domain:      servingDomain,
		component:   "",
		expectedErr: "metrics component name cannot be empty",
	}, {
		name: "invalidReportingPeriod",
		cm: map[string]string{
			"metrics.backend-destination":      "prometheus",
			"metrics.reporting-period-seconds": "test",
		},
		domain:      servingDomain,
		component:   testComponent,
		expectedErr: "invalid metrics.reporting-period-seconds value \"test\"",
	}, {
		name: "invalidAllowStackdriverCustomMetrics",
		cm: map[string]string{
			"metrics.backend-destination":              "stackdriver",
			"metrics.allow-stackdriver-custom-metrics": "test",
		},
		domain:      servingDomain,
		component:   testComponent,
		expectedErr: "invalid metrics.allow-stackdriver-custom-metrics value \"test\"",
	}}
	successTests = []struct {
		name                string
		cm                  map[string]string
		domain              string
		component           string
		expectedConfig      metricsConfig
		expectedNewExporter bool // Whether the config requires a new exporter compared to previous test case
	}{
		// Note the first unit test is skipped in TestUpdateExporterFromConfigMap since
		// unit test does not have application default credentials.
		{
			name:      "stackdriverProjectIDMissing",
			cm:        map[string]string{"metrics.backend-destination": "stackdriver"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, testComponent),
			},
			expectedNewExporter: true,
		}, {
			name:      "backendKeyMissing",
			cm:        map[string]string{"": ""},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
			},
			expectedNewExporter: true,
		}, {
			name: "validStackdriver",
			cm: map[string]string{"metrics.backend-destination": "stackdriver",
				"metrics.stackdriver-project-id": anotherProj},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              anotherProj,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, testComponent),
			},
			expectedNewExporter: true,
		}, {
			name:      "validPrometheus",
			cm:        map[string]string{"metrics.backend-destination": "prometheus"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
			},
			expectedNewExporter: true,
		}, {
			name: "validCapitalStackdriver",
			cm: map[string]string{"metrics.backend-destination": "Stackdriver",
				"metrics.stackdriver-project-id": testProj},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              testProj,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, testComponent),
			},
			expectedNewExporter: true,
		}, {
			name:      "overriddenReportingPeriodPrometheus",
			cm:        map[string]string{"metrics.backend-destination": "prometheus", "metrics.reporting-period-seconds": "12"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    12 * time.Second,
			},
			expectedNewExporter: true,
		}, {
			name: "overriddenReportingPeriodStackdriver",
			cm: map[string]string{"metrics.backend-destination": "stackdriver",
				"metrics.stackdriver-project-id": "test2", "metrics.reporting-period-seconds": "7"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   7 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, testComponent),
			},
			expectedNewExporter: true,
		}, {
			name: "overriddenReportingPeriodStackdriver2",
			cm: map[string]string{"metrics.backend-destination": "stackdriver",
				"metrics.stackdriver-project-id": "test2", "metrics.reporting-period-seconds": "3"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   3 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, testComponent),
			},
		}, {
			name:      "emptyReportingPeriodPrometheus",
			cm:        map[string]string{"metrics.backend-destination": "prometheus", "metrics.reporting-period-seconds": ""},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
			},
			expectedNewExporter: true,
		}, {
			name: "emptyReportingPeriodStackdriver",
			cm: map[string]string{"metrics.backend-destination": "stackdriver",
				"metrics.stackdriver-project-id": "test2", "metrics.reporting-period-seconds": ""},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, testComponent),
			},
			expectedNewExporter: true,
		}, {
			name: "allowStackdriverCustomMetric",
			cm: map[string]string{
				"metrics.backend-destination":              "stackdriver",
				"metrics.stackdriver-project-id":           "test2",
				"metrics.reporting-period-seconds":         "",
				"metrics.allow-stackdriver-custom-metrics": "true",
			},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, testComponent),
				allowStackdriverCustomMetrics:     true,
			},
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
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, testComponent),
			},
		}, {
			name:      "validPrometheus",
			cm:        map[string]string{"metrics.backend-destination": "prometheus"},
			domain:    servingDomain,
			component: testComponent,
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
			},
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

func TestIsNewExporterRequired(t *testing.T) {
	setCurMetricsConfig(nil)
	for _, test := range successTests {
		t.Run(test.name, func(t *testing.T) {
			mc, err := getMetricsConfig(test.cm, test.domain, test.component, TestLogger(t))
			if err != nil {
				t.Errorf("Wanted valid config %v, got error %v", test.expectedConfig, err)
			}
			changed := isNewExporterRequired(mc)
			if changed != test.expectedNewExporter {
				t.Errorf("isMetricsConfigChanged=%v wanted %v", changed, test.expectedNewExporter)
			}
			setCurMetricsConfig(mc)
		})
	}

	setCurMetricsConfig(&metricsConfig{
		domain:             servingDomain,
		component:          testComponent,
		backendDestination: Prometheus})
	newConfig := &metricsConfig{
		domain:               servingDomain,
		component:            testComponent,
		backendDestination:   Prometheus,
		stackdriverProjectID: testProj,
	}
	changed := isNewExporterRequired(newConfig)
	if changed {
		t.Error("isNewExporterRequired should be false if stackdriver project ID changes for prometheus backend")
	}
}

func TestUpdateExporterFromConfigMap(t *testing.T) {
	setCurMetricsConfig(nil)
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
	setCurMetricsConfig(nil)
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
