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

	"github.com/google/go-cmp/cmp"
	"knative.dev/pkg/ptr"
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
		name: "http - bad URL",
		c: Config{
			Protocol: ProtocolHTTPProtobuf,
			Endpoint: "://hello",
		},
		wantErr: true,
	}, {
		name: "prometheus",
		c:    Config{Protocol: ProtocolPrometheus},
	}, {
		name: "prometheus push with path in endpoint",
		c: Config{
			Protocol: ProtocolHTTPProtobuf,
			Endpoint: "http://example.com:9090/api/v1/otlp/v1/metrics",
		},
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

func TestPrometheusEndpoint(t *testing.T) {
	cases := []struct {
		name    string
		c       Config
		wantErr bool
	}{{
		name: "ipv4",
		c: Config{
			Protocol: ProtocolPrometheus,
			Endpoint: "0.0.0.0:9000", // IPv4 only on port 9000
		},
	}, {
		name: "ipv6",
		c: Config{
			Protocol: ProtocolPrometheus,
			Endpoint: "[::]:9000", // IPv6 only on port 9000
		},
	}, {
		name: "bad endpoint",
		c: Config{
			Protocol: ProtocolPrometheus,
			Endpoint: "[:::9000", // IPv6 only on port 9000
		},
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

			opt, err := endpointFor(Config{Endpoint: "https://otel.example.com"}, optFunc)
			if err != nil {
				t.Fatal("unexpected err", err)
			}

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
		opt, err := endpointFor(Config{Endpoint: "https://otel.example.com"}, optFunc)
		if err != nil {
			t.Fatal("unexpected err", err)
		}
		if *opt != "result" {
			t.Error("expected option to work when no env vars are present")
		}
	})

	t.Run("missing scheme - defaults to https", func(t *testing.T) {
		optFunc := func(r string) *string {
			return &r
		}
		got, err := endpointFor(Config{Endpoint: "otel.example.com:8080"}, optFunc)
		if err != nil {
			t.Fatal("unexpected err", err)
		}

		want := ptr.String("https://otel.example.com:8080")
		if diff := cmp.Diff(want, got); diff != "" {
			t.Error("expected option to work when no env vars are present: (-want +got): ", diff)
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
