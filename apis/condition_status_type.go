/*
Copyright 2018 The Knative Authors

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
	"reflect"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	// ConditionReady specifies that the resource is ready.
	// For long-running resources.
	ConditionReady ConditionType = "Ready"
	// ConditionSucceeded specifies that the resource has finished.
	// For resource which run to completion.
	ConditionSucceeded ConditionType = "Succeeded"
)

const (
	ConditionTrue    = corev1.ConditionTrue
	ConditionFalse   = corev1.ConditionFalse
	ConditionUnknown = corev1.ConditionUnknown
)

// Conditions communicates the observed state of the Knative resource (from the controller).
type Conditions struct {
	// Conditions communicates information about ongoing/complete
	// reconciliation processes that bring the "spec" inline with the observed
	// state of the world.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

// Conditions defines a readiness condition for a Knative resource.
// See: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#typical-status-properties
type Condition struct {
	// Type of condition.
	// +required
	Type ConditionType `json:"type" description:"type of status condition"`

	// Status of the condition, one of True, False, Unknown.
	// +required
	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// LastTransitionTime is the last time the condition transitioned from one status to another.
	// We use VolatileTime in place of metav1.Time to exclude this from creating equality.Semantic
	// differences (all other things held constant).
	// +optional
	LastTransitionTime VolatileTime `json:"lastTransitionTime,omitempty" description:"last time the condition transit from one status to another"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

func (c *Condition) IsTrue() bool {
	return c.Status == ConditionTrue
}

// IsReady looks at the conditions on the Conditions.
// Returns true if condition[Ready].Status is true.
func (cs *Conditions) IsReady() bool {

	// check all conditions, call IsTrue on them each.
	if len(cs.Conditions) > 0 {
		for _, c := range cs.Conditions {
			if !c.IsTrue() {
				return false
			}
		}

		// The resource must also be Ready or Succeeded.
		for _, t := range []ConditionType{ConditionReady, ConditionSucceeded} {
			if c := cs.GetCondition(t); c != nil && c.IsTrue() {
				return true
			}
		}
	}
	return false
}

// GetCondition finds and returns the Condition that matches the ConditionType previously set on Conditions.
func (cs *Conditions) GetCondition(t ConditionType) *Condition {
	for _, cond := range cs.Conditions {
		if cond.Type == t {
			return &cond
		}
	}
	return nil
}

// setCondition sets or updates the Condition on Conditions for Condition.Type.
func (cs *Conditions) setCondition(new *Condition) {
	if new == nil {
		return
	}
	t := new.Type
	var conditions []Condition
	for _, cond := range cs.Conditions {
		if cond.Type != t {
			conditions = append(conditions, cond)
		} else {
			// If we'd only update the LastTransitionTime, then return.
			new.LastTransitionTime = cond.LastTransitionTime
			if reflect.DeepEqual(new, &cond) {
				return
			}
		}
	}
	new.LastTransitionTime = VolatileTime{metav1.NewTime(time.Now())}
	conditions = append(conditions, *new)
	sort.Slice(conditions, func(i, j int) bool { return conditions[i].Type < conditions[j].Type })
	cs.Conditions = conditions
}

// MarkTrue sets the status of t to true.
func (cs *Conditions) MarkTrue(t ConditionType) {
	cs.setCondition(&Condition{
		Type:   t,
		Status: corev1.ConditionTrue,
	})
}

func (cs *Conditions) MarkUnknown(t ConditionType, reason, message string) {
	cs.setCondition(&Condition{
		Type:    t,
		Status:  corev1.ConditionUnknown,
		Reason:  reason,
		Message: message,
	})
}

func (cs *Conditions) MarkFalse(t ConditionType, reason, message string) {
	cs.setCondition(&Condition{
		Type:    t,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

// InitializeConditions updates the Ready Condition to unknown if not set.
func (cs *Conditions) InitializeConditions() {
	if rc := cs.GetCondition(ConditionReady); rc == nil {
		cs.setCondition(&Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionUnknown,
		})
	}
}
