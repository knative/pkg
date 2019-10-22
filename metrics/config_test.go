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
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/stats/view"

	corev1 "k8s.io/api/core/v1"

	. "knative.dev/pkg/logging/testing"
	sdconfig "knative.dev/pkg/stackdriver/config"
)

// TODO UTs should move to eventing and serving, as appropriate.
// 	See https://github.com/knative/pkg/issues/608

const (
	servingDomain   = "knative.dev/serving"
	eventingDomain  = "knative.dev/eventing"
	customSubDomain = "test.domain"
	testComponent   = "testComponent"
	testProj        = "test-project"
	anotherProj     = "another-project"
)

var (
	errorTests = []struct {
		name        string
		ops         ExporterOptions
		expectedErr string
	}{{
		name: "emptyConfig",
		ops: ExporterOptions{
			Domain:    servingDomain,
			Component: testComponent,
		},
		expectedErr: "metrics config map cannot be empty",
	}, {
		name: "unsupportedBackend",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				"metrics.backend-destination":    "unsupported",
				"metrics.stackdriver-project-id": testProj,
			},
			Domain:    servingDomain,
			Component: testComponent,
		},
		expectedErr: "unsupported metrics backend value \"unsupported\"",
	}, {
		name: "emptyDomain",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				"metrics.backend-destination": "prometheus",
			},
			Domain:    "",
			Component: testComponent,
		},
		expectedErr: "metrics domain cannot be empty",
	}, {
		name: "invalidComponent",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				"metrics.backend-destination": "prometheus",
			},
			Domain:    servingDomain,
			Component: "",
		},
		expectedErr: "metrics component name cannot be empty",
	}, {
		name: "invalidReportingPeriod",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				"metrics.backend-destination":      "prometheus",
				"metrics.reporting-period-seconds": "test",
			},
			Domain:    servingDomain,
			Component: testComponent,
		},
		expectedErr: "invalid metrics.reporting-period-seconds value \"test\"",
	}, {
		name: "invalidAllowStackdriverCustomMetrics",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				"metrics.backend-destination":              "stackdriver",
				"metrics.allow-stackdriver-custom-metrics": "test",
			},
			Domain:    servingDomain,
			Component: testComponent,
		},
		expectedErr: "invalid metrics.allow-stackdriver-custom-metrics value \"test\"",
	}, {
		name: "tooSmallPrometheusPort",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				"metrics.backend-destination": "prometheus",
			},
			Domain:         servingDomain,
			Component:      testComponent,
			PrometheusPort: 1023,
		},
		expectedErr: "invalid port 1023, should between 1024 and 65535",
	}, {
		name: "tooBigPrometheusPort",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				"metrics.backend-destination": "prometheus",
			},
			Domain:         servingDomain,
			Component:      testComponent,
			PrometheusPort: 65536,
		},
		expectedErr: "invalid port 65536, should between 1024 and 65535",
	}}
	successTests = []struct {
		name                string
		ops                 ExporterOptions
		expectedConfig      metricsConfig
		expectedNewExporter bool // Whether the config requires a new exporter compared to previous test case
	}{
		// Note the first unit test is skipped in TestUpdateExporterFromConfigMap since
		// unit test does not have application default credentials.
		{
			name: "stackdriverProjectIDMissing",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination": "stackdriver",
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
			},
			expectedNewExporter: true,
		}, {
			name: "backendKeyMissing",
			ops: ExporterOptions{
				ConfigMap: map[string]string{},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
				prometheusPort:     defaultPrometheusPort,
			},
			expectedNewExporter: true,
		}, {
			name: "validStackdriver",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":    "stackdriver",
					"metrics.stackdriver-project-id": anotherProj,
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              anotherProj,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
			},
			expectedNewExporter: true,
		}, {
			name: "validPrometheus",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination": "prometheus",
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
				prometheusPort:     defaultPrometheusPort,
			},
			expectedNewExporter: true,
		}, {
			name: "validCapitalStackdriver",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":    "Stackdriver",
					"metrics.stackdriver-project-id": testProj,
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              testProj,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
			},
			expectedNewExporter: true,
		}, {
			name: "overriddenReportingPeriodPrometheus",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":      "prometheus",
					"metrics.reporting-period-seconds": "12",
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    12 * time.Second,
				prometheusPort:     defaultPrometheusPort,
			},
			expectedNewExporter: true,
		}, {
			name: "overriddenReportingPeriodStackdriver",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":      "stackdriver",
					"metrics.stackdriver-project-id":   "test2",
					"metrics.reporting-period-seconds": "7",
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   7 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
			},
			expectedNewExporter: true,
		}, {
			name: "overriddenReportingPeriodStackdriver2",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":      "stackdriver",
					"metrics.stackdriver-project-id":   "test2",
					"metrics.reporting-period-seconds": "3",
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   3 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
			},
		}, {
			name: "emptyReportingPeriodPrometheus",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":      "prometheus",
					"metrics.reporting-period-seconds": "",
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
				prometheusPort:     defaultPrometheusPort,
			},
			expectedNewExporter: true,
		}, {
			name: "emptyReportingPeriodStackdriver",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":      "stackdriver",
					"metrics.stackdriver-project-id":   "test2",
					"metrics.reporting-period-seconds": "",
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
			},
			expectedNewExporter: true,
		}, {
			name: "allowStackdriverCustomMetric",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":              "stackdriver",
					"metrics.stackdriver-project-id":           "test2",
					"metrics.reporting-period-seconds":         "",
					"metrics.allow-stackdriver-custom-metrics": "true",
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				allowStackdriverCustomMetrics:     true,
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
			},
		}, {
			name: "allowStackdriverCustomMetric with subdomain",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination":                  "stackdriver",
					"metrics.stackdriver-project-id":               "test2",
					"metrics.reporting-period-seconds":             "",
					"metrics.stackdriver-custom-metrics-subdomain": customSubDomain,
				},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				stackdriverProjectID:              "test2",
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, customSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: customSubDomain,
			},
		}, {
			name: "overridePrometheusPort",
			ops: ExporterOptions{
				ConfigMap: map[string]string{
					"metrics.backend-destination": "prometheus",
				},
				Domain:         servingDomain,
				Component:      testComponent,
				PrometheusPort: 9091,
			},
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
				prometheusPort:     9091,
			},
			expectedNewExporter: true,
		}}
	envTests = []struct {
		name           string
		ops            ExporterOptions
		expectedConfig metricsConfig
	}{
		{
			name: "stackdriverFromEnv",
			ops: ExporterOptions{
				ConfigMap: map[string]string{},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
			},
		}, {
			name: "validPrometheus",
			ops: ExporterOptions{
				ConfigMap: map[string]string{"metrics.backend-destination": "prometheus"},
				Domain:    servingDomain,
				Component: testComponent,
			},
			expectedConfig: metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
				prometheusPort:     defaultPrometheusPort,
			},
		}}

	sdConfigUpdate = ExporterOptions{
		StackdriverConfigMap: map[string]string{
			"project-id":      "project",
			"gcp-location":    "us-west1",
			"cluster-name":    "cluster",
			"gcp-secret-name": "secret",
		},
	}

	partialSdConfigUpdate = ExporterOptions{
		StackdriverConfigMap: map[string]string{
			"project-id":   "partial-project",
			"cluster-name": "partial-cluster",
		},
	}

	prometheusBackendOption = ExporterOptions{
		ConfigMap: map[string]string{
			"metrics.backend-destination": "prometheus",
		},
		Domain:    servingDomain,
		Component: testComponent,
	}

	sdBackendOption = ExporterOptions{
		ConfigMap: map[string]string{
			"metrics.backend-destination": "stackdriver",
		},
		Domain:    servingDomain,
		Component: testComponent,
	}

	modifyBothMapsInOneUpdate = ExporterOptions{
		ConfigMap: map[string]string{
			"metrics.backend-destination": "stackdriver",
		},
		Domain:    servingDomain,
		Component: testComponent,
		StackdriverConfigMap: map[string]string{
			"project-id":      "project",
			"gcp-location":    "us-west1",
			"cluster-name":    "cluster",
			"gcp-secret-name": "secret",
		},
	}

	// These tests require modifying two config maps which both
	// influence the state of the Stackdriver metrics exporter.
	stackdriverTests = []struct {
		name                      string
		updateList                []ExporterOptions
		expectedMetricsConfig     *metricsConfig
		expectedStackdriverConfig *sdconfig.Config
		expectedNewExporter       bool
	}{
		{
			name: "updateStackdriverConfigOnly",
			updateList: []ExporterOptions{
				sdConfigUpdate,
			},
			expectedMetricsConfig: nil,
			expectedNewExporter:   false,
		},
		{
			name: "backendPrometheusUpdateSdConfigFirst",
			updateList: []ExporterOptions{
				sdConfigUpdate,
				prometheusBackendOption,
			},
			expectedMetricsConfig: &metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
				prometheusPort:     defaultPrometheusPort,
				stackdriverConfig: sdconfig.Config{
					ProjectID:     "project",
					GcpLocation:   "us-west1",
					ClusterName:   "cluster",
					GcpSecretName: "secret",
				},
			},
			expectedNewExporter: true,
		},
		{
			name: "backendPrometheusUpdateSdConfigSecond",
			updateList: []ExporterOptions{
				prometheusBackendOption,
				sdConfigUpdate,
			},
			expectedMetricsConfig: &metricsConfig{
				domain:             servingDomain,
				component:          testComponent,
				backendDestination: Prometheus,
				reportingPeriod:    5 * time.Second,
				prometheusPort:     defaultPrometheusPort,
				stackdriverConfig: sdconfig.Config{
					ProjectID:     "project",
					GcpLocation:   "us-west1",
					ClusterName:   "cluster",
					GcpSecretName: "secret",
				},
			},
			expectedNewExporter: true,
		},
		{
			name: "backendSdUpdateSdConfigFirst",
			updateList: []ExporterOptions{
				sdBackendOption,
				sdConfigUpdate,
			},
			expectedMetricsConfig: &metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
				stackdriverConfig: sdconfig.Config{
					ProjectID:     "project",
					GcpLocation:   "us-west1",
					ClusterName:   "cluster",
					GcpSecretName: "secret",
				},
			},
			expectedNewExporter: true,
		},
		{
			name: "backendSdUpdateSdConfigSecond",
			updateList: []ExporterOptions{
				sdConfigUpdate,
				sdBackendOption,
			},
			expectedMetricsConfig: &metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
				stackdriverConfig: sdconfig.Config{
					ProjectID:     "project",
					GcpLocation:   "us-west1",
					ClusterName:   "cluster",
					GcpSecretName: "secret",
				},
			},
			expectedNewExporter: true,
		},
		{
			name: "partialSdConfig",
			updateList: []ExporterOptions{
				sdBackendOption,
				partialSdConfigUpdate,
			},
			expectedMetricsConfig: &metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
				stackdriverConfig: sdconfig.Config{
					ProjectID:   "partial-project",
					ClusterName: "partial-cluster",
				},
			},
			expectedNewExporter: true,
		},
		{
			name: "updateToNewestSdConfig",
			updateList: []ExporterOptions{
				sdBackendOption,
				sdConfigUpdate,
				partialSdConfigUpdate,
			},
			expectedMetricsConfig: &metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
				stackdriverConfig: sdconfig.Config{
					ProjectID:   "partial-project",
					ClusterName: "partial-cluster",
				},
			},
			expectedNewExporter: true,
		},
		{
			name: "bothConfigMaps",
			updateList: []ExporterOptions{
				modifyBothMapsInOneUpdate,
			},
			expectedMetricsConfig: &metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
				stackdriverConfig: sdconfig.Config{
					ProjectID:     "project",
					GcpLocation:   "us-west1",
					ClusterName:   "cluster",
					GcpSecretName: "secret",
				},
			},
			expectedNewExporter: true,
		},
		{
			name: "invalidGcpLocation",
			updateList: []ExporterOptions{
				sdBackendOption,
				ExporterOptions{
					StackdriverConfigMap: map[string]string{
						"gcp-location": "narnia",
					},
				},
			},
			expectedMetricsConfig: &metricsConfig{
				domain:                            servingDomain,
				component:                         testComponent,
				backendDestination:                Stackdriver,
				reportingPeriod:                   60 * time.Second,
				isStackdriverBackend:              true,
				stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
				stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
				stackdriverCustomMetricsSubDomain: defaultCustomMetricSubDomain,
				stackdriverConfig: sdconfig.Config{
					GcpLocation: "narnia",
				},
			},
			expectedNewExporter: true,
		},
	}

	updateExporterFromStackdriverConfigMapTests = []struct {
		name             string
		config           corev1.ConfigMap
		expectedSdConfig sdconfig.Config
	}{
		{
			name: "fullSdConfig",
			config: corev1.ConfigMap{
				Data: map[string]string{
					"project-id":      "project",
					"gcp-location":    "us-west1",
					"cluster-name":    "cluster",
					"gcp-secret-name": "secret",
				},
			},
			expectedSdConfig: sdconfig.Config{
				ProjectID:     "project",
				GcpLocation:   "us-west1",
				ClusterName:   "cluster",
				GcpSecretName: "secret",
			},
		},
		{
			name:             "emptySdConfig",
			config:           corev1.ConfigMap{},
			expectedSdConfig: sdconfig.Config{},
		},
		{
			name: "partialSdConfig",
			config: corev1.ConfigMap{
				Data: map[string]string{
					"project-id":   "project",
					"gcp-location": "us-west1",
					"cluster-name": "cluster",
				},
			},
			expectedSdConfig: sdconfig.Config{
				ProjectID:   "project",
				GcpLocation: "us-west1",
				ClusterName: "cluster",
			},
		},
		{
			name: "invalidGcpLocation",
			config: corev1.ConfigMap{
				Data: map[string]string{
					"gcp-location": "narnia",
				},
			},
			expectedSdConfig: sdconfig.Config{
				GcpLocation: "narnia",
			},
		},
	}
)

