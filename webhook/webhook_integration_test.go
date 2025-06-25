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

package webhook

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "knative.dev/pkg/client/injection/kube/client/fake"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"

	"knative.dev/pkg/system"
	pkgtest "knative.dev/pkg/testing"
	certresources "knative.dev/pkg/webhook/certificates/resources"

	_ "knative.dev/pkg/injection/clients/namespacedkube/informers/core/v1/secret/fake"
)

// createResource creates a testing.Resource with the given name in the system namespace.
func createResource(name string) *pkgtest.Resource {
	return &pkgtest.Resource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: system.Namespace(),
			Name:      name,
		},
		Spec: pkgtest.ResourceSpec{
			FieldWithValidation: "magic value",
		},
	}
}

const testTimeout = 10 * time.Second

func TestMissingContentType(t *testing.T) {
	test := testSetup(t)

	eg, _ := errgroup.WithContext(test.ctx)
	eg.Go(func() error { return test.webhook.Run(test.ctx.Done()) })
	test.webhook.InformersHaveSynced()
	defer func() {
		test.cancel()
		if err := eg.Wait(); err != nil {
			t.Error("Unable to run controller:", err)
		}
	}()

	if err := waitForServerAvailable(t, test.addr, testTimeout); err != nil {
		t.Fatal("waitForServerAvailable() =", err)
	}

	tlsClient, err := createSecureTLSClient(t, kubeclient.Get(test.ctx), &test.webhook.Options)
	if err != nil {
		t.Fatal("createSecureTLSClient() =", err)
	}

	req, err := http.NewRequest(http.MethodGet, "https://"+test.addr, nil)
	if err != nil {
		t.Fatal("http.NewRequest() =", err)
	}

	response, err := tlsClient.Do(req)
	if err != nil {
		t.Fatalf("Received %v error from server %s", err, test.addr)
	}

	if got, want := response.StatusCode, http.StatusUnsupportedMediaType; got != want {
		t.Errorf("Response status code = %v, wanted %v", got, want)
	}

	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal("Failed to read response body", err)
	}

	if !strings.Contains(string(responseBody), "invalid Content-Type") {
		t.Errorf("Response body to contain 'invalid Content-Type' , got = '%s'", string(responseBody))
	}
}

func TestServerWithCustomSecret(t *testing.T) {
	test := testSetup(t, withServerCertificateName("tls.crt"), withServerPrivateKeyName("tls.key"))

	eg, _ := errgroup.WithContext(test.ctx)
	eg.Go(func() error { return test.webhook.Run(test.ctx.Done()) })
	test.webhook.InformersHaveSynced()
	defer func() {
		test.cancel()
		if err := eg.Wait(); err != nil {
			t.Error("Unable to run controller:", err)
		}
	}()

	pollErr := waitForServerAvailable(t, test.addr, testTimeout)
	if pollErr != nil {
		t.Fatal("waitForServerAvailable() =", pollErr)
	}
}

func testEmptyRequestBody(t *testing.T, controller any) {
	test := testSetup(t, withController(controller))

	eg, _ := errgroup.WithContext(test.ctx)
	eg.Go(func() error { return test.webhook.Run(test.ctx.Done()) })
	test.webhook.InformersHaveSynced()
	defer func() {
		test.cancel()
		if err := eg.Wait(); err != nil {
			t.Error("Unable to run controller:", err)
		}
	}()

	if err := waitForServerAvailable(t, test.addr, testTimeout); err != nil {
		t.Fatal("waitForServerAvailable() =", err)
	}

	tlsClient, err := createSecureTLSClient(t, kubeclient.Get(test.ctx), &test.webhook.Options)
	if err != nil {
		t.Fatal("createSecureTLSClient() =", err)
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/bazinga", test.addr), nil)
	if err != nil {
		t.Fatal("http.NewRequest() =", err)
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := tlsClient.Do(req)
	if err != nil {
		t.Fatal("failed to get resp", err)
	}

	if got, want := response.StatusCode, http.StatusBadRequest; got != want {
		t.Errorf("Response status code = %v, wanted %v", got, want)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal("Failed to read response body", err)
	}

	if !strings.Contains(string(responseBody), "could not decode body") {
		t.Errorf("Response body to contain 'decode failure information' , got = %q", string(responseBody))
	}
}

func TestSetupWebhookHTTPServerError(t *testing.T) {
	defaultOpts := newDefaultOptions()
	defaultOpts.Port = -1 // invalid port
	ctx, wh, cancel := newNonRunningTestWebhook(t, defaultOpts)
	defer cancel()
	kubeClient := kubeclient.Get(ctx)

	nsErr := createNamespace(t, kubeClient, metav1.NamespaceSystem)
	if nsErr != nil {
		t.Fatal("createNamespace() =", nsErr)
	}
	cMapsErr := createTestConfigMap(t, kubeClient)
	if cMapsErr != nil {
		t.Fatal("createTestConfigMap() =", cMapsErr)
	}

	stopCh := make(chan struct{})
	errCh := make(chan error)
	go func() {
		if err := wh.Run(stopCh); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-time.After(6 * time.Second):
		t.Error("Timeout in testing bootstrap webhook http server failed")
	case errItem := <-errCh:
		if !strings.Contains(errItem.Error(), "bootstrap failed") {
			t.Error("Expected bootstrap webhook http server failed")
		}
	}
}

func testSetup(t *testing.T, opts ...func(*testOptions)) testContext {
	t.Helper()

	// ephemeral port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal("unable to get ephemeral port: ", err)
	}

	testOpts := &testOptions{
		Options: newDefaultOptions(),
	}

	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	resetPackageMetrics()

	for _, opt := range opts {
		opt(testOpts)
	}

	ctx, wh, cancel := newNonRunningTestWebhook(t, testOpts.Options, testOpts.controllers...)
	wh.testListener = l

	// Create certificate
	secret, err := certresources.MakeSecret(ctx, testOpts.SecretName, system.Namespace(), testOpts.ServiceName)
	if err != nil {
		t.Fatalf("failed to create certificate")
	}

	if testOpts.ServerCertificateName != "" {
		secret.Data[testOpts.ServerCertificateName] = secret.Data[certresources.ServerCert]
		delete(secret.Data, certresources.ServerCert)
	}

	if testOpts.ServerPrivateKeyName != "" {
		secret.Data[testOpts.ServerPrivateKeyName] = secret.Data[certresources.ServerKey]
		delete(secret.Data, certresources.ServerKey)
	}

	kubeClient := kubeclient.Get(ctx)

	if _, err := kubeClient.CoreV1().Secrets(secret.Namespace).Create(context.Background(), secret, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create secret")
	}

	return testContext{
		webhook:      wh,
		addr:         l.Addr().String(),
		ctx:          ctx,
		cancel:       cancel,
		metricReader: reader,
	}
}

type testContext struct {
	webhook      *Webhook
	addr         string
	ctx          context.Context
	cancel       context.CancelFunc
	metricReader *metric.ManualReader
}

type testOptions struct {
	Options
	controllers []any
}

func withController(controller any) func(o *testOptions) {
	return func(o *testOptions) {
		o.controllers = append(o.controllers, controller)
	}
}

func withServerCertificateName(name string) func(o *testOptions) {
	return func(o *testOptions) {
		o.ServerCertificateName = name
	}
}

func withServerPrivateKeyName(name string) func(o *testOptions) {
	return func(o *testOptions) {
		o.ServerPrivateKeyName = name
	}
}

func withNoTLS() func(o *testOptions) {
	return func(o *testOptions) {
		o.SecretName = ""
	}
}
