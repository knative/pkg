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

package validation_test

import (
	"context"
	"github.com/knative/pkg/apis"
	"github.com/knative/pkg/ptr"
	. "github.com/knative/pkg/testing"
	"github.com/knative/pkg/validation"
	"strings"
	"testing"
)

// In strict mode, you are not allowed to set a deprecated filed when doing a Create.
func TestStrictValidation(t *testing.T) {

	testCases := map[string]struct {
		strict   bool
		obj      interface{}
		org      interface{}
		wantErrs []string
	}{
		"create strict, string": {
			strict: true,
			obj: &InnerDefaultSubSpec{
				DeprecatedString: "an error",
			},
			wantErrs: []string{
				"must not set",
				"DeprecatedString",
			},
		},
		"create strict, stringptr": {
			strict: true,
			obj: &InnerDefaultSubSpec{
				DeprecatedStringPtr: ptr.String("test string"),
			},
			wantErrs: []string{
				"must not set",
				"DeprecatedStringPtr",
			},
		},
		"create strict, int": {
			strict: true,
			obj: &InnerDefaultSubSpec{
				DeprecatedInt: 42,
			},
			wantErrs: []string{
				"must not set",
				"DeprecatedInt",
			},
		},
		"create strict, intptr": {
			strict: true,
			obj: &InnerDefaultSubSpec{
				DeprecatedIntPtr: ptr.Int64(42),
			},
			wantErrs: []string{
				"must not set",
				"DeprecatedIntPtr",
			},
		},
		"create strict, map": {
			strict: true,
			obj: &InnerDefaultSubSpec{
				DeprecatedMap: map[string]string{"hello": "failure"},
			},
			wantErrs: []string{
				"must not set",
				"DeprecatedMap",
			},
		},
		"create strict, slice": {
			strict: true,
			obj: &InnerDefaultSubSpec{
				DeprecatedSlice: []string{"hello", "failure"},
			},
			wantErrs: []string{
				"must not set",
				"DeprecatedSlice",
			},
		},
		"create strict, struct": {
			strict: true,
			obj: &InnerDefaultSubSpec{
				DeprecatedStruct: InnerDefaultStruct{FieldAsString: "not ok"},
			},
			wantErrs: []string{
				"must not set",
				"DeprecatedStruct",
			},
		},
		"create strict, structptr": {
			strict: true,
			obj: &InnerDefaultSubSpec{
				DeprecatedStructPtr: &InnerDefaultStruct{
					FieldAsString: "fail",
				},
			},
			wantErrs: []string{
				"must not set",
				"DeprecatedStructPtr",
			},
		},

		"update strict, intptr": {
			strict: true,
			org:    &InnerDefaultSubSpec{},
			obj: &InnerDefaultSubSpec{
				DeprecatedIntPtr: ptr.Int64(42),
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedIntPtr",
			},
		},
		"update strict, map": {
			strict: true,
			org:    &InnerDefaultSubSpec{},
			obj: &InnerDefaultSubSpec{
				DeprecatedMap: map[string]string{"hello": "failure"},
			},

			wantErrs: []string{
				"must not update",
				"DeprecatedMap",
			},
		},
		"update strict, slice": {
			strict: true,
			org:    &InnerDefaultSubSpec{},
			obj: &InnerDefaultSubSpec{
				DeprecatedSlice: []string{"hello", "failure"},
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedSlice",
			},
		},
		"update strict, struct": {
			strict: true,
			org:    &InnerDefaultSubSpec{},
			obj: &InnerDefaultSubSpec{
				DeprecatedStruct: InnerDefaultStruct{
					FieldAsString: "fail",
				},
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedStruct",
			},
		},
		"update strict, structptr": {
			strict: true,
			org:    &InnerDefaultSubSpec{},
			obj: &InnerDefaultSubSpec{
				DeprecatedStructPtr: &InnerDefaultStruct{
					FieldAsString: "fail",
				},
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedStructPtr",
			},
		},

		"overwrite strict, string": {
			strict: true,
			org: &InnerDefaultSubSpec{
				DeprecatedString: "original setting.",
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedString: "fail setting.",
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedString",
			},
		},
		"overwrite strict, stringptr": {
			strict: true,
			org: &InnerDefaultSubSpec{
				DeprecatedStringPtr: ptr.String("original string"),
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedStringPtr: ptr.String("fail string"),
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedStringPtr",
			},
		},
		"overwrite strict, int": {
			strict: true,
			org: &InnerDefaultSubSpec{
				DeprecatedInt: 10,
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedInt: 42,
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedInt",
			},
		},
		"overwrite strict, intptr": {
			strict: true,
			org: &InnerDefaultSubSpec{
				DeprecatedIntPtr: ptr.Int64(10),
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedIntPtr: ptr.Int64(42),
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedIntPtr",
			},
		},
		"overwrite strict, map": {
			strict: true,
			org: &InnerDefaultSubSpec{
				DeprecatedMap: map[string]string{"goodbye": "existing"},
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedMap: map[string]string{"hello": "failure"},
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedMap",
			},
		},
		"overwrite strict, slice": {
			strict: true,
			org: &InnerDefaultSubSpec{
				DeprecatedSlice: []string{"hello", "existing"},
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedSlice: []string{"hello", "failure"},
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedSlice",
			},
		},
		"overwrite strict, struct": {
			strict: true,
			org: &InnerDefaultSubSpec{
				DeprecatedStruct: InnerDefaultStruct{
					FieldAsString: "original",
				},
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedStruct: InnerDefaultStruct{
					FieldAsString: "fail",
				},
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedStruct",
			},
		},
		"overwrite strict, structptr": {
			strict: true,
			org: &InnerDefaultSubSpec{
				DeprecatedStructPtr: &InnerDefaultStruct{
					FieldAsString: "original",
				},
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedStructPtr: &InnerDefaultStruct{
					FieldAsString: "fail",
				},
			},
			wantErrs: []string{
				"must not update",
				"DeprecatedStructPtr",
			},
		},

		"create, not strict": {
			strict: false,
			obj: &InnerDefaultSubSpec{
				DeprecatedString: "fail setting.",
			},
		},
		"update, not strict": {
			strict: false,
			org:    &InnerDefaultSubSpec{},
			obj: &InnerDefaultSubSpec{
				DeprecatedString: "it's k",
			},
		},
		"overwrite, not strict": {
			strict: false,
			org: &InnerDefaultSubSpec{
				DeprecatedString: "org",
			},
			obj: &InnerDefaultSubSpec{
				DeprecatedString: "it's k",
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			ctx := context.TODO()
			if tc.strict {
				ctx = apis.DisallowDeprecated(ctx)
			}
			resp := validation.CheckDeprecated(ctx, tc.obj, tc.org)

			if len(tc.wantErrs) > 0 {
				for _, err := range tc.wantErrs {
					var gotErr string
					if resp != nil {
						gotErr = resp.Error()
					}
					if !strings.Contains(gotErr, err) {
						t.Errorf("Expected failure containing %q got %q", err, gotErr)
					}
				}
			} else if resp != nil {
				t.Errorf("Expected no error, got %q", resp.Error())
			}
		})
	}
}
