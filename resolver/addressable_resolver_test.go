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

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/client/injection/ducks/duck/v1/addressable"
	fakedynamicclient "knative.dev/pkg/injection/clients/dynamicclient/fake"
	"knative.dev/pkg/resolver"
	"knative.dev/pkg/tracker"
)

const (
	addressableDNS                           = "http://addressable.sink.svc.cluster.local"
	addressableDNS1                          = "http://addressable.sink1.svc.cluster.local"
	addressableDNS2                          = "http://addressable.sink2.svc.cluster.local"
	addressableDNS3                          = "http://addressable.sink3.svc.cluster.local"
	addressableDNSWithPathAndTrailingSlash   = "http://addressable.sink.svc.cluster.local/bar/"
	addressableDNSWithPathAndNoTrailingSlash = "http://addressable.sink.svc.cluster.local/bar"

	addressableName       = "testsink"
	addressableName1      = "testsink1"
	addressableName2      = "testsink2"
	addressableName3      = "testsink3"
	addressableKind       = "Sink"
	addressableAPIVersion = "duck.knative.dev/v1"

	unaddressableName       = "testunaddressable"
	unaddressableKind       = "KResource"
	unaddressableAPIVersion = "duck.knative.dev/v1alpha1"
	unaddressableResource   = "kresources.duck.knative.dev"

	testNS = "testnamespace"

	CACert = `-----BEGIN CERTIFICATE-----
MIICNDCCAaECEAKtZn5ORf5eV288mBle3cAwDQYJKoZIhvcNAQECBQAwXzELMAkG
A1UEBhMCVVMxIDAeBgNVBAoTF1JTQSBEYXRhIFNlY3VyaXR5LCBJbmMuMS4wLAYD
VQQLEyVTZWN1cmUgU2VydmVyIENlcnRpZmljYXRpb24gQXV0aG9yaXR5MB4XDTk0
MTEwOTAwMDAwMFoXDTEwMDEwNzIzNTk1OVowXzELMAkGA1UEBhMCVVMxIDAeBgNV
BAoTF1JTQSBEYXRhIFNlY3VyaXR5LCBJbmMuMS4wLAYDVQQLEyVTZWN1cmUgU2Vy
dmVyIENlcnRpZmljYXRpb24gQXV0aG9yaXR5MIGbMA0GCSqGSIb3DQEBAQUAA4GJ
ADCBhQJ+AJLOesGugz5aqomDV6wlAXYMra6OLDfO6zV4ZFQD5YRAUcm/jwjiioII
0haGN1XpsSECrXZogZoFokvJSyVmIlZsiAeP94FZbYQHZXATcXY+m3dM41CJVphI
uR2nKRoTLkoRWZweFdVJVCxzOmmCsZc5nG1wZ0jl3S3WyB57AgMBAAEwDQYJKoZI
hvcNAQECBQADfgBl3X7hsuyw4jrg7HFGmhkRuNPHoLQDQCYCPgmc4RKz0Vr2N6W3
YQO2WxZpO8ZECAyIUwxrl0nHPjXcbLm7qt9cuzovk2C2qUtN8iD3zV9/ZHuO3ABc
1/p3yjkWWW8O6tO1g39NTUJWdrTJXwT4OPjr0l91X817/OWOgHz8UA==
-----END CERTIFICATE-----`
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
		wantErr: "destination missing Ref and URI, expected at least one",
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
			wantErr: fmt.Sprintf("URL missing in address of %+v", unaddressableKnativeRef()),
		},
		"nil address": {
			objects: []runtime.Object{
				addressableNilAddress(),
			},
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("address not set for %+v", unaddressableKnativeRef()),
		}, "missing host": {
			objects: []runtime.Object{
				addressableNoHostURL(),
			},
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("hostname missing in address of %+v", unaddressableKnativeRef()),
		}, "missing status": {
			objects: []runtime.Object{
				addressableNoStatus(),
			},
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("address not set for %+v", unaddressableKnativeRef()),
		}, "notFound": {
			dest:    duckv1beta1.Destination{Ref: unaddressableRef()},
			wantErr: fmt.Sprintf("failed to get object %s/%s: %s %q not found", testNS, unaddressableName, unaddressableResource, unaddressableName),
		}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tc.objects...)
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
		wantErr: fmt.Sprintf("URI is not absolute (both scheme and host should be non-empty): %q", "//example.com"),
	}, "URI with no host": {
		dest: duckv1.Destination{
			URI: &apis.URL{
				Scheme: "http",
			},
		},
		wantErr: fmt.Sprintf("URI is not absolute (both scheme and host should be non-empty): %q", "http:"),
	},
		"happy ref": {
			objects: []runtime.Object{
				getAddressable(),
			},
			dest:    duckv1.Destination{Ref: addressableKnativeRef()},
			wantURI: addressableDNS,
		}, "happy ref to k8s service": {
			objects: []runtime.Object{
				getAddressableFromKRef(k8sServiceRef()),
			},
			dest:    duckv1.Destination{Ref: k8sServiceRef()},
			wantURI: "http://testsink.testnamespace.svc.cluster.local",
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
		},
		"nil url": {
			objects: []runtime.Object{
				addressableNilURL(),
			},
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("URL missing in address of %+v", unaddressableKnativeRef()),
		},
		"nil address": {
			objects: []runtime.Object{
				addressableNilAddress(),
			},
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("address not set for %+v", unaddressableKnativeRef()),
		}, "missing host": {
			objects: []runtime.Object{
				addressableNoHostURL(),
			},
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("hostname missing in address of %+v", unaddressableKnativeRef()),
		}, "missing status": {
			objects: []runtime.Object{
				addressableNoStatus(),
			},
			dest:    duckv1.Destination{Ref: unaddressableKnativeRef()},
			wantErr: fmt.Sprintf("address not set for %+v", unaddressableKnativeRef()),
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
		}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tc.objects...)
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
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tc.objects...)
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

