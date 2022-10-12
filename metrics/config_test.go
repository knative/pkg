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
	"context"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	. "knative.dev/pkg/logging/testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	metricsDomain = "knative.dev/project"
)

var (
	errorTests = []struct {
		name        string
		ops         ExporterOptions
		expectedErr string
	}{{
		name: "empty config",
		ops: ExporterOptions{
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedErr: "metrics config map cannot be empty",
	}, {
		name: "unsupportedBackend",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: "unsupported",
			},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedErr: `unsupported metrics backend value "unsupported"`,
	}, {
		name: "emptyDomain",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(prometheus),
			},
			Component: testComponent,
		},
		expectedErr: "metrics domain cannot be empty",
	}, {
		name: "invalidComponent",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(openCensus),
			},
			Domain: metricsDomain,
		},
		expectedErr: "metrics component name cannot be empty",
	}, {
		name: "invalidReportingPeriod",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(openCensus),
				reportingPeriodKey:    "test",
			},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedErr: "invalid " + reportingPeriodKey + ` value "test"`,
	}, {
		name: "invalidOpenCensusSecuritySetting",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(openCensus),
				collectorSecureKey:    "yep",
			},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedErr: "invalid " + collectorSecureKey + ` value "yep"`,
	}, {
		name: "tooSmallPrometheusPort",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(prometheus),
			},
			Domain:         metricsDomain,
			Component:      testComponent,
			PrometheusPort: 1023,
		},
		expectedErr: "invalid port 1023, should be between 1024 and 65535",
	}, {
		name: "tooBigPrometheusPort",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(prometheus),
			},
			Domain:         metricsDomain,
			Component:      testComponent,
			PrometheusPort: 65536,
		},
		expectedErr: "invalid port 65536, should be between 1024 and 65535",
	}}

	successTests = []struct {
		name                string
		ops                 ExporterOptions
		expectedConfig      metricsConfig
		expectedNewExporter bool // Whether the config requires a new exporter compared to previous test case
	}{{
		name: "backendKeyMissing",
		ops: ExporterOptions{
			ConfigMap: map[string]string{},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    5 * time.Second,
			prometheusPort:     defaultPrometheusPort,
			prometheusHost:     defaultPrometheusHost,
		},
		expectedNewExporter: true,
	}, {
		name: "validOpenCensusSettings",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(openCensus),
				collectorAddressKey:   "localhost:55678",
				collectorSecureKey:    "true",
			},
			Domain:    metricsDomain,
			Component: testComponent,
			Secrets: fakeSecretList(corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "opencensus",
				},
				Data: map[string][]byte{
					"client-cert.pem": {},
					"client-key.pem":  {},
				},
			}).Get,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: openCensus,
			reportingPeriod:    time.Minute,
			collectorAddress:   "localhost:55678",
			requireSecure:      true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "opencensus",
				},
				Data: map[string][]byte{
					"client-cert.pem": {},
					"client-key.pem":  {},
				},
			},
		},
		expectedNewExporter: true,
	}, {
		name: "validPrometheus",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(prometheus),
			},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    5 * time.Second,
			prometheusPort:     defaultPrometheusPort,
			prometheusHost:     defaultPrometheusHost,
		},
		expectedNewExporter: true,
	}, {
		name: "overriddenReportingPeriodPrometheus",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(prometheus),
				reportingPeriodKey:    "12",
			},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    12 * time.Second,
			prometheusPort:     defaultPrometheusPort,
			prometheusHost:     defaultPrometheusHost,
		},
		expectedNewExporter: true,
	}, {
		name: "overriddenReportingPeriodOpencensus",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(openCensus),
				reportingPeriodKey:    "8",
			},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: openCensus,
			reportingPeriod:    8 * time.Second,
		},
		expectedNewExporter: true,
	}, {
		name: "emptyReportingPeriodPrometheus",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(prometheus),
				reportingPeriodKey:    "",
			},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    5 * time.Second,
			prometheusPort:     defaultPrometheusPort,
			prometheusHost:     defaultPrometheusHost,
		},
		expectedNewExporter: true,
	}, {
		name: "overridePrometheusPort",
		ops: ExporterOptions{
			ConfigMap: map[string]string{
				BackendDestinationKey: string(prometheus),
			},
			Domain:         metricsDomain,
			Component:      testComponent,
			PrometheusPort: 9091,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    5 * time.Second,
			prometheusPort:     9091,
			prometheusHost:     defaultPrometheusHost,
		},
		expectedNewExporter: true,
	}}
)