func TestUpdateExporterFromStackdriverConfigMap(t *testing.T) {
	for _, test := range updateExporterFromStackdriverConfigMapTests {
		t.Run(test.name, func(t *testing.T) {
			resetState()
			updateFunc := UpdateExporterFromStackdriverConfigMap(TestLogger(t))
			updateFunc(&test.config)

			cc := curStackdriverConfig
			if test.expectedSdConfig != *cc {
				t.Errorf("Incorrect stackdriver config. Expected: [%v], Got: [%v]", test.expectedSdConfig, *cc)
			}
		})
	}

	resetState()
}

func resetState() {
	curMetricsConfig = nil
	view.UnregisterExporter(curMetricsExporter)
	curMetricsExporter = nil
	curStackdriverConfig = &sdconfig.Config{}
}

func TestStackdriverConfigChangeCreatesNewExporter(t *testing.T) {
	getStackdriverSecretFunc = fakeGetStackdriverSecret
	logger := TestLogger(t)
	UpdateExporter(sdBackendOption, logger)
	c1 := getCurMetricsConfig()

	old := c1

	updates := []ExporterOptions{sdConfigUpdate, partialSdConfigUpdate}
	for _, update := range updates {
		UpdateExporter(update, logger)
		new := getCurMetricsConfig()

		setCurMetricsConfig(old)
		if !isNewExporterRequired(new) {
			t.Errorf("Stackdriver config change should have created a new exporter. Old config [%v]. New config [%v].", old, new)
		}

		old = new
	}
}

