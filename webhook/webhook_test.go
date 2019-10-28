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
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	fakekc "knative.dev/pkg/client/injection/kube/client/fake"

	"knative.dev/pkg/configmap"
	. "knative.dev/pkg/logging/testing"
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
	kubeClient *fakekubeclientset.Clientset,
	ac *Webhook) {
	t.Helper()

	// Create fake clients
	ctx := TestContextWithLogger(t)
	ctx, kubeClient = fakekc.With(ctx, initialConfigWebhook, initialResourceWebhook)
	ctx = WithOptions(ctx, options)

	ac, err := NewTestWebhook(ctx)
	if err != nil {
		t.Fatalf("Failed to create new admission controller: %v", err)
	}
	return
}

func TestRegistrationStopChanFire(t *testing.T) {
	opts := newDefaultOptions()
	_, ac := newNonRunningTestWebhook(t, opts)

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
	kubeClient, ac := newNonRunningTestWebhook(t, opts)

	ctx := TestContextWithLogger(t)
	newSecret, err := generateSecret(ctx, &opts)
	if err != nil {
		t.Fatalf("Failed to generate secret: %v", err)
	}
	_, err = kubeClient.CoreV1().Secrets(ns).Create(newSecret)
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	createNamespace(t, kubeClient, metav1.NamespaceSystem)
	createTestConfigMap(t, kubeClient)

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
	opts := newDefaultOptions()
	opts.SecretName = secretName
	kubeClient, ac := newNonRunningTestWebhook(t, opts)

	ctx := TestContextWithLogger(t)
	createNamespace(t, kubeClient, metav1.NamespaceSystem)
	createTestConfigMap(t, kubeClient)

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
