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

package leaderelection

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/kmeta"
)

const (
	controllerOrdinalEnv = "STATEFUL_CONTROLLER_ORDINAL"
	serviceNameEnv       = "STATEFUL_SERVICE_NAME"
	servicePortEnv       = "STATEFUL_SERVICE_PORT"
	serviceProtocolEnv   = "STATEFUL_SERVICE_PROTOCOL"
)

func okConfig() *Config {
	return &Config{
		Buckets:       1,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
	}
}

func okData() map[string]string {
	return map[string]string{
		"buckets": "1",
		// values in this data come from the defaults suggested in the
		// code:
		// https://github.com/kubernetes/client-go/blob/kubernetes-1.16.0/tools/leaderelection/leaderelection.go
		"leaseDuration": "15s",
		"renewDeadline": "10s",
		"retryPeriod":   "2s",
	}
}

func TestNewConfigMapFromData(t *testing.T) {
	cases := []struct {
		name     string
		data     map[string]string
		expected *Config
		err      string
	}{{
		name:     "OK config - controller enabled",
		data:     okData(),
		expected: okConfig(),
	}, {
		name: "OK config - controller enabled with multiple buckets",
		data: kmeta.UnionMaps(okData(), map[string]string{
			"buckets": "5",
		}),
		expected: func() *Config {
			config := okConfig()
			config.Buckets = 5
			return config
		}(),
	}, {
		name: "invalid leaseDuration",
		data: kmeta.UnionMaps(okData(), map[string]string{
			"leaseDuration": "flops",
		}),
		err: `failed to parse "leaseDuration": time: invalid duration`,
	}, {
		name: "invalid renewDeadline",
		data: kmeta.UnionMaps(okData(), map[string]string{
			"renewDeadline": "flops",
		}),
		err: `failed to parse "renewDeadline": time: invalid duration`,
	}, {
		name: "invalid retryPeriod",
		data: kmeta.UnionMaps(okData(), map[string]string{
			"retryPeriod": "flops",
		}),
		err: `failed to parse "retryPeriod": time: invalid duration`,
	}, {
		name: "invalid buckets - not an int",
		data: kmeta.UnionMaps(okData(), map[string]string{
			"buckets": "not-an-int",
		}),
		err: `failed to parse "buckets": strconv.ParseUint: parsing "not-an-int": invalid syntax`,
	}, {
		name: "invalid buckets - too small",
		data: kmeta.UnionMaps(okData(), map[string]string{
			"buckets": "0",
		}),
		err: fmt.Sprint("buckets: value must be between 1 <= 0 <= ", MaxBuckets),
	}, {
		name: "invalid buckets - too large",
		data: kmeta.UnionMaps(okData(), map[string]string{
			"buckets": strconv.Itoa(int(MaxBuckets + 1)),
		}),
		err: fmt.Sprintf("buckets: value must be between 1 <= %d <= %d", MaxBuckets+1, MaxBuckets),
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actualConfig, actualErr := NewConfigFromConfigMap(
				&corev1.ConfigMap{
					Data: tc.data,
				})

			if actualErr != nil {
				if got, want := actualErr.Error(), tc.err; !strings.HasPrefix(got, want) {
					t.Fatalf("Err = '%s', want: '%s'", got, want)
				}
			} else if tc.err != "" {
				t.Fatal("Expected an error, got none")
			}

			if got, want := actualConfig, tc.expected; !cmp.Equal(got, want) {
				t.Errorf("Config = %#v, want: %#v, diff(-want,+got):\n%s", got, want, cmp.Diff(want, got))
			}
		})
	}
}

func TestGetComponentConfig(t *testing.T) {
	const expectedName = "the-component"
	cases := []struct {
		name     string
		config   Config
		expected ComponentConfig
	}{{
		name: "component enabled",
		config: Config{
			LeaseDuration: 15 * time.Second,
			RenewDeadline: 10 * time.Second,
			RetryPeriod:   2 * time.Second,
		},
		expected: ComponentConfig{
			Component:     expectedName,
			LeaseDuration: 15 * time.Second,
			RenewDeadline: 10 * time.Second,
			RetryPeriod:   2 * time.Second,
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.config.GetComponentConfig(expectedName)
			if got, want := actual, tc.expected; !cmp.Equal(got, want) {
				t.Errorf("Incorrect config: diff(-want,+got):\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestNewStatefulSetConfig(t *testing.T) {
	cases := []struct {
		name     string
		pod      string
		service  string
		port     string
		protocol string
		wantErr  string
		expected statefulSetConfig
	}{{
		name:    "success with default",
		pod:     "as-42",
		service: "autoscaler",
		expected: statefulSetConfig{
			StatefulSetID: statefulSetID{
				ssName:  "as",
				ordinal: 42,
			},
			ServiceName: "autoscaler",
			Port:        "80",
			Protocol:    "http",
		},
	}, {
		name:     "success with overriding",
		pod:      "as-42",
		service:  "autoscaler",
		port:     "8080",
		protocol: "ws",
		expected: statefulSetConfig{
			StatefulSetID: statefulSetID{
				ssName:  "as",
				ordinal: 42,
			},
			ServiceName: "autoscaler",
			Port:        "8080",
			Protocol:    "ws",
		},
	}, {
		name:    "failure with empty envs",
		wantErr: "required key STATEFUL_CONTROLLER_ORDINAL missing value",
	}, {
		name:    "failure with invalid name",
		pod:     "as-abcd",
		wantErr: `envconfig.Process: assigning STATEFUL_CONTROLLER_ORDINAL to StatefulSetID: converting 'as-abcd' to type leaderelection.statefulSetID. details: strconv.Atoi: parsing "abcd": invalid syntax`,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.pod != "" {
				os.Setenv(controllerOrdinalEnv, tc.pod)
				defer os.Unsetenv(controllerOrdinalEnv)
			}
			if tc.service != "" {
				os.Setenv(serviceNameEnv, tc.service)
				defer os.Unsetenv(serviceNameEnv)
			}
			if tc.port != "" {
				os.Setenv(servicePortEnv, tc.port)
				defer os.Unsetenv(servicePortEnv)
			}
			if tc.protocol != "" {
				os.Setenv(serviceProtocolEnv, tc.protocol)
				defer os.Unsetenv(serviceProtocolEnv)
			}

			ssc, err := newStatefulSetConfig()
			if err != nil {
				if got, want := err.Error(), tc.wantErr; got != want {
					t.Errorf("Got error: %s. want: %s", got, want)
				}
			} else {
				if got, want := *ssc, tc.expected; !cmp.Equal(got, want, cmp.AllowUnexported(statefulSetID{})) {
					t.Errorf("Incorrect config: diff(-want,+got):\n%s", cmp.Diff(want, got))
				}
			}
		})
	}
}
