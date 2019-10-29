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
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekc "knative.dev/pkg/client/injection/kube/client/fake"
	kubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	certresources "knative.dev/pkg/webhook/certificates/resources"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	. "knative.dev/pkg/reconciler/testing"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

func newDefaultOptions() Options {
	return Options{
		ServiceName: "webhook",
		Port:        443,
		SecretName:  "webhook-certs",
	}
}

const (
	testNamespace    = "test-namespace"
	testResourceName = "test-resource"
	user1            = "brutto@knative.dev"
	user2            = "arrabbiato@knative.dev"
)

func newNonRunningTestWebhook(t *testing.T, options Options) (
	ctx context.Context, ac *Webhook, cancel context.CancelFunc) {
	t.Helper()

	// Create fake clients
	ctx, cancel, informers := SetupFakeContextWithCancel(t)
	ctx = WithOptions(ctx, options)

	if err := controller.StartInformers(ctx.Done(), informers...); err != nil {
		t.Fatalf("StartInformers() = %v", err)
	}
	kc := kubeclient.Get(ctx)

	_, err := kc.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(
		initialResourceWebhook)
	if err != nil {
		t.Errorf("Unable to create %q: %v", initialResourceWebhook.Name, err)
	}
	_, err = kc.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(
		initialConfigWebhook)
	if err != nil {
		t.Errorf("Unable to create %q: %v", initialConfigWebhook.Name, err)
	}

	ac, err = NewTestWebhook(ctx)
	if err != nil {
		t.Fatalf("Failed to create new admission controller: %v", err)
	}
	return
}

func TestRegistrationStopChanFire(t *testing.T) {
	opts := newDefaultOptions()
	_, ac, cancel := newNonRunningTestWebhook(t, opts)
	defer cancel()

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

func TestCertConfigurationForAlreadyGeneratedSecret(t *testing.T) {
	secretName := "test-secret"
	ns := system.Namespace()
	opts := newDefaultOptions()
	opts.SecretName = secretName
	ctx, ac, cancel := newNonRunningTestWebhook(t, opts)
	defer cancel()
	kubeClient := fakekc.Get(ctx)

	newSecret, err := certresources.MakeSecret(
		ctx, opts.SecretName, system.Namespace(), opts.ServiceName)
	if err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}
	_, err = kubeClient.CoreV1().Secrets(ns).Create(newSecret)
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	createNamespace(t, kubeClient, metav1.NamespaceSystem)
	createTestConfigMap(t, kubeClient)

	expectedCert, err := tls.X509KeyPair(newSecret.Data[certresources.ServerCert], newSecret.Data[certresources.ServerKey])
	if err != nil {
		t.Fatalf("Failed to create cert from x509 key pair: %v", err)
	}

	serverKey, serverCert, caCert, err := getOrGenerateKeyCertsFromSecret(ctx, kubeClient, &ac.Options)
	if err != nil {
		t.Fatalf("Failed to configure secret: %v", err)
	}
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		t.Fatalf("Expected to build key pair: %v", err)
	}

	if diff := cmp.Diff(expectedCert.Certificate, cert.Certificate, cmp.AllowUnexported()); diff != "" {
		t.Fatalf("Unexpected cert diff (-want, +got) %v", diff)
	}
	if diff := cmp.Diff(newSecret.Data[certresources.CACert], caCert, cmp.AllowUnexported()); diff != "" {
		t.Fatalf("Unexpected CA cert diff (-want, +got) %v", diff)
	}
}

func TestCertConfigurationForGeneratedSecret(t *testing.T) {
	secretName := "test-secret"
	opts := newDefaultOptions()
	opts.SecretName = secretName
	ctx, ac, cancel := newNonRunningTestWebhook(t, opts)
	defer cancel()
	kubeClient := fakekc.Get(ctx)

	createNamespace(t, kubeClient, metav1.NamespaceSystem)
	createTestConfigMap(t, kubeClient)

	_, _, caCert, err := getOrGenerateKeyCertsFromSecret(ctx, kubeClient, &ac.Options)
	if err != nil {
		t.Fatalf("Failed to configure secret: %v", err)
	}

	p, _ := pem.Decode(caCert)
	if p == nil {
		t.Fatalf("Expected PEM encoded CA cert ")
	}
	if p.Type != "CERTIFICATE" {
		t.Fatalf("Expectet type to be CERTIFICATE but got %s", string(p.Type))
	}
}

func NewTestWebhook(ctx context.Context) (*Webhook, error) {
	validations := configmap.Constructors{"test-config": newConfigFromConfigMap}

	admissionControllers := []AdmissionController{
		NewResourceAdmissionController(
			testResourceValidationName, testResourceValidationPath, handlers, true,
			func(ctx context.Context) context.Context {
				return ctx
			}),
		NewConfigValidationController(
			testConfigValidationName, testConfigValidationPath, validations),
	}
	return New(ctx, admissionControllers)
}