func TestAddressableFromDestinationV1(t *testing.T) {
	tests := map[string]struct {
		objects         []runtime.Object
		dest            duckv1.Destination
		addr            *unstructured.Unstructured
		customResolvers []resolver.RefResolverFunc
		wantAddress     string
		wantErr         string
	}{"address and addresses are not set": {
		dest: duckv1.Destination{
			Ref: addressableKnativeRef(),
			URI: &apis.URL{
				Path: "/foo",
			},
		},
		objects: []runtime.Object{
			addressableNoAddresses(),
		},
		addr:    addressableNoAddresses(),
		wantErr: fmt.Sprintf("address not set for %+v", addressableKnativeRef()),
	}, "ref.address is set on destination and is present in target addressable addresses": {
		dest: duckv1.Destination{
			Ref: addressableKnativeRefWithAddress(),
			URI: &apis.URL{
				Path: "/foo",
			},
		},
		objects: []runtime.Object{
			addressableWithAddresses(),
		},
		wantAddress: *addressableKnativeRefWithAddress().Address,
		addr:        addressableWithAddresses(),
	}, "ref.address is set on destination and is NOT present in target addressable addresses": {
		dest: duckv1.Destination{
			Ref: addressableKnativeRefWithAddress(),
			URI: &apis.URL{
				Path: "/foo",
			},
		},
		objects: []runtime.Object{
			addressableWithDifferentAddresses(),
		},
		addr:    addressableWithDifferentAddresses(),
		wantErr: fmt.Sprintf("address with name %q not found for %+v", *addressableKnativeRefWithAddress().Address, addressableKnativeRefWithAddress()),
	}, "address and addresses are set but no destination.ref.address is set": {
		dest: duckv1.Destination{
			Ref: addressableKnativeRef(),
			URI: &apis.URL{
				Path: "/foo",
			},
		},
		objects: []runtime.Object{
			addressableWithAddresses(),
		},
		addr:        addressableWithAddresses(),
		wantAddress: addressableName,
	}, "only address is set and no destination.ref.address is set": {
		dest: duckv1.Destination{
			Ref: addressableKnativeRef(),
			URI: &apis.URL{
				Path: "/foo",
			},
		},
		objects: []runtime.Object{
			addressableWithAddressOnly(),
		},
		addr:        addressableWithAddressOnly(),
		wantAddress: addressableName,
	}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tc.objects...)
			ctx = addressable.WithDuck(ctx)
			r := resolver.NewURIResolverFromTracker(ctx, tracker.New(func(types.NamespacedName) {}, 0), tc.customResolvers...)

			// Run it twice since this should be idempotent. URI Resolver should
			// not modify the cache's copy.
			_, _ = r.AddressableFromDestinationV1(ctx, tc.dest, tc.addr)
			addr, gotErr := r.AddressableFromDestinationV1(ctx, tc.dest, tc.addr)

			if gotErr != nil {
				if tc.wantErr != "" {
					if got, want := gotErr.Error(), tc.wantErr; got != want {
						t.Errorf("Unexpected error (-want, +got) =\n%s", cmp.Diff(want, got))
					}
				} else {
					t.Error("Unexpected error:", gotErr)
				}
			}
			if got, want := *addr.Name, tc.wantAddress; got != want {
				t.Errorf("Unexpected object (-want, +got) =\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestAddressableFromDestinationScheme(t *testing.T) {
	cert := CACert
	tests := map[string]struct {
		objects         []runtime.Object
		dest            duckv1.Destination
		addr            *unstructured.Unstructured
		customResolvers []resolver.RefResolverFunc
		wantScheme      string
		wantErr         string
	}{"destination is a raw k8s serving and there is CACerts set ": {
		dest: duckv1.Destination{
			Ref: k8sServiceRef(),
			URI: &apis.URL{
				Path: "/foo",
			},
			CACerts: &cert,
		},
		objects: []runtime.Object{
			addressableWithAddresses(),
		},
		wantScheme: "https",
		addr:       addressableWithAddresses(),
	}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tc.objects...)
			ctx = addressable.WithDuck(ctx)
			r := resolver.NewURIResolverFromTracker(ctx, tracker.New(func(types.NamespacedName) {}, 0), tc.customResolvers...)

			// Run it twice since this should be idempotent. URI Resolver should
			// not modify the cache's copy.
			_, _ = r.AddressableFromDestinationV1(ctx, tc.dest, tc.addr)
			addr, gotErr := r.AddressableFromDestinationV1(ctx, tc.dest, tc.addr)

			if gotErr != nil {
				if tc.wantErr != "" {
					if got, want := gotErr.Error(), tc.wantErr; got != want {
						t.Errorf("Unexpected error (-want, +got) =\n%s", cmp.Diff(want, got))
					}
				} else {
					t.Error("Unexpected error:", gotErr)
				}
			}
			if got, want := addr.URL.Scheme, tc.wantScheme; got != want {
				t.Errorf("Unexpected object (-want, +got) =\n%s", cmp.Diff(want, got))
			}
		})
	}
}