func TestGetMetricsConfig(t *testing.T) {
	ctx := context.Background()
	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			_, err := createMetricsConfig(ctx, test.ops)
			if err == nil || err.Error() != test.expectedErr {
				t.Errorf("Err = %v, want: %v", err, test.expectedErr)
			}
		})
	}

	for _, test := range successTests {
		t.Run(test.name, func(t *testing.T) {
			mc, err := createMetricsConfig(ctx, test.ops)
			if err != nil {
				t.Errorf("Wanted valid config %v, got error %v", test.expectedConfig, err)
			}
			if diff := cmp.Diff(test.expectedConfig, *mc, cmp.AllowUnexported(*mc)); diff != "" {
				t.Errorf("Invalid config (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetMetricsConfig_fromEnv(t *testing.T) {
	successTests := []struct {
		name           string
		varName        string
		varValue       string
		ops            ExporterOptions
		expectedConfig metricsConfig
	}{{
		name:     "OpenCensus backend from env, no config",
		varName:  defaultBackendEnvName,
		varValue: string(openCensus),
		ops: ExporterOptions{
			ConfigMap: map[string]string{},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: openCensus,
			reportingPeriod:    time.Minute,
		},
	}, {
		name:     "OpenCensus sbackend from env, Prometheus backend from config",
		varName:  defaultBackendEnvName,
		varValue: string(openCensus),
		ops: ExporterOptions{
			ConfigMap: map[string]string{BackendDestinationKey: string(prometheus)},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    5 * time.Second,
			prometheusPort:     defaultPrometheusPort,
			prometheusHost:     defaultPrometheusHost,
		},
	}, {
		name:     "PrometheusPort from env",
		varName:  prometheusPortEnvName,
		varValue: "9999",
		ops: ExporterOptions{
			ConfigMap: map[string]string{},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    5 * time.Second,
			prometheusPort:     9999,
			prometheusHost:     defaultPrometheusHost,
		},
	}}

	failureTests := []struct {
		name                string
		varName             string
		varValue            string
		ops                 ExporterOptions
		expectedErrContains string
	}{{
		name:     "Invalid PrometheusPort from env",
		varName:  prometheusPortEnvName,
		varValue: strconv.Itoa(math.MaxUint16 + 1),
		ops: ExporterOptions{
			ConfigMap: map[string]string{},
			Domain:    metricsDomain,
			Component: testComponent,
		},
		expectedErrContains: "value out of range",
	}}

	ctx := context.Background()

	for _, test := range successTests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.varName, test.varValue)

			mc, err := createMetricsConfig(ctx, test.ops)
			if err != nil {
				t.Errorf("Wanted valid config %v, got error %v", test.expectedConfig, err)
			}
			if diff := cmp.Diff(test.expectedConfig, *mc, cmp.AllowUnexported(*mc)); diff != "" {
				t.Errorf("Invalid config (-want +got):\n%s", diff)
			}
		})
	}

	for _, test := range failureTests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.varName, test.varValue)

			mc, err := createMetricsConfig(ctx, test.ops)
			if mc != nil {
				t.Error("Wanted no config, got", mc)
			}
			if err == nil || !strings.Contains(err.Error(), test.expectedErrContains) {
				t.Errorf("Wanted err to contain: %q, got: %v", test.expectedErrContains, err)
			}
		})
	}
}

func TestIsNewExporterRequiredFromNilConfig(t *testing.T) {
	ctx := context.Background()

	for _, test := range successTests {
		t.Run(test.name, func(t *testing.T) {
			setCurMetricsConfig(nil)
			mc, err := createMetricsConfig(ctx, test.ops)
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
}

func TestIsNewExporterRequired(t *testing.T) {
	tests := []struct {
		name                string
		oldConfig           metricsConfig
		newConfig           metricsConfig
		newExporterRequired bool
	}{{
		name: "changeMetricsBackend",
		oldConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: openCensus,
			reportingPeriod:    time.Minute,
		},
		newConfig: metricsConfig{
			domain:             metricsDomain,
			component:          testComponent,
			backendDestination: prometheus,
			reportingPeriod:    time.Minute,
		},
		newExporterRequired: true,
	}, {
		name: "changeComponent",
		oldConfig: metricsConfig{
			domain:    metricsDomain,
			component: "component1",
		},
		newConfig: metricsConfig{
			domain:    metricsDomain,
			component: "component2",
		},
		newExporterRequired: false,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setCurMetricsConfig(&test.oldConfig)
			actualNewExporterRequired := isNewExporterRequired(&test.newConfig)
			if test.newExporterRequired != actualNewExporterRequired {
				t.Errorf("isNewExporterRequired returned incorrect value. Expected: [%v], Got: [%v]. Old config: [%v], New config: [%v]", test.newExporterRequired, actualNewExporterRequired, test.oldConfig, test.newConfig)
			}
		})
	}
}

func TestUpdateExporter(t *testing.T) {
	setCurMetricsConfig(nil)
	oldConfig := getCurMetricsConfig()
	ctx := context.Background()

	for _, test := range successTests[1:] {
		t.Run(test.name, func(t *testing.T) {
			UpdateExporter(ctx, test.ops, TestLogger(t))
			mConfig := getCurMetricsConfig()
			if mConfig == oldConfig {
				t.Error("Expected metrics config change")
			}
			if diff := cmp.Diff(test.expectedConfig, *mConfig, cmp.AllowUnexported(*mConfig)); diff != "" {
				t.Errorf("Invalid config (-want +got):\n%s", diff)
			}
			oldConfig = mConfig
		})
	}

	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			UpdateExporter(ctx, test.ops, TestLogger(t))
			mConfig := getCurMetricsConfig()
			if mConfig != oldConfig {
				t.Error("mConfig should not change")
			}
		})
	}
}

