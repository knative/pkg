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

package duck_test

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/google/go-cmp/cmp"

	"github.com/knative/pkg/apis/duck"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	duckv1beta1 "github.com/knative/pkg/apis/duck/v1beta1"
	"github.com/knative/pkg/system"

	fakedynamicclient "github.com/knative/pkg/injection/clients/dynamicclient/fake"
)

var (
	addressableDNS = "addressable.sink.svc.cluster.local"
	addressableURI = "http://addressable.sink.svc.cluster.local"

	addressableName = "testsink"
	addressableKind = "Sink"

	unaddressableName       = "testunaddressable"
	unaddressableKind       = "KResource"
	unaddressableAPIVersion = "duck.knative.dev/v1alpha1"
	unaddressableResource   = "kresources.duck.knative.dev"

	testNS = "testnamespace"
)

func init() {
	// Add types to scheme
	duckv1alpha1.AddToScheme(scheme.Scheme)
	duckv1beta1.AddToScheme(scheme.Scheme)
}

func TestGetSinkURI_v1alpha1(t *testing.T) {
	GetSinkURITest(t, v1alpha1)
}

func TestGetSinkURI_v1beta1(t *testing.T) {
	GetSinkURITest(t, v1beta1)
}

func GetSinkURITest(t *testing.T, goose fowl) {

	testCases := map[string]struct {
		objects   []runtime.Object
		namespace string
		want      string
		wantErr   error
		ref       *corev1.ObjectReference
	}{
		"happy": {
			objects: []runtime.Object{
				goose.getAddressable(),
			},
			namespace: testNS,
			ref:       goose.getAddressableRef(),
			want:      fmt.Sprintf("http://%s", addressableDNS),
		},
		"nil hostname": {
			objects: []runtime.Object{
				goose.getAddressableNilAddressInner(),
			},
			namespace: testNS,
			ref:       goose.getAddressableRef(),
			wantErr:   fmt.Errorf(`object "testsink/testnamespace Sink.duck.knative.dev/%s" contains an empty hostname`, goose),
		},
		"nil ref": {
			objects: []runtime.Object{
				goose.getAddressableNilAddressInner(),
			},
			namespace: testNS,
			ref:       nil,
			wantErr:   fmt.Errorf("addressable ref is nil"),
		},
		"nil address": {
			objects: []runtime.Object{
				goose.getAddressableNilAddress(),
			},
			namespace: testNS,
			ref:       goose.getAddressableRef(),
			wantErr:   fmt.Errorf(`object "testsink/testnamespace Sink.duck.knative.dev/%s" does not contain address`, goose),
		},
		"notAddressable": {
			objects: []runtime.Object{
				goose.getAddressableNoStatus(),
			},
			namespace: testNS,
			ref:       goose.getUnaddressableRef(),
			wantErr:   fmt.Errorf(`failed to fetch addressable "testunaddressable/testnamespace KResource.duck.knative.dev/%s" for observer "testunaddressable": kresources.duck.knative.dev "testunaddressable" not found`, goose),
		},
		"notFound": {
			namespace: testNS,
			ref:       goose.getUnaddressableRef(),
			wantErr:   fmt.Errorf(`failed to fetch addressable "testunaddressable/testnamespace KResource.duck.knative.dev/%s" for observer "testunaddressable": kresources.duck.knative.dev "testunaddressable" not found`, goose),
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tc.objects...)
			ar := duck.NewAddressableResolver(ctx, func(string) {})
			obsName := "nilRef"
			if tc.ref != nil {
				obsName = tc.ref.Name
			}
			uri, gotErr := ar.Resolve(tc.ref, goose.getAddressable(), obsName)
			if gotErr != nil {
				if tc.wantErr != nil {
					if diff := cmp.Diff(tc.wantErr.Error(), gotErr.Error()); diff != "" {
						t.Errorf("%s: unexpected error (-want, +got) = %v", n, diff)
					}
				} else {
					t.Errorf("%s: unexpected error %v", n, gotErr.Error())
				}
			}
			if gotErr == nil {
				got := uri
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("%s: unexpected object (-want, +got) = %v", n, diff)
				}
			}
		})
	}
}

type fowl string

var v1alpha1 fowl = "v1alpha1"
var v1beta1 fowl = "v1beta1"

func (g fowl) getApiVersion() string {
	return "duck.knative.dev/" + string(g)
}

func (g fowl) getAddressable() *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": g.getApiVersion(),
		"kind":       addressableKind,
		"metadata": map[string]interface{}{
			"namespace": testNS,
			"name":      addressableName,
		},
	}

	switch string(g) {
	case "v1alpha1":
		obj["status"] = map[string]interface{}{
			"address": map[string]interface{}{
				"hostname": addressableDNS,
			},
		}
	case "v1beta1":
		obj["status"] = map[string]interface{}{
			"address": map[string]interface{}{
				"url": addressableURI,
			},
		}
	}

	return &unstructured.Unstructured{Object: obj}
}

func (g fowl) getAddressableNoStatus() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": g.getApiVersion(),
			"kind":       addressableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      addressableName,
			},
		},
	}
}

func (g fowl) getAddressableNilAddress() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": g.getApiVersion(),
			"kind":       addressableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      addressableName,
			},
			"status": map[string]interface{}{
				"address": map[string]interface{}(nil),
			},
		},
	}
}

func (g fowl) getAddressableNilAddressInner() *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": g.getApiVersion(),
		"kind":       addressableKind,
		"metadata": map[string]interface{}{
			"namespace": testNS,
			"name":      addressableName,
		},
	}

	switch string(g) {
	case "v1alpha1":
		obj["status"] = map[string]interface{}{
			"address": map[string]interface{}{
				"hostname": nil,
			},
		}
	case "v1beta1":
		obj["status"] = map[string]interface{}{
			"address": map[string]interface{}{
				"url": nil,
			},
		}
	}

	return &unstructured.Unstructured{Object: obj}
}

func (g fowl) getAddressableRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       addressableKind,
		Name:       addressableName,
		APIVersion: g.getApiVersion(),
		Namespace:  testNS,
	}
}

func (g fowl) getUnaddressableRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       unaddressableKind,
		Name:       unaddressableName,
		APIVersion: g.getApiVersion(),
		Namespace:  testNS,
	}
}

func TestNames(t *testing.T) {
	testCases := []struct {
		Name string
		F    func() string
		Want string
	}{{
		Name: "ServiceHostName",
		F: func() string {
			return duck.ServiceHostName("foo", "namespace")
		},
		Want: "foo.namespace.svc." + system.GetClusterDomainName(),
	}}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if got := tc.F(); got != tc.Want {
				t.Errorf("want %v, got %v", tc.Want, got)
			}
		})
	}
}
