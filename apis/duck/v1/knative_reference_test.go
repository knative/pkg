/*
Copyright 2020 The Knative Authors

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestValidate(t *testing.T) {
	ctx := context.Background()

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
				fe := apis.ErrMissingField("name", "kind", "apiVersion")
				return fe
			}(),
		},
		"valid ref": {
			ref:  &validRef,
			want: nil,
		},
		"invalid ref, empty": {
			ref:  &KnativeReference{},
			want: apis.ErrMissingField("name", "kind", "apiVersion"),
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
					t.Errorf("Got: %v wanted %v", gotErr, tc.want)
				}
			} else if gotErr != nil {
				t.Errorf("Validate() = %v, wanted nil", gotErr)
			}
		})
	}
}

func TestKnativeReferenceSetDefaults(t *testing.T) {
	ctx := context.Background()

	parentNamespace := "parentNamespace"

	tests := map[string]struct {
		ref  *KnativeReference
		ctx  context.Context
		want string
	}{
		"namespace set, nothing in context, not modified ": {
			ref:  &KnativeReference{Namespace: namespace},
			ctx:  ctx,
			want: namespace,
		},
		"namespace set, context set, not modified ": {
			ref:  &KnativeReference{Namespace: namespace},
			ctx:  apis.WithinParent(ctx, metav1.ObjectMeta{Namespace: parentNamespace}),
			want: namespace,
		},
		"namespace not set, context set, defaulted": {
			ref:  &KnativeReference{},
			ctx:  apis.WithinParent(ctx, metav1.ObjectMeta{Namespace: parentNamespace}),
			want: parentNamespace,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.ref.SetDefaults(tc.ctx)
			if tc.ref.Namespace != tc.want {
				t.Errorf("Got: %s wanted %s", tc.ref.Namespace, tc.want)
			}
		})
	}
}
