/*
Copyright 2021 The Knative Authors

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

package kmap

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestKeyPriority(t *testing.T) {
	tests := []struct {
		name     string
		keys     KeyPriority
		in       map[string]string
		outKey   string
		outValue string
		outOk    bool
	}{{
		name:     "single key in map",
		keys:     KeyPriority{"old", "new"},
		in:       map[string]string{"old": "1"},
		outKey:   "old",
		outValue: "1",
		outOk:    true,
	}, {
		name:     "another single key in map",
		keys:     KeyPriority{"old", "new"},
		in:       map[string]string{"new": "1"},
		outKey:   "new",
		outValue: "1",
		outOk:    true,
	}, {
		name:     "lower ordinal takes priority",
		keys:     KeyPriority{"old", "new"},
		in:       map[string]string{"new": "1", "old": "2"},
		outKey:   "old",
		outValue: "2",
		outOk:    true,
	}, {
		name: "missing key in map",
		keys: []string{"old", "new"},
		in:   map[string]string{},

		// We still return what key we used to access the values
		// and we first key since it has priority
		outKey: "old",
		outOk:  false,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k, v, ok := tc.keys.Get(tc.in)

			if tc.outOk != ok {
				t.Error("expected ok to be", tc.outOk)
			}

			if tc.outValue != v {
				t.Errorf("expected value to be %q got: %q", tc.outValue, v)
			}

			if tc.outKey != k {
				t.Errorf("expected key to be %q got: %q", tc.outKey, k)
			}

			if want, got := v, tc.keys.Value(tc.in); got != want {
				t.Errorf("Value() diff %q != %q", want, got)
			}
		})
	}
}

func TestUpdateKeys(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]string
		out  map[string]string
		ols  []KeyPriority
	}{{
		name: "identity",
		in:   map[string]string{"1": "2"},
		out:  map[string]string{"1": "2"},
	}, {
		name: "single replacement",
		in:   map[string]string{"1": "2", "4": "5"},
		ols: []KeyPriority{
			{"3", "1"},
		},
		out: map[string]string{"3": "2", "4": "5"},
	}, {
		name: "multiple replacement",
		in:   map[string]string{"1": "2", "6": "5", "8": "9"},
		ols: []KeyPriority{
			{"3", "1"},
			{"4", "6"},
		},
		out: map[string]string{"3": "2", "4": "5", "8": "9"},
	}, {
		name: "default not replaced",
		in:   map[string]string{"1": "2", "4": "5"},
		ols: []KeyPriority{
			{"1", "3"},
		},
		out: map[string]string{"1": "2", "4": "5"},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			out := UpdateKeys(tc.in, tc.ols...)
			if diff := cmp.Diff(tc.out, out); diff != "" {
				t.Error("Migrate diff (-want,+got):", diff)
			}
		})
	}
}
