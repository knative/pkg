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

package validators

import (
	"testing"

	"github.com/knative/pkg/apis"
)

type foo_k8s struct {
	Default      string `json:"default,omitempty"`
	OptionalName string `json:"optionalName" validate:"QualifiedName"`
	RequiredName string `json:"requiredName" validate:"QualifiedName;Required"`
}

type non_json_k8s struct {
	OptionalName string `validate:"QualifiedName"`
	RequiredName string `validate:"QualifiedName;Required"`
}

func TestK8sQualifiedNameValidate(t *testing.T) {
	tests := []ValidateTest{{
		name: "valid k8s",
		obj: foo_k8s{
			Default:      "default",
			OptionalName: "valid",
			RequiredName: "valid",
		},
		want: nil,
	}, {
		name: "missing required k8s name",
		obj:  foo_k8s{},
		want: &apis.FieldError{
			Message: `missing field(s)`,
			Paths:   []string{"requiredName"},
		},
	}, {
		name: "invalid optional k8s name",
		obj: foo_k8s{
			Default:      "default",
			OptionalName: "v@lid",
			RequiredName: "valid",
		},
		want: &apis.FieldError{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"optionalName"},
			Details: invalidQualifiedNameError,
		},
	}, {
		name: "invalid required k8s name",
		obj: foo_k8s{
			RequiredName: "v@lid",
		},
		want: &apis.FieldError{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"requiredName"},
			Details: invalidQualifiedNameError,
		},
	}, {
		name: "invalid optional and required k8s names",
		obj: foo_k8s{
			OptionalName: "val!d",
			RequiredName: "v@lid",
		},
		want: (&apis.FieldError{
			Message: `invalid key name "val!d"`,
			Paths:   []string{"optionalName"},
			Details: invalidQualifiedNameError,
		}).Also(&apis.FieldError{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"requiredName"},
			Details: invalidQualifiedNameError,
		}),
	}, {
		name: "non-json invalid optional and required k8s names",
		obj: non_json_k8s{
			OptionalName: "val!d",
			RequiredName: "v@lid",
		},
		want: (&apis.FieldError{
			Message: `invalid key name "val!d"`,
			Paths:   []string{"OptionalName"},
			Details: invalidQualifiedNameError,
		}).Also(&apis.FieldError{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"RequiredName"},
			Details: invalidQualifiedNameError,
		}),
	}}
	doTestValidate(t, tests)
}
