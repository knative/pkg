/*
Copyright 2018 The Knative Authors

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

package duck_test

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	"github.com/knative/pkg/apis/duck"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	. "github.com/knative/pkg/testing"
)

func TestSimpleList(t *testing.T) {
	scheme := runtime.NewScheme()
	AddToScheme(scheme)
	duckv1alpha1.AddToScheme(scheme)

	namespace, name := "foo", "bar"
	var want int64 = 1234
	// Despite the signature allowing `...runtime.Object`, this method
	// will not work properly unless the passed objects are `unstructured.Unstructured`
	client := fake.NewSimpleDynamicClient(scheme, &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "pkg.knative.dev/v2",
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
			"spec": map[string]interface{}{
				"generation": want,
			},
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	tif := &duck.TypedInformerFactory{
		Client:       client,
		Type:         &duckv1alpha1.Generational{},
		ResyncPeriod: 1 * time.Second,
		StopChannel:  stopCh,
	}

	// This hangs without:
	// https://github.com/kubernetes/kubernetes/pull/68552
	_, lister, err := tif.Get(SchemeGroupVersion.WithResource("resources"))
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}

	elt, err := lister.ByNamespace(namespace).Get(name)
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}

	got, ok := elt.(*duckv1alpha1.Generational)
	if !ok {
		t.Fatalf("Get() = %T, wanted *duckv1alpha1.Generational", elt)
	}

	if want != int64(got.Spec.Generation) {
		t.Errorf("Get().Spec.Generation = %v, wanted %v", got.Spec.Generation, want)
	}

	// TODO(mattmoor): Access through informer
}
