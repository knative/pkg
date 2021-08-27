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

package client

import (
	context "context"
	json "encoding/json"
	errors "errors"
	fmt "fmt"

	apisapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	typedapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	typedapiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	discovery "k8s.io/client-go/discovery"
	dynamic "k8s.io/client-go/dynamic"
	rest "k8s.io/client-go/rest"
	injection "knative.dev/pkg/injection"
	dynamicclient "knative.dev/pkg/injection/clients/dynamicclient"
	logging "knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterClient(withClientFromConfig)
	injection.Default.RegisterClientFetcher(func(ctx context.Context) interface{} {
		return Get(ctx)
	})
	injection.Dynamic.RegisterDynamicClient(withClientFromDynamic)
}

// Key is used as the key for associating information with a context.Context.
type Key struct{}

func withClientFromConfig(ctx context.Context, cfg *rest.Config) context.Context {
	return context.WithValue(ctx, Key{}, clientset.NewForConfigOrDie(cfg))
}

func withClientFromDynamic(ctx context.Context) context.Context {
	return context.WithValue(ctx, Key{}, &wrapClient{dyn: dynamicclient.Get(ctx)})
}

// Get extracts the clientset.Interface client from the context.
func Get(ctx context.Context) clientset.Interface {
	untyped := ctx.Value(Key{})
	if untyped == nil {
		if injection.GetConfig(ctx) == nil {
			logging.FromContext(ctx).Panic(
				"Unable to fetch k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset.Interface from context. This context is not the application context (which is typically given to constructors via sharedmain).")
		} else {
			logging.FromContext(ctx).Panic(
				"Unable to fetch k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset.Interface from context.")
		}
	}
	return untyped.(clientset.Interface)
}

type wrapClient struct {
	dyn dynamic.Interface
}

var _ clientset.Interface = (*wrapClient)(nil)

func (w *wrapClient) Discovery() discovery.DiscoveryInterface {
	panic("Discovery called on dynamic client!")
}

func convert(from interface{}, to runtime.Object) error {
	bs, err := json.Marshal(from)
	if err != nil {
		return fmt.Errorf("Marshal() = %w", err)
	}
	if err := json.Unmarshal(bs, to); err != nil {
		return fmt.Errorf("Unmarshal() = %w", err)
	}
	return nil
}

// ApiextensionsV1beta1 retrieves the ApiextensionsV1beta1Client
func (w *wrapClient) ApiextensionsV1beta1() typedapiextensionsv1beta1.ApiextensionsV1beta1Interface {
	return &wrapApiextensionsV1beta1{
		dyn: w.dyn,
	}
}

type wrapApiextensionsV1beta1 struct {
	dyn dynamic.Interface
}

func (w *wrapApiextensionsV1beta1) RESTClient() rest.Interface {
	panic("RESTClient called on dynamic client!")
}

func (w *wrapApiextensionsV1beta1) CustomResourceDefinitions() typedapiextensionsv1beta1.CustomResourceDefinitionInterface {
	return &wrapApiextensionsV1beta1CustomResourceDefinitionImpl{
		dyn: w.dyn.Resource(schema.GroupVersionResource{
			Group:    "apiextensions.k8s.io",
			Version:  "v1beta1",
			Resource: "customresourcedefinitions",
		}),
	}
}

type wrapApiextensionsV1beta1CustomResourceDefinitionImpl struct {
	dyn dynamic.NamespaceableResourceInterface
}

var _ typedapiextensionsv1beta1.CustomResourceDefinitionInterface = (*wrapApiextensionsV1beta1CustomResourceDefinitionImpl)(nil)

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) Create(ctx context.Context, in *apiextensionsv1beta1.CustomResourceDefinition, opts v1.CreateOptions) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	in.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1beta1",
		Kind:    "CustomResourceDefinition",
	})
	uo := &unstructured.Unstructured{}
	if err := convert(in, uo); err != nil {
		return nil, err
	}
	uo, err := w.dyn.Create(ctx, uo, opts)
	if err != nil {
		return nil, err
	}
	out := &apiextensionsv1beta1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return w.dyn.Delete(ctx, name, opts)
}

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	return w.dyn.DeleteCollection(ctx, opts, listOpts)
}

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) Get(ctx context.Context, name string, opts v1.GetOptions) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	uo, err := w.dyn.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	out := &apiextensionsv1beta1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) List(ctx context.Context, opts v1.ListOptions) (*apiextensionsv1beta1.CustomResourceDefinitionList, error) {
	uo, err := w.dyn.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	out := &apiextensionsv1beta1.CustomResourceDefinitionList{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *apiextensionsv1beta1.CustomResourceDefinition, err error) {
	uo, err := w.dyn.Patch(ctx, name, pt, data, opts)
	if err != nil {
		return nil, err
	}
	out := &apiextensionsv1beta1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) Update(ctx context.Context, in *apiextensionsv1beta1.CustomResourceDefinition, opts v1.UpdateOptions) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	in.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1beta1",
		Kind:    "CustomResourceDefinition",
	})
	uo := &unstructured.Unstructured{}
	if err := convert(in, uo); err != nil {
		return nil, err
	}
	uo, err := w.dyn.Update(ctx, uo, opts)
	if err != nil {
		return nil, err
	}
	out := &apiextensionsv1beta1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) UpdateStatus(ctx context.Context, in *apiextensionsv1beta1.CustomResourceDefinition, opts v1.UpdateOptions) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	in.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1beta1",
		Kind:    "CustomResourceDefinition",
	})
	uo := &unstructured.Unstructured{}
	if err := convert(in, uo); err != nil {
		return nil, err
	}
	uo, err := w.dyn.UpdateStatus(ctx, uo, opts)
	if err != nil {
		return nil, err
	}
	out := &apiextensionsv1beta1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1beta1CustomResourceDefinitionImpl) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("NYI: Watch")
}

