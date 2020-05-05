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

func makeResource() *duckv1.KResource {
	fooCond := apis.Condition{
		Type:    "Foo",
		Status:  corev1.ConditionTrue,
		Message: "Something something foo",
	}
	readyCond := apis.Condition{
		Type:    "Ready",
		Status:  corev1.ConditionTrue,
		Message: "Something something bar",
	}

	return &duckv1.KResource{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 1,
		},

		Status: v1.Status{
			ObservedGeneration: 0,
			Conditions:         v1.Conditions{fooCond, readyCond},
		},
	}
}

func TestPreProcessResetsReady(t *testing.T) {
	resource := makeResource()

	krShape := duckv1.KRShaped(resource)
	PreProcessReconcile(context.Background(), krShape)

	// Expect ready to be reset to Unknown
	if rc := resource.Status.GetCondition(apis.ConditionReady); rc.Status != "Unknown" {
		t.Errorf("Expected unchanged ready status got=%s want=Unknown", rc.Status)
	}
}

func TestPostProcessReconcileBumpsGeneration(t *testing.T) {
	resource := makeResource()

	krShape := duckv1.KRShaped(resource)
	PostProcessReconcile(context.Background(), krShape)

	if resource.Status.ObservedGeneration != resource.Generation {
		t.Errorf("Expected observed generation bump got=%d want=%d", resource.Status.ObservedGeneration, resource.Generation)
	}

	if krShape.GetStatus().ObservedGeneration != krShape.GetObjectMeta().GetGeneration() {
		t.Errorf("Expected observed generation bump got=%d want=%d", resource.Status.ObservedGeneration, resource.Generation)
	}
}
