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
	"errors"
	"fmt"
	"testing"

	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/service/fake"

	"github.com/google/go-cmp/cmp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kubeinformers "k8s.io/client-go/informers"
	fakekubeclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/client/injection/ducks/duck/v1/addressable"
	"knative.dev/pkg/client/injection/kube/informers/core/v1/service"
	"knative.dev/pkg/injection"
	fakedynamicclient "knative.dev/pkg/injection/clients/dynamicclient/fake"
	"knative.dev/pkg/resolver"
	"knative.dev/pkg/tracker"
)

const (
	addressableDNS                           = "http://addressable.sink.svc.cluster.local"
	addressableDNSWithPathAndTrailingSlash   = "http://addressable.sink.svc.cluster.local/bar/"
	addressableDNSWithPathAndNoTrailingSlash = "http://addressable.sink.svc.cluster.local/bar"

	addressableName       = "testsink"
	addressableKind       = "Sink"
	addressableAPIVersion = "duck.knative.dev/v1"

	unaddressableName       = "testunaddressable"
	unaddressableKind       = "KResource"
	unaddressableAPIVersion = "duck.knative.dev/v1alpha1"
	unaddressableResource   = "kresources.duck.knative.dev"

	testNS = "testnamespace"

	targetPortName = "target-port"
)

func init() {
	// Add types to scheme
	duckv1alpha1.AddToScheme(scheme.Scheme)
	duckv1beta1.AddToScheme(scheme.Scheme)

	scheme.Scheme.AddKnownTypeWithName(
		schema.FromAPIVersionAndKind(unaddressableAPIVersion, unaddressableKind),
		&unstructured.Unstructured{},
	)
	scheme.Scheme.AddKnownTypeWithName(
		schema.FromAPIVersionAndKind(unaddressableAPIVersion, unaddressableKind+"List"),
		&unstructured.UnstructuredList{},
	)
	scheme.Scheme.AddKnownTypeWithName(
		schema.FromAPIVersionAndKind(addressableAPIVersion, addressableKind),
		&unstructured.Unstructured{},
	)
	scheme.Scheme.AddKnownTypeWithName(
		schema.FromAPIVersionAndKind(addressableAPIVersion, addressableKind+"List"),
		&unstructured.UnstructuredList{},
	)
}

