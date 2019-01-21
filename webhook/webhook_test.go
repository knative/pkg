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

package webhook

import (
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/knative/pkg/apis/duck"
	"golang.org/x/sync/errgroup"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mattbaird/jsonpatch"
	"go.uber.org/zap"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	// corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	. "github.com/knative/pkg/logging/testing"
	. "github.com/knative/pkg/testing"
)

func newDefaultOptions() ControllerOptions {
	return ControllerOptions{
		Namespace:   "knative-something",
		ServiceName: "webhook",
		Port:        443,
		SecretName:  "webhook-certs",
		WebhookName: "webhook.knative.dev",
	}
}

const (
	testNamespace    = "test-namespace"
	testResourceName = "test-resource"
	user1            = "brutto@knative.dev"
	user2            = "arrabbiato@knative.dev"
)

func newNonRunningTestAdmissionController(t *testing.T, options ControllerOptions) (
	kubeClient *fakekubeclientset.Clientset,
	ac *AdmissionController) {
	t.Helper()
	// Create fake clients
	kubeClient = fakekubeclientset.NewSimpleClientset()

	ac, err := NewAdmissionController(kubeClient, options, TestLogger(t))
	if err != nil {
		t.Fatalf("Failed to create new admission controller: %v", err)
	}
	return
}

func TestDeleteAllowed(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())

	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Delete,
	}

	if resp := ac.admit(TestContextWithLogger(t), req); !resp.Allowed {
		t.Fatal("Unexpected denial of delete")
	}
}

func TestConnectAllowed(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())

	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Connect,
	}

	resp := ac.admit(TestContextWithLogger(t), req)
	if !resp.Allowed {
		t.Fatalf("Unexpected denial of connect")
	}
}

func TestUnknownKindFails(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())

	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Garbage",
		},
	}

	expectFailsWith(t, ac.admit(TestContextWithLogger(t), req), "unhandled kind")
}

func TestUnknownVersionFails(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1beta2",
			Kind:    "Resource",
		},
	}
	expectFailsWith(t, ac.admit(TestContextWithLogger(t), req), "unhandled kind")
}

func TestValidCreateResourceSucceeds(t *testing.T) {
	r := createResource("a name")
	for _, v := range []string{"v1alpha1", "v1beta1"} {
		r.TypeMeta.APIVersion = v
		r.SetDefaults() // Fill in defaults to check that there are no patches.
		_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
		resp := ac.admit(TestContextWithLogger(t), createCreateResource(r))
		expectAllowed(t, resp)
		expectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{
			setUserAnnotation(user1, user1),
		})
	}
}

func TestValidCreateResourceSucceedsWithDefaultPatch(t *testing.T) {
	r := createResource("a name")
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createCreateResource(r))
	expectAllowed(t, resp)
	expectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{
		{
			Operation: "add",
			Path:      "/spec/fieldThatsImmutableWithDefault",
			Value:     "this is another default value",
		}, {
			Operation: "add",
			Path:      "/spec/fieldWithDefault",
			Value:     "I'm a default.",
		},
		setUserAnnotation(user1, user1),
	})
}

func TestValidCreateResourceSucceedsWithRoundTripAndDefaultPatch(t *testing.T) {
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "InnerDefaultResource",
		},
	}
	req.Object.Raw = createInnerDefaultResourceWithoutSpec(t)

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), req)
	expectAllowed(t, resp)
	expectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{
		{
			Operation: "add",
			Path:      "/spec",
			Value:     map[string]interface{}{},
		},
		{
			Operation: "add",
			Path:      "/spec/fieldWithDefault",
			Value:     "I'm a default.",
		},
	})
}

func createInnerDefaultResourceWithoutSpec(t *testing.T) []byte {
	t.Helper()
	r := InnerDefaultResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
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

func TestInvalidCreateResourceFails(t *testing.T) {
	r := createResource("a name")

	// Put a bad value in.
	r.Spec.FieldWithValidation = "not what's expected"

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createCreateResource(r))
	expectFailsWith(t, resp, "invalid value")
}

func TestNopUpdateResourceSucceeds(t *testing.T) {
	r := createResource("a name")
	r.SetDefaults() // Fill in defaults to check that there are no patches.
	nr := r.DeepCopyObject().(*Resource)
	r.AnnotateUserInfo(nil, &authenticationv1.UserInfo{Username: user1})
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createUpdateResource(r, nr))
	expectAllowed(t, resp)
	expectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{})
}

