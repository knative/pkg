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

	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigurationIsReady(t *testing.T) {
	cases := []struct {
		name    string
		status  ConditionStatus
		isReady bool
	}{{
		name:    "empty status should not be ready",
		status:  ConditionStatus{},
		isReady: false,
	}, {
		name: "Different condition type should not be ready",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:   "Foo",
				Status: corev1.ConditionTrue,
			}},
		},
		isReady: false,
	}, {
		name: "False condition status should not be ready",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:   ConditionReady,
				Status: corev1.ConditionFalse,
			}},
		},
		isReady: false,
	}, {
		name: "Unknown condition status should not be ready",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:   ConditionReady,
				Status: corev1.ConditionUnknown,
			}},
		},
		isReady: false,
	}, {
		name: "Missing condition status should not be ready",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type: ConditionReady,
			}},
		},
		isReady: false,
	}, {
		name: "True condition status should be ready",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:   ConditionReady,
				Status: corev1.ConditionTrue,
			}},
		},
		isReady: true,
	}, {
		name: "Multiple conditions with ready status should be ready",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:   "Foo",
				Status: corev1.ConditionTrue,
			}, {
				Type:   ConditionReady,
				Status: corev1.ConditionTrue,
			}},
		},
		isReady: true,
	}, {
		name: "Multiple conditions with ready status false should not be ready",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:   "Foo",
				Status: corev1.ConditionTrue,
			}, {
				Type:   ConditionReady,
				Status: corev1.ConditionFalse,
			}},
		},
		isReady: false,
	}}

	for _, tc := range cases {
		if e, a := tc.isReady, tc.status.IsReady(); e != a {
			t.Errorf("%q expected: %v got: %v", tc.name, e, a)
		}
	}
}

func TestUpdateLastTransitionTime(t *testing.T) {

	cases := []struct {
		name      string
		status    ConditionStatus
		condition Condition
		update    bool
	}{{
		name: "LastTransitionTime should be set",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:   ConditionReady,
				Status: corev1.ConditionFalse,
			}},
		},
		condition: Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionTrue,
		},
		update: true,
	}, {
		name: "LastTransitionTime should update",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:               ConditionReady,
				Status:             corev1.ConditionFalse,
				LastTransitionTime: VolatileTime{metav1.NewTime(time.Unix(1337, 0))},
			}},
		},
		condition: Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionTrue,
		},
		update: true,
	}, {
		name: "if LastTransitionTime is the only chance, don't do it",
		status: ConditionStatus{
			Conditions: []Condition{{
				Type:               ConditionReady,
				Status:             corev1.ConditionFalse,
				LastTransitionTime: VolatileTime{metav1.NewTime(time.Unix(1337, 0))},
			}},
		},
		condition: Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionFalse,
		},
		update: false,
	}}

	for _, tc := range cases {
		was := tc.status.GetCondition(tc.condition.Type)
		tc.status.SetCondition(&tc.condition)
		now := tc.status.GetCondition(tc.condition.Type)

		if tc.update {
			if e, a := was.LastTransitionTime, now.LastTransitionTime; e == a {
				t.Errorf("%q expected: %v to not match %v", tc.name, e, a)
			}
		} else {
			if e, a := was.LastTransitionTime, now.LastTransitionTime; e != a {
				t.Errorf("%q expected: %v to match %v", tc.name, e, a)
			}
		}
	}
}

type testResource struct {
	Status ConditionStatus
}

func TestResourceConditions(t *testing.T) {
	config := &testResource{}
	foo := &Condition{
		Type:   "Foo",
		Status: "True",
	}
	bar := &Condition{
		Type:   "Bar",
		Status: "True",
	}

	// Add a new condition.
	config.Status.SetCondition(foo)

	if got, want := len(config.Status.Conditions), 1; got != want {
		t.Fatalf("Unexpected Condition length; got %d, want %d", got, want)
	}

	// Add a second condition.
	config.Status.SetCondition(bar)

	if got, want := len(config.Status.Conditions), 2; got != want {
		t.Fatalf("Unexpected Condition length; got %d, want %d", got, want)
	}

	// Add nil condition.
	config.Status.SetCondition(nil)

	if got, want := len(config.Status.Conditions), 2; got != want {
		t.Fatalf("Unexpected Condition length; got %d, want %d", got, want)
	}
}

func TestInitializeConditions(t *testing.T) {

	s := ConditionStatus{}
	s.InitializeConditions()

	e := &Condition{
		Type:   ConditionReady,
		Status: corev1.ConditionUnknown,
	}

	a := s.GetCondition(ConditionReady)

	if !equality.Semantic.DeepEqual(e, a) {
		t.Errorf("expected: %v got:  %v", e, a)
	}
}