// ApiextensionsV1 retrieves the ApiextensionsV1Client
func (w *wrapClient) ApiextensionsV1() typedapiextensionsv1.ApiextensionsV1Interface {
	return &wrapApiextensionsV1{
		dyn: w.dyn,
	}
}

type wrapApiextensionsV1 struct {
	dyn dynamic.Interface
}

func (w *wrapApiextensionsV1) RESTClient() rest.Interface {
	panic("RESTClient called on dynamic client!")
}

func (w *wrapApiextensionsV1) CustomResourceDefinitions() typedapiextensionsv1.CustomResourceDefinitionInterface {
	return &wrapApiextensionsV1CustomResourceDefinitionImpl{
		dyn: w.dyn.Resource(schema.GroupVersionResource{
			Group:    "apiextensions.k8s.io",
			Version:  "v1",
			Resource: "customresourcedefinitions",
		}),
	}
}

type wrapApiextensionsV1CustomResourceDefinitionImpl struct {
	dyn dynamic.NamespaceableResourceInterface
}

var _ typedapiextensionsv1.CustomResourceDefinitionInterface = (*wrapApiextensionsV1CustomResourceDefinitionImpl)(nil)

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) Create(ctx context.Context, in *apisapiextensionsv1.CustomResourceDefinition, opts v1.CreateOptions) (*apisapiextensionsv1.CustomResourceDefinition, error) {
	in.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	})
	uo := &unstructured.Unstructured{}
	if err := convert(in, uo); err != nil {
		return nil, err
	}
	uo, err := w.dyn.Create(ctx, uo, opts)
	if err != nil {
		return nil, err
	}
	out := &apisapiextensionsv1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return w.dyn.Delete(ctx, name, opts)
}

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	return w.dyn.DeleteCollection(ctx, opts, listOpts)
}

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) Get(ctx context.Context, name string, opts v1.GetOptions) (*apisapiextensionsv1.CustomResourceDefinition, error) {
	uo, err := w.dyn.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}
	out := &apisapiextensionsv1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) List(ctx context.Context, opts v1.ListOptions) (*apisapiextensionsv1.CustomResourceDefinitionList, error) {
	uo, err := w.dyn.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	out := &apisapiextensionsv1.CustomResourceDefinitionList{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *apisapiextensionsv1.CustomResourceDefinition, err error) {
	uo, err := w.dyn.Patch(ctx, name, pt, data, opts)
	if err != nil {
		return nil, err
	}
	out := &apisapiextensionsv1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) Update(ctx context.Context, in *apisapiextensionsv1.CustomResourceDefinition, opts v1.UpdateOptions) (*apisapiextensionsv1.CustomResourceDefinition, error) {
	in.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	})
	uo := &unstructured.Unstructured{}
	if err := convert(in, uo); err != nil {
		return nil, err
	}
	uo, err := w.dyn.Update(ctx, uo, opts)
	if err != nil {
		return nil, err
	}
	out := &apisapiextensionsv1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) UpdateStatus(ctx context.Context, in *apisapiextensionsv1.CustomResourceDefinition, opts v1.UpdateOptions) (*apisapiextensionsv1.CustomResourceDefinition, error) {
	in.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	})
	uo := &unstructured.Unstructured{}
	if err := convert(in, uo); err != nil {
		return nil, err
	}
	uo, err := w.dyn.UpdateStatus(ctx, uo, opts)
	if err != nil {
		return nil, err
	}
	out := &apisapiextensionsv1.CustomResourceDefinition{}
	if err := convert(uo, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *wrapApiextensionsV1CustomResourceDefinitionImpl) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return nil, errors.New("NYI: Watch")
}