func TestUpdateExporterFromConfigMapWithOpts(t *testing.T) {
	setCurMetricsConfig(nil)
	oldConfig := getCurMetricsConfig()
	ctx := context.Background()

	for _, test := range successTests[1:] {
		t.Run(test.name, func(t *testing.T) {
			opts := ExporterOptions{
				Component:      test.ops.Component,
				Domain:         test.ops.Domain,
				PrometheusPort: test.ops.PrometheusPort,
				Secrets:        test.ops.Secrets,
			}
			updateFunc, err := UpdateExporterFromConfigMapWithOpts(ctx, opts, TestLogger(t))
			if err != nil {
				t.Error("failed to call UpdateExporterFromConfigMapWithOpts:", err)
			}
			updateFunc(&corev1.ConfigMap{Data: test.ops.ConfigMap})
			mConfig := getCurMetricsConfig()
			if mConfig == oldConfig {
				t.Error("Expected metrics config change")
			}
			if diff := cmp.Diff(test.expectedConfig, *mConfig, cmp.AllowUnexported(*mConfig)); diff != "" {
				t.Errorf("Invalid config (-want +got):\n%s", diff)
			}
			oldConfig = mConfig
		})
	}

	t.Run("ConfigMapSetErr", func(t *testing.T) {
		opts := ExporterOptions{
			Component:      testComponent,
			Domain:         metricsDomain,
			PrometheusPort: defaultPrometheusPort,
			ConfigMap:      map[string]string{"some": "data"},
		}
		_, err := UpdateExporterFromConfigMapWithOpts(ctx, opts, TestLogger(t))
		if err == nil {
			t.Error("got err=nil want err")
		}
	})

	t.Run("MissingComponentErr", func(t *testing.T) {
		opts := ExporterOptions{
			Component:      "",
			Domain:         metricsDomain,
			PrometheusPort: defaultPrometheusPort,
		}
		_, err := UpdateExporterFromConfigMapWithOpts(ctx, opts, TestLogger(t))
		if err == nil {
			t.Error("got err=nil want err")
		}
	})
}

func TestUpdateExporter_doesNotCreateExporter(t *testing.T) {
	setCurMetricsConfig(nil)
	ctx := context.Background()

	for _, test := range errorTests {
		t.Run(test.name, func(t *testing.T) {
			UpdateExporter(ctx, test.ops, TestLogger(t))
			mConfig := getCurMetricsConfig()
			if mConfig != nil {
				t.Error("mConfig should not be created")
			}
		})
	}
}

func TestMetricsOptions(t *testing.T) {
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
		"happy": {
			opts: &ExporterOptions{
				Domain:         "domain",
				Component:      "component",
				PrometheusPort: 9090,
				PrometheusHost: "0.0.0.0",
				ConfigMap: map[string]string{
					"foo":   "bar",
					"boosh": "kakow",
				},
			},
			want: `{"Domain":"domain","Component":"component","PrometheusPort":9090,"PrometheusHost":"0.0.0.0","ConfigMap":{"boosh":"kakow","foo":"bar"}}`,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			jsonOpts, err := OptionsToJSON(tc.opts)
			if err != nil {
				t.Error("error while converting metrics config to json:", err)
			}
			// Test to json.
			{
				want := tc.want
				got := jsonOpts
				if diff := cmp.Diff(want, got); diff != "" {
					t.Error("unexpected (-want, +got) =", diff)
					t.Log(got)
				}
			}
			// Test to options.
			{
				want := tc.opts
				got, gotErr := JSONToOptions(jsonOpts)

				if gotErr != nil {
					if diff := cmp.Diff(tc.wantErr, gotErr.Error()); diff != "" {
						t.Error("unexpected err (-want, +got) =", diff)
					}
				} else if tc.wantErr != "" {
					t.Error("expected err", tc.wantErr)
				}

				if diff := cmp.Diff(want, got); diff != "" {
					t.Error("unexpected (-want, +got) =", diff)
					t.Log(got)
				}
			}
		})
	}
}
