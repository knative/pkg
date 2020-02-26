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
	"errors"
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
)

func okConfig() *Config {
	return &Config{
		ResourceLock:      "leases",
		LeaseDuration:     15 * time.Second,
		RenewDeadline:     10 * time.Second,
		RetryPeriod:       2 * time.Second,
		EnabledComponents: sets.NewString(),
	}
}

func okData() map[string]string {
	return map[string]string{
		"resourceLock": "leases",
		// values in this data come from the defaults suggested in the
		// code:
		// https://github.com/kubernetes/client-go/blob/kubernetes-1.16.0/tools/leaderelection/leaderelection.go
		"leaseDuration":     "15s",
		"renewDeadline":     "10s",
		"retryPeriod":       "2s",
		"enabledComponents": "controller",
	}
}

func TestNewConfigMapFromData(t *testing.T) {
	cases := []struct {
		name     string
		data     map[string]string
		expected *Config
		err      error
	}{
		{
			name: "disabled but OK config",
			data: func() map[string]string {
				data := okData()
				delete(data, "enabledComponents")
				return data
			}(),
			expected: okConfig(),
		},
		{
			name: "OK config - controller enabled",
			data: okData(),
			expected: func() *Config {
				config := okConfig()
				config.EnabledComponents.Insert("controller")
				return config
			}(),
		},
		{
			name: "missing resourceLock",
			data: func() map[string]string {
				data := okData()
				delete(data, "resourceLock")
				return data
			}(),
			err: errors.New(`resourceLock: invalid value "": valid values are "leases","configmaps","endpoints"`),
		},
		{
			name: "invalid resourceLock",
			data: func() map[string]string {
				data := okData()
				data["resourceLock"] = "flarps"
				return data
			}(),
			err: errors.New(`resourceLock: invalid value "flarps": valid values are "leases","configmaps","endpoints"`),
		},
		{
			name: "missing leaseDuration",
			data: func() map[string]string {
				data := okData()
				delete(data, "leaseDuration")
				return data
			}(),
			err: errors.New(`leaseDuration: invalid duration: ""`),
		},
		{
			name: "invalid leaseDuration",
			data: func() map[string]string {
				data := okData()
				data["leaseDuration"] = "flops"
				return data
			}(),
			err: errors.New(`leaseDuration: invalid duration: "flops"`),
		},
		{
			name: "missing renewDeadline",
			data: func() map[string]string {
				data := okData()
				delete(data, "renewDeadline")
				return data
			}(),
			err: errors.New(`renewDeadline: invalid duration: ""`),
		},
		{
			name: "invalid renewDeadline",
			data: func() map[string]string {
				data := okData()
				data["renewDeadline"] = "flops"
				return data
			}(),
			err: errors.New(`renewDeadline: invalid duration: "flops"`),
		},
		{
			name: "missing retryPeriod",
			data: func() map[string]string {
				data := okData()
				delete(data, "retryPeriod")
				return data
			}(),
			err: errors.New(`retryPeriod: invalid duration: ""`),
		},
		{
			name: "invalid retryPeriod",
			data: func() map[string]string {
				data := okData()
				data["retryPeriod"] = "flops"
				return data
			}(),
			err: errors.New(`retryPeriod: invalid duration: "flops"`),
		},
	}

	for i := range cases {
		tc := cases[i]
		actualConfig, actualErr := NewConfigFromMap(tc.data)
		if !reflect.DeepEqual(tc.err, actualErr) {
			t.Errorf("%v: expected error %v, got %v", tc.name, tc.err, actualErr)
			continue
		}

		if !reflect.DeepEqual(tc.expected, actualConfig) {
			t.Errorf("%v: expected config:\n%+v\ngot:\n%+v", tc.name, tc.expected, actualConfig)
			continue
		}
	}
}

func TestGetComponentConfig(t *testing.T) {
	cases := []struct {
		name     string
		config   Config
		expected ComponentConfig
	}{
		{
			name: "component enabled",
			config: Config{
				ResourceLock:      "leases",
				LeaseDuration:     15 * time.Second,
				RenewDeadline:     10 * time.Second,
				RetryPeriod:       2 * time.Second,
				EnabledComponents: sets.NewString("component"),
			},
			expected: ComponentConfig{
				LeaderElect:   true,
				ResourceLock:  "leases",
				LeaseDuration: 15 * time.Second,
				RenewDeadline: 10 * time.Second,
				RetryPeriod:   2 * time.Second,
			},
		},
		{
			name: "component disabled",
			config: Config{
				ResourceLock:      "leases",
				LeaseDuration:     15 * time.Second,
				RenewDeadline:     10 * time.Second,
				RetryPeriod:       2 * time.Second,
				EnabledComponents: sets.NewString("not-the-component"),
			},
			expected: ComponentConfig{
				LeaderElect: false,
			},
		},
	}

	for i := range cases {
		tc := cases[i]
		actual := tc.config.GetComponentConfig("component")
		if !reflect.DeepEqual(tc.expected, actual) {
			t.Errorf("%v: expected:\n%+v\ngot:\n%+v", tc.name, tc.expected, actual)
		}
	}
}
