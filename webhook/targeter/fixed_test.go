/*
Copyright 2022 The Knative Authors

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

package targeter

import (
	"testing"

	fakesecret "knative.dev/pkg/injection/clients/namespacedkube/informers/core/v1/secret/fake"
	certresources "knative.dev/pkg/webhook/certificates/resources"

	"github.com/google/go-cmp/cmp"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook"

	. "knative.dev/pkg/reconciler/testing"
)

func TestFixed(t *testing.T) {
	ctx, _ := SetupFakeContext(t)

	ctx = webhook.WithOptions(ctx, webhook.Options{
		ServiceName: "webhook",
		SecretName:  "precious",
	})

	wantBundle := []byte("pem")

	fakesecret.Get(ctx).Informer().GetIndexer().Add(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: system.Namespace(),
			Name:      "precious",
		},
		Data: map[string][]byte{
			certresources.CACert: wantBundle,
		},
	})

	wantBasePath := "/mypath"
	l := NewFixed(ctx, wantBasePath)

	l.AddEventHandlers(ctx, func(interface{}) {})

	if gotBasePath := l.BasePath(); gotBasePath != wantBasePath {
		t.Errorf("BasePath() = %s, wanted %s", gotBasePath, wantBasePath)
	}

	got, err := l.WebhookClientConfig(ctx)
	if err != nil {
		t.Fatalf("WebhookClientConfig() = %v", err)
	}
	want := &admissionregistrationv1.WebhookClientConfig{
		Service: &admissionregistrationv1.ServiceReference{
			Namespace: system.Namespace(),
			Name:      "webhook",
			Path:      ptr.String(wantBasePath),
		},
		CABundle: wantBundle,
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("WebhookClientConfig (-got, +want) = %s", diff)
	}
}

func TestFixedMissingSecret(t *testing.T) {
	ctx, _ := SetupFakeContext(t)

	ctx = webhook.WithOptions(ctx, webhook.Options{
		ServiceName: "webhook",
		SecretName:  "not-found",
	})

	l := NewFixed(ctx, "/mypath")

	if cc, err := l.WebhookClientConfig(ctx); err == nil {
		t.Errorf("WebhookClientConfig() = %+v, wanted error", cc)
	}
}

func TestFixedMalformedSecret(t *testing.T) {
	ctx, _ := SetupFakeContext(t)

	ctx = webhook.WithOptions(ctx, webhook.Options{
		ServiceName: "webhook",
		SecretName:  "precious",
	})

	fakesecret.Get(ctx).Informer().GetIndexer().Add(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: system.Namespace(),
			Name:      "precious",
		},
		Data: map[string][]byte{
			"wrong-key": []byte("doesn't matter"),
		},
	})

	l := NewFixed(ctx, "/mypath")

	if cc, err := l.WebhookClientConfig(ctx); err == nil {
		t.Errorf("WebhookClientConfig() = %+v, wanted error", cc)
	}
}
