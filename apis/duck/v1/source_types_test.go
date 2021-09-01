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
	"fmt"
	"testing"

	"knative.dev/pkg/apis"
)

func TestSourceValidate(t *testing.T) {
	for _, tt := range []struct {
		name string
		src  *Source
		want *apis.FieldError
	}{{
		name: "empty source validation",
		src:  nil,
		want: nil,
	}, {
		name: "empty source ceOverrides extensions validation",
		src: &Source{Spec: SourceSpec{
			CloudEventOverrides: &CloudEventOverrides{Extensions: map[string]string{}},
		}},
		want: nil,
	}, {
		name: "empty extension name error",
		src: &Source{Spec: SourceSpec{
			CloudEventOverrides: &CloudEventOverrides{Extensions: map[string]string{"": "test"}},
		}},
		want: apis.ErrInvalidKeyName(
			"",
			"spec.ceOverrides.extensions",
			"CloudEvents attribute names MUST NOT be empty",
		),
	}, {
		name: "extension name too long",
		src: &Source{Spec: SourceSpec{
			CloudEventOverrides: &CloudEventOverrides{
				Extensions: map[string]string{"nameLongerThan20Characters": "test"},
			},
		}},
		want: apis.ErrInvalidKeyName(
			"nameLongerThan20Characters",
			"spec.ceOverrides.extensions",
			fmt.Sprintf("CloudEvents attribute name is longer than %d characters", MaxExtensionNameLength),
		),
	}, {
		name: "invalid extension name ",
		src: &Source{Spec: SourceSpec{
			CloudEventOverrides: &CloudEventOverrides{Extensions: map[string]string{"invalid_name": "test"}},
		}},
		want: apis.ErrInvalidKeyName(
			"invalid_name",
			"spec.ceOverrides.extensions",
			"CloudEvents attribute names MUST consist of lower-case letters ('a' to 'z') or digits ('0' to '9') from the ASCII character set",
		),
	}, {
		name: "valid extension name",
		src: &Source{Spec: SourceSpec{
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

			if tt.want == nil && tt.want != got {
				t.Errorf("Unexpected error want:\nnil\ngot:\n%+s", got)
			}
		})
	}
}
