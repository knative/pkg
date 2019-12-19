/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Veroute.on 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconciler

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
)

var exampleStatusFailed = "ExampleStatusFailed"

func TestNil_Is(t *testing.T) {
	var err error
	if errors.Is(err, NewReconcilerEvent(corev1.EventTypeWarning, exampleStatusFailed, "")) {
		t.Error("not a ReconcilerEvent")
	}
}

func TestError_Is(t *testing.T) {
	err := errors.New("some other error")
	if errors.Is(err, NewReconcilerEvent(corev1.EventTypeWarning, exampleStatusFailed, "")) {
		t.Error("not a ReconcilerEvent")
	}
}

func TestNew_Is(t *testing.T) {
	err := NewReconcilerEvent(corev1.EventTypeWarning, exampleStatusFailed, "this is an example error, %s", "yep")
	if !errors.Is(err, NewReconcilerEvent(corev1.EventTypeWarning, exampleStatusFailed, "")) {
		t.Error("not a ReconcilerEvent")
	}
}

func TestNewOtherType_Is(t *testing.T) {
	err := NewReconcilerEvent(corev1.EventTypeNormal, exampleStatusFailed, "this is an example error, %s", "yep")
	if errors.Is(err, NewReconcilerEvent(corev1.EventTypeWarning, exampleStatusFailed, "")) {
		t.Error("not a Warn, ExampleStatusFailed")
	}
}

func TestNewOtherReason_Is(t *testing.T) {
	err := NewReconcilerEvent(corev1.EventTypeWarning, "otherReason", "this is an example error, %s", "yep")
	if errors.Is(err, NewReconcilerEvent(corev1.EventTypeWarning, exampleStatusFailed, "")) {
		t.Error("not a Warn, ExampleStatusFailed")
	}
}

func TestNew_As(t *testing.T) {
	err := NewReconcilerEvent(corev1.EventTypeWarning, exampleStatusFailed, "this is an example error, %s", "yep")

	var event *ReconcilerEvent
	if errors.As(err, &event) {
		if event.EventType != "Warning" {
			t.Error("mismatched reason")
		}
		if event.Reason != exampleStatusFailed {
			t.Error("mismatched eventtype")
		}
	} else {
		t.Error("not a ReconcilerEvent")
	}
}

func TestNil_As(t *testing.T) {
	var err error

	var event *ReconcilerEvent
	if errors.As(err, &event) {
		t.Error("not a ReconcilerEvent")
	}
}

func TestNew_Error(t *testing.T) {
	err := NewReconcilerEvent(corev1.EventTypeWarning, exampleStatusFailed, "this is an example error, %s", "yep")

	want := "this is an example error, yep"
	got := err.Error()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected diff (-want, +got) = %v", diff)
	}
}
