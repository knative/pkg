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

package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewTrackerProviderProtocols(t *testing.T) {
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
		name:    "bad protocol",
		c:       Config{Protocol: "bad"},
		wantErr: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := NewTracerProvider(ctx, tc.c)

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
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
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

func TestTracerProviderShutdown(t *testing.T) {
	want := errors.New("some error")
	invoked := false
	shutdown := func(context.Context) error {
		invoked = true
		return want
	}

	p := TracerProvider{shutdown: shutdown}
	got := p.Shutdown(context.Background())

	if !invoked {
		t.Fatal("expected shutdown to be invoked")
	}

	if !errors.Is(got, want) {
		t.Error("unexpected error (-want +got): ", cmp.Diff(want, got))
	}
}

func TestSampleFor(t *testing.T) {
	cfg := Config{
		SamplingRate: 0.85,
	}

	cases := []struct {
		name           string
		env            map[string]string
		wantNilSampler bool
		wantErr        bool
	}{{
		name: "sampler override",
		env: map[string]string{
			"OTEL_TRACES_SAMPLER": "some-sampler",
		},
		wantNilSampler: true,
	}, {
		name:           "standard sampler",
		wantNilSampler: false,
	}, {
		name:           "sample arg override",
		wantNilSampler: false,
		env: map[string]string{
			"OTEL_TRACES_SAMPLER_ARG": "1.0",
		},
	}, {
		name:           "sample arg override - bad input",
		wantNilSampler: true,
		wantErr:        true,
		env: map[string]string{
			"OTEL_TRACES_SAMPLER_ARG": "bad-ratio",
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			got, err := sampleFor(cfg)
			if tc.wantNilSampler && got != nil {
				t.Error("expected a nil sampler")
			} else if !tc.wantNilSampler && got == nil {
				t.Error("expected a non-nil sampler")
			}

			if tc.wantErr && err == nil {
				t.Error("expected an error")
			} else if !tc.wantErr && err != nil {
				t.Error("unexpected error")
			}
		})
	}
}
