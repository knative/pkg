/*
Copyright 2020 The Knative Authors

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

package conversion

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/apis"
	apixclient "knative.dev/pkg/client/injection/apiextensions/client"
	crdinformer "knative.dev/pkg/client/injection/apiextensions/informers/apiextensions/v1beta1/customresourcedefinition"
	secretinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook"
)

type ConvertibleObject interface {
	runtime.Object
	apis.Convertible
	SetGroupVersionKind(gvk schema.GroupVersionKind)
}

type GroupKindConversion struct {
	DefinitionName string
	HubVersion     string
	Zygotes        map[string]ConvertibleObject
}

func NewConversionController(
	ctx context.Context,
	path string,
	kinds map[schema.GroupKind]GroupKindConversion,
	withContext func(context.Context) context.Context,
) *controller.Impl {

	logger := logging.FromContext(ctx)
	secretInformer := secretinformer.Get(ctx)
	crdInformer := crdinformer.Get(ctx)
	client := apixclient.Get(ctx)
	options := webhook.GetOptions(ctx)

	r := &reconciler{
		kinds:       kinds,
		path:        path,
		secretName:  options.SecretName,
		withContext: withContext,

		client:       client,
		secretLister: secretInformer.Lister(),
		crdLister:    crdInformer.Lister(),
	}

	c := controller.NewImpl(r, logger, "ConversionWebhook")

	// Reconciler when the named CRDs change.
	for _, gkc := range kinds {
		name := gkc.DefinitionName

		crdInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterWithName(name),
			Handler:    controller.HandleAll(c.Enqueue),
		})

		sentinel := c.EnqueueSentinel(types.NamespacedName{Name: name})

		// Reconcile when the cert bundle changes.
		secretInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: controller.FilterWithNameAndNamespace(system.Namespace(), options.SecretName),
			Handler:    controller.HandleAll(sentinel),
		})
	}

	return c
}
