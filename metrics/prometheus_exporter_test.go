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
	"context"
	"testing"
	"time"

	. "knative.dev/pkg/logging/testing"
)

func TestNewPrometheusExporter(t *testing.T) {
	testCases := []struct {
		name         string
		config       metricsConfig
		expectedAddr string
	}{{
		name: "port 9090",
		config: metricsConfig{
			domain:             "does not matter",
			component:          testComponent,
			backendDestination: prometheus,
			prometheusPort:     9090,
			prometheusHost:     "0.0.0.0",
		},
		expectedAddr: "0.0.0.0:9090",
	}, {
		name: "port 9091",
		config: metricsConfig{
			domain:             "does not matter",
			component:          testComponent,
			backendDestination: prometheus,
			prometheusPort:     9091,
			prometheusHost:     "127.0.0.1",
		},
		expectedAddr: "127.0.0.1:9091",
	}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e, _, err := newPrometheusExporter(&tc.config, TestLogger(t))
			if err != nil {
				t.Error(err)
			}
			if e == nil {
				t.Fatal("expected a non-nil metrics exporter")
			}
			expectPromSrv(t, tc.expectedAddr)
		})
	}
}

func TestNewPrometheusExporter_fromEnv(t *testing.T) {
	exporterOptions := ExporterOptions{
		ConfigMap: map[string]string{},
		Domain:    "does not matter",
		Component: testComponent,
	}
	testCases := []struct {
		name                   string
		prometheusPortVarName  string
		prometheusPortVarValue string
		prometheusHostVarName  string
		prometheusHostVarValue string
		ops                    ExporterOptions
		expectedAddr           string
	}{{
		name:                   "port from env var with no host set",
		prometheusPortVarName:  prometheusPortEnvName,
		prometheusPortVarValue: "9092",
		ops:                    exporterOptions,
		expectedAddr:           "0.0.0.0:9092",
	}, {
		name:                   "no port set with host from env var",
		prometheusHostVarName:  prometheusHostEnvName,
		prometheusHostVarValue: "127.0.0.1",
		ops:                    exporterOptions,
		expectedAddr:           "127.0.0.1:9090",
	}, {
		name:                   "port set and host set to empty string",
		prometheusPortVarName:  prometheusPortEnvName,
		prometheusPortVarValue: "",
		prometheusHostVarName:  prometheusHostEnvName,
		prometheusHostVarValue: "",
		ops:                    exporterOptions,
		expectedAddr:           "0.0.0.0:9090",
	}, {
		name:         "no port or host from the env",
		ops:          exporterOptions,
		expectedAddr: "0.0.0.0:9090",
	}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.prometheusPortVarName != "" {
				t.Setenv(tc.prometheusPortVarName, tc.prometheusPortVarValue)
			}
			if tc.prometheusHostVarName != "" {
				t.Setenv(tc.prometheusHostVarName, tc.prometheusHostVarValue)
			}
			mc, err := createMetricsConfig(context.Background(), tc.ops)
			if err != nil {
				t.Fatal("Failed to create the metrics config:", err)
			}
			e, _, err := newPrometheusExporter(mc, TestLogger(t))
			if err != nil {
				t.Fatal("Failed to create a new Prometheus exporter:", err)
			}
			if e == nil {
				t.Fatal("expected a non-nil metrics exporter")
			}
			expectPromSrv(t, tc.expectedAddr)
		})
	}
}

func expectPromSrv(t *testing.T, expectedAddr string) {
	time.Sleep(200 * time.Millisecond)
	srv := getCurPromSrv()
	if srv == nil {
		t.Fatal("expected a server for prometheus exporter")
	}
	if got, want := srv.Addr, expectedAddr; got != want {
		t.Errorf("metrics port addresses diff, got=%v, want=%v", got, want)
	}
}
