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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

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

func TestNewServerTLSRequiresBothCertAndKey(t *testing.T) {
	t.Run("only cert set", func(t *testing.T) {
		t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
		t.Setenv(prometheusTLSKeyEnvName, "") // ensure key is unset
		_, err := NewServer()
		if err == nil {
			t.Error("expected NewServer to fail when only TLS cert is set")
		}
		if err != nil && !strings.Contains(err.Error(), "must be set or neither") {
			t.Errorf("expected error about both cert and key; got %v", err)
		}
	})
	t.Run("only key set", func(t *testing.T) {
		t.Setenv(prometheusTLSCertEnvName, "")
		t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")
		_, err := NewServer()
		if err == nil {
			t.Error("expected NewServer to fail when only TLS key is set")
		}
		if err != nil && !strings.Contains(err.Error(), "must be set or neither") {
			t.Errorf("expected error about both cert and key; got %v", err)
		}
	})
}

func TestNewServerWithTLSConfigFromEnv(t *testing.T) {
	t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
	t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")
	t.Setenv("METRICS_PROMETHEUS_TLS_MIN_VERSION", "1.2")

	s, err := NewServer()
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

	if s.http.TLSConfig == nil {
		t.Fatal("expected TLSConfig to be set when cert/key env vars are set")
	}
	if s.http.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS 1.2 from env, got %v", s.http.TLSConfig.MinVersion)
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

func TestTLSConfigWithCertFilesFromEnv(t *testing.T) {
	t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
	t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")
	t.Setenv("METRICS_PROMETHEUS_TLS_MIN_VERSION", "1.3")
	t.Setenv("METRICS_PROMETHEUS_TLS_MAX_VERSION", "1.3")
	t.Setenv("METRICS_PROMETHEUS_TLS_CIPHER_SUITES", "TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256")

	s, err := NewServer()
	if err != nil {
		t.Fatal("NewServer() =", err)
	}

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
	if s.certFile != "/etc/tls/tls.crt" {
		t.Errorf("expected certFile=/etc/tls/tls.crt, got %s", s.certFile)
	}
	if s.keyFile != "/etc/tls/tls.key" {
		t.Errorf("expected keyFile=/etc/tls/tls.key, got %s", s.keyFile)
	}
}

func TestPrometheusMTLSFromEnv(t *testing.T) {
	t.Run("require without client CA file returns error", func(t *testing.T) {
		t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
		t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")
		t.Setenv(prometheusTLSClientAuthEnvName, "require")
		t.Setenv(prometheusTLSClientCAFileEnv, "") // unset
		_, err := NewServer()
		if err == nil {
			t.Fatal("expected NewServer to fail when client auth is require but client CA file is unset")
		}
		if !strings.Contains(err.Error(), prometheusTLSClientCAFileEnv) {
			t.Errorf("expected error to mention %s; got %v", prometheusTLSClientCAFileEnv, err)
		}
	})

	t.Run("invalid client auth value returns error", func(t *testing.T) {
		t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
		t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")
		t.Setenv(prometheusTLSClientAuthEnvName, "invalid")
		_, err := NewServer()
		if err == nil {
			t.Fatal("expected NewServer to fail with invalid client auth value")
		}
		if !strings.Contains(err.Error(), prometheusTLSClientAuthEnvName) {
			t.Errorf("expected error to mention %s; got %v", prometheusTLSClientAuthEnvName, err)
		}
	})

	t.Run("optional with valid client CA file sets ClientAuth and ClientCAs", func(t *testing.T) {
		caFile := createTempCACertFile(t)
		t.Cleanup(func() { _ = os.Remove(caFile) })
		t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
		t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")
		t.Setenv(prometheusTLSClientAuthEnvName, "optional")
		t.Setenv(prometheusTLSClientCAFileEnv, caFile)

		s, err := NewServer()
		if err != nil {
			t.Fatal("NewServer() =", err)
		}
		if s.http.TLSConfig == nil {
			t.Fatal("expected TLSConfig to be set")
		}
		if s.http.TLSConfig.ClientAuth != tls.VerifyClientCertIfGiven {
			t.Errorf("expected ClientAuth VerifyClientCertIfGiven, got %v", s.http.TLSConfig.ClientAuth)
		}
		if s.http.TLSConfig.ClientCAs == nil {
			t.Error("expected ClientCAs to be set")
		}
	})

	t.Run("require with valid client CA file sets RequireAndVerifyClientCert", func(t *testing.T) {
		caFile := createTempCACertFile(t)
		t.Cleanup(func() { _ = os.Remove(caFile) })
		t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
		t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")
		t.Setenv(prometheusTLSClientAuthEnvName, "require")
		t.Setenv(prometheusTLSClientCAFileEnv, caFile)

		s, err := NewServer()
		if err != nil {
			t.Fatal("NewServer() =", err)
		}
		if s.http.TLSConfig == nil {
			t.Fatal("expected TLSConfig to be set")
		}
		if s.http.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert {
			t.Errorf("expected ClientAuth RequireAndVerifyClientCert, got %v", s.http.TLSConfig.ClientAuth)
		}
		if s.http.TLSConfig.ClientCAs == nil {
			t.Error("expected ClientCAs to be set")
		}
	})

	t.Run("explicit none does not set client auth", func(t *testing.T) {
		t.Setenv(prometheusTLSCertEnvName, "/etc/tls/tls.crt")
		t.Setenv(prometheusTLSKeyEnvName, "/etc/tls/tls.key")
		t.Setenv(prometheusTLSClientAuthEnvName, "none")
		t.Setenv(prometheusTLSClientCAFileEnv, "")

		s, err := NewServer()
		if err != nil {
			t.Fatal("NewServer() =", err)
		}
		if s.http.TLSConfig == nil {
			t.Fatal("expected TLSConfig to be set")
		}
		if s.http.TLSConfig.ClientAuth != tls.NoClientCert {
			t.Errorf("expected ClientAuth NoClientCert, got %v", s.http.TLSConfig.ClientAuth)
		}
		if s.http.TLSConfig.ClientCAs != nil {
			t.Error("expected ClientCAs to be nil")
		}
	})

	t.Run("mTLS env vars without TLS enabled returns error", func(t *testing.T) {
		t.Setenv(prometheusTLSCertEnvName, "")
		t.Setenv(prometheusTLSKeyEnvName, "")
		t.Setenv(prometheusTLSClientAuthEnvName, "require")
		t.Setenv(prometheusTLSClientCAFileEnv, "/etc/tls/ca.pem")
		_, err := NewServer()
		if err == nil {
			t.Fatal("expected NewServer to fail when mTLS is set but TLS is not enabled")
		}
		if !strings.Contains(err.Error(), "require TLS to be enabled") {
			t.Errorf("expected error about TLS not enabled; got %v", err)
		}
	})

	t.Run("client CA file without TLS enabled returns error", func(t *testing.T) {
		t.Setenv(prometheusTLSCertEnvName, "")
		t.Setenv(prometheusTLSKeyEnvName, "")
		t.Setenv(prometheusTLSClientAuthEnvName, "")
		t.Setenv(prometheusTLSClientCAFileEnv, "/etc/tls/ca.pem")
		_, err := NewServer()
		if err == nil {
			t.Fatal("expected NewServer to fail when client CA file is set but TLS is not enabled")
		}
		if !strings.Contains(err.Error(), "require TLS to be enabled") {
			t.Errorf("expected error about TLS not enabled; got %v", err)
		}
	})
}

// createTempCACertFile writes a minimal self-signed CA cert to a temp file and returns its path.
func createTempCACertFile(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp("", "prometheus-mtls-ca-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}
