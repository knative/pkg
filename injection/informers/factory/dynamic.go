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

package factory

import (
	"context"

	"k8s.io/client-go/dynamic/dynamicinformer"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterInformerFactory(withInformerFactory)
}

// Key is used as the key for associating information
// with a context.Context.
type Key struct{}

func withInformerFactory(ctx context.Context) context.Context {
	kc := dynamicclient.Get(ctx)

	namespace := ""
	if injection.HasNamespaceScope(ctx) {
		namespace = injection.GetNamespaceScope(ctx)
	}

	return context.WithValue(ctx, Key{},
		dynamicinformer.NewFilteredDynamicSharedInformerFactory(kc, controller.GetResyncPeriod(ctx), namespace, nil))
}

// Get extracts the Kubernetes InformerFactory from the context.
func Get(ctx context.Context) dynamicinformer.DynamicSharedInformerFactory {
	untyped := ctx.Value(Key{})
	if untyped == nil {
		logging.FromContext(ctx).Panicf(
			"Unable to fetch %T from context.", (dynamicinformer.DynamicSharedInformerFactory)(nil))
	}
	return untyped.(dynamicinformer.DynamicSharedInformerFactory)
}
