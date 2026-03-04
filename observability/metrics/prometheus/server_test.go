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
	"context"
	"crypto/tls"
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

func TestNewServerWithTLSConfig(t *testing.T) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
	}

	s, err := NewServer(WithTLSConfig(tlsConfig))
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

	if s.http.TLSConfig == nil {
		t.Error("expected TLSConfig to be set on http.Server")
	}

	if s.http.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected MinVersion to be TLS 1.3, got %v", s.http.TLSConfig.MinVersion)
	}
}

func TestNewServerWithTLSCertFiles(t *testing.T) {
	s, err := NewServer(WithTLSCertFiles("/path/to/cert.pem", "/path/to/key.pem"))
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

	if s.certFile != "/path/to/cert.pem" {
		t.Errorf("expected certFile to be /path/to/cert.pem, got %s", s.certFile)
	}

	if s.keyFile != "/path/to/key.pem" {
		t.Errorf("expected keyFile to be /path/to/key.pem, got %s", s.keyFile)
	}
}

func TestNewServerWithTLSEnvVars(t *testing.T) {
	t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
	t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")

	s, err := NewServer()
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

	if s.certFile != "/etc/tls/tls.crt" {
		t.Errorf("expected certFile to be /etc/tls/tls.crt, got %s", s.certFile)
	}

	if s.keyFile != "/etc/tls/tls.key" {
		t.Errorf("expected keyFile to be /etc/tls/tls.key, got %s", s.keyFile)
	}
}

func TestNewServerTLSEnvOverridesOption(t *testing.T) {
	t.Setenv(prometheusTLSCertEnvName, "/env/cert.pem")
	t.Setenv(prometheusTLSKeyEnvName, "/env/key.pem")

	s, err := NewServer(WithTLSCertFiles("/opt/cert.pem", "/opt/key.pem"))
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

	if s.certFile != "/env/cert.pem" {
		t.Errorf("expected env var to override option, got certFile=%s", s.certFile)
	}

	if s.keyFile != "/env/key.pem" {
		t.Errorf("expected env var to override option, got keyFile=%s", s.keyFile)
	}
}

func TestContextWithTLSConfig(t *testing.T) {
	ctx := context.Background()

	if cfg := TLSConfigFromContext(ctx); cfg != nil {
		t.Error("expected TLSConfigFromContext to return nil for empty context")
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	ctx = ContextWithTLSConfig(ctx, tlsConfig)

	retrieved := TLSConfigFromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected TLSConfigFromContext to return the config")
	}

	if retrieved.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion to be TLS 1.2, got %v", retrieved.MinVersion)
	}
}

func TestTLSConfigWithCertFiles(t *testing.T) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	s, err := NewServer(
		WithTLSConfig(tlsConfig),
		WithTLSCertFiles("/etc/tls/tls.crt", "/etc/tls/tls.key"),
	)
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

	// Verify TLSConfig is set with the specified settings
	if s.http.TLSConfig == nil {
		t.Fatal("expected TLSConfig to be set")
	}

	if s.http.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected MinVersion TLS 1.3, got %v", s.http.TLSConfig.MinVersion)
	}

	if s.http.TLSConfig.MaxVersion != tls.VersionTLS13 {
		t.Errorf("expected MaxVersion TLS 1.3, got %v", s.http.TLSConfig.MaxVersion)
	}

	if len(s.http.TLSConfig.CipherSuites) != 2 {
		t.Errorf("expected 2 cipher suites, got %d", len(s.http.TLSConfig.CipherSuites))
	}

	// Verify cert files are also set
	if s.certFile != "/etc/tls/tls.crt" {
		t.Errorf("expected certFile=/etc/tls/tls.crt, got %s", s.certFile)
	}

	if s.keyFile != "/etc/tls/tls.key" {
		t.Errorf("expected keyFile=/etc/tls/tls.key, got %s", s.keyFile)
	}

}
