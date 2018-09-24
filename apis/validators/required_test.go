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

type foo struct {
	Default string `validate:"Default"`
}

type complex_types struct {
	Name         string
	RequiredName string `validate:"Required"`
	Num          int
	RequiredNum  int `validate:"Required"`
	Foo          foo
	RequiredFoo  foo `validate:"Required"`
}

func TestValidate(t *testing.T) {
	tests := []ValidateTest{{
		name: "default",
		obj: foo{
			Default: "default",
		},
		want: nil,
	}, {
		name: "complex types",
		obj: complex_types{
			RequiredName: "foo",
			RequiredNum:  42,
			RequiredFoo: foo{
				Default: "hi",
			},
		},
		want: nil,
	}, {
		name: "empty complex types",
		obj:  complex_types{},
		want: (&apis.FieldError{
			Message: `missing field(s)`,
			Paths:   []string{"RequiredName"},
		}).Also(&apis.FieldError{
			Message: `missing field(s)`,
			Paths:   []string{"RequiredNum"},
		}).Also(&apis.FieldError{
			Message: `missing field(s)`,
			Paths:   []string{"RequiredFoo"},
		}),
	}}
	doTestValidate(t, tests)
}
