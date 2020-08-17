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

package validation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	// Injection stuff
	_ "knative.dev/pkg/client/injection/kube/client/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/admissionregistration/v1/validatingwebhookconfiguration/fake"
	_ "knative.dev/pkg/injection/clients/namespacedkube/informers/core/v1/secret/fake"
	pkgreconciler "knative.dev/pkg/reconciler"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook"

	_ "knative.dev/pkg/system/testing"

	. "knative.dev/pkg/logging/testing"
	. "knative.dev/pkg/reconciler/testing"
	. "knative.dev/pkg/testing"
	"knative.dev/pkg/webhook/resourcesemantics"
	. "knative.dev/pkg/webhook/testing"
)

const (
	testResourceValidationPath = "/foo"
	testResourceValidationName = "webhook.knative.dev"
	user1                      = "brutto@knative.dev"
	user2                      = "arrabbiato@knative.dev"
)

var (
	handlers = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
		{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		}: &Resource{},
		{
			Group:   "pkg.knative.dev",
			Version: "v1beta1",
			Kind:    "Resource",
		}: &Resource{},
		{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "InnerDefaultResource",
		}: &InnerDefaultResource{},
		{
			Group:   "pkg.knative.io",
			Version: "v1alpha1",
			Kind:    "InnerDefaultResource",
		}: &InnerDefaultResource{},
	}

	callbacks = map[schema.GroupVersionKind]Callback{
		{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		}: NewCallback(resourceCallback, webhook.Create, webhook.Update, webhook.Delete),
		{
			Group:   "pkg.knative.dev",
			Version: "v1beta1",
			Kind:    "Resource",
		}: NewCallback(resourceCallback, webhook.Create, webhook.Update, webhook.Delete),
	}
	initialResourceWebhook = &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "webhook.knative.dev",
			OwnerReferences: []metav1.OwnerReference{{
				Name: "asdf",
			}},
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{{
			Name: "webhook.knative.dev",
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				Service: &admissionregistrationv1.ServiceReference{
					Namespace: system.Namespace(),
					Name:      "webhook",
				},
			},
		}},
	}
)

func newNonRunningTestResourceAdmissionController(t *testing.T) (
	kubeClient *fakekubeclientset.Clientset,
	ac *reconciler) {

	t.Helper()
	// Create fake clients
	kubeClient = fakekubeclientset.NewSimpleClientset(initialResourceWebhook)

	ac = NewTestResourceAdmissionController(t)
	return
}

func TestDeleteAllowed(t *testing.T) {
	_, ac := newNonRunningTestResourceAdmissionController(t)

	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Delete,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
	}

	if resp := ac.Admit(TestContextWithLogger(t), req); !resp.Allowed {
		t.Fatal("Unexpected denial of delete")
	}
}

func TestConnectAllowed(t *testing.T) {
	_, ac := newNonRunningTestResourceAdmissionController(t)

	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Connect,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
	}

	resp := ac.Admit(TestContextWithLogger(t), req)
	if !resp.Allowed {
		t.Fatalf("Unexpected denial of connect")
	}
}

func TestUnknownKindFails(t *testing.T) {
	_, ac := newNonRunningTestResourceAdmissionController(t)

	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Garbage",
		},
	}

	ExpectFailsWith(t, ac.Admit(TestContextWithLogger(t), req), "unhandled kind")
}

func TestUnknownVersionFails(t *testing.T) {
	_, ac := newNonRunningTestResourceAdmissionController(t)
	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1beta2",
			Kind:    "Resource",
		},
	}
	ExpectFailsWith(t, ac.Admit(TestContextWithLogger(t), req), "unhandled kind")
}

func TestUnknownFieldFails(t *testing.T) {
	_, ac := newNonRunningTestResourceAdmissionController(t)
	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
	}

	marshaled, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"foo": "bar",
		},
	})
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}
	req.Object.Raw = marshaled

	ExpectFailsWith(t, ac.Admit(TestContextWithLogger(t), req),
		`decoding request failed: cannot decode incoming new object: json: unknown field "foo"`)
}

func TestAdmitCreates(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(context.Context, *Resource)
		rejection string
	}{{
		name: "test simple creation (alpha, no diff)",
		setup: func(ctx context.Context, r *Resource) {
			r.TypeMeta.APIVersion = "v1alpha1"
			r.SetDefaults(ctx)
			r.Annotations = map[string]string{
				"pkg.knative.dev/creator":      user1,
				"pkg.knative.dev/lastModifier": user1,
			}
		},
	}, {
		name: "test simple creation (beta, no diff)",
		setup: func(ctx context.Context, r *Resource) {
			r.TypeMeta.APIVersion = "v1beta1"
			r.SetDefaults(ctx)
			r.Annotations = map[string]string{
				"pkg.knative.dev/creator":      user1,
				"pkg.knative.dev/lastModifier": user1,
			}
		},
	}, {
		name: "with bad field",
		setup: func(ctx context.Context, r *Resource) {
			// Put a bad value in.
			r.Spec.FieldWithValidation = "not what's expected"
		},
		rejection: "invalid value",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := CreateResource("a name")
			ctx := apis.WithinCreate(apis.WithUserInfo(
				TestContextWithLogger(t),
				&authenticationv1.UserInfo{Username: user1}))

			// Setup the resource.
			tc.setup(ctx, r)

			_, ac := newNonRunningTestResourceAdmissionController(t)
			resp := ac.Admit(ctx, createCreateResource(ctx, t, r))

			if tc.rejection == "" {
				ExpectAllowed(t, resp)
			} else {
				ExpectFailsWith(t, resp, tc.rejection)
			}
		})
	}
}

