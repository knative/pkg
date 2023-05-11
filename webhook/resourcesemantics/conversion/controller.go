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

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	apixclient "knative.dev/pkg/client/injection/apiextensions/client"
	crdinformer "knative.dev/pkg/client/injection/apiextensions/informers/apiextensions/v1/customresourcedefinition"
	"knative.dev/pkg/controller"
	secretinformer "knative.dev/pkg/injection/clients/namespacedkube/informers/core/v1/secret"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/resourcesemantics/common"
)

// NewConversionController returns a K8s controller that will
// will reconcile CustomResourceDefinitions and update their
// conversion webhook attributes such as path & CA bundle.
//
// Additionally the controller's Reconciler implements
// webhook.ConversionController for the purposes of converting
// resources between different versions
func NewConversionController(
	ctx context.Context,
	path string,
	kinds map[schema.GroupKind]common.GroupKindConversion,
	withContext func(context.Context) context.Context,
) *controller.Impl {

	opts := []common.OptionFunc{
		common.WithPath(path),
		common.WithWrapContext(withContext),
		common.WithKinds(kinds),
	}

	return NewController(ctx, opts...)
}

func NewController(ctx context.Context, optsFunc ...common.OptionFunc) *controller.Impl {
	secretInformer := secretinformer.Get(ctx)
	crdInformer := crdinformer.Get(ctx)
	client := apixclient.Get(ctx)
	options := webhook.GetOptions(ctx)

	opts := common.NewOptions()

	for _, f := range optsFunc {
		f(opts)
	}

	r := &reconciler{
		LeaderAwareFuncs: pkgreconciler.LeaderAwareFuncs{
			// Have this reconciler enqueue our types whenever it becomes leader.
			PromoteFunc: func(bkt pkgreconciler.Bucket, enq func(pkgreconciler.Bucket, types.NamespacedName)) error {
				for _, gkc := range opts.Kinds() {
					name := gkc.DefinitionName
					enq(bkt, types.NamespacedName{Name: name})
				}
				return nil
			},
		},

		kinds:       opts.Kinds(),
		path:        opts.Path(),
		secretName:  options.SecretName,
		withContext: opts.WrapContext(),

		client:       client,
		secretLister: secretInformer.Lister(),
		crdLister:    crdInformer.Lister(),
	}

	logger := logging.FromContext(ctx)
	controllerOptions := options.ControllerOptions
	if controllerOptions == nil {
		const queueName = "ConversionWebhook"
		controllerOptions = &controller.ControllerOptions{WorkQueueName: queueName, Logger: logger.Named(queueName)}
	}
	c := controller.NewContext(ctx, r, *controllerOptions)

	// Reconciler when the named CRDs change.
	for _, gkc := range opts.Kinds() {
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
