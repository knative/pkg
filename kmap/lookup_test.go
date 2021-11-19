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

func TestNewOrderedLookup(t *testing.T) {
	keys := []string{"1", "2", "3"}
	a := NewOrderedLookup(keys...)

	if diff := cmp.Diff(keys, a.Keys); diff != "" {
		t.Error("NewOrderedLookup unexpected diff (-want, +got):", diff)
	}
}

func TestNewOrderedLookup_BadInput(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected no keys to panic")
		}
	}()
	NewOrderedLookup()
}

func TestOrderedLookup_Get(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		in       map[string]string
		outKey   string
		outValue string
		outOk    bool
	}{{
		name:     "single key in map",
		keys:     []string{"old", "new"},
		in:       map[string]string{"old": "1"},
		outKey:   "old",
		outValue: "1",
		outOk:    true,
	}, {
		name:     "another single key in map",
		keys:     []string{"old", "new"},
		in:       map[string]string{"new": "1"},
		outKey:   "new",
		outValue: "1",
		outOk:    true,
	}, {
		name:     "lower ordinal takes priority",
		keys:     []string{"old", "new"},
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
			a := NewOrderedLookup(tc.keys...)

			k, v, ok := a.Get(tc.in)

			if tc.outOk != ok {
				t.Error("expected ok to be", tc.outOk)
			}

			if tc.outValue != v {
				t.Errorf("expected value to be %q got: %q", tc.outValue, v)
			}

			if tc.outKey != k {
				t.Errorf("expected key to be %q got: %q", tc.outKey, k)
			}

		})
	}

}