func resourceCallback(ctx context.Context, uns *unstructured.Unstructured) error {
	var resource Resource
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uns.UnstructuredContent(), &resource); err != nil {
		return err
	}

	if apis.IsInDelete(ctx) {
		if resource.Spec.FieldForCallbackValidation != "magic delete" {
			return errors.New("no magic delete")
		}
		return nil
	}

	if apis.IsDryRun(ctx) {
		return errors.New("dryRun fail")
	}

	if resource.Spec.FieldForCallbackValidation != "" &&
		resource.Spec.FieldForCallbackValidation != "magic value" {
		return errors.New(resource.Spec.FieldForCallbackValidation)
	}
	return nil
}

func TestValidationCreateCallback(t *testing.T) {
	tests := []struct {
		name      string
		dryRun    bool
		setup     func(context.Context, *Resource)
		rejection string
	}{{
		name:      "with dryRun reject",
		dryRun:    true,
		setup:     func(ctx context.Context, r *Resource) {},
		rejection: "validation callback failed: dryRun fail",
	}, {
		name:   "with dryRun off",
		dryRun: false,
		setup:  func(ctx context.Context, r *Resource) {},
	}, {
		name:   "with field magic value",
		dryRun: false,
		setup: func(ctx context.Context, r *Resource) {
			// Put a good value in.
			r.Spec.FieldForCallbackValidation = "magic value"
		},
	}, {
		name:   "with field reject value",
		dryRun: false,
		setup: func(ctx context.Context, r *Resource) {
			// Put a bad value in.
			r.Spec.FieldForCallbackValidation = "callbacks hate this"
		},
		rejection: "validation callback failed: callbacks hate this",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := CreateResource("a name")
			ctx := apis.WithinCreate(apis.WithUserInfo(
				TestContextWithLogger(t),
				&authenticationv1.UserInfo{Username: user1}))

			// Setup the resource.
			tc.setup(ctx, r)

			_, ac := newNonRunningTestResourceAdmissionController(t)
			req := createCreateResource(ctx, t, r)
			if tc.dryRun {
				truePoint := true
				req.DryRun = &truePoint
			}

			resp := ac.Admit(ctx, req)

			if tc.rejection == "" {
				ExpectAllowed(t, resp)
			} else {
				ExpectFailsWith(t, resp, tc.rejection)
			}
		})
	}
}

func TestValidationDeleteCallback(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(context.Context, *Resource)
		rejection string
	}{{
		name: "with field magic delete",
		setup: func(ctx context.Context, r *Resource) {
			// Put a good value in.
			r.Spec.FieldForCallbackValidation = "magic delete"
		},
	}, {
		name: "with field reject value",
		setup: func(ctx context.Context, r *Resource) {
			// Put a bad value in.
			r.Spec.FieldForCallbackValidation = "no magic delete"
		},
		rejection: "validation callback failed: no magic delete",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := CreateResource("a name")
			ctx := apis.WithinCreate(apis.WithUserInfo(
				TestContextWithLogger(t),
				&authenticationv1.UserInfo{Username: user1}))

			// Setup the resource.
			tc.setup(ctx, r)

			_, ac := newNonRunningTestResourceAdmissionController(t)
			req := createDeleteResource(ctx, t, r)

			resp := ac.Admit(ctx, req)

			if tc.rejection == "" {
				ExpectAllowed(t, resp)
			} else {
				ExpectFailsWith(t, resp, tc.rejection)
			}
		})
	}
}

func createDeleteResource(ctx context.Context, t *testing.T, old *Resource) *admissionv1.AdmissionRequest {
	t.Helper()
	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Delete,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
		UserInfo: *apis.GetUserInfo(ctx),
	}
	marshaledOld, err := json.Marshal(old)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}
	req.OldObject.Raw = marshaledOld
	req.Resource.Group = "pkg.knative.dev"
	return req
}

func createCreateResource(ctx context.Context, t *testing.T, r *Resource) *admissionv1.AdmissionRequest {
	t.Helper()
	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
		UserInfo: *apis.GetUserInfo(ctx),
	}
	marshaled, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}
	req.Object.Raw = marshaled
	req.Resource.Group = "pkg.knative.dev"
	return req
}