func TestValidUpdateResourcePreserveAnnotations(t *testing.T) {
	old := createResource("a name")
	old.SetDefaults() // Fill in defaults to check that there are no patches.
	old.AnnotateUserInfo(nil, &authenticationv1.UserInfo{Username: user1})
	new := createResource("a name")
	new.SetDefaults()
	// User set annotations on the resource.
	new.ObjectMeta.SetAnnotations(map[string]string{
		"key": "to-my-heart",
	})

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createUpdateResource(old, new))
	expectAllowed(t, resp)
	// No spec change => no user info change.
	expectPatches(t, resp.Patch, duck.JSONPatch{})
}

func TestValidBigChangeResourceSucceeds(t *testing.T) {
	old := createResource("a name")
	old.SetDefaults() // Fill in defaults to check that there are no patches.
	old.AnnotateUserInfo(nil, &authenticationv1.UserInfo{Username: user1})
	new := createResource("a name")
	new.Spec.FieldWithDefault = "melon collie and the infinite sadness"

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createUpdateResource(old, new))
	expectAllowed(t, resp)
	expectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/fieldThatsImmutableWithDefault",
		Value:     "this is another default value",
	}, setUserAnnotation(user1, user2),
	})
}

func TestValidUpdateResourceSucceeds(t *testing.T) {
	old := createResource("a name")
	old.SetDefaults() // Fill in defaults to check that there are no patches.
	old.AnnotateUserInfo(nil, &authenticationv1.UserInfo{Username: user1})
	new := createResource("a name")

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createUpdateResource(old, new))
	expectAllowed(t, resp)
	expectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/fieldThatsImmutableWithDefault",
		Value:     "this is another default value",
	}, {
		Operation: "add",
		Path:      "/spec/fieldWithDefault",
		Value:     "I'm a default.",
	}})
}

func TestInvalidUpdateResourceFailsValidation(t *testing.T) {
	old := createResource("a name")
	new := createResource("a name")

	// Try to update to a bad value.
	new.Spec.FieldWithValidation = "not what's expected"

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createUpdateResource(old, new))
	expectFailsWith(t, resp, "invalid value")
}

func TestInvalidUpdateResourceFailsImmutability(t *testing.T) {
	old := createResource("a name")
	new := createResource("a name")

	// Try to change the value
	new.Spec.FieldThatsImmutable = "a different value"
	new.Spec.FieldThatsImmutableWithDefault = "another different value"

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createUpdateResource(old, new))
	expectFailsWith(t, resp, "Immutable field changed")
}

func TestDefaultingImmutableFields(t *testing.T) {
	old := createResource("a name")
	old.AnnotateUserInfo(nil, &authenticationv1.UserInfo{Username: user1})
	new := createResource("a name")

	// If we don't specify the new, but immutable field, we default it,
	// and it is not rejected.

	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.admit(TestContextWithLogger(t), createUpdateResource(old, new))
	expectAllowed(t, resp)
	expectPatches(t, resp.Patch,
		[]jsonpatch.JsonPatchOperation{{
			Operation: "add",
			Path:      "/spec/fieldThatsImmutableWithDefault",
			Value:     "this is another default value",
		}, {
			Operation: "add",
			Path:      "/spec/fieldWithDefault",
			Value:     "I'm a default.",
		},
			// From WebHook perspective an update is happening here,
			// so the annotations must be set.
			setUserAnnotation(user1, user2),
		})
}

func TestValidWebhook(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	createDeployment(ac)
	ac.register(TestContextWithLogger(t), ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations(), []byte{})
	_, err := ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(ac.Options.WebhookName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to create webhook: %s", err)
	}
}

func TestUpdatingWebhook(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	webhook := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.Options.WebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{{
			Name:         ac.Options.WebhookName,
			Rules:        []admissionregistrationv1beta1.RuleWithOperations{{}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{},
		}},
	}

	createDeployment(ac)
	createWebhook(ac, webhook)
	ac.register(TestContextWithLogger(t), ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations(), []byte{})
	currentWebhook, _ := ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(ac.Options.WebhookName, metav1.GetOptions{})
	if reflect.DeepEqual(currentWebhook.Webhooks, webhook.Webhooks) {
		t.Fatalf("Expected webhook to be updated")
	}
}

