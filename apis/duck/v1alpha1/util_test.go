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

package v1alpha1

import (
	"fmt"
	"reflect"
	"testing"

	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getValidReference() corev1.ObjectReference {
	return corev1.ObjectReference{
		Kind:       "Channel",
		APIVersion: "channels.eventing.knative.dev/v1alpha1",
		Name:       "mychannel"}
}

type StatusWithExtraFields struct {
	Value        string        `json:"value",omitempty"`
	Another      string        `json:"another",omitempty"`
	Subscribable *Subscribable `json:"subscribable,omitempty"`
}

type StatusWithMissingSubscribable struct {
	Value   string
	Another string
}

func getUnstructured() unstructured.Unstructured {
	g := Generational{
		metav1.TypeMeta{
			Kind:       "test_kind",
			APIVersion: "test_kind",
		},
		metav1.ObjectMeta{Name: "test_name"},
		GenerationalSpec{1234},
	}
	raw, err := json.Marshal(g)
	if err != nil {
		panic("failed to marshal")
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		fmt.Printf("Failed to unmarshal: %s", err)
		panic("failed to unmarshal")
	}
	return unstructured.Unstructured{Object: m}

}

func TestFromUnstructuredSubscription(t *testing.T) {
	tcs := []struct {
		name      string
		in        unstructured.Unstructured
		want      SubscribableStatus
		wantError error
	}{{
		name: "Works with valid status",
		in: unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "test",
				"kind":       "test_kind",
				"name":       "test_name",
				"status":     SubscribableStatus{&Subscribable{getValidReference()}},
			}},
		want: SubscribableStatus{&Subscribable{corev1.ObjectReference{
			Kind:       "Channel",
			APIVersion: "channels.eventing.knative.dev/v1alpha1",
			Name:       "mychannel"}}},
		wantError: nil,
	}, {
		name: "does not work with missing subscribable status",
		in: unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "test",
				"kind":       "test_kind",
				"name":       "test_name",
				"status":     StatusWithMissingSubscribable{"first", "second"},
			}},
		want:      SubscribableStatus{},
		wantError: nil,
	}, {
		name:      "empty unstructured",
		in:        unstructured.Unstructured{},
		want:      SubscribableStatus{},
		wantError: nil,
	}}
	for _, tc := range tcs {
		raw, err := json.Marshal(tc.in)
		if err != nil {
			panic("failed to marshal")
		}
		fmt.Printf("Marshalled : %s", string(raw))

		got := Subscription{}
		err = FromUnstructured(tc.in, &got)
		if err != tc.wantError {
			t.Errorf("Unexpected error for %q: %v", string(tc.name), err)
			continue
		}

		if !reflect.DeepEqual(tc.want, got.Status) {
			t.Errorf("Decode(%q) want: %+v\ngot: %+v", string(tc.name), tc.want, got)
		}
	}
}
