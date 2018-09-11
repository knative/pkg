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
	ConditionReady ConditionType = "Ready"
)

// ConditionStatus communicates the observed state of the Knative resource (from the controller).
type ConditionStatus struct {
	// Conditions communicates information about ongoing/complete
	// reconciliation processes that bring the "spec" inline with the observed
	// state of the world.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

// ConditionStatus defines a readiness condition for a Knative resource.
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

// IsReady looks at the conditions on the ConditionStatus.
// Returns true if condition[Ready].Status is true.
func (cs *ConditionStatus) IsReady() bool {
	if c := cs.GetCondition(ConditionReady); c != nil {
		return c.Status == corev1.ConditionTrue
	}
	return false
}

// GetCondition finds and returns the Condition that matches the ConditionType previously set on ConditionStatus.
func (cs *ConditionStatus) GetCondition(t ConditionType) *Condition {
	for _, cond := range cs.Conditions {
		if cond.Type == t {
			return &cond
		}
	}
	return nil
}

// SetCondition sets or updates the Condition on ConditionStatus for Condition.Type.
func (cs *ConditionStatus) SetCondition(new *Condition) {
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

// InitializeConditions updates the Ready Condition to unknown if not set.
func (cs *ConditionStatus) InitializeConditions() {
	if rc := cs.GetCondition(ConditionReady); rc == nil {
		cs.SetCondition(&Condition{
			Type:   ConditionReady,
			Status: corev1.ConditionUnknown,
		})
	}
}
