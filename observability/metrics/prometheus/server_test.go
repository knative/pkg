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

package prometheus

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewServerWithOptions(t *testing.T) {
	s, err := NewServer(
		WithHost("127.0.0.1"),
		WithPort("57289"),
	)
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

	got := s.http.Addr
	want := "127.0.0.1:57289"

	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("unexpected diff (-want, +got) : ", diff)
	}
}

func TestNewServerEnvOverride(t *testing.T) {
	t.Setenv(prometheusHostEnvName, "0.0.0.0")
	t.Setenv(prometheusPortEnvName, "1028")

	s, err := NewServer(
		WithHost("127.0.0.1"),
		WithPort("57289"),
	)
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

	got := s.http.Addr
	want := "0.0.0.0:1028"

	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("unexpected diff (-want, +got) : ", diff)
	}
}

func TestNewServerFailure(t *testing.T) {
	if _, err := NewServer(WithPort("1000000")); err == nil {
		t.Error("expected port parsing to fail")
	}

	if _, err := NewServer(WithPort("80")); err == nil {
		t.Error("expected below port range to fail")
	}

	if _, err := NewServer(WithPort("65536")); err == nil {
		t.Error("expected above port range to fail")
	}
}
