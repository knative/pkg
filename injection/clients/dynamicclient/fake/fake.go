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
	"encoding/json"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
)

func init() {
	injection.Fake.RegisterClient(withClient)
}

func withClient(ctx context.Context, cfg *rest.Config) context.Context {
	scheme := runtime.NewScheme()
	k8sscheme.AddToScheme(scheme)
	ctx, _ = With(ctx, scheme)
	return ctx
}

func With(ctx context.Context, scheme *runtime.Scheme, objects ...runtime.Object) (context.Context, *fake.FakeDynamicClient) {
	// We create a scheme were we define all our types and lists
	// and have them map to unstructured types
	//
	// This was a K8s 1.20 breaking change
	unstructuredScheme := runtime.NewScheme()
	for gvk := range scheme.AllKnownTypes() {
		if unstructuredScheme.Recognizes(gvk) {
			continue
		}
		if strings.HasSuffix(gvk.Kind, "List") {
			unstructuredScheme.AddKnownTypeWithName(gvk, &unstructured.UnstructuredList{})
			continue
		}
		unstructuredScheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
	}

	objects = ToUnstructured(scheme, objects)
	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()
		if !unstructuredScheme.Recognizes(gvk) {
			unstructuredScheme.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		}
		gvk.Kind += "List"
		if !unstructuredScheme.Recognizes(gvk) {
			unstructuredScheme.AddKnownTypeWithName(gvk, &unstructured.UnstructuredList{})
		}
	}

	cs := fake.NewSimpleDynamicClient(unstructuredScheme, objects...)
	return context.WithValue(ctx, dynamicclient.Key{}, cs), cs
}

// Get extracts the Kubernetes client from the context.
func Get(ctx context.Context) *fake.FakeDynamicClient {
	untyped := ctx.Value(dynamicclient.Key{})
	if untyped == nil {
		logging.FromContext(ctx).Panicf(
			"Unable to fetch %T from context.", (*fake.FakeDynamicClient)(nil))
	}
	return untyped.(*fake.FakeDynamicClient)
}

// ToUnstructured takes a list of k8s resources and converts them to
// Unstructured objects.
// We must pass objects as Unstructured to the dynamic client fake, or it
// won't handle them properly.
func ToUnstructured(sch *runtime.Scheme, objs []runtime.Object) (us []runtime.Object) {
	for _, obj := range objs {
		// Don't mess with the primary copy
		obj = obj.DeepCopyObject()

		ta, err := meta.TypeAccessor(obj)
		if err != nil {
			panic("Unable to create type accessor: " + err.Error())
		}
		if ta.GetAPIVersion() == "" || ta.GetKind() == "" {
			// Determine and set the TypeMeta for this object based on our test scheme.
			gvks, _, err := sch.ObjectKinds(obj)
			if err != nil {
				panic("Unable to determine kind for type: " + err.Error())
			}
			apiv, k := gvks[0].ToAPIVersionAndKind()
			ta.SetAPIVersion(apiv)
			ta.SetKind(k)
		}

		b, err := json.Marshal(obj)
		if err != nil {
			panic("Unable to marshal: " + err.Error())
		}
		u := &unstructured.Unstructured{}
		if err := json.Unmarshal(b, u); err != nil {
			panic("Unable to unmarshal: " + err.Error())
		}
		us = append(us, u)
	}
	return
}
