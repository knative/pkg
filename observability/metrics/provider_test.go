/*
Copyright 2025 The Knative Authors

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
)

func TestNewMeterProviderProtocols(t *testing.T) {
	cases := []struct {
		name    string
		c       Config
		wantErr bool
	}{{
		name: "grpc",
		c:    Config{Protocol: ProtocolGRPC},
	}, {
		name: "none",
		c:    Config{Protocol: ProtocolNone},
	}, {
		name: "http",
		c:    Config{Protocol: ProtocolHTTPProtobuf},
	}, {
		name: "prometheus",
		c:    Config{Protocol: ProtocolPrometheus},
	}, {
		name:    "bad protocol",
		c:       Config{Protocol: "bad"},
		wantErr: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := NewMeterProvider(ctx, tc.c)

			if !tc.wantErr && err != nil {
				t.Error("unexpected failed", err)
			} else if tc.wantErr && err == nil {
				t.Error("expected failure")
			}
		})
	}
}

func TestEndpointFor(t *testing.T) {
	cases := []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
	}

	optFunc := func(string) *string {
		panic("unexpected call")
	}

	for _, envKey := range cases {
		t.Run("override "+envKey, func(t *testing.T) {
			t.Setenv(envKey, "https://override.example.com")

			opt := endpointFor(Config{Endpoint: "https://otel.example.com"}, optFunc)

			if opt != nil {
				t.Error("expected the option to not be present when 'OTEL_EXPORTER_OTLP_ENDPOINT' is set")
			}
		})
	}

	t.Run("normal", func(t *testing.T) {
		optFunc := func(string) *string {
			result := "result"
			return &result
		}
		opt := endpointFor(Config{Endpoint: "https://otel.example.com"}, optFunc)
		if *opt != "result" {
			t.Error("expected option to work when no env vars are present")
		}
	})
}

func TestIntervalFor(t *testing.T) {
	cases := []string{
		"OTEL_METRIC_EXPORT_INTERVAL",
	}

	for _, envKey := range cases {
		t.Run("override "+envKey, func(t *testing.T) {
			t.Setenv(envKey, "https://override.example.com")

			opt := intervalFor(Config{ExportInterval: time.Second})

			if opt != nil {
				t.Error("expected the option to not be present when 'OTEL_EXPORTER_OTLP_ENDPOINT' is set")
			}
		})
	}

	t.Run("normal", func(t *testing.T) {
		opt := intervalFor(Config{ExportInterval: time.Second})

		if opt == nil {
			t.Error("expected the interval option be present when env vars are not set")
		}
	})
}
