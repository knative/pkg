package webhook

import (
	. "github.com/knative/pkg/logging/testing"
	. "github.com/knative/pkg/testing"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
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
				"deprecated field set: Spec.DeprecatedField",
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
				"deprecated field set: Spec.SubFields.DeprecatedString",
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
				"deprecated field set: Spec.SubFields.DeprecatedStringPtr",
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
				"deprecated field set: Spec.SubFields.DeprecatedInt",
			},
		},
		"create strict, spec.sub.intptr": {
			strict: true,
			req: newCreateReq(createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
				SubFields: &InnerDefaultSubSpec{
					DeprecatedIntPtr: func() *int {
						i := 42
						return &i
					}(),
				},
			}, nil)),
			wantErrs: []string{
				"deprecated field set: Spec.SubFields.DeprecatedIntPtr",
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
				"deprecated field set: Spec.SubFields.DeprecatedMap",
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
				"deprecated field set: Spec.SubFields.DeprecatedSlice",
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
				"deprecated field set: Spec.SubFields.DeprecatedStruct",
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
				"deprecated field set: Spec.SubFields.DeprecatedStructPtr",
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
				"deprecated field updated: Spec.DeprecatedField",
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
				"deprecated field set: Spec.SubFields.DeprecatedString",
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
				"deprecated field set: Spec.SubFields.DeprecatedStringPtr",
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
				"deprecated field set: Spec.SubFields.DeprecatedInt",
			},
		},
		"update strict, spec.sub.intptr": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedIntPtr: func() *int {
							i := 42
							return &i
						}(),
					},
				}, nil)),
			wantErrs: []string{
				"deprecated field set: Spec.SubFields.DeprecatedIntPtr",
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
				"deprecated field set: Spec.SubFields.DeprecatedMap",
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
				"deprecated field set: Spec.SubFields.DeprecatedSlice",
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
				"deprecated field set: Spec.SubFields.DeprecatedStruct",
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
				"deprecated field set: Spec.SubFields.DeprecatedStructPtr",
			},
		},
		"update strict, spec.sub.slicestruct": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithoutSpec(t),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						SliceStruct: []InnerDefaultStruct{
							InnerDefaultStruct{DeprecatedField: "error"},
						},
					},
				}, nil)),
			wantErrs: []string{
				"deprecated field set: Spec.SubFields.SliceStruct[0].DeprecatedField",
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
				"deprecated field updated: Spec.DeprecatedField",
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
				"deprecated field updated: Spec.SubFields.DeprecatedString",
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
				"deprecated field updated: Spec.SubFields.DeprecatedStringPtr",
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
				"deprecated field updated: Spec.SubFields.DeprecatedInt",
			},
		},
		"overwrite strict, spec.sub.intptr": {
			strict: true,
			req: newUpdateReq(
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedIntPtr: func() *int {
							i := 10
							return &i
						}(),
					},
				}, nil),
				createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
					SubFields: &InnerDefaultSubSpec{
						DeprecatedIntPtr: func() *int {
							i := 42
							return &i
						}(),
					},
				}, nil)),
			wantErrs: []string{
				"deprecated field updated: Spec.SubFields.DeprecatedIntPtr",
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
				"deprecated field updated: Spec.SubFields.DeprecatedMap",
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
				"deprecated field updated: Spec.SubFields.DeprecatedSlice",
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
				"deprecated field updated: Spec.SubFields.DeprecatedStruct",
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
				"deprecated field updated: Spec.SubFields.DeprecatedStructPtr",
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

			opts := newDefaultOptions()
			opts.Strict = tc.strict

			_, ac := newNonRunningTestAdmissionController(t, opts)
			resp := ac.admit(TestContextWithLogger(t), tc.req)

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

	opts := newDefaultOptions()
	opts.Strict = true

	_, ac := newNonRunningTestAdmissionController(t, opts)
	resp := ac.admit(TestContextWithLogger(t), req)

	expectFailsWith(t, resp, "deprecated")
	expectFailsWith(t, resp, "Spec.DeprecatedField")
}

// In strict mode, you are not allowed to set a deprecated filed when doing a Create.
func TestStrictValidation_Status_Create(t *testing.T) {
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "InnerDefaultResource",
		},
	}
	req.Object.Raw = createInnerDefaultResourceWithSpecAndStatus(t, nil, &InnerDefaultStatus{
		DeprecatedField: "fail setting.",
	})

	opts := newDefaultOptions()
	opts.Strict = true

	_, ac := newNonRunningTestAdmissionController(t, opts)
	resp := ac.admit(TestContextWithLogger(t), req)

	expectFailsWith(t, resp, "deprecated")
	expectFailsWith(t, resp, "Status.DeprecatedField")
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

	opts := newDefaultOptions()
	opts.Strict = true

	_, ac := newNonRunningTestAdmissionController(t, opts)
	resp := ac.admit(TestContextWithLogger(t), req)

	expectFailsWith(t, resp, "deprecated")
	expectFailsWith(t, resp, "Spec.DeprecatedField")

}

// In strict mode, you are not allowed to update a deprecated filed when doing a Update.
func TestStrictValidation_Status_Update(t *testing.T) {
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Update,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "InnerDefaultResource",
		},
	}
	req.OldObject.Raw = createInnerDefaultResourceWithoutSpec(t)
	req.Object.Raw = createInnerDefaultResourceWithSpecAndStatus(t, nil, &InnerDefaultStatus{
		DeprecatedField: "fail setting.",
	})

	opts := newDefaultOptions()
	opts.Strict = true

	_, ac := newNonRunningTestAdmissionController(t, opts)
	resp := ac.admit(TestContextWithLogger(t), req)

	expectFailsWith(t, resp, "deprecated")
	expectFailsWith(t, resp, "Status.DeprecatedField")
}
