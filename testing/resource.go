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

package testing

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knative/pkg/apis"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Resource is a simple resource that's compatible with our webhook
type Resource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourceSpec `json:"spec,omitempty"`
}

// Check that Resource may be validated and defaulted.
var _ apis.Validatable = (*Resource)(nil)
var _ apis.Defaultable = (*Resource)(nil)
var _ apis.Immutable = (*Resource)(nil)

type ResourceSpec struct {
	Generation int64 `json:"generation,omitempty"`

	FieldWithDefault    string `json:"fieldWithDefault,omitempty"`
	FieldWithValidation string `json:"fieldWithValidation,omitempty"`
	FieldThatsImmutable string `json:"fieldThatsImmutable,omitempty"`
}

func (r *Resource) GetGeneration() int64 {
	return r.Spec.Generation
}

func (r *Resource) SetGeneration(generation int64) {
	r.Spec.Generation = generation
}

func (r *Resource) GetSpecJSON() ([]byte, error) {
	return json.Marshal(r.Spec)
}

func (c *Resource) SetDefaults() {
	c.Spec.SetDefaults()
}

func (cs *ResourceSpec) SetDefaults() {
	if cs.FieldWithDefault == "" {
		cs.FieldWithDefault = "I'm a default."
	}
}

func (c *Resource) Validate() *apis.FieldErrors {
	return c.Spec.Validate().ViaField("spec")
}

func (cs *ResourceSpec) Validate() *apis.FieldErrors {
	if cs.FieldWithValidation != "magic value" {
		return apis.ErrInvalidValue(cs.FieldWithValidation, "fieldWithValidation")
	}
	return nil
}

func (current *Resource) CheckImmutableFields(og apis.Immutable) *apis.FieldErrors {
	original, ok := og.(*Resource)
	if !ok {
		return apis.FieldError{Message: "The provided original was not a Resource"}.Wrap()
	}

	if original.Spec.FieldThatsImmutable != current.Spec.FieldThatsImmutable {
		return apis.FieldError{
			Message: "Immutable field changed",
			Paths:   []string{"spec.fieldThatsImmutable"},
			Details: fmt.Sprintf("got: %v, want: %v", current.Spec.FieldThatsImmutable,
				original.Spec.FieldThatsImmutable),
		}.Wrap()
	}
	return nil
}
