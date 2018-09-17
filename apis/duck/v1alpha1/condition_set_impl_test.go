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

package v1alpha1

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/knative/pkg/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestConds struct {
	Conditions Conditions
}

func (ts *TestConds) GetConditions() Conditions {
	return ts.Conditions
}

func (ts *TestConds) SetConditions(conditions Conditions) {
	ts.Conditions = conditions
}

func TestIsHappy(t *testing.T) {
	cases := []struct {
		name    string
		conds   TestConds
		condSet ConditionSet
		isHappy bool
	}{{
		name: "empty status should not be ready",
		conds: TestConds{
			Conditions: Conditions(nil),
		},
		condSet: NewLivingConditionSet(),
		isHappy: false,
	}, {
		name: "Different condition type should not be ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type:   "Foo",
				Status: corev1.ConditionTrue,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: false,
	}, {
		name: "False condition status should not be ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type:   ConditionReady,
				Status: corev1.ConditionFalse,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: false,
	}, {
		name: "Unknown condition status should not be ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type:   ConditionReady,
				Status: corev1.ConditionUnknown,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: false,
	}, {
		name: "Missing condition status should not be ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type: ConditionReady,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: false,
	}, {
		name: "True condition status should be ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type:   ConditionReady,
				Status: corev1.ConditionTrue,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: true,
	}, {
		name: "Multiple conditions with ready status should be ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type:   "Foo",
				Status: corev1.ConditionTrue,
			}, {
				Type:   ConditionReady,
				Status: corev1.ConditionTrue,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: true,
	}, {
		name: "Multiple conditions with ready status false should not be ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type:   "Foo",
				Status: corev1.ConditionTrue,
			}, {
				Type:   ConditionReady,
				Status: corev1.ConditionFalse,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: false,
	}, {
		name: "Multiple conditions with mixed ready status, some don't matter,  ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type:   "Foo",
				Status: corev1.ConditionTrue,
			}, {
				Type:   "Bar",
				Status: corev1.ConditionFalse,
			}, {
				Type:   ConditionReady,
				Status: corev1.ConditionTrue,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: true,
	}, {
		name: "Multiple conditions with mixed ready status, some don't matter, not ready",
		conds: TestConds{
			Conditions: Conditions{{
				Type:   "Foo",
				Status: corev1.ConditionTrue,
			}, {
				Type:   "Bar",
				Status: corev1.ConditionTrue,
			}, {
				Type:   ConditionReady,
				Status: corev1.ConditionFalse,
			}},
		},
		condSet: NewLivingConditionSet(),
		isHappy: false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if e, a := tc.isHappy, tc.condSet.Manage(&tc.conds).IsHappy(); e != a {
				t.Errorf("%q expected: %v got: %v", tc.name, e, a)
			}
		})
	}
}

func TestUpdateLastTransitionTime(t *testing.T) {
	condSet := NewLivingConditionSet()

	cases := []struct {
		name       string
		conditions Conditions
		condition  Condition
		update     bool
	}{{
		name: "LastTransitionTime should be set",
		conditions: Conditions{{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		}},

		condition: Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionTrue,
		},
		update: true,
	}, {
		name: "LastTransitionTime should update",
		conditions: Conditions{{
			Type:               ConditionReady,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: apis.VolatileTime{metav1.NewTime(time.Unix(1337, 0))},
		}},
		condition: Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionTrue,
		},
		update: true,
	}, {
		name: "if LastTransitionTime is the only chance, don't do it",
		conditions: Conditions{{
			Type:               ConditionReady,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: apis.VolatileTime{metav1.NewTime(time.Unix(1337, 0))},
		}},

		condition: Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		},
		update: false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			conds := &TestConds{Conditions: tc.conditions}

			was := condSet.Manage(conds).GetCondition(tc.condition.Type)
			condSet.Manage(conds).SetCondition(tc.condition)
			now := condSet.Manage(conds).GetCondition(tc.condition.Type)

			if e, a := tc.condition.Status, now.Status; e != a {
				t.Errorf("%q expected: %v to match %v", tc.name, e, a)
			}

			if tc.update {
				if e, a := was.LastTransitionTime, now.LastTransitionTime; e == a {
					t.Errorf("%q expected: %v to not match %v", tc.name, e, a)
				}
			} else {
				if e, a := was.LastTransitionTime, now.LastTransitionTime; e != a {
					t.Errorf("%q expected: %v to match %v", tc.name, e, a)
				}
			}
		})
	}
}

func TestResourceConditions(t *testing.T) {
	condSet := NewLivingConditionSet()

	config := &TestConds{}

	foo := Condition{
		Type:   "Foo",
		Status: "True",
	}
	bar := Condition{
		Type:   "Bar",
		Status: "True",
	}

	// Add a new condition.
	condSet.Manage(config).SetCondition(foo)

	if got, want := len(config.Conditions), 1; got != want {
		t.Fatalf("Unexpected Condition length; got %d, want %d", got, want)
	}

	// Add a second condition.
	condSet.Manage(config).SetCondition(bar)

	if got, want := len(config.Conditions), 2; got != want {
		t.Fatalf("Unexpected Condition length; got %d, want %d", got, want)
	}
}

