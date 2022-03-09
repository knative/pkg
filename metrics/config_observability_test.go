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
			RequestMetricsBackend:  "opencensus",
		},
		data: map[string]string{
			EnableProbeReqLogKey:                          "true",
			"logging.enable-var-log-collection":           "true",
			ReqLogTemplateKey:                             `{"requestMethod": "{{.Request.Method}}"}`,
			"logging.revision-url-template":               "https://logging.io",
			EnableReqLogKey:                               "true",
			"metrics.request-metrics-backend-destination": "opencensus",
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
			EnableProbeReqLogKey:            "true",
			EnableReqLogKey:                 "true",
			"logging.revision-url-template": "https://logging.io",
		},
		wantConfig: func() *ObservabilityConfig {
			oc := defaultConfig()
			oc.EnableProbeRequestLog = true
			oc.EnableRequestLog = true
			oc.LoggingURLTemplate = "https://logging.io"
			return oc
		}(),
	}, {
		name: "observability configuration with request log and template not set",
		wantConfig: func() *ObservabilityConfig {
			oc := defaultConfig()
			oc.RequestLogTemplate = ""
			oc.EnableProbeRequestLog = true
			return oc
		}(),
		data: map[string]string{
			EnableProbeReqLogKey: "true",
			EnableReqLogKey:      "false", // Explicit default.
			ReqLogTemplateKey:    "",
		},
	}, {
		name:    "observability configuration with request log set and template not set",
		wantErr: true,
		data: map[string]string{
			EnableProbeReqLogKey:                "true",
			EnableReqLogKey:                     "true",
			"logging.enable-var-log-collection": "true",
			ReqLogTemplateKey:                   "",
		},
	}, {
		name: "observability configuration with request log not set and with template set",
		wantConfig: func() *ObservabilityConfig {
			oc := defaultConfig()
			oc.EnableProbeRequestLog = true
			oc.EnableVarLogCollection = true
			oc.RequestLogTemplate = `{"requestMethod": "{{.Request.Method}}"}`
			return oc
		}(),
		data: map[string]string{
			EnableProbeReqLogKey:                "true",
			"logging.enable-var-log-collection": "true",
			ReqLogTemplateKey:                   `{"requestMethod": "{{.Request.Method}}"}`,
		},
	}, {
		name: "observability configuration with collector address",
		wantConfig: func() *ObservabilityConfig {
			oc := defaultConfig()
			oc.RequestMetricsBackend = "opencensus"
			oc.MetricsCollectorAddress = "otel:55678"
			return oc
		}(),
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
	t.Setenv(configMapNameEnv, "")
	if got, want := ConfigMapName(), "config-observability"; got != want {
		t.Errorf("ConfigMapName = %q, want: %q", got, want)
	}
	t.Setenv(configMapNameEnv, "why-is-living-so-hard?")
	if got, want := ConfigMapName(), "why-is-living-so-hard?"; got != want {
		t.Errorf("ConfigMapName = %q, want: %q", got, want)
	}
}
