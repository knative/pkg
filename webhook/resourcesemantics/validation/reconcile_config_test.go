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

package validation

import (
	"context"
	"testing"
	"time"

	kubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	_ "knative.dev/pkg/injection/clients/namespacedkube/informers/core/v1/secret/fake"
	pkgreconciler "knative.dev/pkg/reconciler"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgotesting "k8s.io/client-go/testing"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook"
	certresources "knative.dev/pkg/webhook/certificates/resources"
	"knative.dev/pkg/webhook/resourcesemantics"

	. "knative.dev/pkg/reconciler/testing"
	. "knative.dev/pkg/webhook/testing"
)

func TestReconcile(t *testing.T) {
	const name, path = "foo.bar.baz", "/blah"
	const secretName = "webhook-secret"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: system.Namespace(),
		},
		Data: map[string][]byte{
			certresources.ServerKey:  []byte("present"),
			certresources.ServerCert: []byte("present"),
			certresources.CACert:     []byte("present"),
		},
	}

	// This is the namespace selector setup
	namespaceSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      "webhooks.knative.dev/exclude",
			Operator: metav1.LabelSelectorOpDoesNotExist,
		}},
	}

	// These are the rules we expect given the context of "handlers".
	expectedRules := []admissionregistrationv1.RuleWithOperations{{
		Operations: []admissionregistrationv1.OperationType{"CREATE", "UPDATE", "DELETE"},
		Rule: admissionregistrationv1.Rule{
			APIGroups:   []string{"pkg.knative.dev"},
			APIVersions: []string{"v1alpha1"},
			Resources:   []string{"innerdefaultresources/*"},
		},
	}, {
		Operations: []admissionregistrationv1.OperationType{"CREATE", "UPDATE", "DELETE"},
		Rule: admissionregistrationv1.Rule{
			APIGroups:   []string{"pkg.knative.dev"},
			APIVersions: []string{"v1alpha1"},
			Resources:   []string{"resources/*"},
		},
	}, {
		Operations: []admissionregistrationv1.OperationType{"CREATE", "UPDATE", "DELETE"},
		Rule: admissionregistrationv1.Rule{
			APIGroups:   []string{"pkg.knative.dev"},
			APIVersions: []string{"v1beta1"},
			Resources:   []string{"resources/*"},
		},
	}, {
		Operations: []admissionregistrationv1.OperationType{"CREATE", "UPDATE", "DELETE"},
		Rule: admissionregistrationv1.Rule{
			APIGroups:   []string{"pkg.knative.io"},
			APIVersions: []string{"v1alpha1"},
			Resources:   []string{"innerdefaultresources/*"},
		},
	}}

	// The key to use, which for this singleton reconciler doesn't matter (although the
	// namespace matters for namespace validation).
	key := system.Namespace() + "/does not matter"

	table := TableTest{{
		Name:    "no secret",
		Key:     key,
		WantErr: true,
	}, {
		Name: "secret missing CA Cert",
		Key:  key,
		Objects: []runtime.Object{&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: system.Namespace(),
			},
			Data: map[string][]byte{
				certresources.ServerKey:  []byte("present"),
				certresources.ServerCert: []byte("present"),
				// certresources.CACert:     []byte("missing"),
			},
		}},
		WantErr: true,
	}, {
		Name:    "secret exists, but VWH does not",
		Key:     key,
		Objects: []runtime.Object{secret},
		WantErr: true,
	}, {
		Name: "secret and VWH exist, missing service reference",
		Key:  key,
		Objects: []runtime.Object{secret,
			&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{{
					Name: name,
				}},
			},
		},
		WantErr: true,
	}, {
		Name: "secret and VWH exist, missing other stuff",
		Key:  key,
		Objects: []runtime.Object{secret,
			&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{{
					Name: name,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: system.Namespace(),
							Name:      "webhook",
						},
					},
				}},
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{{
					Name: name,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: system.Namespace(),
							Name:      "webhook",
							// Path is added.
							Path: ptr.String(path),
						},
						// CABundle is added.
						CABundle: []byte("present"),
					},
					// Rules are added.
					Rules:             expectedRules,
					NamespaceSelector: namespaceSelector,
				}},
			},
		}},
	}, {
		Name: "secret and VWH exist, added fields are incorrect",
		Key:  key,
		Objects: []runtime.Object{secret,
			&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{{
					Name: name,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: system.Namespace(),
							Name:      "webhook",
							// Incorrect
							Path: ptr.String("incorrect"),
						},
						// Incorrect
						CABundle: []byte("incorrect"),
					},
					// Incorrect (really just incomplete)
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Operations: []admissionregistrationv1.OperationType{"CREATE", "UPDATE"},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"pkg.knative.dev"},
							APIVersions: []string{"v1alpha1"},
							Resources:   []string{"innerdefaultresources/*"},
						},
					}},
				}},
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{{
					Name: name,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: system.Namespace(),
							Name:      "webhook",
							// Path is fixed.
							Path: ptr.String(path),
						},
						// CABundle is fixed.
						CABundle: []byte("present"),
					},
					// Rules are fixed.
					Rules:             expectedRules,
					NamespaceSelector: namespaceSelector,
				}},
			},
		}},
	}, {
		Name:    "failure updating VWH",
		Key:     key,
		WantErr: true,
		WithReactors: []clientgotesting.ReactionFunc{
			InduceFailure("update", "validatingwebhookconfigurations"),
		},
		Objects: []runtime.Object{secret,
			&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{{
					Name: name,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: system.Namespace(),
							Name:      "webhook",
							// Incorrect
							Path: ptr.String("incorrect"),
						},
						// Incorrect
						CABundle: []byte("incorrect"),
					},
					// Incorrect (really just incomplete)
					Rules: []admissionregistrationv1.RuleWithOperations{{
						Operations: []admissionregistrationv1.OperationType{"CREATE", "UPDATE"},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"pkg.knative.dev"},
							APIVersions: []string{"v1alpha1"},
							Resources:   []string{"innerdefaultresources/*"},
						},
					}},
				}},
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{{
			Object: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{{
					Name: name,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: system.Namespace(),
							Name:      "webhook",
							// Path is fixed.
							Path: ptr.String(path),
						},
						// CABundle is fixed.
						CABundle: []byte("present"),
					},
					// Rules are fixed.
					Rules:             expectedRules,
					NamespaceSelector: namespaceSelector,
				}},
			},
		}},
	}, {
		Name: ":fire: everything is fine :fire:",
		Key:  key,
		Objects: []runtime.Object{secret,
			&admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{{
					Name: name,
					ClientConfig: admissionregistrationv1.WebhookClientConfig{
						Service: &admissionregistrationv1.ServiceReference{
							Namespace: system.Namespace(),
							Name:      "webhook",
							// Path is fine.
							Path: ptr.String(path),
						},
						// CABundle is fine.
						CABundle: []byte("present"),
					},
					// Rules are fine.
					Rules:             expectedRules,
					NamespaceSelector: namespaceSelector,
				}},
			},
		},
	}}

	table.Test(t, MakeFactory(func(ctx context.Context, listers *Listers, cmw configmap.Watcher) controller.Reconciler {
		return &reconciler{
			key: types.NamespacedName{
				Name: name,
			},
			path: path,

			handlers: handlers,

			client:       kubeclient.Get(ctx),
			vwhlister:    listers.GetValidatingWebhookConfigurationLister(),
			secretlister: listers.GetSecretLister(),

			secretName: secretName,
		}
	}))
}

