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

const invalidQualifiedNameError = `name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`

type ValidateTest struct {
	name string
	obj  interface{}
	want *apis.FieldError
}

func doTestValidate(t *testing.T, tests []ValidateTest) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.obj)
			if diff := cmp.Diff(tt.want.Error(), got.Error()); diff != "" {
				t.Errorf("Validate() (-want, +got) = %v", diff)
			}
		})
	}
}
