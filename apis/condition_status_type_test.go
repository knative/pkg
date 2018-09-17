/*
Copyright 2017 The Knative Authors

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

package apis

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestConfigurationIsTrue(t *testing.T) {
	cases := []struct {
		name      string
		condition *Condition
		truth     bool
	}{{
		name:      "empty should be false",
		condition: &Condition{},
		truth:     false,
	}, {
		name: "True should be true",
		condition: &Condition{
			Status: corev1.ConditionTrue,
		},
		truth: true,
	}, {
		name: "False should be false",
		condition: &Condition{
			Status: corev1.ConditionFalse,
		},
		truth: false,
	}, {
		name: "Unknown should be false",
		condition: &Condition{
			Status: corev1.ConditionUnknown,
		},
		truth: false,
	}, {
		name:      "Nil should be false",
		condition: nil,
		truth:     false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if e, a := tc.truth, tc.condition.IsTrue(); e != a {
				t.Errorf("%q expected: %v got: %v", tc.name, e, a)
			}
		})
	}
}

func TestConfigurationIsFalse(t *testing.T) {
	cases := []struct {
		name      string
		condition *Condition
		truth     bool
	}{{
		name:      "empty should be false",
		condition: &Condition{},
		truth:     false,
	}, {
		name: "True should be false",
		condition: &Condition{
			Status: corev1.ConditionTrue,
		},
		truth: false,
	}, {
		name: "False should be true",
		condition: &Condition{
			Status: corev1.ConditionFalse,
		},
		truth: true,
	}, {
		name: "Unknown should be false",
		condition: &Condition{
			Status: corev1.ConditionUnknown,
		},
		truth: false,
	}, {
		name:      "Nil should be false",
		condition: nil,
		truth:     false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if e, a := tc.truth, tc.condition.IsFalse(); e != a {
				t.Errorf("%q expected: %v got: %v", tc.name, e, a)
			}
		})
	}
}

func TestConfigurationIsUnknown(t *testing.T) {
	cases := []struct {
		name      string
		condition *Condition
		truth     bool
	}{{
		name:      "empty should be false",
		condition: &Condition{},
		truth:     false,
	}, {
		name: "True should be false",
		condition: &Condition{
			Status: corev1.ConditionTrue,
		},
		truth: false,
	}, {
		name: "False should be false",
		condition: &Condition{
			Status: corev1.ConditionFalse,
		},
		truth: false,
	}, {
		name: "Unknown should be true",
		condition: &Condition{
			Status: corev1.ConditionUnknown,
		},
		truth: true,
	}, {
		name:      "Nil should be true",
		condition: nil,
		truth:     true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if e, a := tc.truth, tc.condition.IsUnknown(); e != a {
				t.Errorf("%q expected: %v got: %v", tc.name, e, a)
			}
		})
	}
}
