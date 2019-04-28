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

package webhook

import (
	"testing"

	"github.com/knative/pkg/apis"
	. "github.com/knative/pkg/logging/testing"
	"github.com/knative/pkg/ptr"
	. "github.com/knative/pkg/testing"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// In strict mode, you are not allowed to set a deprecated filed when doing a Create.
func TestStrictValidation(t *testing.T) {

	newCreateReq := func(new []byte) *admissionv1beta1.AdmissionRequest {
		req := &admissionv1beta1.AdmissionRequest{
			Operation: admissionv1beta1.Create,
			Kind: metav1.GroupVersionKind{
				Group:   "pkg.knative.dev",
				Version: "v1alpha1",
				Kind:    "InnerDefaultResource",
			},
		}
		req.Object.Raw = new
		return req
	}

	newUpdateReq := func(old, new []byte) *admissionv1beta1.AdmissionRequest {
		req := &admissionv1beta1.AdmissionRequest{
			Operation: admissionv1beta1.Create,
			Kind: metav1.GroupVersionKind{
				Group:   "pkg.knative.dev",
				Version: "v1alpha1",
				Kind:    "InnerDefaultResource",
			},
		}
		req.OldObject.Raw = old
		req.Object.Raw = new
		return req
	}

	testCases := map[string]struct {
		strict   bool
		req      *admissionv1beta1.AdmissionRequest
		wantErrs []string
	}{
		"create, strict": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				DeprecatedField: "fail setting.",
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.field",
			},
		},
		"create strict, spec.sub.string": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedString: "an error",
				},
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.string",
			},
		},
		"create strict, spec.sub.stringptr": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedStringPtr: func() *string {
						s := "test string"
						return &s
					}(),
				},
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.stringPtr",
			},
		},
		"create strict, spec.sub.int": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedInt: 42,
				},
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.int",
			},
		},
		"create strict, spec.sub.intptr": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedIntPtr: ptr.Int64(42),
				},
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.intPtr",
			},
		},
		"create strict, spec.sub.map": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedMap: map[string]string{"hello": "failure"},
				},
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.map",
			},
		},
		"create strict, spec.sub.slice": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedSlice: []string{"hello", "failure"},
				},
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.slice",
			},
		},
		"create strict, spec.sub.struct": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedStruct: InnerDefaultStruct{
						FieldAsString: "fail",
					},
				},
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.struct",
			},
		},
		"create strict, spec.sub.structptr": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedStructPtr: &InnerDefaultStruct{
						FieldAsString: "fail",
					},
				},
			}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.structPtr",
			},
		},

		"update, strict": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					DeprecatedField: "fail setting.",
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.field",
			},
		},
		"update strict, spec.sub.string": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedString: "an error",
					},
				}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.string",
			},
		},
		"update strict, spec.sub.stringptr": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStringPtr: func() *string {
							s := "test string"
							return &s
						}(),
					},
				}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.stringPtr",
			},
		},
		"update strict, spec.sub.int": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedInt: 42,
					},
				}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.int",
			},
		},
		"update strict, spec.sub.intptr": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedIntPtr: ptr.Int64(42),
					},
				}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.intPtr",
			},
		},
		"update strict, spec.sub.map": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedMap: map[string]string{"hello": "failure"},
					},
				}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.map",
			},
		},
		"update strict, spec.sub.slice": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedSlice: []string{"hello", "failure"},
					},
				}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.slice",
			},
		},
		"update strict, spec.sub.struct": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStruct: InnerDefaultStruct{
							FieldAsString: "fail",
						},
					},
				}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.struct",
			},
		},
		"update strict, spec.sub.structptr": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStructPtr: &InnerDefaultStruct{
							FieldAsString: "fail",
						},
					},
				}, nil)),
			wantErrs: []string{
				"must not set",
				"spec.subFields.structPtr",
			},
		},

		"overwrite, strict": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					DeprecatedField: "original setting.",
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					DeprecatedField: "fail setting.",
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.field",
			},
		},
		"overwrite strict, spec.sub.string": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedString: "original string",
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedString: "an error",
					},
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.subFields.string",
			},
		},
		"overwrite strict, spec.sub.stringptr": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStringPtr: func() *string {
							s := "original string"
							return &s
						}(),
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStringPtr: func() *string {
							s := "test string"
							return &s
						}(),
					},
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.subFields.stringPtr",
			},
		},
		"overwrite strict, spec.sub.int": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedInt: 10,
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedInt: 42,
					},
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.subFields.int",
			},
		},
		"overwrite strict, spec.sub.intptr": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedIntPtr: ptr.Int64(10),
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedIntPtr: ptr.Int64(42),
					},
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.subFields.intPtr",
			},
		},
		"overwrite strict, spec.sub.map": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedMap: map[string]string{"goodbye": "existing"},
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedMap: map[string]string{"hello": "failure"},
					},
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.subFields.map",
			},
		},
		"overwrite strict, spec.sub.slice": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedSlice: []string{"hello", "existing"},
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedSlice: []string{"hello", "failure"},
					},
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.subFields.slice",
			},
		},
		"overwrite strict, spec.sub.struct": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStruct: InnerDefaultStruct{
							FieldAsString: "original",
						},
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStruct: InnerDefaultStruct{
							FieldAsString: "fail",
						},
					},
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.subFields.struct",
			},
		},
		"overwrite strict, spec.sub.structptr": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStructPtr: &InnerDefaultStruct{
							FieldAsString: "original",
						},
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedStructPtr: &InnerDefaultStruct{
							FieldAsString: "fail",
						},
					},
				}, nil)),
			wantErrs: []string{
				"must not update",
				"spec.subFields.structPtr",
			},
		},

		"create, not strict": {
			strict: false,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				DeprecatedField: "fail setting.",
			}, nil)),
		},
		"update, not strict": {
			strict: false,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					DeprecatedField: "fail setting.",
				}, nil)),
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			ctx := TestContextWithLogger(t)
			if tc.strict {
				ctx = apis.DisallowDeprecated(ctx)
			}

			_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
			resp := ac.admit(ctx, tc.req)

			if len(tc.wantErrs) > 0 {
				for _, err := range tc.wantErrs {
					expectFailsWith(t, resp, err)
				}
			} else {
				expectAllowed(t, resp)
			}
		})
	}
}

// In strict mode, you are not allowed to set a deprecated filed when doing a Create.
func TestStrictValidation_Spec_Create(t *testing.T) {
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "InnerDefaultResource",
		},
	}
	req.Object.Raw = createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
		DeprecatedField: "fail setting.",
	}, nil)

	ctx := apis.DisallowDeprecated(TestContextWithLogger(t))

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(ctx, req)

	expectFailsWith(t, resp, "must not set")
	expectFailsWith(t, resp, "spec.field")
}

// In strict mode, you are not allowed to update a deprecated filed when doing a Update.
func TestStrictValidation_Spec_Update(t *testing.T) {
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Update,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "InnerDefaultResource",
		},
	}
	req.OldObject.Raw = createInnerDefaultResourceWithoutSpec(t)
	req.Object.Raw = createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
		DeprecatedField: "fail setting.",
	}, nil)

	ctx := apis.DisallowDeprecated(TestContextWithLogger(t))

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(ctx, req)

	expectFailsWith(t, resp, "must not update")
	expectFailsWith(t, resp, "spec.field")

}
