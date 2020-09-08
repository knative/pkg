/*
Copyright 2019 The Knative Authors.

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

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"

	_ "knative.dev/pkg/system/testing"
)

func TestObservabilityConfiguration(t *testing.T) {
	observabilityConfigTests := []struct {
		name       string
		data       map[string]string
		wantErr    bool
		wantConfig *ObservabilityConfig
	}{{
		name: "observability configuration with all inputs",
		wantConfig: &ObservabilityConfig{
			EnableProbeRequestLog:  true,
			EnableProfiling:        true,
			EnableVarLogCollection: true,
			EnableRequestLog:       true,
			LoggingURLTemplate:     "https://logging.io",
			RequestLogTemplate:     `{"requestMethod": "{{.Request.Method}}"}`,
			RequestMetricsBackend:  "stackdriver",
		},
		data: map[string]string{
			"logging.enable-probe-request-log":            "true",
			"logging.enable-var-log-collection":           "true",
			ReqLogTemplateKey:                             `{"requestMethod": "{{.Request.Method}}"}`,
			"logging.revision-url-template":               "https://logging.io",
			EnableReqLogKey:                               "true",
			"metrics.request-metrics-backend-destination": "stackdriver",
			"profiling.enable":                            "true",
		},
	}, {
		name:       "observability config with no map",
		wantConfig: defaultConfig(),
	}, {
		name:    "invalid request log template",
		wantErr: true,
		data: map[string]string{
			ReqLogTemplateKey: `{{ something }}`,
		},
	}, {
		name: "observability configuration with request log set and template default",
		data: map[string]string{
			"logging.enable-probe-request-log":            "true",
			EnableReqLogKey:                               "true",
			"logging.enable-var-log-collection":           "true",
			"logging.revision-url-template":               "https://logging.io",
			"metrics.request-metrics-backend-destination": "stackdriver",
			"profiling.enable":                            "true",
		},
		wantConfig: &ObservabilityConfig{
			EnableProbeRequestLog:  true,
			EnableProfiling:        true,
			EnableRequestLog:       true,
			EnableVarLogCollection: true,
			LoggingURLTemplate:     "https://logging.io",
			RequestLogTemplate:     DefaultRequestLogTemplate,
			RequestMetricsBackend:  "stackdriver",
		},
	}, {
		name: "observability configuration with request log and template not set",
		wantConfig: &ObservabilityConfig{
			EnableProbeRequestLog:  true,
			EnableProfiling:        true,
			EnableVarLogCollection: true,
			LoggingURLTemplate:     "https://logging.io",
			RequestMetricsBackend:  "stackdriver",
		},
		data: map[string]string{
			"logging.enable-probe-request-log":            "true",
			EnableReqLogKey:                               "false",
			"logging.enable-var-log-collection":           "true",
			ReqLogTemplateKey:                             "",
			"logging.revision-url-template":               "https://logging.io",
			"metrics.request-metrics-backend-destination": "stackdriver",
			"profiling.enable":                            "true",
		},
	}, {
		name:    "observability configuration with request log set and template not set",
		wantErr: true,
		data: map[string]string{
			"logging.enable-probe-request-log":            "true",
			EnableReqLogKey:                               "true",
			"logging.enable-var-log-collection":           "true",
			ReqLogTemplateKey:                             "",
			"logging.revision-url-template":               "https://logging.io",
			"metrics.request-metrics-backend-destination": "stackdriver",
			"profiling.enable":                            "true",
		},
	}, {
		name: "observability configuration with request log not set and with template set",
		wantConfig: &ObservabilityConfig{
			EnableProbeRequestLog:  true,
			EnableProfiling:        true,
			EnableVarLogCollection: true,
			LoggingURLTemplate:     "https://logging.io",
			RequestLogTemplate:     `{"requestMethod": "{{.Request.Method}}"}`,
			RequestMetricsBackend:  "stackdriver",
		},
		data: map[string]string{
			"logging.enable-probe-request-log":            "true",
			"logging.enable-var-log-collection":           "true",
			ReqLogTemplateKey:                             `{"requestMethod": "{{.Request.Method}}"}`,
			"logging.revision-url-template":               "https://logging.io",
			"metrics.request-metrics-backend-destination": "stackdriver",
			"profiling.enable":                            "true",
		},
	}, {
		name: "observability configuration with collector address",
		wantConfig: &ObservabilityConfig{
			LoggingURLTemplate:      DefaultLogURLTemplate,
			RequestLogTemplate:      DefaultRequestLogTemplate,
			RequestMetricsBackend:   "opencensus",
			MetricsCollectorAddress: "otel:55678",
		},
		data: map[string]string{
			"metrics.request-metrics-backend-destination": "opencensus",
			"metrics.opencensus-address":                  "otel:55678",
		},
	}}

	for _, tt := range observabilityConfigTests {
		t.Run(tt.name, func(t *testing.T) {
			obsConfig, err := NewObservabilityConfigFromConfigMap(&corev1.ConfigMap{
				Data: tt.data,
			})

			if (err != nil) != tt.wantErr {
				t.Fatalf("NewObservabilityFromConfigMap() error = %v, WantErr %v", err, tt.wantErr)
			}

			if got, want := obsConfig, tt.wantConfig; !cmp.Equal(got, want) {
				t.Errorf("Got = %v, want: %v, diff(-want,+got)\n%s", got, want, cmp.Diff(want, got))
			}
		})
	}
}

func TestConfigMapName(t *testing.T) {
	if got, want := ConfigMapName(), "config-observability"; got != want {
		t.Errorf("ConfigMapName = %q, want: %q", got, want)
	}
	t.Cleanup(func() {
		os.Unsetenv(configMapNameEnv)
	})
	os.Setenv(configMapNameEnv, "")
	if got, want := ConfigMapName(), "config-observability"; got != want {
		t.Errorf("ConfigMapName = %q, want: %q", got, want)
	}
	os.Setenv(configMapNameEnv, "why-is-living-so-hard?")
	if got, want := ConfigMapName(), "why-is-living-so-hard?"; got != want {
		t.Errorf("ConfigMapName = %q, want: %q", got, want)
	}
}
