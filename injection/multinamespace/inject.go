/*
Copyright 2025 The Knative Authors

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

// Package multinamespace provides an opt-in kube SharedInformerFactory override
// that scopes the secret informer to multiple namespaces while leaving other
// informer types on a self-contained default factory (cluster-wide, or
// single-namespace when injection.WithNamespaceScope is set).
//
// Usage from main():
//
//	import (
//	    "knative.dev/pkg/injection"
//	    "knative.dev/pkg/injection/multinamespace"
//	    "knative.dev/pkg/injection/sharedmain"
//	    "knative.dev/pkg/signals"
//	)
//
//	func main() {
//	    ctx := signals.NewContext()
//	    ctx = injection.WithNamespaceScopes(ctx, "tenant-a", "tenant-b")
//	    multinamespace.RegisterScopeOverride(10 * time.Minute)
//	    sharedmain.MainWithContext(ctx, "mycomponent", controllers...)
//	}
//
// RegisterScopeOverride must be called after the generated kube factory is
// linked (typically from main before sharedmain) so its injector runs last and
// overwrites kubefactory.Key{} when len(injection.GetNamespaceScopes(ctx)) > 1.
package multinamespace

import (
	"context"
	"time"

	kubeclient "knative.dev/pkg/client/injection/kube/client"
	kubefactory "knative.dev/pkg/client/injection/kube/informers/factory"
	"knative.dev/pkg/injection"
)

// RegisterScopeOverride registers an InformerFactoryInjector with
// injection.Default that, when more than one namespace scope is set on the
// context, replaces the kube SharedInformerFactory with a self-contained
// scopedFactory (secrets merged across namespaces; other types unchanged).
func RegisterScopeOverride(resync time.Duration) {
	injection.Default.RegisterInformerFactory(func(ctx context.Context) context.Context {
		namespaces := injection.GetNamespaceScopes(ctx)
		if len(namespaces) <= 1 {
			return ctx
		}
		scoped := NewScopedFactory(
			kubeclient.Get(ctx),
			resync,
			namespaces,
			injection.GetNamespaceScope(ctx),
		)
		return context.WithValue(ctx, kubefactory.Key{}, scoped)
	})
}
