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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewFromMap(t *testing.T) {
	got, err := NewFromMap(nil)
	want := DefaultConfig()

	if err != nil {
		t.Error("unexpected error:", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("unexpected diff (-want +got): ", diff)
	}
}

func TestNewFromMapBadInput(t *testing.T) {
	cases := []struct {
		name string
		m    map[string]string
	}{{
		name: "unexpected endpoint set",
		m:    map[string]string{"tracing-endpoint": "https://blah.example.com"},
	}, {
		name: "missing endpoint",
		m:    map[string]string{"tracing-protocol": "grpc"},
	}, {
		name: "unsupported protocol",
		m:    map[string]string{"tracing-protocol": "bad-protocol"},
	}, {
		name: "bad sampling rate - less than zero",
		m:    map[string]string{"tracing-sampling-rate": "-1"},
	}, {
		name: "bad sampling rate - greater than one",
		m:    map[string]string{"tracing-sampling-rate": "1.1"},
	}, {
		name: "bad sampling rate - not a float",
		m:    map[string]string{"tracing-sampling-rate": "bad-rate"},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewFromMap(tc.m); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestNewFromMapWithPrefix(t *testing.T) {
	got, err := NewFromMapWithPrefix("request-", map[string]string{
		"request-tracing-protocol":      ProtocolGRPC,
		"request-tracing-endpoint":      "https://blah.example.com",
		"request-tracing-sampling-rate": "0.8",
	})
	if err != nil {
		t.Error("unexpected error", err)
	}

	want := Config{
		Protocol:     ProtocolGRPC,
		Endpoint:     "https://blah.example.com",
		SamplingRate: 0.8,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("unexpected diff (-want +got): ", diff)
	}
}
