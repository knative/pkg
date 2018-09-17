///*
//Copyright 2017 The Knative Authors
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//*/
//
package apis

//
//import (
//	"testing"
//
//	"time"
//
//	"github.com/google/go-cmp/cmp"
//	"github.com/google/go-cmp/cmp/cmpopts"
//	corev1 "k8s.io/api/core/v1"
//	"k8s.io/apimachinery/pkg/api/equality"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//)
//
//func TestConfigurationIsReady(t *testing.T) {
//	cases := []struct {
//		name       string
//		conditions Conditions
//		requires   []ConditionType
//		isReady    bool
//	}{{
//		name:       "empty status should not be ready",
//		conditions: Conditions(nil),
//		requires:   []ConditionType{ConditionReady},
//		isReady:    false,
//	}, {
//		name: "Different condition type should not be ready",
//		conditions: Conditions{{
//			Type:   "Foo",
//			Status: corev1.ConditionTrue,
//		}},
//		requires: []ConditionType{ConditionReady},
//		isReady:  false,
//	}, {
//		name: "False condition status should not be ready",
//		conditions: Conditions{{
//			Type:   ConditionReady,
//			Status: corev1.ConditionFalse,
//		}},
//		requires: []ConditionType{ConditionReady},
//		isReady:  false,
//	}, {
//		name: "Unknown condition status should not be ready",
//		conditions: Conditions{{
//			Type:   ConditionReady,
//			Status: corev1.ConditionUnknown,
//		}},
//		requires: []ConditionType{ConditionReady},
//		isReady:  false,
//	}, {
//		name: "Missing condition status should not be ready",
//		conditions: Conditions{{
//			Type: ConditionReady,
//		}},
//		requires: []ConditionType{ConditionReady},
//		isReady:  false,
//	}, {
//		name: "True condition status should be ready",
//		conditions: Conditions{{
//			Type:   ConditionReady,
//			Status: corev1.ConditionTrue,
//		}},
//		requires: []ConditionType{ConditionReady},
//		isReady:  true,
//	}, {
//		name: "Multiple conditions with ready status should be ready",
//		conditions: Conditions{{
//			Type:   "Foo",
//			Status: corev1.ConditionTrue,
//		}, {
//			Type:   ConditionReady,
//			Status: corev1.ConditionTrue,
//		}},
//		requires: []ConditionType{ConditionReady, "Foo"},
//		isReady:  true,
//	}, {
//		name: "Multiple conditions with ready status false should not be ready",
//		conditions: Conditions{{
//			Type:   "Foo",
//			Status: corev1.ConditionTrue,
//		}, {
//			Type:   ConditionReady,
//			Status: corev1.ConditionFalse,
//		}},
//		requires: []ConditionType{ConditionReady, "Foo"},
//		isReady:  false,
//	}, {
//		name: "Multiple conditions with mixed ready status, some don't matter,  ready",
//		conditions: Conditions{{
//			Type:   "Foo",
//			Status: corev1.ConditionTrue,
//		}, {
//			Type:   "Bar",
//			Status: corev1.ConditionFalse,
//		}, {
//			Type:   ConditionReady,
//			Status: corev1.ConditionTrue,
//		}},
//		requires: []ConditionType{ConditionReady, "Foo"},
//		isReady:  true,
//	}, {
//		name: "Multiple conditions with mixed ready status, some don't matter, not ready",
//		conditions: Conditions{{
//			Type:   "Foo",
//			Status: corev1.ConditionTrue,
//		}, {
//			Type:   "Bar",
//			Status: corev1.ConditionTrue,
//		}, {
//			Type:   ConditionReady,
//			Status: corev1.ConditionFalse,
//		}},
//		requires: []ConditionType{ConditionReady, "Foo"},
//		isReady:  false,
//	}}
//
//	for _, tc := range cases {
//		t.Run(tc.name, func(t *testing.T) {
//			if e, a := tc.isReady, tc.conditions.IsReady(tc.requires); e != a {
//				t.Errorf("%q expected: %v got: %v", tc.name, e, a)
//			}
//		})
//	}
//}
//
//func TestUpdateLastTransitionTime(t *testing.T) {
//
//	cases := []struct {
//		name       string
//		conditions Conditions
//		condition  Condition
//		update     bool
//	}{{
//		name: "LastTransitionTime should be set",
//		conditions: Conditions{{
//			Type:   ConditionReady,
//			Status: corev1.ConditionFalse,
//		}},
//		condition: Condition{
//			Type:   ConditionReady,
//			Status: corev1.ConditionTrue,
//		},
//		update: true,
//	}, {
//		name: "LastTransitionTime should update",
//		conditions: Conditions{{
//			Type:               ConditionReady,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: VolatileTime{metav1.NewTime(time.Unix(1337, 0))},
//		}},
//		condition: Condition{
//			Type:   ConditionReady,
//			Status: corev1.ConditionTrue,
//		},
//		update: true,
//	}, {
//		name: "if LastTransitionTime is the only chance, don't do it",
//		conditions: Conditions{{
//			Type:               ConditionReady,
//			Status:             corev1.ConditionFalse,
//			LastTransitionTime: VolatileTime{metav1.NewTime(time.Unix(1337, 0))},
//		}},
//		condition: Condition{
//			Type:   ConditionReady,
//			Status: corev1.ConditionFalse,
//		},
//		update: false,
//	}}
//
//	for _, tc := range cases {
//		t.Run(tc.name, func(t *testing.T) {
//			was := tc.conditions.GetCondition(tc.condition.Type)
//			tc.conditions = tc.conditions.SetCondition(&tc.condition)
//			now := tc.conditions.GetCondition(tc.condition.Type)
//
//			if tc.update {
//				if e, a := was.LastTransitionTime, now.LastTransitionTime; e == a {
//					t.Errorf("%q expected: %v to not match %v", tc.name, e, a)
//				}
//			} else {
//				if e, a := was.LastTransitionTime, now.LastTransitionTime; e != a {
//					t.Errorf("%q expected: %v to match %v", tc.name, e, a)
//				}
//			}
//		})
//	}
//}
//
//type testResource struct {
//	Conditions Conditions
//}
//
//func TestResourceConditions(t *testing.T) {
//	config := &testResource{}
//	foo := &Condition{
//		Type:   "Foo",
//		Status: "True",
//	}
//	bar := &Condition{
//		Type:   "Bar",
//		Status: "True",
//	}
//
//	// Add a new condition.
//	config.Conditions = config.Conditions.SetCondition(foo)
//
//	if got, want := len(config.Conditions), 1; got != want {
//		t.Fatalf("Unexpected Condition length; got %d, want %d", got, want)
//	}
//
//	// Add a second condition.
//	config.Conditions = config.Conditions.SetCondition(bar)
//
//	if got, want := len(config.Conditions), 2; got != want {
//		t.Fatalf("Unexpected Condition length; got %d, want %d", got, want)
//	}
//
//	// Add nil condition.
//	config.Conditions.SetCondition(nil)
//
//	if got, want := len(config.Conditions), 2; got != want {
//		t.Fatalf("Unexpected Condition length; got %d, want %d", got, want)
//	}
//}
//
//func TestMarkTrue(t *testing.T) {
//	c := Conditions{{
//		Type:   ConditionReady,
//		Status: corev1.ConditionFalse,
//	}}
//	c = c.MarkTrue(ConditionReady)
//
//	if e, a := true, c.IsReady([]ConditionType{ConditionReady}); e != a {
//		t.Errorf("%q expected: %v got: %v", "mark true", e, a)
//	}
//
//	expected := &Condition{
//		Type:   ConditionReady,
//		Status: corev1.ConditionTrue,
//	}
//
//	e, a := expected, c.GetCondition(ConditionReady)
//	ignoreArguments := cmpopts.IgnoreFields(Condition{}, "LastTransitionTime")
//	if diff := cmp.Diff(e, a, ignoreArguments); diff != "" {
//		t.Errorf("markTrue (-want, +got) = %v", diff)
//	}
//}
//
//func TestMarkFalse(t *testing.T) {
//	c := Conditions{{
//		Type:   ConditionReady,
//		Status: corev1.ConditionTrue,
//	}}
//	c = c.MarkFalse(ConditionReady, "false-reason", "false-message")
//
//	if e, a := false, c.IsReady([]ConditionType{ConditionReady}); e != a {
//		t.Errorf("%q expected: %v got: %v", "mark false", e, a)
//	}
//
//	expected := &Condition{
//		Type:    ConditionReady,
//		Status:  corev1.ConditionFalse,
//		Reason:  "false-reason",
//		Message: "false-message",
//	}
//
//	e, a := expected, c.GetCondition(ConditionReady)
//	ignoreArguments := cmpopts.IgnoreFields(Condition{}, "LastTransitionTime")
//	if diff := cmp.Diff(e, a, ignoreArguments); diff != "" {
//		t.Errorf("markFalse (-want, +got) = %v", diff)
//	}
//}
//
//func TestMarkUnknown(t *testing.T) {
//	c := Conditions{{
//		Type:   ConditionReady,
//		Status: corev1.ConditionTrue,
//	}}
//	c = c.MarkUnknown(ConditionReady, "unknown-reason", "unknown-message")
//
//	if e, a := false, c.IsReady([]ConditionType{ConditionReady}); e != a {
//		t.Errorf("%q expected: %v got: %v", "mark unknown", e, a)
//	}
//
//	expected := &Condition{
//		Type:    ConditionReady,
//		Status:  corev1.ConditionUnknown,
//		Reason:  "unknown-reason",
//		Message: "unknown-message",
//	}
//
//	e, a := expected, c.GetCondition(ConditionReady)
//	ignoreArguments := cmpopts.IgnoreFields(Condition{}, "LastTransitionTime")
//	if diff := cmp.Diff(e, a, ignoreArguments); diff != "" {
//		t.Errorf("markUnknown (-want, +got) = %v", diff)
//	}
//}
//
//func TestInitializeConditions(t *testing.T) {
//	cases := []struct {
//		name       string
//		conditions Conditions
//		condition  *Condition
//	}{{
//		name: "initialized",
//		condition: &Condition{
//			Type:   ConditionReady,
//			Status: corev1.ConditionUnknown,
//		},
//	}, {
//		name: "already initialized",
//		conditions: Conditions{{
//			Type:   ConditionReady,
//			Status: corev1.ConditionFalse,
//		}},
//		condition: &Condition{
//			Type:   ConditionReady,
//			Status: corev1.ConditionFalse,
//		},
//	}}
//
//	for _, tc := range cases {
//		t.Run(tc.name, func(t *testing.T) {
//			tc.conditions = tc.conditions.InitializeConditions([]ConditionType{ConditionReady})
//			if e, a := tc.condition, tc.conditions.GetCondition(ConditionReady); !equality.Semantic.DeepEqual(e, a) {
//				t.Errorf("%q expected: %v got: %v", tc.name, e, a)
//			}
//		})
//	}
//}
