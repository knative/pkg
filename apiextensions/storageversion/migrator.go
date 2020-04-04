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

package storageversion

import (
	"context"
	"fmt"

	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apixclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/pager"
)

type Migrator struct {
	dynamicClient dynamic.Interface
	apixClient    apixclient.Interface
}

func NewMigrator(d dynamic.Interface, a apixclient.Interface) *Migrator {
	return &Migrator{
		dynamicClient: d,
		apixClient:    a,
	}
}

func (m *Migrator) Migrate(ctx context.Context, gr schema.GroupResource) error {
	crdClient := m.apixClient.ApiextensionsV1beta1().CustomResourceDefinitions()

	crd, err := crdClient.Get(gr.String(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to fetch crd %s - %w", gr, err)
	}

	version := storageVersion(crd)

	if err := m.migrateResources(ctx, gr.WithVersion(version)); err != nil {
		return err
	}

	patch := fmt.Sprintf(`{"status":{"storedVersions":["` + version + `"]}}`)
	_, err = crdClient.Patch(crd.Name, types.StrategicMergePatchType, []byte(patch), "status")
	if err != nil {
		return fmt.Errorf("unable to drop storage version definition %s - %w", gr, err)
	}

	return nil
}

func (m *Migrator) migrateResources(ctx context.Context, gvr schema.GroupVersionResource) error {
	client := m.dynamicClient.Resource(gvr)

	listFunc := func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return client.Namespace(metav1.NamespaceAll).List(opts)
	}

	onEach := func(obj runtime.Object) error {
		item := obj.(metav1.Object)

		_, err := client.Namespace(item.GetNamespace()).
			Patch(item.GetName(), types.MergePatchType, []byte("{}"), metav1.PatchOptions{})

		if err != nil {
			return fmt.Errorf("unable to patch resource %s/%s (gvr: %s) - %w",
				item.GetNamespace(), item.GetName(),
				gvr, err)
		}

		return nil
	}

	pager := pager.New(listFunc)
	return pager.EachListItem(ctx, metav1.ListOptions{}, onEach)
}

func storageVersion(crd *apix.CustomResourceDefinition) string {
	var version string

	for _, v := range crd.Spec.Versions {
		if v.Storage {
			version = v.Name
			break
		}
	}

	if version == "" {
		version = crd.Spec.Version
	}

	return version
}
