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
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/configmap"
	. "knative.dev/pkg/logging/testing"
)

func newNonRunningTestConfigValidationController(t *testing.T, options ControllerOptions) (
	kubeClient *fakekubeclientset.Clientset,
	ac AdmissionController) {
	t.Helper()
	// Create fake clients
	kubeClient = fakekubeclientset.NewSimpleClientset()

	ac = NewTestConfigValidationController(options)
	return
}

func NewTestConfigValidationController(options ControllerOptions) AdmissionController {
	validations := configmap.Constructors{"test-config": newConfigFromConfigMap}
	return NewConfigValidationController(validations, options)
}

func TestValidConfigValidationController(t *testing.T) {
	kubeClient, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())
	createDeployment(kubeClient)
	err := ac.Register(TestContextWithLogger(t), kubeClient, []byte{})
	if err != nil {
		t.Fatalf("Failed to create webhook: %s", err)
	}
}

func TestUpdatingConfigValidationController(t *testing.T) {
	kubeClient, c := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	ac := c.(*ConfigValidationController)
	webhook := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.options.ConfigValidationWebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.ValidatingWebhook{{
			Name:         ac.options.ConfigValidationWebhookName,
			Rules:        []admissionregistrationv1beta1.RuleWithOperations{{}},
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{},
		}},
	}

	createDeployment(kubeClient)
	createConfigValidationWebhook(kubeClient, webhook)
	err := ac.Register(TestContextWithLogger(t), kubeClient, []byte{})
	if err != nil {
		t.Fatalf("Failed to create webhook: %s", err)
	}

	currentWebhook, _ := kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(ac.options.ConfigValidationWebhookName, metav1.GetOptions{})
	if reflect.DeepEqual(currentWebhook.Webhooks, webhook.Webhooks) {
		t.Fatalf("Expected webhook to be updated")
	}
}

func TestDeleteAllowedForConfigMap(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Delete,
	}

	if resp := ac.Admit(TestContextWithLogger(t), req); !resp.Allowed {
		t.Fatal("Unexpected denial of delete")
	}
}

func TestConnectAllowedForConfigMap(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Connect,
	}

	resp := ac.Admit(TestContextWithLogger(t), req)
	if !resp.Allowed {
		t.Fatalf("Unexpected denial of connect")
	}
}

func TestNonConfigMapKindFails(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Garbage",
		},
	}

	expectFailsWith(t, ac.Admit(TestContextWithLogger(t), req), "unhandled kind")
}

func TestAdmitCreateValidConfigMap(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	r := createValidConfigMap()
	ctx := apis.WithinCreate(apis.WithUserInfo(
		TestContextWithLogger(t),
		&authenticationv1.UserInfo{Username: user1}))

	resp := ac.Admit(ctx, createCreateConfigMapRequest(ctx, r))

	expectAllowed(t, resp)
}

func TestDenyInvalidCreateConfigMapWithWrongType(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	r := createWrongTypeConfigMap()
	ctx := apis.WithinCreate(apis.WithUserInfo(
		TestContextWithLogger(t),
		&authenticationv1.UserInfo{Username: user1}))

	resp := ac.Admit(ctx, createCreateConfigMapRequest(ctx, r))

	expectFailsWith(t, resp, "invalid syntax")
}

func TestDenyInvalidCreateConfigMapOutOfRange(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	r := createWrongValueConfigMap()
	ctx := apis.WithinCreate(apis.WithUserInfo(
		TestContextWithLogger(t),
		&authenticationv1.UserInfo{Username: user1}))

	resp := ac.Admit(ctx, createCreateConfigMapRequest(ctx, r))

	expectFailsWith(t, resp, "out of range")
}

func TestAdmitUpdateValidConfigMap(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	r := createValidConfigMap()
	ctx := apis.WithinCreate(apis.WithUserInfo(
		TestContextWithLogger(t),
		&authenticationv1.UserInfo{Username: user1}))

	resp := ac.Admit(ctx, updateCreateConfigMapRequest(ctx, r))

	expectAllowed(t, resp)
}

func TestDenyInvalidUpdateConfigMapWithWrongType(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	r := createWrongTypeConfigMap()
	ctx := apis.WithinCreate(apis.WithUserInfo(
		TestContextWithLogger(t),
		&authenticationv1.UserInfo{Username: user1}))

	resp := ac.Admit(ctx, createCreateConfigMapRequest(ctx, r))

	expectFailsWith(t, resp, "invalid syntax")
}

func TestDenyInvalidUpdateConfigMapOutOfRange(t *testing.T) {
	_, ac := newNonRunningTestConfigValidationController(t, newDefaultOptions())

	r := createWrongValueConfigMap()
	ctx := apis.WithinCreate(apis.WithUserInfo(
		TestContextWithLogger(t),
		&authenticationv1.UserInfo{Username: user1}))

	resp := ac.Admit(ctx, createCreateConfigMapRequest(ctx, r))

	expectFailsWith(t, resp, "out of range")
}

func createConfigValidationWebhook(kubeClient kubernetes.Interface, webhook *admissionregistrationv1beta1.ValidatingWebhookConfiguration) {
	client := kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations()
	_, err := client.Create(webhook)
	if err != nil {
		panic(fmt.Sprintf("failed to create test webhook: %s", err))
	}
}

type config struct {
	value float64
}

func newConfigFromConfigMap(configMap *corev1.ConfigMap) (*config, error) {
	data := configMap.Data
	cfg := &config{}
	for _, b := range []struct {
		key   string
		field *float64
	}{{
		key:   "value",
		field: &cfg.value,
	}} {
		if raw, ok := data[b.key]; !ok {
			return nil, fmt.Errorf("not found")
		} else if val, err := strconv.ParseFloat(raw, 64); err != nil {
			return nil, err
		} else {
			*b.field = val
		}
	}

	// some sample validation on the value
	if cfg.value > 2.0 || cfg.value < 0.0 {
		return nil, fmt.Errorf("out of range")
	}

	return cfg, nil
}

func createValidConfigMap() *corev1.ConfigMap {
	return createConfigMap("1.5")
}

func createWrongTypeConfigMap() *corev1.ConfigMap {
	return createConfigMap("bad")
}

func createWrongValueConfigMap() *corev1.ConfigMap {
	return createConfigMap("2.5")
}

func createConfigMap(value string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "test-config",
		},
		Data: map[string]string{
			"value": value,
		},
	}
}

func createCreateConfigMapRequest(ctx context.Context, r *corev1.ConfigMap) *admissionv1beta1.AdmissionRequest {
	return configMapRequest(r, admissionv1beta1.Create, *apis.GetUserInfo(ctx))
}

func updateCreateConfigMapRequest(ctx context.Context, r *corev1.ConfigMap) *admissionv1beta1.AdmissionRequest {
	return configMapRequest(r, admissionv1beta1.Update, *apis.GetUserInfo(ctx))
}

func configMapRequest(
	r *corev1.ConfigMap,
	o admissionv1beta1.Operation,
	u authenticationv1.UserInfo,
) *admissionv1beta1.AdmissionRequest {
	req := &admissionv1beta1.AdmissionRequest{
		Operation: o,
		Kind: metav1.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "ConfigMap",
		},
		UserInfo: u,
	}
	marshaled, err := json.Marshal(r)
	if err != nil {
		panic("failed to marshal resource")
	}
	req.Object.Raw = marshaled
	req.Resource.Group = ""
	return req
}