func TestGetURIDestinationV1Beta1(t *testing.T) {
	tests := map[string]struct {
		objects []runtime.Object
		dest    duckv1beta1.Destination
		wantURI string
		wantErr string
	}{"nil everything": {
		wantErr: "destination missing Ref, [apiVersion, kind, name] and URI, expected at least one",
	}, "Happy URI with path": {
		dest: duckv1beta1.Destination{
			URI: &apis.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/foo",
			},
		},
		wantURI: "http://example.com/foo",
	}, "URI is not absolute URL": {
		dest: duckv1beta1.Destination{
			URI: &apis.URL{
				Host: "example.com",
			},
		},
		wantErr: fmt.Sprintf("URI is not absolute (both scheme and host should be non-empty): %q", "//example.com"),
	}, "URI with no host": {
		dest: duckv1beta1.Destination{
			URI: &apis.URL{
				Scheme: "http",
			},
		},
		wantErr: fmt.Sprintf("URI is not absolute (both scheme and host should be non-empty): %q", "http:"),
	},
		"Ref and [apiVersion, kind, name] both exists": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1beta1.Destination{Ref: addressableRef(),
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS,
			},
			wantErr: "ref and [apiVersion, kind, name] can't be both present",
		},
		"happy ref": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest:    duckv1beta1.Destination{Ref: addressableRef()},
			wantURI: addressableDNS,
		}, "ref with relative uri": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1beta1.Destination{
				Ref: addressableRef(),
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref with relative URI without leading slash": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1beta1.Destination{
				Ref: addressableRef(),
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref ends with path and trailing slash and relative URI without leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndTrailingSlash(),
			},
			dest: duckv1beta1.Destination{
				Ref: addressableRef(),
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNSWithPathAndTrailingSlash + "foo",
		}, "ref ends with path and trailing slash and relative URI with leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndTrailingSlash(),
			},
			dest: duckv1beta1.Destination{
				Ref: addressableRef(),
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref ends with path and no trailing slash and relative URI without leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndNoTrailingSlash(),
			},
			dest: duckv1beta1.Destination{
				Ref: addressableRef(),
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref ends with path and no trailing slash and relative URI with leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndNoTrailingSlash(),
			},
			dest: duckv1beta1.Destination{
				Ref: addressableRef(),
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref with URI which is absolute URL": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1beta1.Destination{
				Ref: addressableRef(),
				URI: &apis.URL{
					Scheme: "http",
					Host:   "example.com",
					Path:   "/foo",
				},
			},
			wantErr: "absolute URI is not allowed when Ref or [apiVersion, kind, name] exists",
		},
		"happy [apiVersion, kind, name]": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1beta1.Destination{
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS},
			wantURI: addressableDNS,
		},
		"[apiVersion, kind, name] with relative uri": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1beta1.Destination{
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS,
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "[apiVersion, kind, name] with relative URI without leading slash": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1beta1.Destination{
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS,
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "[apiVersion, kind, name] ends with path and trailing slash and relative URI without leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndTrailingSlash(),
			},
			dest: duckv1beta1.Destination{
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS,
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNSWithPathAndTrailingSlash + "foo",
		}, "[apiVersion, kind, name] ends with path and trailing slash and relative URI with leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndTrailingSlash(),
			},
			dest: duckv1beta1.Destination{
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS,
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "[apiVersion, kind, name] ends with path and no trailing slash and relative URI without leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndNoTrailingSlash(),
			},
			dest: duckv1beta1.Destination{
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS,
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "[apiVersion, kind, name] ends with path and no trailing slash and relative URI with leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndNoTrailingSlash(),
			},
			dest: duckv1beta1.Destination{
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS,
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "[apiVersion, kind, name] with URI which is absolute URL": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1beta1.Destination{
				DeprecatedKind:       addressableKind,
				DeprecatedName:       addressableName,
				DeprecatedAPIVersion: addressableAPIVersion,
				DeprecatedNamespace:  testNS,
				URI: &apis.URL{
					Scheme: "http",
					Host:   "example.com",
					Path:   "/foo",
				},
			},
			wantErr: "absolute URI is not allowed when Ref or [apiVersion, kind, name] exists",
		},
		"nil url": {
			objects: []runtime.Object{
				addressableNilURL(),
			},
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("URL missing in address of %+v", unaddressableRef()),
		},
		"nil address": {
			objects: []runtime.Object{
				addressableNilAddress(),
			},
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("address not set for %+v", unaddressableRef()),
		}, "missing host": {
			objects: []runtime.Object{
				addressableNoHostURL(),
			},
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("hostname missing in address of %+v", unaddressableRef()),
		}, "missing status": {
			objects: []runtime.Object{
				addressableNoStatus(),
			},
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("address not set for %+v", unaddressableRef()),
		}, "notFound": {
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("failed to get object %s/%s: %s %q not found", testNS, unaddressableName, unaddressableResource, unaddressableName),
		}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx, _ := injection.Fake.SetupInformers(context.Background(), &rest.Config{})
			ctx, _ = fakedynamicclient.With(ctx, scheme.Scheme, tc.objects...)
			ctx = addressable.WithDuck(ctx)
			r := resolver.NewURIResolverFromTracker(ctx, tracker.New(func(types.NamespacedName) {}, 0))

			// Run it twice since this should be idempotent. URI Resolver should
			// not modify the cache's copy.
			_, _ = r.URIFromDestination(ctx, tc.dest, getAddressable())
			uri, gotErr := r.URIFromDestination(ctx, tc.dest, getAddressable())

			if gotErr != nil {
				if tc.wantErr != "" {
					if got, want := gotErr.Error(), tc.wantErr; got != want {
						t.Errorf("Unexpected error (-want, +got) =\n%s", cmp.Diff(want, got))
					}
				} else {
					t.Error("Unexpected error:", gotErr)
				}
				return
			}
			if got, want := uri, tc.wantURI; got != want {
				t.Errorf("Unexpected object (-want, +got) =\n%s", cmp.Diff(got, want))
			}
		})
	}
}

