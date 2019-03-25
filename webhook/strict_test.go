package webhook

import (
	. "github.com/knative/pkg/logging/testing"
	. "github.com/knative/pkg/testing"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

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
		FieldWithDeprecation: "fail setting.",
	}, nil)

	opts := newDefaultOptions()
	opts.Strict = true

	_, ac := newNonRunningTestAdmissionController(t, opts)
	resp := ac.admit(TestContextWithLogger(t), req)

	expectFailsWith(t, resp, "deprecated")
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
		FieldWithDeprecation: "fail setting.",
	})

	opts := newDefaultOptions()
	opts.Strict = true

	_, ac := newNonRunningTestAdmissionController(t, opts)
	resp := ac.admit(TestContextWithLogger(t), req)

	expectFailsWith(t, resp, "deprecated")
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
	req.Object.Raw = createInnerDefaultResourceWithSpecAndStatus(t, &InnerDefaultSpec{
		FieldWithDeprecation: "fail setting.",
	}, nil)

	opts := newDefaultOptions()
	opts.Strict = true

	_, ac := newNonRunningTestAdmissionController(t, opts)
	resp := ac.admit(TestContextWithLogger(t), req)

	expectFailsWith(t, resp, "deprecated")
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
		FieldWithDeprecation: "fail setting.",
	})

	opts := newDefaultOptions()
	opts.Strict = true

	_, ac := newNonRunningTestAdmissionController(t, opts)
	resp := ac.admit(TestContextWithLogger(t), req)

	expectFailsWith(t, resp, "deprecated")
}