func TestRegistrationStopChanFire(t *testing.T) {
	opts := newDefaultOptions()
	_, ac := newNonRunningTestAdmissionController(t, opts)
	webhook := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.Options.WebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name:         ac.Options.WebhookName,
				Rules:        []admissionregistrationv1beta1.RuleWithOperations{{}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{},
			},
		},
	}
	createWebhook(ac, webhook)

	ac.Options.RegistrationDelay = 1 * time.Minute
	stopCh := make(chan struct{})

	var g errgroup.Group
	g.Go(func() error {
		return ac.Run(stopCh)
	})
	close(stopCh)

	if err := g.Wait(); err != nil {
		t.Fatal("Error during run: ", err)
	}
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", opts.Port))
	if err == nil {
		conn.Close()
		t.Errorf("Unexpected success to dial to port %d", opts.Port)
	}
}

func TestRegistrationForAlreadyExistingWebhook(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	webhook := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.Options.WebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name:         ac.Options.WebhookName,
				Rules:        []admissionregistrationv1beta1.RuleWithOperations{{}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{},
			},
		},
	}
	createWebhook(ac, webhook)

	ac.Options.RegistrationDelay = 1 * time.Millisecond
	stopCh := make(chan struct{})

	var g errgroup.Group
	g.Go(func() error {
		return ac.Run(stopCh)
	})
	err := g.Wait()
	if err == nil {
		t.Fatal("Expected webhook controller to fail")
	}

	if ac.Options.ClientAuth >= tls.VerifyClientCertIfGiven && !strings.Contains(err.Error(), "configmaps") {
		t.Fatal("Expected error msg to contain configmap key missing error")
	}
}

func TestCertConfigurationForAlreadyGeneratedSecret(t *testing.T) {
	secretName := "test-secret"
	ns := "test-namespace"
	opts := newDefaultOptions()
	opts.SecretName = secretName
	opts.Namespace = ns
	kubeClient, ac := newNonRunningTestAdmissionController(t, opts)

	ctx := TestContextWithLogger(t)
	newSecret, err := generateSecret(ctx, &opts)
	if err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}
	_, err = kubeClient.CoreV1().Secrets(ns).Create(newSecret)
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	createNamespace(t, ac.Client, metav1.NamespaceSystem)
	createTestConfigMap(t, ac.Client)

	tlsConfig, caCert, err := configureCerts(ctx, kubeClient, &ac.Options)
	if err != nil {
		t.Fatalf("Failed to configure secret: %v", err)
	}
	expectedCert, err := tls.X509KeyPair(newSecret.Data[secretServerCert], newSecret.Data[secretServerKey])
	if err != nil {
		t.Fatalf("Failed to create cert from x509 key pair: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("Expected TLS config not to be nil")
	}
	if len(tlsConfig.Certificates) < 1 {
		t.Fatalf("Expected TLS Config Cert to be set")
	}

	if diff := cmp.Diff(expectedCert.Certificate, tlsConfig.Certificates[0].Certificate, cmp.AllowUnexported()); diff != "" {
		t.Fatalf("Unexpected cert diff (-want, +got) %v", diff)
	}
	if diff := cmp.Diff(newSecret.Data[secretCACert], caCert, cmp.AllowUnexported()); diff != "" {
		t.Fatalf("Unexpected CA cert diff (-want, +got) %v", diff)
	}
}

func TestCertConfigurationForGeneratedSecret(t *testing.T) {
	secretName := "test-secret"
	ns := "test-namespace"
	opts := newDefaultOptions()
	opts.SecretName = secretName
	opts.Namespace = ns
	kubeClient, ac := newNonRunningTestAdmissionController(t, opts)

	ctx := TestContextWithLogger(t)
	createNamespace(t, ac.Client, metav1.NamespaceSystem)
	createTestConfigMap(t, ac.Client)

	tlsConfig, caCert, err := configureCerts(ctx, kubeClient, &ac.Options)
	if err != nil {
		t.Fatalf("Failed to configure certificates: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("Expected TLS config not to be nil")
	}
	if len(tlsConfig.Certificates) < 1 {
		t.Fatalf("Expected TLS Certfificate to be set on webhook server")
	}

	p, _ := pem.Decode(caCert)
	if p == nil {
		t.Fatalf("Expected PEM encoded CA cert ")
	}
	if p.Type != "CERTIFICATE" {
		t.Fatalf("Expectet type to be CERTIFICATE but got %s", string(p.Type))
	}
}

func TestSettingWebhookClientAuth(t *testing.T) {
	opts := newDefaultOptions()
	if opts.ClientAuth != tls.NoClientCert {
		t.Fatalf("Expected default ClientAuth to be NoClientCert (%v) but got (%v)",
			tls.NoClientCert, opts.ClientAuth)
	}
}

func createDeployment(ac *AdmissionController) {
	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "whatever",
			Namespace: "knative-something",
		},
	}
	ac.Client.ExtensionsV1beta1().Deployments("knative-something").Create(deployment)
}