func TestAddressableFromDestinationV1CACerts(t *testing.T) {
	certDestination := "CA CERT FOR DESTINATION"
	tests := map[string]struct {
		objects         []runtime.Object
		dest            duckv1.Destination
		addr            *unstructured.Unstructured
		customResolvers []resolver.RefResolverFunc
		wantCert        string
		wantErr         string
	}{"CACerts is set on the target addressable": {
		dest: duckv1.Destination{
			Ref: addressableKnativeRef(),
			URI: &apis.URL{
				Path: "/foo",
			},
		},
		objects: []runtime.Object{
			addressableWithCACert(),
		},
		addr:     addressableWithCACert(),
		wantErr:  fmt.Sprintf("address with name %q not found for %+v", *addressableKnativeRefWithAddress().Address, addressableKnativeRefWithAddress()),
		wantCert: CACert,
	}, "CACerts is not set on the target addressable but it is set on the destination": {
		dest: duckv1.Destination{
			Ref: addressableKnativeRef(),
			URI: &apis.URL{
				Path: "/foo",
			},
			CACerts: &certDestination,
		},
		objects: []runtime.Object{
			addressableWithAddresses(),
		},
		addr:     addressableWithAddresses(),
		wantCert: certDestination,
	}, "CACerts is set on the target addressable and it is set on the destination": {
		dest: duckv1.Destination{
			Ref: addressableKnativeRef(),
			URI: &apis.URL{
				Path: "/foo",
			},
			CACerts: &certDestination,
		},
		objects: []runtime.Object{
			addressableWithCACert(),
		},
		addr:     addressableWithCACert(),
		wantCert: certDestination,
	}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tc.objects...)
			ctx = addressable.WithDuck(ctx)
			r := resolver.NewURIResolverFromTracker(ctx, tracker.New(func(types.NamespacedName) {}, 0), tc.customResolvers...)

			// Run it twice since this should be idempotent. URI Resolver should
			// not modify the cache's copy.
			_, _ = r.AddressableFromDestinationV1(ctx, tc.dest, tc.addr)
			addr, gotErr := r.AddressableFromDestinationV1(ctx, tc.dest, tc.addr)

			if gotErr != nil {
				if tc.wantErr != "" {
					if got, want := gotErr.Error(), tc.wantErr; got != want {
						t.Errorf("Unexpected error (-want, +got) =\n%s", cmp.Diff(want, got))
					}
				} else {
					t.Error("Unexpected error:", gotErr)
				}
			}
			if got, want := *addr.CACerts, tc.wantCert; got != want {
				t.Errorf("Unexpected object (-want, +got) =\n%s", cmp.Diff(want, got))
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

func addressableKnativeRefWithAddress() *duckv1.KReference {
	address := addressableName
	return &duckv1.KReference{
		Kind:       addressableKind,
		Name:       addressableName,
		APIVersion: addressableAPIVersion,
		Namespace:  testNS,
		Address:    &address,
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

func getAddressableFromKRef(ref *duckv1.KReference) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": ref.APIVersion,
			"kind":       ref.Kind,
			"metadata": map[string]interface{}{
				"namespace": ref.Namespace,
				"name":      ref.Name,
			},
			"status": map[string]interface{}{
				"address": map[string]interface{}{
					"url":  addressableDNS,
					"name": addressableDNS,
				},
			},
		},
	}
}

func addressableNoAddresses() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": addressableAPIVersion,
			"kind":       addressableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      addressableName,
			},
			"status": map[string]interface{}{
				"address":   map[string]interface{}(nil),
				"addresses": []map[string]interface{}{},
			},
		},
	}
}

func addressableWithAddresses() *unstructured.Unstructured {
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
					"url":  addressableDNS,
					"name": addressableName,
				},
				"addresses": []map[string]interface{}{{
					"url":  addressableDNS,
					"name": addressableName,
				}, {
					"url":  addressableDNS1,
					"name": addressableName1,
				}, {
					"url":  addressableDNS2,
					"name": addressableName2,
				}, {
					"url":  addressableDNS3,
					"name": addressableName3,
				}},
			},
		},
	}
}

func addressableWithAddressOnly() *unstructured.Unstructured {
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
					"url":  addressableDNS,
					"name": addressableName,
				},
				"addresses": []map[string]interface{}{},
			},
		},
	}
}

func addressableWithDifferentAddresses() *unstructured.Unstructured {
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
					"url":  addressableDNS,
					"name": addressableName,
				},
				"addresses": []map[string]interface{}{{
					"url":  addressableDNS1,
					"name": addressableName1,
				}, {
					"url":  addressableDNS2,
					"name": addressableName2,
				}},
			},
		},
	}
}

func addressableWithCACert() *unstructured.Unstructured {
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
					"url":     addressableDNS,
					"name":    addressableName,
					"CACerts": CACert,
				},
			},
		},
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
