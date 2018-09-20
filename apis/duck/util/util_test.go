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

package util

import (
	"fmt"
	"reflect"
	"testing"

	"encoding/json"

	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getValidReference() corev1.ObjectReference {
	return corev1.ObjectReference{
		Kind:       "Channel",
		APIVersion: "channels.eventing.knative.dev/v1alpha1",
		Name:       "mychannel"}
}

type StatusWithExtraFields struct {
	Value        string                     `json:"value",omitempty"`
	Another      string                     `json:"another",omitempty"`
	Subscribable *duckv1alpha1.Subscribable `json:"subscribable,omitempty"`
}

type StatusWithMissingSubscribable struct {
	Value   string
	Another string
}

func TestFromUnstructuredSubscription(t *testing.T) {
	tcs := []struct {
		name      string
		in        unstructured.Unstructured
		want      duckv1alpha1.SubscribableStatus
		wantError error
	}{{
		name: "Works with valid status",
		in: unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "test",
				"kind":       "test_kind",
				"name":       "test_name",
				"status":     duckv1alpha1.SubscribableStatus{&duckv1alpha1.Subscribable{getValidReference()}},
			}},
		want: duckv1alpha1.SubscribableStatus{&duckv1alpha1.Subscribable{corev1.ObjectReference{
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
		want:      duckv1alpha1.SubscribableStatus{},
		wantError: nil,
	}, {
		name:      "empty unstructured",
		in:        unstructured.Unstructured{},
		want:      duckv1alpha1.SubscribableStatus{},
		wantError: nil,
	}}
	for _, tc := range tcs {
		raw, err := json.Marshal(tc.in)
		if err != nil {
			panic("failed to marshal")
		}
		fmt.Printf("Marshalled : %s", string(raw))

		got := duckv1alpha1.Subscription{}
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
