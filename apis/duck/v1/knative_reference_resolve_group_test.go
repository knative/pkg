package v1_test

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
			err := tc.input.ResolveGroup(fakeCrdInformer.Lister())
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
				if !cmp.Equal(tc.output, tc.input) {
					t.Errorf("ResolveGroup diff: (-want, +got) =\n%s", cmp.Diff(tc.input, tc.output))
				}
			}
		})
	}
}
