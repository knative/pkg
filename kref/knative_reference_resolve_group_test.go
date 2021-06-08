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

package kref

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	. "knative.dev/pkg/apis/duck/v1"
	customresourcedefinitioninformer "knative.dev/pkg/client/injection/apiextensions/informers/apiextensions/v1/customresourcedefinition/fake"
	"knative.dev/pkg/injection"
)

func TestResolveGroup(t *testing.T) {
	const crdGroup = "messaging.knative.dev"
	const crdName = "inmemorychannels." + crdGroup

	ctx, _ := injection.Fake.SetupInformers(context.TODO(), &rest.Config{})

	fakeCrdInformer := customresourcedefinitioninformer.Get(ctx)
	fakeCrdInformer.Informer().GetIndexer().Add(
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: crdName,
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
					Name:    "v1beta1",
					Storage: false,
				}, {
					Name:    "v1",
					Storage: true,
				}},
			},
		},
	)

	tests := map[string]struct {
		input   *KReference
		output  *KReference
		wantErr bool
	}{
		"No group": {
			input: &KReference{
				Kind:       "Abc",
				Name:       "123",
				APIVersion: "something/v1",
			},
			output: &KReference{
				Kind:       "Abc",
				Name:       "123",
				APIVersion: "something/v1",
			},
		},
		"No group nor api version": {
			input: &KReference{
				Kind: "Abc",
				Name: "123",
			},
			output: &KReference{
				Kind: "Abc",
				Name: "123",
			},
		},
		"imc channel": {
			input: &KReference{
				Kind:  "InMemoryChannel",
				Name:  "my-cool-channel",
				Group: crdGroup,
			},
			output: &KReference{
				Kind:       "InMemoryChannel",
				Name:       "my-cool-channel",
				Group:      crdGroup,
				APIVersion: crdGroup + "/v1",
			},
		},
		"unknown CRD": {
			input: &KReference{
				Kind:  "MyChannel",
				Name:  "my-cool-channel",
				Group: crdGroup,
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			kr, err := ResolveGroup(tc.input, fakeCrdInformer.Lister())
			if err != nil {
				if !tc.wantErr {
					t.Error("ResolveGroup() =", err)
				}
				return
			} else if tc.wantErr {
				t.Errorf("ResolveGroup() = %v, wanted error", err)
				return
			}

			if tc.output != nil {
				if !cmp.Equal(tc.output, kr) {
					t.Errorf("ResolveGroup diff: (-want, +got) =\n%s", cmp.Diff(tc.input, tc.output))
				}
			}
		})
	}
}
