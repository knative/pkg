/*
Copyright 2020 The Knative Authors

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

package webhook

import (
	"crypto/tls"
	"testing"
)

const (
	testMissingInputName = "MissingInput"

	testDefaultPort          = 8888
	testDefaultSecretName    = "webhook-certs"
	testDefaultTLSMinVersion = tls.VersionTLS12
)

type portTest struct {
	name      string
	in        string
	want      int
	wantPanic bool
}

type webhookNameTest struct {
	name      string
	in        string
	want      string
	wantPanic bool
}

type secretNameTest struct {
	name      string
	in        string
	want      string
	wantPanic bool
}

type tlsMinVersionTest struct {
	name      string
	in        string
	want      uint16
	wantPanic bool
}

func TestPort(t *testing.T) {
	tests := []portTest{{
		name: testMissingInputName,
		want: testDefaultPort,
	}, {
		name: "EmptyInput",
		in:   "",
		want: testDefaultPort,
	}, {
		name:      "InvalidInputNonNumeric",
		in:        "invalid",
		wantPanic: true,
	}, {
		name:      "InvalidInputTrailingSpace",
		in:        "8443 ",
		wantPanic: true,
	}, {
		name:      "InvalidInputZero",
		in:        "0",
		wantPanic: true,
	}, {
		name: "ValidInput",
		in:   "443",
		want: 443,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// portEnvKey is unset when testing missing input.
			if tc.name != testMissingInputName {
				t.Setenv(portEnvKey, tc.in)
			}

			defer func() {
				if r := recover(); r == nil && tc.wantPanic {
					t.Error("Did not panic")
				} else if r != nil && !tc.wantPanic {
					t.Error("Got unexpected panic")
				}
			}()

			if got := PortFromEnv(testDefaultPort); got != tc.want {
				t.Errorf("PortFromEnv = %d, want: %d", got, tc.want)
			}
		})
	}
}

func TestWebhookName(t *testing.T) {
	tests := []webhookNameTest{{
		name:      "EmptyInput",
		in:        "",
		wantPanic: true,
	}, {
		name: "ValidInput",
		in:   "mywebhook",
		want: "mywebhook",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// webhookNameEnv is unset when testing missing input.
			if tc.name != testMissingInputName {
				t.Setenv(webhookNameEnvKey, tc.in)
			}

			defer func() {
				if r := recover(); r == nil && tc.wantPanic {
					t.Error("Did not panic")
				} else if r != nil && !tc.wantPanic {
					t.Error("Got unexpected panic")
				}
			}()

			if got := NameFromEnv(); got != tc.want {
				t.Errorf("NameFromEnv = %s, want: %s", got, tc.want)
			}
		})
	}
}

func TestSecretName(t *testing.T) {
	tests := []secretNameTest{{
		name: testMissingInputName,
		want: testDefaultSecretName,
	}, {
		name: "EmptyInput",
		in:   "",
		want: testDefaultSecretName,
	}, {
		name: "ValidInput",
		in:   "my-webhook-certs",
		want: "my-webhook-certs",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// secretNameEnvKey is unset when testing missing input.
			if tc.name != testMissingInputName {
				t.Setenv(secretNameEnvKey, tc.in)
			}

			defer func() {
				if r := recover(); r == nil && tc.wantPanic {
					t.Error("Did not panic")
				} else if r != nil && !tc.wantPanic {
					t.Error("Got unexpected panic")
				}
			}()

			if got := SecretNameFromEnv(testDefaultSecretName); got != tc.want {
				t.Errorf("SecretNameFromEnv = %s, want: %s", got, tc.want)
			}
		})
	}
}

func TestTLSMinVersion(t *testing.T) {
	tests := []tlsMinVersionTest{{
		name: testMissingInputName,
		want: testDefaultTLSMinVersion,
	}, {
		name: "EmptyInput",
		in:   "",
		want: testDefaultTLSMinVersion,
	}, {
		name:      "InvalidInputTrailingSpace",
		in:        "1.2  ",
		wantPanic: true,
	}, {
		name:      "InvalidInput",
		in:        "1.0",
		wantPanic: true,
	}, {
		name: "ValidInputTLS12",
		in:   "1.2",
		want: tls.VersionTLS12,
	}, {
		name: "ValidInputTLS13",
		in:   "1.3",
		want: tls.VersionTLS13,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// tlsMinVersionEnvKey is unset when testing missing input.
			if tc.name != testMissingInputName {
				t.Setenv(tlsMinVersionEnvKey, tc.in)
			}

			defer func() {
				if r := recover(); r == nil && tc.wantPanic {
					t.Error("Did not panic")
				} else if r != nil && !tc.wantPanic {
					t.Error("Got unexpected panic")
				}
			}()

			if got := TLSMinVersionFromEnv(testDefaultTLSMinVersion); got != tc.want {
				t.Errorf("TLSMinVersionFromEnv = %d, want: %d", got, tc.want)
			}
		})
	}
}