func TestMarkTrue(t *testing.T) {
	condSet := NewLivingConditionSet()

	cases := []struct {
		name       string
		conditions Conditions
		mark       ConditionType
		happy      bool
	}{{
		name:  "no deps",
		mark:  ConditionReady,
		happy: true,
	}, {
		name: "existing conditions, turns happy",
		conditions: Conditions{{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		}},
		mark:  ConditionReady,
		happy: true,
	}, {
		name: "with deps, happy",
		conditions: Conditions{{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		}, {
			Type:   "Foo",
			Status: corev1.ConditionTrue,
		}},
		mark:  ConditionReady,
		happy: true,
	}, {
		name: "with deps, not happy",
		conditions: Conditions{{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		}, {
			Type:   "Foo",
			Status: corev1.ConditionFalse,
		}},
		mark:  ConditionReady,
		happy: true,
	}, {
		name: "update dep, turns happy",
		conditions: Conditions{{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		}, {
			Type:   "Foo",
			Status: corev1.ConditionFalse,
		}},
		mark:  "Foo",
		happy: true,
	}, {
		name: "update dep, happy was unknown, turns happy",
		conditions: Conditions{{
			Type:   ConditionReady,
			Status: corev1.ConditionUnknown,
		}, {
			Type:   "Foo",
			Status: corev1.ConditionFalse,
		}},
		mark:  "Foo",
		happy: true,
	}, {
		name: "update dep 1/2, still not happy",
		conditions: Conditions{{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		}, {
			Type:   "Foo",
			Status: corev1.ConditionFalse,
		}, {
			Type:   "Bar",
			Status: corev1.ConditionFalse,
		}},
		mark:  "Foo",
		happy: true,
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			config := &TestConds{}
			condSet.Manage(config).InitializeConditions()

			condSet.Manage(config).MarkTrue(tc.mark)

			if e, a := true, condSet.Manage(config).IsHappy(); e != a {
				t.Errorf("%q expected: %v got: %v", tc.name, e, a)
			}

			expected := &Condition{
				Type:   ConditionReady,
				Status: corev1.ConditionTrue,
			}

			e, a := expected, condSet.Manage(config).GetCondition(ConditionReady)
			ignoreArguments := cmpopts.IgnoreFields(Condition{}, "LastTransitionTime")
			if diff := cmp.Diff(e, a, ignoreArguments); diff != "" {
				t.Errorf("markTrue (-want, +got) = %v", diff)
			}
		})
	}
}

func TestMarkFalse(t *testing.T) {
	condSet := NewLivingConditionSet()
	config := &TestConds{}

	condSet.Manage(config).InitializeConditions()
	condSet.Manage(config).MarkFalse(ConditionReady, "false-reason", "false-message")

	if e, a := false, condSet.Manage(config).IsHappy(); e != a {
		t.Errorf("%q expected: %v got: %v", "mark false", e, a)
	}

	expected := &Condition{
		Type:    ConditionReady,
		Status:  corev1.ConditionFalse,
		Reason:  "false-reason",
		Message: "false-message",
	}

	e, a := expected, condSet.Manage(config).GetCondition(ConditionReady)
	ignoreArguments := cmpopts.IgnoreFields(Condition{}, "LastTransitionTime")
	if diff := cmp.Diff(e, a, ignoreArguments); diff != "" {
		t.Errorf("markFalse (-want, +got) = %v", diff)
	}
}

func TestMarkUnknown(t *testing.T) {
	condSet := NewLivingConditionSet()
	config := &TestConds{}

	condSet.Manage(config).InitializeConditions()
	condSet.Manage(config).MarkUnknown(ConditionReady, "unknown-reason", "unknown-message")

	if e, a := false, condSet.Manage(config).IsHappy(); e != a {
		t.Errorf("%q expected: %v got: %v", "mark unknown", e, a)
	}

	expected := &Condition{
		Type:    ConditionReady,
		Status:  corev1.ConditionUnknown,
		Reason:  "unknown-reason",
		Message: "unknown-message",
	}

	e, a := expected, condSet.Manage(config).GetCondition(ConditionReady)
	ignoreArguments := cmpopts.IgnoreFields(Condition{}, "LastTransitionTime")
	if diff := cmp.Diff(e, a, ignoreArguments); diff != "" {
		t.Errorf("markUnknown (-want, +got) = %v", diff)
	}
}

func TestInitializeConditions(t *testing.T) {
	condSet := NewLivingConditionSet()

	cases := []struct {
		name       string
		conditions Conditions
		condition  *Condition
	}{{
		name: "initialized",
		condition: &Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionUnknown,
		},
	}, {
		name: "already initialized",
		conditions: Conditions{{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		}},
		condition: &Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := &TestConds{Conditions: tc.conditions}
			condSet.Manage(config).InitializeConditions()
			if e, a := tc.condition, condSet.Manage(config).GetCondition(ConditionReady); !equality.Semantic.DeepEqual(e, a) {
				t.Errorf("%q expected: %v got: %v", tc.name, e, a)
			}
		})
	}
}
