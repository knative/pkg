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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knative/pkg/apis/duck"
)

// Conditions is the schema for the conditions portion of the payload
type Conditions []Condition

type Condition struct {
	// TODO(n3wscott): Give me a schema!
	Field string `json:"field,omitempty"`
}

// Implementations can verify that they implement Conditions via:
var _ = duck.VerifyType(&KResource{}, &Conditions{})

// Conditions is an Implementable "duck type".
var _ duck.Implementable = (*Conditions)(nil)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KResource is a skeleton type wrapping Conditions in the manner we expect
// resource writers defining compatible resources to embed it.  We will
// typically use this type to deserialize Conditions ObjectReferences and
// access the Conditions data.  This is not a real resource.
type KResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status KResourceStatus `json:"status"`
}

// KResourceStatus shows how we expect folks to embed Conditions in
// their Status field.
type KResourceStatus struct {
	Conditions Conditions `json:"conditions,omitempty"`
}

// In order for Conditions to be Implementable, KResource must be Populatable.
var _ duck.Populatable = (*KResource)(nil)

// GetFullType implements duck.Implementable
func (_ *Conditions) GetFullType() duck.Populatable {
	return &KResource{}
}

// Populate implements duck.Populatable
func (t *KResource) Populate() {
	t.Status.Conditions = Conditions{{
		// Populate ALL fields
		Field: "this is not empty",
	}}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KResourceList is a list of KResource resources
type KResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KResource `json:"items"`
}
