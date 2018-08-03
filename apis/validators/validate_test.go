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

	"github.com/google/go-cmp/cmp"
	"github.com/knative/pkg/apis"
)

type foo struct {
	Default string `validate:"-"`
}

type foo_k8s struct {
	Default      string `validate:"-"`
	OptionalName string `validate:"QualifiedName"`
	RequiredName string `validate:"QualifiedName,Required"`
}

func TestValidate(t *testing.T) {
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantOk  bool
		wantErr []*apis.FieldError
	}{{
		name: "default",
		args: args{
			obj: foo{
				Default: "default",
			},
		},
		wantOk:  true,
		wantErr: nil,
	}, {
		name: "valid k8s",
		args: args{
			obj: foo_k8s{
				Default:      "default",
				OptionalName: "valid",
				RequiredName: "valid",
			},
		},
		wantOk:  true,
		wantErr: nil,
	}, {
		name: "missing required k8s name",
		args: args{
			obj: foo_k8s{},
		},
		wantOk: false,
		wantErr: []*apis.FieldError{{
			Message: `missing field(s)`,
			Paths:   []string{"RequiredName"},
		}},
	}, {
		name: "invalid required k8s name",
		args: args{
			obj: foo_k8s{
				RequiredName: "v@lid",
			},
		},
		wantOk: false,
		wantErr: []*apis.FieldError{{
			Message: `invalid key name "v@lid"`,
			Paths:   []string{"RequiredName"},
			Details: `name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOk, gotErr := Validate(tt.args.obj)
			if gotOk != tt.wantOk {
				t.Errorf("Validate() got Ok = %v, want %v", gotOk, tt.wantOk)
			}

			if diff := cmp.Diff(tt.wantErr, gotErr); diff != "" {
				t.Errorf("Validate() got Err (-want, +got) = %v", diff)
			}
		})
	}
}
