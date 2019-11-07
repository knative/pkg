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

package fake

import (
	"context"

	"k8s.io/client-go/dynamic/dynamicinformer"

	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
	fake "knative.dev/pkg/injection/clients/dynamicclient/fake"
	"knative.dev/pkg/injection/informers/factory"
)

func init() {
	injection.Fake.RegisterInformerFactory(withInformerFactory)
}

func withInformerFactory(ctx context.Context) context.Context {
	kc := fake.Get(ctx)

	namespace := ""
	if injection.HasNamespaceScope(ctx) {
		namespace = injection.GetNamespaceScope(ctx)
	}

	return context.WithValue(ctx, factory.Key{},
		dynamicinformer.NewFilteredDynamicSharedInformerFactory(kc, controller.GetResyncPeriod(ctx), namespace, nil))
}