func TestGetURIDestinationV1(t *testing.T) {
	tests := map[string]struct {
		objects         []runtime.Object
		dest            duckv1.Destination
		customResolvers []resolver.RefResolverFunc
		wantURI         string
		wantErr         string
	}{"nil everything": {
		wantErr: "destination missing Ref and URI, expected at least one",
	}, "Happy URI with path": {
		dest: duckv1.Destination{
			URI: &apis.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/foo",
			},
		},
		wantURI: "http://example.com/foo",
	}, "URI is not absolute URL": {
		dest: duckv1.Destination{
			URI: &apis.URL{
				Host: "example.com",
			},
		},
		wantErr: fmt.Sprintf("URI is not absolute(both scheme and host should be non-empty): %q", "//example.com"),
	}, "URI with no host": {
		dest: duckv1.Destination{
			URI: &apis.URL{
				Scheme: "http",
			},
		},
		wantErr: fmt.Sprintf("URI is not absolute(both scheme and host should be non-empty): %q", "http:"),
	},
		"happy ref": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest:    duckv1.Destination{Ref: addressableKnativeRef()},
			wantURI: addressableDNS,
		}, "ref with relative uri": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1.Destination{
				Ref: addressableKnativeRef(),
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref with relative URI without leading slash": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1.Destination{
				Ref: addressableKnativeRef(),
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref ends with path and trailing slash and relative URI without leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndTrailingSlash(),
			},
			dest: duckv1.Destination{
				Ref: addressableKnativeRef(),
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNSWithPathAndTrailingSlash + "foo",
		}, "ref ends with path and trailing slash and relative URI with leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndTrailingSlash(),
			},
			dest: duckv1.Destination{
				Ref: addressableKnativeRef(),
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref ends with path and no trailing slash and relative URI without leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndNoTrailingSlash(),
			},
			dest: duckv1.Destination{
				Ref: addressableKnativeRef(),
				URI: &apis.URL{
					Path: "foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref ends with path and no trailing slash and relative URI with leading slash ": {
			objects: []runtime.Object{
				addressableWithPathAndNoTrailingSlash(),
			},
			dest: duckv1.Destination{
				Ref: addressableKnativeRef(),
				URI: &apis.URL{
					Path: "/foo",
				},
			},
			wantURI: addressableDNS + "/foo",
		}, "ref with URI which is absolute URL": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest: duckv1.Destination{
				Ref: addressableKnativeRef(),
				URI: &apis.URL{
					Scheme: "http",
					Host:   "example.com",
					Path:   "/foo",
				},
			},
			wantErr: "absolute URI is not allowed when Ref or [apiVersion, kind, name] exists",
		}, "nil url": {
			objects: []runtime.Object{
				addressableNilURL(),
			},
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("URL missing in address of %+v", unaddressableRef()),
		}, "nil address": {
			objects: []runtime.Object{
				addressableNilAddress(),
			},
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("address not set for %+v", unaddressableRef()),
		}, "missing host": {
			objects: []runtime.Object{
				addressableNoHostURL(),
			},
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("hostname missing in address of %+v", unaddressableRef()),
		}, "missing status": {
			objects: []runtime.Object{
				addressableNoStatus(),
			},
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("address not set for %+v", unaddressableRef()),
		}, "notFound": {
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("failed to get object %s/%s: %s %q not found", testNS, unaddressableName, unaddressableResource, unaddressableName),
		}, "notFound k8s service": {
			dest:    duckv1.Destination{Ref: k8sServiceRef()},
			wantErr: fmt.Sprintf("failed to get object %s/%s: services %q not found", testNS, addressableName, addressableName),
		}, "with sample resolver": {
			dest:            duckv1.Destination{Ref: k8sServiceRef()},
			customResolvers: []resolver.RefResolverFunc{sampleURIResolver},
			wantURI:         "ref://" + addressableName + ".Service.v1",
		}, "happy ref with sample resolver": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest:            duckv1.Destination{Ref: addressableKnativeRef()},
			wantURI:         addressableDNS,
			customResolvers: []resolver.RefResolverFunc{sampleURIResolver},
		}, "unaddressable with sample resolver": {
			dest:            duckv1.Destination{Ref: unaddressableKnativeRef()},
			customResolvers: []resolver.RefResolverFunc{sampleURIResolver},
			wantErr:         "cannot be referenced",
		}, "happy with two sample resolvers, first one passes": {
			dest:            duckv1.Destination{Ref: k8sServiceRef()},
			customResolvers: []resolver.RefResolverFunc{sampleURIResolver, noopURIResolver},
			wantURI:         "ref://" + addressableName + ".Service.v1",
		}, "happy with two sample resolvers, second one passes": {
			dest:            duckv1.Destination{Ref: k8sServiceRef()},
			customResolvers: []resolver.RefResolverFunc{noopURIResolver, sampleURIResolver},
			wantURI:         "ref://" + addressableName + ".Service.v1",
		}, "happy with single service port": {
			dest:    duckv1.Destination{Ref: k8sServiceRef()},
			objects: k8sServiceWithSinglePort(),
			wantURI: "http://" + addressableName + "." + testNS + ".svc.cluster.local:81",
		}, "happy with single HTTPS service port": {
			dest:    duckv1.Destination{Ref: k8sServiceRef()},
			objects: k8sServiceWithSingleHTTPSPort(),
			wantURI: "https://" + addressableName + "." + testNS + ".svc.cluster.local",
		}, "happy with multiple service ports and annotation": {
			dest:    duckv1.Destination{Ref: k8sServiceRef()},
			objects: k8sServiceWithPortAnnotation(),
			wantURI: "http://" + addressableName + "." + testNS + ".svc.cluster.local:8080",
		}, "happy with multiple service ports and http port": {
			dest:    duckv1.Destination{Ref: k8sServiceRef()},
			objects: k8sServiceWithHTTPPort(),
			wantURI: "http://" + addressableName + "." + testNS + ".svc.cluster.local",
		}, "k8s service with no destination annotation": {
			dest:    duckv1.Destination{Ref: k8sServiceRef()},
			objects: k8sServiceWithoutPortAnnotation(),
			wantErr: fmt.Sprintf("service \"%s/%s\" does not have target port annotation %q",
				testNS, addressableName, resolver.ServicePortAnnotation),
		}, "k8s service with missing destination port": {
			dest:    duckv1.Destination{Ref: k8sServiceRef()},
			objects: k8sServiceWithMissingDstPort(),
			wantErr: fmt.Sprintf("port %q not found in Service \"%s/%s\"",
				targetPortName, testNS, addressableName),
		},
	}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx := context.Background()

			kubeInformerFactory := kubeinformers.NewSharedInformerFactory(
				fakekubeclient.NewSimpleClientset(tc.objects...), 0,
			)
			svcInformerIface := kubeInformerFactory.Core().V1().Services()
			svcInformerIface.Informer() // register informer in factory
			go kubeInformerFactory.Start(ctx.Done())
			kubeInformerFactory.WaitForCacheSync(ctx.Done())

			ctx = context.WithValue(ctx, service.Key{}, svcInformerIface)
			ctx, _ = fakedynamicclient.With(ctx, scheme.Scheme, tc.objects...)
			ctx = addressable.WithDuck(ctx)
			r := resolver.NewURIResolverFromTracker(ctx, tracker.New(func(types.NamespacedName) {}, 0), tc.customResolvers...)

			// Run it twice since this should be idempotent. URI Resolver should
			// not modify the cache's copy.
			_, _ = r.URIFromDestinationV1(ctx, tc.dest, getAddressable())
			uri, gotErr := r.URIFromDestinationV1(ctx, tc.dest, getAddressable())

			if gotErr != nil {
				if tc.wantErr != "" {
					if got, want := gotErr.Error(), tc.wantErr; got != want {
						t.Errorf("Unexpected error (-want, +got) =\n%s", cmp.Diff(want, got))
					}
				} else {
					t.Error("Unexpected error:", gotErr)
				}
			}
			if got, want := uri.String(), tc.wantURI; got != want {
				t.Errorf("Unexpected object (-want, +got) =\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestURIFromObjectReferenceErrors(t *testing.T) {
	tests := map[string]struct {
		objects []runtime.Object
		ref     *corev1.ObjectReference
		wantErr string
	}{"nil": {
		wantErr: "ref is nil",
	}, "fail tracker with bad object": {
		ref: invalidObjectRef(),
		wantErr: `failed to track reference duck.knative.dev/v1, Resource=sinks -bad/testsink: invalid Reference:
Namespace: a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character (e.g. 'my-name',  or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')`,
	}, "fail get": {
		ref:     addressableRef(),
		wantErr: `failed to get object testnamespace/testsink: sinks.duck.knative.dev "testsink" not found`,
	}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx, _ := injection.Fake.SetupInformers(context.Background(), &rest.Config{})
			ctx, _ = fakedynamicclient.With(ctx, scheme.Scheme, tc.objects...)
			ctx = addressable.WithDuck(ctx)
			r := resolver.NewURIResolverFromTracker(ctx, tracker.New(func(types.NamespacedName) {}, 0))

			// Run it twice since this should be idempotent. URI Resolver should
			// not modify the cache's copy.
			_, err1 := r.URIFromObjectReference(ctx, tc.ref, getAddressable())
			_, err2 := r.URIFromObjectReference(ctx, tc.ref, getAddressable())

			if err2 == nil {
				t.Fatal("Expected failure")
			}
			if got, want := err2.Error(), tc.wantErr; got != want {
				t.Errorf("Unexpected error (-want, +got) =\n%s", cmp.Diff(want, got))
			}
			if err1.Error() != err2.Error() {
				t.Errorf("Idempotency fail: first err = %+v, second err = %+v", err1, err2)
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

func addressableWithPathAndTrailingSlash() *unstructured.Unstructured {
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
					"url": addressableDNSWithPathAndTrailingSlash,
				},
			},
		},
	}
}

