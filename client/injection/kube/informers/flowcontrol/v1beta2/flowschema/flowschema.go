/*
Copyright 2021 The Knative Authors

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

// Code generated by injection-gen. DO NOT EDIT.

package flowschema

import (
	context "context"

	apiflowcontrolv1beta2 "k8s.io/api/flowcontrol/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	v1beta2 "k8s.io/client-go/informers/flowcontrol/v1beta2"
	kubernetes "k8s.io/client-go/kubernetes"
	flowcontrolv1beta2 "k8s.io/client-go/listers/flowcontrol/v1beta2"
	cache "k8s.io/client-go/tools/cache"
	client "knative.dev/pkg/client/injection/kube/client"
	factory "knative.dev/pkg/client/injection/kube/informers/factory"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
	logging "knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterInformer(withInformer)
	injection.Dynamic.RegisterDynamicInformer(withDynamicInformer)
}

// Key is used for associating the Informer inside the context.Context.
type Key struct{}

func withInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := factory.Get(ctx)
	inf := f.Flowcontrol().V1beta2().FlowSchemas()
	return context.WithValue(ctx, Key{}, inf), inf.Informer()
}

func withDynamicInformer(ctx context.Context) context.Context {
	inf := &wrapper{client: client.Get(ctx), resourceVersion: injection.GetResourceVersion(ctx)}
	return context.WithValue(ctx, Key{}, inf)
}

// Get extracts the typed informer from the context.
func Get(ctx context.Context) v1beta2.FlowSchemaInformer {
	untyped := ctx.Value(Key{})
	if untyped == nil {
		logging.FromContext(ctx).Panic(
			"Unable to fetch k8s.io/client-go/informers/flowcontrol/v1beta2.FlowSchemaInformer from context.")
	}
	return untyped.(v1beta2.FlowSchemaInformer)
}

type wrapper struct {
	client kubernetes.Interface

	resourceVersion string
}

var _ v1beta2.FlowSchemaInformer = (*wrapper)(nil)
var _ flowcontrolv1beta2.FlowSchemaLister = (*wrapper)(nil)

func (w *wrapper) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(nil, &apiflowcontrolv1beta2.FlowSchema{}, 0, nil)
}

func (w *wrapper) Lister() flowcontrolv1beta2.FlowSchemaLister {
	return w
}

// SetResourceVersion allows consumers to adjust the minimum resourceVersion
// used by the underlying client.  It is not accessible via the standard
// lister interface, but can be accessed through a user-defined interface and
// an implementation check e.g. rvs, ok := foo.(ResourceVersionSetter)
func (w *wrapper) SetResourceVersion(resourceVersion string) {
	w.resourceVersion = resourceVersion
}

func (w *wrapper) List(selector labels.Selector) (ret []*apiflowcontrolv1beta2.FlowSchema, err error) {
	lo, err := w.client.FlowcontrolV1beta2().FlowSchemas().List(context.TODO(), v1.ListOptions{
		LabelSelector:   selector.String(),
		ResourceVersion: w.resourceVersion,
	})
	if err != nil {
		return nil, err
	}
	for idx := range lo.Items {
		ret = append(ret, &lo.Items[idx])
	}
	return ret, nil
}

func (w *wrapper) Get(name string) (*apiflowcontrolv1beta2.FlowSchema, error) {
	return w.client.FlowcontrolV1beta2().FlowSchemas().Get(context.TODO(), name, v1.GetOptions{
		ResourceVersion: w.resourceVersion,
	})
}