func TestStackdriverConfig(t *testing.T) {
	getStackdriverSecretFunc = fakeGetStackdriverSecret
	for _, test := range stackdriverTests {
		t.Run(test.name, func(t *testing.T) {
			// clean slate for test
			resetState()

			for _, op := range test.updateList {
				UpdateExporter(op, TestLogger(t))
			}
			mc := getCurMetricsConfig()

			mismatch := false
			if test.expectedMetricsConfig != nil && mc != nil {
				fmt.Println(test.name)
				if *test.expectedMetricsConfig != *mc {
					mismatch = true
				}
			} else if test.expectedMetricsConfig != mc {
				mismatch = true
			}

			if mismatch {
				t.Errorf("Incorrect metrics config. Expected config: [%v], Got: [%v]", test.expectedMetricsConfig, mc)
			}

			ne := getCurMetricsExporter()
			if test.expectedNewExporter != (ne != nil) {
				t.Errorf("Unexpected exporter state. Expected new exporter? [%v], got [%v]", test.expectedNewExporter, ne)
			}
		})
	}

	resetState()
}

func TestGetMetricsConfig(t *testing.T) {
	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			defer ClearAll()
			_, err := createMetricsConfig(test.ops, TestLogger(t))
			if err.Error() != test.expectedErr {
				t.Errorf("Wanted err: %v, got: %v", test.expectedErr, err)
			}
		})
	}

	for _, test := range successTests {
		t.Run(test.name, func(t *testing.T) {
			defer ClearAll()
			mc, err := createMetricsConfig(test.ops, TestLogger(t))
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
			defer ClearAll()
			mc, err := createMetricsConfig(test.ops, TestLogger(t))
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
			defer ClearAll()
			mc, err := createMetricsConfig(test.ops, TestLogger(t))
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

func TestUpdateExporter(t *testing.T) {
	setCurMetricsConfig(nil)
	oldConfig := getCurMetricsConfig()
	for _, test := range successTests[1:] {
		t.Run(test.name, func(t *testing.T) {
			defer ClearAll()
			UpdateExporter(test.ops, TestLogger(t))
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
			defer ClearAll()
			err := UpdateExporter(test.ops, TestLogger(t))
			if err != nil {
				fmt.Println(err)
			}
			mConfig := getCurMetricsConfig()
			if *mConfig != *oldConfig {
				t.Error("mConfig should not change")
			}
		})
	}
}

func TestUpdateExporter_doesNotCreateExporter(t *testing.T) {
	setCurMetricsConfig(nil)
	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			defer ClearAll()
			UpdateExporter(test.ops, TestLogger(t))
			mConfig := getCurMetricsConfig()
			if mConfig != nil {
				t.Error("mConfig should not be created")
			}
		})
	}
}

