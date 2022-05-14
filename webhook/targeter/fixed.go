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
	"context"
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
	nsfactory "knative.dev/pkg/injection/clients/namespacedkube/informers/factory"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook"
	certresources "knative.dev/pkg/webhook/certificates/resources"
)

// NewFixed returns an Interface that expects us to run on a fixed path local
// to the cluster.
func NewFixed(ctx context.Context, path string) Interface {
	options := webhook.GetOptions(ctx)
	// We rely on something else to have set this up and start it
	// via injection, typically the cert controller.
	secretInformer := nsfactory.Get(ctx).Core().V1().Secrets()
	return &Fixed{
		Path:         path,
		SecretName:   options.SecretName,
		ServiceName:  options.ServiceName,
		SecretLister: secretInformer.Lister(),
	}
}

// Fixed is public to make testing easier in other packages.  Code should
// use NewFixed to construct Fixed instances.
type Fixed struct {
	Path         string
	SecretName   string
	ServiceName  string
	SecretLister corelisters.SecretLister
}

// Assert that Fixed implements Interface
var _ Interface = (*Fixed)(nil)

// BasePath implements Interface
func (lt *Fixed) BasePath() string {
	return lt.Path
}

// WebhookClientConfig implements Interface
func (lt *Fixed) WebhookClientConfig(ctx context.Context) (*admissionregistrationv1.WebhookClientConfig, error) {
	// Look up the webhook secret, and fetch the CA cert bundle.
	secret, err := lt.SecretLister.Secrets(system.Namespace()).Get(lt.SecretName)
	if err != nil {
		return nil, err
	}
	cacert, ok := secret.Data[certresources.CACert]
	if !ok {
		return nil, fmt.Errorf("secret %q is missing %q key", lt.SecretName, certresources.CACert)
	}

	return &admissionregistrationv1.WebhookClientConfig{
		Service: &admissionregistrationv1.ServiceReference{
			Namespace: system.Namespace(),
			Name:      lt.ServiceName,
			Path:      ptr.String(lt.Path),
		},
		CABundle: cacert,
	}, nil
}

// AddEventHandlers implements Interface
func (lt *Fixed) AddEventHandlers(ctx context.Context, f func(interface{})) {
	// We rely on something else to have set this up and start it
	// via injection, typically the cert controller.
	secretInformer := nsfactory.Get(ctx).Core().V1().Secrets()

	// Reconcile when the cert bundle changes.
	secretInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterWithNameAndNamespace(system.Namespace(), lt.SecretName),
		// It doesn't matter what we enqueue because we will always Reconcile
		// the named resource.
		Handler: controller.HandleAll(f),
	})
}
