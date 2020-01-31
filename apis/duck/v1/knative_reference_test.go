/*
Copyright 2019 The Knative Authors

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

package v1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"knative.dev/pkg/apis"
)

func TestValidate(t *testing.T) {
	ctx := context.TODO()

	validRef := KnativeReference{
		Kind:       kind,
		APIVersion: apiVersion,
		Name:       name,
		Namespace:  namespace,
	}

	tests := map[string]struct {
		ref  *KnativeReference
		want *apis.FieldError
	}{
		"nil valid": {
			ref: nil,
			want: func() *apis.FieldError {
				fe := apis.ErrMissingField("name", "namespace", "kind", "apiVersion")
				return fe
			}(),
		},
		"valid ref": {
			ref:  &validRef,
			want: nil,
		},
		"invalid ref, empty": {
			ref:  &KnativeReference{},
			want: apis.ErrMissingField("name", "namespace", "kind", "apiVersion"),
		},
		"invalid ref, missing namespace": {
			ref: &KnativeReference{
				Name:       name,
				Kind:       kind,
				APIVersion: apiVersion,
			},
			want: func() *apis.FieldError {
				fe := apis.ErrMissingField("namespace")
				return fe
			}(),
		},
		"invalid ref, missing kind": {
			ref: &KnativeReference{
				Namespace:  namespace,
				Name:       name,
				APIVersion: apiVersion,
			},
			want: func() *apis.FieldError {
				fe := apis.ErrMissingField("kind")
				return fe
			}(),
		},
		"invalid ref, missing api version": {
			ref: &KnativeReference{
				Namespace: namespace,
				Name:      name,
				Kind:      kind,
			},
			want: func() *apis.FieldError {
				return apis.ErrMissingField("apiVersion")
			}(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotErr := tc.ref.Validate(ctx)

			if tc.want != nil {
				if diff := cmp.Diff(tc.want.Error(), gotErr.Error()); diff != "" {
					t.Errorf("%s: got: %v wanted %v", name, gotErr, tc.want)
				}
			} else if gotErr != nil {
				t.Errorf("%s: Validate() = %v, wanted nil", name, gotErr)
			}
		})
	}
}