func TestNew(t *testing.T) {
	ctx, cancel, _ := SetupFakeContextWithCancel(t)
	defer cancel()
	ctx = webhook.WithOptions(ctx, webhook.Options{})

	c := NewAdmissionController(ctx, "foo", "/bar",
		map[schema.GroupVersionKind]resourcesemantics.GenericCRD{},
		func(ctx context.Context) context.Context {
			return ctx
		}, true /* disallow unknown field */)
	if c == nil {
		t.Fatal("Expected NewController to return a non-nil value")
	}

	if want, got := 0, c.WorkQueue.Len(); want != got {
		t.Errorf("WorkQueue.Len() = %d, wanted %d", got, want)
	}

	la, ok := c.Reconciler.(pkgreconciler.LeaderAware)
	if !ok {
		t.Fatalf("%T is not leader aware", c.Reconciler)
	}

	if err := la.Promote(pkgreconciler.UniversalBucket(), c.MaybeEnqueueBucketKey); err != nil {
		t.Errorf("Promote() = %v", err)
	}

	// Queue has async moving parts so if we check at the wrong moment, thist might still be 0.
	if wait.PollImmediate(10*time.Millisecond, 250*time.Millisecond, func() (bool, error) {
		return c.WorkQueue.Len() == 1, nil
	}) != nil {
		t.Error("Queue length was never 1")
	}

}