func addressableWithPathAndNoTrailingSlash() *unstructured.Unstructured {
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
					"url": addressableDNSWithPathAndNoTrailingSlash,
				},
			},
		},
	}
}

func addressableNoStatus() *unstructured.Unstructured {
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

func addressableNilAddress() *unstructured.Unstructured {
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

func addressableNilURL() *unstructured.Unstructured {
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

func addressableNoHostURL() *unstructured.Unstructured {
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

func invalidObjectRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       addressableKind,
		Name:       addressableName,
		APIVersion: addressableAPIVersion,
		Namespace:  "-bad",
	}

}

func k8sServiceRef() *duckv1.KReference {
	return &duckv1.KReference{
		Kind:       "Service",
		Name:       addressableName,
		APIVersion: "v1",
		Namespace:  testNS,
	}

}

func addressableKnativeRef() *duckv1.KReference {
	return &duckv1.KReference{
		Kind:       addressableKind,
		Name:       addressableName,
		APIVersion: addressableAPIVersion,
		Namespace:  testNS,
	}
}

func addressableRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       addressableKind,
		Name:       addressableName,
		APIVersion: addressableAPIVersion,
		Namespace:  testNS,
	}
}

func unaddressableKnativeRef() *duckv1.KReference {
	return &duckv1.KReference{
		Kind:       unaddressableKind,
		Name:       unaddressableName,
		APIVersion: unaddressableAPIVersion,
		Namespace:  testNS,
	}
}

func unaddressableRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       unaddressableKind,
		Name:       unaddressableName,
		APIVersion: unaddressableAPIVersion,
		Namespace:  testNS,
	}
}

func sampleURIResolver(ctx context.Context, ref *corev1.ObjectReference) (bool, *apis.URL, error) {
	if ref.Kind == "Service" {
		parsed, err := apis.ParseURL(fmt.Sprintf("ref://%s.%s.%s", ref.Name, ref.Kind, ref.APIVersion))
		return true, parsed, err
	}
	if ref.Kind == unaddressableKind {
		return true, nil, errors.New("cannot be referenced")
	}
	return false, nil, nil
}

func noopURIResolver(ctx context.Context, ref *corev1.ObjectReference) (bool, *apis.URL, error) {
	return false, nil, nil
}

func k8sServiceWithAnnotationsAndPorts(annotations map[string]string, ports ...corev1.ServicePort) []runtime.Object {
	return []runtime.Object{
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        addressableName,
				Namespace:   testNS,
				Annotations: annotations,
			},
			Spec: corev1.ServiceSpec{
				Ports: ports,
			},
			Status: corev1.ServiceStatus{},
		},
	}
}

func k8sServiceWithPortAnnotation() []runtime.Object {
	return k8sServiceWithAnnotationsAndPorts(
		map[string]string{
			resolver.ServicePortAnnotation: targetPortName,
		},
		corev1.ServicePort{
			Name:     "http",
			Protocol: "TCP",
			Port:     80,
		},
		corev1.ServicePort{
			Name:     targetPortName,
			Protocol: "TCP",
			Port:     8080,
		})
}

func k8sServiceWithMissingDstPort() []runtime.Object {
	return k8sServiceWithAnnotationsAndPorts(
		map[string]string{
			resolver.ServicePortAnnotation: targetPortName,
		},
		corev1.ServicePort{
			Name:     "http",
			Protocol: "TCP",
			Port:     80,
		})
}

func k8sServiceWithHTTPPort() []runtime.Object {
	return k8sServiceWithAnnotationsAndPorts(
		nil,
		corev1.ServicePort{
			Name:     "https",
			Protocol: "TCP",
			Port:     443,
		},
		corev1.ServicePort{
			Name:     "http",
			Protocol: "TCP",
			Port:     80,
		})
}

func k8sServiceWithSinglePort() []runtime.Object {
	return k8sServiceWithAnnotationsAndPorts(
		nil,
		corev1.ServicePort{
			Name:     "http",
			Protocol: "TCP",
			Port:     81,
		})
}

func k8sServiceWithSingleHTTPSPort() []runtime.Object {
	return k8sServiceWithAnnotationsAndPorts(
		nil,
		corev1.ServicePort{
			Name:     "https",
			Protocol: "TCP",
			Port:     443,
		})
}

func k8sServiceWithoutPortAnnotation() []runtime.Object {
	return k8sServiceWithAnnotationsAndPorts(
		nil,
		corev1.ServicePort{
			Name:     "foo",
			Protocol: "TCP",
			Port:     81,
		},
		corev1.ServicePort{
			Name:     "bar",
			Protocol: "TCP",
			Port:     8080,
		},
	)
}
