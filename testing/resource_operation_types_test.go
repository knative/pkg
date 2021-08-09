/*
Copyright 2021 The Knative Authors

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

package testing

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func TestResourceOperationTypesDefaultingOperationTypes(t *testing.T) {
	r := &ResourceOperationTypes{}

	operations := r.DefaultingOperationTypes()
	expected := []admissionregistrationv1.OperationType{admissionregistrationv1.Create}

	if diff := cmp.Diff(expected, operations); diff != "" {
		t.Error("(-want, got)", diff)
	}
}

func TestResourceOperationTypesValidatingOperationTypes(t *testing.T) {
	r := &ResourceOperationTypes{}

	operations := r.ValidatingOperationTypes()
	expected := []admissionregistrationv1.OperationType{admissionregistrationv1.Create}

	if diff := cmp.Diff(expected, operations); diff != "" {
		t.Error("(-want, got)", diff)
	}
}
