/*
Copyright 2017 The Knative Authors

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

package defaulting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	// Injection stuff
	_ "knative.dev/pkg/client/injection/kube/client/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/admissionregistration/v1/mutatingwebhookconfiguration/fake"
	_ "knative.dev/pkg/injection/clients/namespacedkube/informers/core/v1/secret/fake"

	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		{
			Group:   "pkg.knative.dev",
			Version: "v1beta1",
			Kind:    "ResourceCallbackDefault",
		}: NewCallback(resourceCallback, webhook.Create, webhook.Update, webhook.Delete),
		{
			Group:   "pkg.knative.dev",
			Version: "v1beta1",
			Kind:    "ResourceCallbackDefaultCreate",
		}: NewCallback(resourceCallback, webhook.Create),
	}

	initialResourceWebhook = &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "webhook.knative.dev",
			OwnerReferences: []metav1.OwnerReference{{
				Name: "asdf",
			}},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{{
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
	ac webhook.AdmissionController) {

	t.Helper()
	// Create fake clients
	kubeClient = fakekubeclientset.NewSimpleClientset(initialResourceWebhook)

	ac = newTestResourceAdmissionController(t)
	return
}

func TestDeleteAllowed(t *testing.T) {
	_, ac := newNonRunningTestResourceAdmissionController(t)

	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Delete,
	}

	if resp := ac.Admit(TestContextWithLogger(t), req); !resp.Allowed {
		t.Fatal("Unexpected denial of delete")
	}
}

func TestConnectAllowed(t *testing.T) {
	_, ac := newNonRunningTestResourceAdmissionController(t)

	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Connect,
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
		t.Fatal("Failed to marshal resource:", err)
	}
	req.Object.Raw = marshaled

	ExpectFailsWith(t, ac.Admit(TestContextWithLogger(t), req),
		`mutation failed: cannot decode incoming new object: json: unknown field "foo"`)
}

func TestUnknownMetadataFieldSucceeds(t *testing.T) {
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
		"apiVersion": "pkg.knative.dev/v1alpha1",
		"kind":       "Resource",
		"metadata": map[string]string{
			"unknown": "property",
		},
		"spec": map[string]string{
			"fieldWithValidation": "magic value",
		},
	})
	if err != nil {
		t.Fatal("Failed to marshal resource:", err)
	}
	req.Object.Raw = marshaled

	ExpectAllowed(t, ac.Admit(TestContextWithLogger(t), req))
}

func TestAdmitCreates(t *testing.T) {
	tests := []struct {
		name              string
		setup             func(context.Context, *Resource)
		rejection         string
		patches           []jsonpatch.JsonPatchOperation
		createRequestFunc func(ctx context.Context, t *testing.T, r *Resource) *admissionv1.AdmissionRequest
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
		patches: []jsonpatch.JsonPatchOperation{},
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
		patches: []jsonpatch.JsonPatchOperation{},
	}, {
		name: "test simple creation (with defaults)",
		setup: func(ctx context.Context, r *Resource) {
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "add",
			Path:      "/metadata/annotations",
			Value: map[string]interface{}{
				"pkg.knative.dev/creator":      user1,
				"pkg.knative.dev/lastModifier": user1,
			},
		}, {
			Operation: "add",
			Path:      "/spec/fieldThatsImmutableWithDefault",
			Value:     "this is another default value",
		}, {
			Operation: "add",
			Path:      "/spec/fieldWithDefault",
			Value:     "I'm a default.",
		}},
	}, {
		name: "test simple creation (with defaults around annotations)",
		setup: func(ctx context.Context, r *Resource) {
			r.Annotations = map[string]string{
				"foo": "bar",
			}
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "add",
			Path:      "/metadata/annotations/pkg.knative.dev~1creator",
			Value:     user1,
		}, {
			Operation: "add",
			Path:      "/metadata/annotations/pkg.knative.dev~1lastModifier",
			Value:     user1,
		}, {
			Operation: "add",
			Path:      "/spec/fieldThatsImmutableWithDefault",
			Value:     "this is another default value",
		}, {
			Operation: "add",
			Path:      "/spec/fieldWithDefault",
			Value:     "I'm a default.",
		}},
	}, {
		name: "test simple creation (with partially overridden defaults)",
		setup: func(ctx context.Context, r *Resource) {
			r.Spec.FieldThatsImmutableWithDefault = "not the default"
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "add",
			Path:      "/metadata/annotations",
			Value: map[string]interface{}{
				"pkg.knative.dev/creator":      user1,
				"pkg.knative.dev/lastModifier": user1,
			},
		}, {
			Operation: "add",
			Path:      "/spec/fieldWithDefault",
			Value:     "I'm a default.",
		}},
	}, {
		name: "test simple creation (webhook corrects user annotation)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
			// THIS IS NOT WHO IS CREATING IT, IT IS LIES!
			r.Annotations = map[string]string{
				"pkg.knative.dev/lastModifier": user2,
			}
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "replace",
			Path:      "/metadata/annotations/pkg.knative.dev~1lastModifier",
			Value:     user1,
		}, {
			Operation: "add",
			Path:      "/metadata/annotations/pkg.knative.dev~1creator",
			Value:     user1,
		}},
	}, {
		name: "test simple creation (callback return error)",
		setup: func(ctx context.Context, resource *Resource) {
			resource.Spec.FieldForCallbackDefaulting = "no magic value"
		},
		rejection: "no magic value",
	}, {
		name: "test simple creation (resource and callback defaults)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
			r.Spec.FieldForCallbackDefaulting = "magic value"
			// THIS IS NOT WHO IS CREATING IT, IT IS LIES!
			r.Annotations = map[string]string{
				"pkg.knative.dev/lastModifier": user2,
			}
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "replace",
			Path:      "/spec/fieldDefaultingCallback",
			Value:     "I'm a default",
		}, {
			Operation: "replace",
			Path:      "/metadata/annotations/pkg.knative.dev~1lastModifier",
			Value:     user1,
		}, {
			Operation: "add",
			Path:      "/metadata/annotations/pkg.knative.dev~1creator",
			Value:     user1,
		}},
	}, {
		name: "test simple creation (only callback defaults)",
		setup: func(ctx context.Context, r *Resource) {
			r.TypeMeta.APIVersion = "pkg.knative.dev/v1beta1"
			r.TypeMeta.Kind = "ResourceCallbackDefault"
			r.Spec.FieldForCallbackDefaulting = "magic value"
			// THIS IS NOT WHO IS CREATING IT, IT LIES!
			r.Annotations = map[string]string{
				"pkg.knative.dev/lastModifier": user2,
			}
		},
		createRequestFunc: func(ctx context.Context, t *testing.T, r *Resource) *admissionv1.AdmissionRequest {
			req := createCreateResource(ctx, t, r)
			req.Kind = r.GetGroupVersionKindMeta()
			return req
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "replace",
			Path:      "/spec/fieldDefaultingCallback",
			Value:     "I'm a default",
		}},
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
			var req *admissionv1.AdmissionRequest
			if tc.createRequestFunc == nil {
				req = createCreateResource(ctx, t, r)
			} else {
				req = tc.createRequestFunc(ctx, t, r)
			}
			resp := ac.Admit(ctx, req)

			if tc.rejection == "" {
				ExpectAllowed(t, resp)
				ExpectPatches(t, resp.Patch, tc.patches)
			} else {
				ExpectFailsWith(t, resp, tc.rejection)
			}
		})
	}
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
		t.Fatal("Failed to marshal resource:", err)
	}
	req.Object.Raw = marshaled
	req.Resource.Group = "pkg.knative.dev"
	return req
}

func TestAdmitUpdates(t *testing.T) {
	tests := []struct {
		name                     string
		setup                    func(context.Context, *Resource)
		mutate                   func(context.Context, *Resource)
		createResourceFunc       func(name string) *Resource
		createUpdateResourceFunc func(ctx context.Context, t *testing.T, old, new *Resource) *admissionv1.AdmissionRequest
		rejection                string
		patches                  []jsonpatch.JsonPatchOperation
	}{{
		name: "test simple update (no diff)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			// If we don't change anything, the updater
			// annotation doesn't change.
		},
		patches: []jsonpatch.JsonPatchOperation{},
	}, {
		name: "test simple update (callback defaults error)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			r.Spec.FieldForCallbackDefaulting = "no magic value"
		},
		rejection: "no magic value",
	}, {
		name: "test simple update (callback defaults)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			r.Spec.FieldForCallbackDefaulting = "magic value"
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "replace",
			Path:      "/spec/fieldDefaultingCallback",
			Value:     "I'm a default",
		}, {

			Operation: "replace",
			Path:      "/metadata/annotations/pkg.knative.dev~1lastModifier",
			Value:     user2,
		}},
	}, {
		name: "test simple update (callback defaults only)",
		setup: func(ctx context.Context, r *Resource) {
			r.TypeMeta.APIVersion = "pkg.knative.dev/v1beta1"
			r.TypeMeta.Kind = "ResourceCallbackDefault"
			r.Spec.FieldForCallbackDefaulting = "magic value"
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			r.Spec.FieldForCallbackDefaulting = "magic value"
		},
		createUpdateResourceFunc: func(ctx context.Context, t *testing.T, old, new *Resource) *admissionv1.AdmissionRequest {
			req := createUpdateResource(ctx, t, old, new)
			req.Kind = new.GetGroupVersionKindMeta()
			return req
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "replace",
			Path:      "/spec/fieldDefaultingCallback",
			Value:     "I'm a default",
		}},
	}, {
		name: "test simple update (callback defaults only, operation not supported)",
		setup: func(ctx context.Context, r *Resource) {
			r.TypeMeta.APIVersion = "pkg.knative.dev/v1beta1"
			r.TypeMeta.Kind = "ResourceCallbackDefaultCreate"
			r.Spec.FieldForCallbackDefaulting = "magic value"
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			r.Spec.FieldForCallbackDefaulting = "magic value"
		},
		createUpdateResourceFunc: func(ctx context.Context, t *testing.T, old, new *Resource) *admissionv1.AdmissionRequest {
			req := createUpdateResource(ctx, t, old, new)
			req.Operation = admissionv1.Update
			req.Kind = new.GetGroupVersionKindMeta()
			return req
		},
	}, {
		name: "test simple update (update updater annotation)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			// When we change the spec, the updater
			// annotation changes.
			r.Spec.FieldWithDefault = "not the default"
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "replace",
			Path:      "/metadata/annotations/pkg.knative.dev~1lastModifier",
			Value:     user2,
		}},
	}, {
		name: "test simple update (annotation change doesn't change updater)",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
		},
		mutate: func(ctx context.Context, r *Resource) {
			// When we change an annotation, the updater doesn't change.
			r.Annotations["foo"] = "bar"
		},
		patches: []jsonpatch.JsonPatchOperation{},
	}, {
		name: "test that updates dropping immutable defaults are filled back in",
		setup: func(ctx context.Context, r *Resource) {
			r.SetDefaults(ctx)
			r.Spec.FieldThatsImmutableWithDefault = ""
		},
		mutate: func(ctx context.Context, r *Resource) {
			r.Spec.FieldThatsImmutableWithDefault = ""
		},
		patches: []jsonpatch.JsonPatchOperation{{
			Operation: "add",
			Path:      "/spec/fieldThatsImmutableWithDefault",
			Value:     "this is another default value",
		}},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := "a name"

			var old *Resource
			if tc.createResourceFunc == nil {
				old = CreateResource(name)
			} else {
				old = tc.createResourceFunc(name)
			}
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

			var req *admissionv1.AdmissionRequest
			if tc.createUpdateResourceFunc == nil {
				req = createUpdateResource(ctx, t, old, new)
			} else {
				req = tc.createUpdateResourceFunc(ctx, t, old, new)
			}

			resp := ac.Admit(ctx, req)

			if tc.rejection == "" {
				ExpectAllowed(t, resp)
				ExpectPatches(t, resp.Patch, tc.patches)
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
		t.Error("Failed to marshal resource:", err)
	}
	req.Object.Raw = marshaled
	marshaledOld, err := json.Marshal(old)
	if err != nil {
		t.Error("Failed to marshal resource:", err)
	}
	req.OldObject.Raw = marshaledOld
	req.Resource.Group = "pkg.knative.dev"
	return req
}

func TestValidCreateResourceSucceedsWithRoundTripAndDefaultPatch(t *testing.T) {
	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "InnerDefaultResource",
		},
	}
	req.Object.Raw = createInnerDefaultResourceWithoutSpec(t)

	_, ac := newNonRunningTestResourceAdmissionController(t)
	resp := ac.Admit(TestContextWithLogger(t), req)
	ExpectAllowed(t, resp)
	ExpectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec",
		Value:     map[string]interface{}{},
	}, {
		Operation: "add",
		Path:      "/spec/fieldWithDefault",
		Value:     "I'm a default.",
	}})
}

func createInnerDefaultResourceWithoutSpec(t *testing.T) []byte {
	t.Helper()
	r := InnerDefaultResource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "InnerDefaultResource",
			APIVersion: fmt.Sprintf("%s/%s", SchemeGroupVersion.Group, SchemeGroupVersion.Version),
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
		t.Fatal("Error marshaling origBytes:", err)
	}
	var q map[string]interface{}
	if err := json.Unmarshal(origBytes, &q); err != nil {
		t.Fatal("Error unmarshaling origBytes:", err)
	}
	delete(q, "spec")
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatal("Error marshaling q:", err)
	}
	return b
}

func newTestResourceAdmissionController(t *testing.T) webhook.AdmissionController {
	ctx, _ := SetupFakeContext(t)
	ctx = webhook.WithOptions(ctx, webhook.Options{
		SecretName: "webhook-secret",
	})
	return NewAdmissionController(
		ctx, testResourceValidationName, testResourceValidationPath,
		handlers, func(ctx context.Context) context.Context {
			return ctx
		}, true, callbacks).Reconciler.(*reconciler)
}

func resourceCallback(_ context.Context, uns *unstructured.Unstructured) error {
	var resource Resource
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uns.UnstructuredContent(), &resource); err != nil {
		return err
	}

	if resource.Spec.FieldForCallbackDefaulting != "" {
		if resource.Spec.FieldForCallbackDefaulting != "magic value" {
			return errors.New(resource.Spec.FieldForCallbackDefaulting)
		}
		resource.Spec.FieldForCallbackDefaulting = "I'm a default"
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&resource)
	if err != nil {
		return err
	}
	uns.Object = u

	return nil
}
