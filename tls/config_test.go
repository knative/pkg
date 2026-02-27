/*
Copyright 2026 The Knative Authors

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

package tls

import (
	cryptotls "crypto/tls"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint16
		wantErr bool
	}{
		{name: "TLS 1.2", input: "1.2", want: cryptotls.VersionTLS12},
		{name: "TLS 1.3", input: "1.3", want: cryptotls.VersionTLS13},
		{name: "unsupported version", input: "1.0", wantErr: true},
		{name: "unsupported version 1.1", input: "1.1", wantErr: true},
		{name: "trailing space", input: "1.2 ", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
		{name: "garbage", input: "abc", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseVersion(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseVersion(%q) = %d, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseVersion(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseVersion(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []uint16
		wantErr bool
	}{
		{
			name:  "single suite",
			input: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			want:  []uint16{cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name:  "multiple suites",
			input: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			want: []uint16{
				cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				cryptotls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
		},
		{
			name:  "whitespace trimmed",
			input: " TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 , TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 ",
			want: []uint16{
				cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				cryptotls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			},
		},
		{
			name:  "empty parts skipped",
			input: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,,",
			want:  []uint16{cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name:    "unknown suite",
			input:   "DOES_NOT_EXIST",
			wantErr: true,
		},
		{
			name: "empty string",
			want: []uint16{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCipherSuites(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseCipherSuites(%q) = %v, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCipherSuites(%q) unexpected error: %v", tc.input, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ParseCipherSuites(%q) returned %d suites, want %d", tc.input, len(got), len(tc.want))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("ParseCipherSuites(%q)[%d] = %d, want %d", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestParseCurvePreferences(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []cryptotls.CurveID
		wantErr bool
	}{
		{
			name:  "Go constant name X25519",
			input: "X25519",
			want:  []cryptotls.CurveID{cryptotls.X25519},
		},
		{
			name:  "Go constant name CurveP256",
			input: "CurveP256",
			want:  []cryptotls.CurveID{cryptotls.CurveP256},
		},
		{
			name:  "standard name P-256",
			input: "P-256",
			want:  []cryptotls.CurveID{cryptotls.CurveP256},
		},
		{
			name:  "multiple curves with mixed naming",
			input: "X25519,P-256,CurveP384",
			want: []cryptotls.CurveID{
				cryptotls.X25519,
				cryptotls.CurveP256,
				cryptotls.CurveP384,
			},
		},
		{
			name:  "whitespace trimmed",
			input: " X25519 , CurveP256 ",
			want: []cryptotls.CurveID{
				cryptotls.X25519,
				cryptotls.CurveP256,
			},
		},
		{
			name:  "all curves by standard name",
			input: "P-256,P-384,P-521,X25519",
			want: []cryptotls.CurveID{
				cryptotls.CurveP256,
				cryptotls.CurveP384,
				cryptotls.CurveP521,
				cryptotls.X25519,
			},
		},
		{
			name:  "post-quantum hybrid X25519MLKEM768",
			input: "X25519MLKEM768",
			want:  []cryptotls.CurveID{cryptotls.X25519MLKEM768},
		},
		{
			name:    "unknown curve",
			input:   "CurveP128",
			wantErr: true,
		},
		{
			name: "empty string",
			want: []cryptotls.CurveID{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCurvePreferences(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseCurvePreferences(%q) = %v, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCurvePreferences(%q) unexpected error: %v", tc.input, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ParseCurvePreferences(%q) returned %d curves, want %d", tc.input, len(got), len(tc.want))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("ParseCurvePreferences(%q)[%d] = %d, want %d", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNewConfigFromEnv(t *testing.T) {
	t.Run("no env vars set returns zero value", func(t *testing.T) {
		cfg, err := NewConfigFromEnv("")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if cfg.MinVersion != 0 {
			t.Errorf("MinVersion = %d, want 0", cfg.MinVersion)
		}
		if cfg.MaxVersion != 0 {
			t.Errorf("MaxVersion = %d, want 0", cfg.MaxVersion)
		}
		if cfg.CipherSuites != nil {
			t.Errorf("CipherSuites = %v, want nil", cfg.CipherSuites)
		}
		if cfg.CurvePreferences != nil {
			t.Errorf("CurvePreferences = %v, want nil", cfg.CurvePreferences)
		}
	})

	t.Run("min version from env", func(t *testing.T) {
		t.Setenv(MinVersionEnvKey, "1.2")
		cfg, err := NewConfigFromEnv("")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if cfg.MinVersion != cryptotls.VersionTLS12 {
			t.Errorf("MinVersion = %d, want %d", cfg.MinVersion, cryptotls.VersionTLS12)
		}
	})

	t.Run("max version from env", func(t *testing.T) {
		t.Setenv(MaxVersionEnvKey, "1.3")
		cfg, err := NewConfigFromEnv("")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if cfg.MaxVersion != cryptotls.VersionTLS13 {
			t.Errorf("MaxVersion = %d, want %d", cfg.MaxVersion, cryptotls.VersionTLS13)
		}
	})

	t.Run("cipher suites from env", func(t *testing.T) {
		t.Setenv(CipherSuitesEnvKey, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")
		cfg, err := NewConfigFromEnv("")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if len(cfg.CipherSuites) != 1 || cfg.CipherSuites[0] != cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 {
			t.Errorf("CipherSuites = %v, want [%d]", cfg.CipherSuites, cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)
		}
	})

	t.Run("curve preferences from env", func(t *testing.T) {
		t.Setenv(CurvePreferencesEnvKey, "X25519,CurveP256")
		cfg, err := NewConfigFromEnv("")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if len(cfg.CurvePreferences) != 2 {
			t.Fatalf("CurvePreferences has %d entries, want 2", len(cfg.CurvePreferences))
		}
		if cfg.CurvePreferences[0] != cryptotls.X25519 {
			t.Errorf("CurvePreferences[0] = %d, want %d", cfg.CurvePreferences[0], cryptotls.X25519)
		}
		if cfg.CurvePreferences[1] != cryptotls.CurveP256 {
			t.Errorf("CurvePreferences[1] = %d, want %d", cfg.CurvePreferences[1], cryptotls.CurveP256)
		}
	})

	t.Run("prefix is prepended to env key", func(t *testing.T) {
		t.Setenv("WEBHOOK_TLS_MIN_VERSION", "1.2")
		cfg, err := NewConfigFromEnv("WEBHOOK_")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if cfg.MinVersion != cryptotls.VersionTLS12 {
			t.Errorf("MinVersion = %d, want %d", cfg.MinVersion, cryptotls.VersionTLS12)
		}
	})

	t.Run("all env vars set", func(t *testing.T) {
		t.Setenv(MinVersionEnvKey, "1.2")
		t.Setenv(MaxVersionEnvKey, "1.3")
		t.Setenv(CipherSuitesEnvKey, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384")
		t.Setenv(CurvePreferencesEnvKey, "X25519,P-256")

		cfg, err := NewConfigFromEnv("")
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		if cfg.MinVersion != cryptotls.VersionTLS12 {
			t.Errorf("MinVersion = %d, want %d", cfg.MinVersion, cryptotls.VersionTLS12)
		}
		if cfg.MaxVersion != cryptotls.VersionTLS13 {
			t.Errorf("MaxVersion = %d, want %d", cfg.MaxVersion, cryptotls.VersionTLS13)
		}
		if len(cfg.CipherSuites) != 2 {
			t.Fatalf("CipherSuites has %d entries, want 2", len(cfg.CipherSuites))
		}
		if len(cfg.CurvePreferences) != 2 {
			t.Fatalf("CurvePreferences has %d entries, want 2", len(cfg.CurvePreferences))
		}
	})

	t.Run("invalid min version", func(t *testing.T) {
		t.Setenv(MinVersionEnvKey, "1.0")
		_, err := NewConfigFromEnv("")
		if err == nil {
			t.Fatal("expected error for invalid min version")
		}
	})

	t.Run("invalid max version", func(t *testing.T) {
		t.Setenv(MaxVersionEnvKey, "bad")
		_, err := NewConfigFromEnv("")
		if err == nil {
			t.Fatal("expected error for invalid max version")
		}
	})

	t.Run("invalid cipher suite", func(t *testing.T) {
		t.Setenv(CipherSuitesEnvKey, "NOT_A_REAL_CIPHER")
		_, err := NewConfigFromEnv("")
		if err == nil {
			t.Fatal("expected error for invalid cipher suite")
		}
	})

	t.Run("invalid curve", func(t *testing.T) {
		t.Setenv(CurvePreferencesEnvKey, "NotACurve")
		_, err := NewConfigFromEnv("")
		if err == nil {
			t.Fatal("expected error for invalid curve")
		}
	})
}

func TestConfig_TLSConfig(t *testing.T) {
	cfg := &Config{
		MinVersion: cryptotls.VersionTLS12,
		MaxVersion: cryptotls.VersionTLS13,
		CipherSuites: []uint16{
			cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		CurvePreferences: []cryptotls.CurveID{
			cryptotls.X25519,
			cryptotls.CurveP256,
		},
	}

	tc := cfg.TLSConfig()

	if tc.MinVersion != cryptotls.VersionTLS12 {
		t.Errorf("MinVersion = %d, want %d", tc.MinVersion, cryptotls.VersionTLS12)
	}
	if tc.MaxVersion != cryptotls.VersionTLS13 {
		t.Errorf("MaxVersion = %d, want %d", tc.MaxVersion, cryptotls.VersionTLS13)
	}
	if len(tc.CipherSuites) != 1 || tc.CipherSuites[0] != cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 {
		t.Errorf("CipherSuites = %v, want [%d]", tc.CipherSuites, cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)
	}
	if len(tc.CurvePreferences) != 2 {
		t.Fatalf("CurvePreferences has %d entries, want 2", len(tc.CurvePreferences))
	}
	if tc.CurvePreferences[0] != cryptotls.X25519 {
		t.Errorf("CurvePreferences[0] = %d, want %d", tc.CurvePreferences[0], cryptotls.X25519)
	}
}