func createResource(name string) *Resource {
	return &Resource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      name,
		},
		Spec: ResourceSpec{
			FieldWithValidation: "magic value",
		},
	}
}

func createBaseUpdateResource() *admissionv1beta1.AdmissionRequest {
	return &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Update,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
		UserInfo: authenticationv1.UserInfo{
			Username: user2,
		}}
}

func createUpdateResource(old, new *Resource) *admissionv1beta1.AdmissionRequest {
	req := createBaseUpdateResource()
	marshaled, err := json.Marshal(new)
	if err != nil {
		panic("failed to marshal resource")
	}
	req.Object.Raw = marshaled
	marshaledOld, err := json.Marshal(old)
	if err != nil {
		panic("failed to marshal resource")
	}
	req.OldObject.Raw = marshaledOld
	return req
}

func createCreateResource(r *Resource) *admissionv1beta1.AdmissionRequest {
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
		UserInfo: authenticationv1.UserInfo{
			Username: user1,
		},
	}
	marshaled, err := json.Marshal(r)
	if err != nil {
		panic("failed to marshal resource")
	}
	req.Object.Raw = marshaled
	return req
}

func createWebhook(ac *AdmissionController, webhook *admissionregistrationv1beta1.MutatingWebhookConfiguration) {
	client := ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations()
	_, err := client.Create(webhook)
	if err != nil {
		panic(fmt.Sprintf("failed to create test webhook: %s", err))
	}
}

func expectAllowed(t *testing.T, resp *admissionv1beta1.AdmissionResponse) {
	t.Helper()
	if !resp.Allowed {
		t.Errorf("Expected allowed, but failed with %+v", resp.Result)
	}
}

func expectFailsWith(t *testing.T, resp *admissionv1beta1.AdmissionResponse, contains string) {
	t.Helper()
	if resp.Allowed {
		t.Error("Expected denial, got allowed")
		return
	}
	if !strings.Contains(resp.Result.Message, contains) {
		t.Errorf("Expected failure containing %q got %q", contains, resp.Result.Message)
	}
}

func expectPatches(t *testing.T, a []byte, e []jsonpatch.JsonPatchOperation) {
	t.Helper()
	var got []jsonpatch.JsonPatchOperation

	err := json.Unmarshal(a, &got)
	if err != nil {
		t.Errorf("Failed to unmarshal patches: %s", err)
		return
	}

	// Give the patch a deterministic ordering.
	// Technically this can change the meaning, but the ordering is otherwise unstable
	// and difficult to test.
	sort.Slice(e, func(i, j int) bool {
		lhs, rhs := e[i], e[j]
		if lhs.Operation != rhs.Operation {
			return lhs.Operation < rhs.Operation
		}
		return lhs.Path < rhs.Path
	})
	sort.Slice(got, func(i, j int) bool {
		lhs, rhs := got[i], got[j]
		if lhs.Operation != rhs.Operation {
			return lhs.Operation < rhs.Operation
		}
		return lhs.Path < rhs.Path
	})

	// Even though diff is useful, seeing the whole objects
	// one under another helps a lot.
	t.Logf("Got Patches:  %#v", got)
	t.Logf("Want Patches: %#v", e)
	if diff := cmp.Diff(e, got, cmpopts.EquateEmpty()); diff != "" {
		t.Logf("diff Patches: %v", diff)
		t.Errorf("expectPatches (-want, +got) = %s", diff)
	}
}

func updateAnnotationsWithUser(userC, userU string) []jsonpatch.JsonPatchOperation {
	// Just keys is being updated, so format is iffy.
	return []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/metadata/annotations/testing.knative.dev~1creator",
		Value:     "brutto@knative.dev",
	}, {
		Operation: "add",
		Path:      "/metadata/annotations/testing.knative.dev~1updater",
		Value:     "arrabbiato@knative.dev",
	}}
}
func setUserAnnotation(userC, userU string) jsonpatch.JsonPatchOperation {
	return jsonpatch.JsonPatchOperation{
		Operation: "add",
		Path:      "/metadata/annotations",
		Value: map[string]interface{}{
			CreatorAnnotation: userC,
			UpdaterAnnotation: userU,
		},
	}
}

func NewAdmissionController(client kubernetes.Interface, options ControllerOptions,
	logger *zap.SugaredLogger) (*AdmissionController, error) {
	return &AdmissionController{
		Client:  client,
		Options: options,
		// Use different versions and domains, for coverage.
		Handlers: map[schema.GroupVersionKind]GenericCRD{
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
		},
		Logger: logger,
	}, nil
}
