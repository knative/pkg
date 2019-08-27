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

package resolver_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/apis"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	apisv1alpha1 "knative.dev/pkg/apis/v1alpha1"
	fakedynamicclient "knative.dev/pkg/injection/clients/dynamicclient/fake"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/resolver"
)

var (
	addressableDNS = "http://addressable.sink.svc.cluster.local"

	addressableName       = "testsink"
	addressableKind       = "Sink"
	addressableAPIVersion = "duck.knative.dev/v1alpha1"

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

func TestGetURI_ObjectReference(t *testing.T) {
	tests := map[string]struct {
		objects []runtime.Object
		dest    apisv1alpha1.Destination
		wantURI string
		wantErr error
	}{"nil everything": {
		wantErr: fmt.Errorf("destination missing ObjectReference and URI, expected exactly one"),
	}, "URI with path": {
		dest: apisv1alpha1.Destination{
			URI: &apis.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/foo",
			},
			Path: ptr.String("/bar"),
		},
		wantURI: "http://example.com/foo/bar",
	}, "URI with path without leading slash": {
		dest: apisv1alpha1.Destination{
			URI: &apis.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/foo",
			},
			Path: ptr.String("bar"),
		},
		wantURI: "http://example.com/foo/bar",
	}, "URI with garbage path": {
		dest: apisv1alpha1.Destination{
			URI: &apis.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/foo",
			},
			Path: ptr.String("////bar///"),
		},
		wantURI: "http://example.com/foo/bar",
	}, "URI with nil path": {
		dest: apisv1alpha1.Destination{
			URI: &apis.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/foo",
			},
			Path: nil,
		},
		wantURI: "http://example.com/foo",
	}, "happy objectref": {
		objects: []runtime.Object{
			getAddressable(),
		},
		dest:    apisv1alpha1.Destination{ObjectReference: getAddressableRef()},
		wantURI: addressableDNS,
	}, "object ref with path": {
		objects: []runtime.Object{
			getAddressable(),
		},
		dest: apisv1alpha1.Destination{
			ObjectReference: getAddressableRef(),
			Path:            ptr.String("/foo"),
		},
		wantURI: addressableDNS + "/foo",
	}, "object ref with path without leading slash": {
		objects: []runtime.Object{
			getAddressable(),
		},
		dest: apisv1alpha1.Destination{
			ObjectReference: getAddressableRef(),
			Path:            ptr.String("foo"),
		},
		wantURI: addressableDNS + "/foo",
	}, "nil url": {
		objects: []runtime.Object{
			getAddressableNilURL(),
		},
		dest:    apisv1alpha1.Destination{ObjectReference: getUnaddressableRef()},
		wantErr: fmt.Errorf(`url missing in address of %+v`, getUnaddressableRef()),
	}, "nil address": {
		objects: []runtime.Object{
			getAddressableNilAddress(),
		},
		dest:    apisv1alpha1.Destination{ObjectReference: getUnaddressableRef()},
		wantErr: fmt.Errorf(`address not set for %+v`, getUnaddressableRef()),
	}, "missing host": {
		objects: []runtime.Object{
			getAddressableNoHostURL(),
		},
		dest:    apisv1alpha1.Destination{ObjectReference: getUnaddressableRef()},
		wantErr: fmt.Errorf(`hostname missing in address of %+v`, getUnaddressableRef()),
	}, "missing status": {
		objects: []runtime.Object{
			getAddressableNoStatus(),
		},
		dest:    apisv1alpha1.Destination{ObjectReference: getUnaddressableRef()},
		wantErr: fmt.Errorf(`address not set for %+v`, getUnaddressableRef()),
	}, "notFound": {
		dest:    apisv1alpha1.Destination{ObjectReference: getUnaddressableRef()},
		wantErr: fmt.Errorf(`failed to get ref %+v: %s "%s" not found`, getUnaddressableRef(), unaddressableResource, unaddressableName),
	}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tc.objects...)
			r := resolver.NewURIResolver(ctx, func(string) {})

			// Run it twice since this should be idempotent. URI Resolver should
			// not modify the cache's copy.
			_, _ = r.URIFromDestination(tc.dest, getAddressable())
			uri, gotErr := r.URIFromDestination(tc.dest, getAddressable())

			if gotErr != nil {
				if tc.wantErr != nil {
					if diff := cmp.Diff(tc.wantErr.Error(), gotErr.Error()); diff != "" {
						t.Errorf("%s: unexpected error (-want, +got) = %v", n, diff)
					}
				} else {
					t.Errorf("%s: unexpected error: %v", n, gotErr.Error())
				}
			}
			if gotErr == nil {
				got := uri
				if diff := cmp.Diff(tc.wantURI, got); diff != "" {
					t.Errorf("%s: unexpected object (-want, +got) = %v", n, diff)
				}
			}
		})
	}
}

func getAddressable() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": addressableAPIVersion,
			"kind":       addressableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      addressableName,
			},
			"status": map[string]interface{}{
				"address": map[string]interface{}{
					"url": addressableDNS,
				},
			},
		},
	}
}

func getAddressableNoStatus() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": unaddressableAPIVersion,
			"kind":       unaddressableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      unaddressableName,
			},
		},
	}
}

func getAddressableNilAddress() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": unaddressableAPIVersion,
			"kind":       unaddressableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      unaddressableName,
			},
			"status": map[string]interface{}{
				"address": map[string]interface{}(nil),
			},
		},
	}
}

func getAddressableNilURL() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": unaddressableAPIVersion,
			"kind":       unaddressableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      unaddressableName,
			},
			"status": map[string]interface{}{
				"address": map[string]interface{}{
					"url": nil,
				},
			},
		},
	}
}

func getAddressableNoHostURL() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": unaddressableAPIVersion,
			"kind":       unaddressableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      unaddressableName,
			},
			"status": map[string]interface{}{
				"address": map[string]interface{}{
					"url": "http://",
				},
			},
		},
	}
}

func getAddressableRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       addressableKind,
		Name:       addressableName,
		APIVersion: addressableAPIVersion,
		Namespace:  testNS,
	}
}

func getUnaddressableRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       unaddressableKind,
		Name:       unaddressableName,
		APIVersion: unaddressableAPIVersion,
		Namespace:  testNS,
	}
}
