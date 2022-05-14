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

package configmaps

import (
	"context"
	"reflect"

	// Injection stuff
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	vwhinformer "knative.dev/pkg/client/injection/kube/informers/admissionregistration/v1/validatingwebhookconfiguration"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/webhook/targeter"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

// NewAdmissionController constructs a reconciler
func NewAdmissionController(
	ctx context.Context,
	name, path string,
	constructors configmap.Constructors,
) *controller.Impl {
	return NewAdmissionControllerFromTargeter(ctx, name, targeter.NewFixed(ctx, path), constructors)
}

// NewAdmissionController constructs a reconciler
func NewAdmissionControllerFromTargeter(
	ctx context.Context, name string,
	targeter targeter.Interface,
	constructors configmap.Constructors,
) *controller.Impl {
	client := kubeclient.Get(ctx)
	vwhInformer := vwhinformer.Get(ctx)

	key := types.NamespacedName{Name: name}

	wh := &reconciler{
		LeaderAwareFuncs: pkgreconciler.LeaderAwareFuncs{
			// Have this reconciler enqueue our singleton whenever it becomes leader.
			PromoteFunc: func(bkt pkgreconciler.Bucket, enq func(pkgreconciler.Bucket, types.NamespacedName)) error {
				enq(bkt, key)
				return nil
			},
		},

		key: key,

		constructors: make(map[string]reflect.Value),

		client:    client,
		vwhlister: vwhInformer.Lister(),

		targeter: targeter,
	}

	for configName, constructor := range constructors {
		wh.registerConfig(configName, constructor)
	}

	const queueName = "ConfigMapWebhook"
	c := controller.NewContext(ctx, wh, controller.ControllerOptions{WorkQueueName: queueName, Logger: logging.FromContext(ctx).Named(queueName)})

	// Reconcile when the named ValidatingWebhookConfiguration changes.
	vwhInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterWithName(name),
		// It doesn't matter what we enqueue because we will always Reconcile
		// the named VWH resource.
		Handler: controller.HandleAll(c.Enqueue),
	})

	targeter.AddEventHandlers(ctx, c.Enqueue)

	return c
}
