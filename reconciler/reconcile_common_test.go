/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconciler

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	v1 "knative.dev/pkg/apis/duck/v1"
)

func makeResources() (*duckv1.KResource, *duckv1.KResource) {
	foo := &apis.Condition{
		Type:    "Foo",
		Status:  corev1.ConditionTrue,
		Message: "Something something foo",
	}
	bar := &apis.Condition{
		Type:    "Ready",
		Status:  corev1.ConditionTrue,
		Message: "Something something bar",
	}

	old := &duckv1.KResource{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 0,
		},

		Status: v1.Status{
			ObservedGeneration: 0,
			Conditions:         v1.Conditions{*foo, *bar},
		},
	}

	new := &duckv1.KResource{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 1,
		},

		Status: v1.Status{
			ObservedGeneration: 0,
			Conditions:         v1.Conditions{*foo, *bar},
		},
	}
	return old, new
}

func TestPostProcessReconcileBumpsGeneration(t *testing.T) {
	old, new := makeResources()

	oldShape := duck
	duckv1.KRShaped(old)
	newShape := duck
	duckv1.KRShaped(new)
	PostProcessReconcile(context.Background(), oldShape, newShape, nil)

	if new.Status.ObservedGeneration != new.Generation {
		t.Errorf("Expected observed generation bump got=%d want=%d", new.Status.ObservedGeneration, new.Generation)
	}

	if newShape.GetStatus().ObservedGeneration != newShape.GetObjectMeta().Generation {
		t.Errorf("Expected observed generation bump got=%d want=%d", new.Status.ObservedGeneration, new.Generation)
	}
}

func TestPostProcessReconcileBumpsWithEvent(t *testing.T) {
	old, new := makeResources()
	reconcilEvent := NewEvent(corev1.EventTypeWarning, exampleStatusFailed, "")

	oldShape := duck
	duckv1.KRShaped(old)
	newShape := duck
	duckv1.KRShaped(new)
	PostProcessReconcile(context.Background(), oldShape, newShape, reconcilEvent)

	// Expect generation bumped
	if new.Status.ObservedGeneration != new.Generation {
		t.Errorf("Expected observed generation bump got=%d want=%d", new.Status.ObservedGeneration, new.Generation)
	}

	if newShape.GetStatus().ObservedGeneration != newShape.GetObjectMeta().Generation {
		t.Errorf("Expected observed generation bump got=%d want=%d", new.Status.ObservedGeneration, new.Generation)
	}

	// The old/new conditions were not changed, expect sets unknown
	if rc := new.Status.GetCondition(apis.ConditionReady); rc.Status != "Unknown" {
		t.Errorf("Expected unknown ready status got=%s", rc.Status)
	}
}

func TestPostProcessWithEventCondChange(t *testing.T) {
	old, new := makeResources()
	originalReadyStatus := old.Status.GetCondition(apis.ConditionReady).Status
	old.Status.Conditions = make([]apis.Condition, 0)
	reconcilEvent := NewEvent(corev1.EventTypeWarning, exampleStatusFailed, "")

	oldShape := duck
	duckv1.KRShaped(old)
	newShape := duck
	duckv1.KRShaped(new)
	PostProcessReconcile(context.Background(), oldShape, newShape, reconcilEvent)

	// Expect generation bumped
	if new.Status.ObservedGeneration != new.Generation {
		t.Errorf("Expected observed generation bump got=%d want=%d", new.Status.ObservedGeneration, new.Generation)
	}

	if newShape.GetStatus().ObservedGeneration != newShape.GetObjectMeta().Generation {
		t.Errorf("Expected observed generation bump got=%d want=%d", new.Status.ObservedGeneration, new.Generation)
	}

	// The old/new conditions were changed, expect that ready remains unchanged
	if rc := new.Status.GetCondition(apis.ConditionReady); rc.Status != originalReadyStatus {
		t.Errorf("Expected unchanged ready status got=%s want=%s", rc.Status, originalReadyStatus)
	}
}
