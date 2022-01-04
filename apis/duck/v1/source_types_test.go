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

package v1

import (
	"context"
	"testing"

	"knative.dev/pkg/apis"
)

func TestSourceValidate(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  *Source
		want *apis.FieldError
	}{{
		name: "nil source validation",
		src:  nil,
		want: nil,
	}, {
		name: "nil source spec validation",
		src:  &Source{},
		want: apis.ErrGeneric("expected at least one, got none", "spec.sink.ref", "spec.sink.uri"),
	}, {
		name: "empty source spec validation",
		src:  &Source{Spec: SourceSpec{}},
		want: apis.ErrGeneric("expected at least one, got none", "spec.sink.ref", "spec.sink.uri"),
	}, {
		name: "empty source ceOverrides extensions validation",
		src: &Source{Spec: SourceSpec{
			Sink: Destination{
				URI: func() *apis.URL {
					u, _ := apis.ParseURL("https://localhost")
					return u
				}(),
			},
			CloudEventOverrides: &CloudEventOverrides{Extensions: map[string]string{}},
		}},
		want: nil,
	}, {
		name: "empty extension name error",
		src: &Source{Spec: SourceSpec{
			Sink: Destination{
				URI: func() *apis.URL {
					u, _ := apis.ParseURL("https://localhost")
					return u
				}(),
			},
			CloudEventOverrides: &CloudEventOverrides{Extensions: map[string]string{"": "test"}},
		}},
		want: apis.ErrInvalidKeyName(
			"",
			"spec.ceOverrides.extensions",
			"keys MUST NOT be empty",
		),
	}, {
		name: "long extension key name is valid",
		src: &Source{Spec: SourceSpec{
			Sink: Destination{
				URI: func() *apis.URL {
					u, _ := apis.ParseURL("https://localhost")
					return u
				}(),
			},
			CloudEventOverrides: &CloudEventOverrides{
				Extensions: map[string]string{"nameLongerThan20Characters": "test"},
			},
		}},
		want: nil,
	}, {
		name: "invalid extension name",
		src: &Source{Spec: SourceSpec{
			Sink: Destination{
				URI: func() *apis.URL {
					u, _ := apis.ParseURL("https://localhost")
					return u
				}(),
			},
			CloudEventOverrides: &CloudEventOverrides{Extensions: map[string]string{"invalid_name": "test"}},
		}},
		want: apis.ErrInvalidKeyName(
			"invalid_name",
			"spec.ceOverrides.extensions",
			"keys are expected to be alphanumeric",
		),
	}, {
		name: "valid extension name",
		src: &Source{Spec: SourceSpec{
			Sink: Destination{
				URI: func() *apis.URL {
					u, _ := apis.ParseURL("https://localhost")
					return u
				}(),
			},
			CloudEventOverrides: &CloudEventOverrides{
				Extensions: map[string]string{"validName": "test"},
			},
		}},
		want: nil,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			t.Parallel()
			got := tt.src.Validate(context.TODO())
			if tt.want != nil && got.Error() != tt.want.Error() {
				t.Errorf("Unexpected error want:\n%+s\ngot:\n%+s", tt.want, got)
			}

			if tt.want == nil && got != nil {
				t.Errorf("Unexpected error want:\nnil\ngot:\n%+s", got)
			}
		})
	}
}
