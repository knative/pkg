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
	"errors"
	"fmt"
	"path"
	"strings"

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

// DispatchingInterface encapsulates how webhooks that host multiple endpoints
// target us.
type DispatchingInterface interface {
	Interface

	// GetName extracts the name of the resource that we should dispatch to.
	GetName(ctx context.Context, path string) (string, error)
}

// NewDynamic returns a DispatchingInterface that handles a path prefix by
// encoding "names" under that path, and allowing implementations to extract
// those names to lookup the appropriate handling logic dynamically.
func NewDynamic(ctx context.Context, path string) DispatchingInterface {
	options := webhook.GetOptions(ctx)
	// We rely on something else to have set this up and start it
	// via injection, typically the cert controller.
	secretInformer := nsfactory.Get(ctx).Core().V1().Secrets()
	return &Dynamic{
		Path:         path,
		SecretName:   options.SecretName,
		ServiceName:  options.ServiceName,
		SecretLister: secretInformer.Lister(),
	}
}

// Dynamic is public to make testing easier in other packages.  Code should
// use NewDynamic to construct Dynamic instances.
type Dynamic struct {
	Path         string
	SecretName   string
	ServiceName  string
	SecretLister corelisters.SecretLister
}

// Assert that Local implements DispatchingInterface
var _ DispatchingInterface = (*Dynamic)(nil)

// BasePath implements Interface
func (lt *Dynamic) BasePath() string {
	return lt.Path
}

var ErrMissingName = errors.New("the WebhookClientConfig expects context to be infuxed with a name")

// WebhookClientConfig implements Interface
func (lt *Dynamic) WebhookClientConfig(ctx context.Context) (*admissionregistrationv1.WebhookClientConfig, error) {
	name := GetName(ctx)
	if name == "" {
		return nil, ErrMissingName
	}

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
			Path:      ptr.String(path.Join(lt.Path, name)),
		},
		CABundle: cacert,
	}, nil
}

// GetName implements Interface
func (lt *Dynamic) GetName(ctx context.Context, path string) (string, error) {
	if !strings.HasPrefix(path, lt.Path) {
		return "", fmt.Errorf("expected path %q to have prefix %q", path, lt.Path)
	}
	return path[len(lt.Path):], nil
}

// AddEventHandlers implements Interface
func (lt *Dynamic) AddEventHandlers(ctx context.Context, f func(interface{})) {
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

type namectx struct{}

// WithName associates a name with the provided context.
func WithName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, namectx{}, name)
}

// GetName extracts a Webhook associated with the provided context.
func GetName(ctx context.Context) string {
	untyped := ctx.Value(namectx{})
	if untyped == nil {
		return ""
	}
	return untyped.(string)
}