func TestAdmitUpdates(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(context.Context, *Resource)
		mutate    func(context.Context, *Resource)
		rejection string
	}{{
		name: "test simple update (no diff)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			// If we don't change anything, the updater
			// annotation doesn't change.
		},
	}, {
		name: "bad mutation (immutable)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			r.Spec.FieldThatsImmutableWithDefault = "something different"
		},
		rejection: "Immutable field changed",
	}, {
		name: "bad mutation (validation)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			r.Spec.FieldWithValidation = "not what's expected"
		},
		rejection: "invalid value",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			old := CreateResource("a name")
			ctx := TestContextWithLogger(t)

			old.Annotations = map[string]string{
				"pkg.knative.dev/creator":      user1,
				"pkg.knative.dev/lastModifier": user1,
			}

			tc.setup(ctx, old)

			new := old.DeepCopy()

			// Mutate the resource using the update context as user2
			ctx = apis.WithUserInfo(apis.WithinUpdate(ctx, old),
				&authenticationv1.UserInfo{Username: user2})
			tc.mutate(ctx, new)

			_, ac := newNonRunningTestResourceAdmissionController(t)
			resp := ac.Admit(ctx, createUpdateResource(ctx, t, old, new))

			if tc.rejection == "" {
				ExpectAllowed(t, resp)
			} else {
				ExpectFailsWith(t, resp, tc.rejection)
			}
		})
	}
}

func createUpdateResource(ctx context.Context, t *testing.T, old, new *Resource) *admissionv1.AdmissionRequest {
	t.Helper()
	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Update,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
		UserInfo: *apis.GetUserInfo(ctx),
	}
	marshaled, err := json.Marshal(new)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}
	req.Object.Raw = marshaled
	marshaledOld, err := json.Marshal(old)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}
	req.OldObject.Raw = marshaledOld
	req.Resource.Group = "pkg.knative.dev"
	return req
}

func createInnerDefaultResourceWithoutSpec(t *testing.T) []byte {
	t.Helper()
	r := InnerDefaultResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "testKind",
			APIVersion: "testAPIVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: system.Namespace(),
			Name:      "a name",
		},
	}
	// Remove the 'spec' field of the generated JSON by marshaling it to JSON, parsing that as a
	// generic map[string]interface{}, removing 'spec', and marshaling it again.
	origBytes, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Error marshaling origBytes: %v", err)
	}
	var q map[string]interface{}
	if err := json.Unmarshal(origBytes, &q); err != nil {
		t.Fatalf("Error unmarshaling origBytes: %v", err)
	}
	delete(q, "spec")
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("Error marshaling q: %v", err)
	}
	return b
}

func createInnerDefaultResourceWithSpecAndStatus(t *testing.T, spec *InnerDefaultSpec, status *InnerDefaultStatus) []byte {
	t.Helper()
	r := InnerDefaultResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "testKind",
			APIVersion: "testAPIVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: system.Namespace(),
			Name:      "a name",
		},
	}
	if spec != nil {
		r.Spec = *spec
	}
	if status != nil {
		r.Status = *status
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Error marshaling bytes: %v", err)
	}
	return b
}

func TestNewResourceAdmissionController(t *testing.T) {
	ctx, _ := SetupFakeContext(t)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected a second callback to panic")
		}
	}()

	invalidSecondCallback := map[schema.GroupVersionKind]Callback{}

	NewAdmissionController(
		ctx, testResourceValidationName, testResourceValidationPath,
		handlers,
		func(ctx context.Context) context.Context {
			return ctx
		}, true,
		callbacks,
		invalidSecondCallback)
}

func TestNewResourceAdmissionControllerDuplciateVerb(t *testing.T) {
	ctx, _ := SetupFakeContext(t)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected a second callback to panic")
		}
	}()

	call := map[schema.GroupVersionKind]Callback{
		{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		}: NewCallback(resourceCallback, webhook.Create, webhook.Create), // Disallow duplicates under test
	}

	NewAdmissionController(
		ctx, testResourceValidationName, testResourceValidationPath,
		handlers,
		func(ctx context.Context) context.Context {
			return ctx
		}, true,
		call)
}

func NewTestResourceAdmissionController(t *testing.T) *reconciler {
	ctx, _ := SetupFakeContext(t)
	ctx = webhook.WithOptions(ctx, webhook.Options{
		SecretName: "webhook-secret",
	})

	c := NewAdmissionController(
		ctx, testResourceValidationName, testResourceValidationPath,
		handlers,
		func(ctx context.Context) context.Context {
			return ctx
		}, true, callbacks)
	if c == nil {
		t.Fatal("Expected NewController to return a non-nil value")
	}

	if want, got := 0, c.WorkQueue().Len(); want != got {
		t.Errorf("WorkQueue.Len() = %d, wanted %d", got, want)
	}

	la, ok := c.Reconciler.(pkgreconciler.LeaderAware)
	if !ok {
		t.Fatalf("%T is not leader aware", c.Reconciler)
	}

	if err := la.Promote(pkgreconciler.UniversalBucket(), c.MaybeEnqueueBucketKey); err != nil {
		t.Errorf("Promote() = %v", err)
	}

	// Queue has async moving parts so if we check at the wrong moment, this might still be 0.
	if wait.PollImmediate(10*time.Millisecond, 250*time.Millisecond, func() (bool, error) {
		return c.WorkQueue().Len() == 1, nil
	}) != nil {
		t.Error("Queue length was never 1")
	}

	return c.Reconciler.(*reconciler)
}