func TestMetricsOptionsToJson(t *testing.T) {
	testCases := map[string]struct {
		opts    *ExporterOptions
		want    string
		wantErr string
	}{
		"nil": {
			opts:    nil,
			want:    "",
			wantErr: "json options string is empty",
		},
		"standardConfig": {
			opts: &ExporterOptions{
				Domain:         "domain",
				Component:      "component",
				PrometheusPort: 9090,
				ConfigMap: map[string]string{
					"foo":   "bar",
					"boosh": "kakow",
				},
			},
			want: `{"Domain":"domain","Component":"component","PrometheusPort":9090,"ConfigMap":{"boosh":"kakow","foo":"bar"},"StackdriverConfigMap":null}`,
		},
		"stackdriverConfig": {
			opts: &ExporterOptions{
				Domain:         "domain",
				Component:      "component",
				PrometheusPort: 9090,
				StackdriverConfigMap: map[string]string{
					"foo":   "bar",
					"boosh": "kakow",
				},
			},
			want: `{"Domain":"domain","Component":"component","PrometheusPort":9090,"ConfigMap":null,"StackdriverConfigMap":{"boosh":"kakow","foo":"bar"}}`,
		},
		"allConfig": {
			opts: &ExporterOptions{
				Domain:         "domain",
				Component:      "component",
				PrometheusPort: 9090,
				ConfigMap: map[string]string{
					"apple":  "orange",
					"banana": "pear",
				},
				StackdriverConfigMap: map[string]string{
					"foo":   "bar",
					"boosh": "kakow",
				},
			},
			want: `{"Domain":"domain","Component":"component","PrometheusPort":9090,"ConfigMap":{"apple":"orange","banana":"pear"},"StackdriverConfigMap":{"boosh":"kakow","foo":"bar"}}`,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			jsonOpts, err := MetricsOptionsToJson(tc.opts)
			if err != nil {
				t.Errorf("error while converting metrics config to json: %v", err)
			}
			// Test to json.
			{
				want := tc.want
				got := jsonOpts
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("unexpected (-want, +got) = %v", diff)
					t.Log(got)
				}
			}
			// Test to options.
			{
				want := tc.opts
				got, gotErr := JsonToMetricsOptions(jsonOpts)

				if gotErr != nil {
					if diff := cmp.Diff(tc.wantErr, gotErr.Error()); diff != "" {
						t.Errorf("unexpected err (-want, +got) = %v", diff)
					}
				} else if tc.wantErr != "" {
					t.Errorf("expected err %v", tc.wantErr)
				}

				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("unexpected (-want, +got) = %v", diff)
					t.Log(got)
				}
			}
		})
	}
}
