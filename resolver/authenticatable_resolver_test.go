package resolver_test

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	fakedynamicclient "knative.dev/pkg/injection/clients/dynamicclient/fake"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/resolver"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/client/injection/ducks/duck/v1/authstatus"
	"knative.dev/pkg/tracker"
)

const (
	authenticatableName       = "testsource"
	authenticatableKind       = "Source"
	authenticatableAPIVersion = "duck.knative.dev/v1"
	authenticatableResource   = "sources.duck.knative.dev"

	authenticatable2Name       = "test2-source"
	authenticatable2Kind       = "AnotherSource"
	authenticatable2APIVersion = "duck.knative.dev/v1"
	authenticatable2Resource   = "anothersources.duck.knative.dev"

	unauthenticatableName       = "testunauthenticatable"
	unauthenticatableKind       = "KResource"
	unauthenticatableAPIVersion = "duck.knative.dev/v1alpha1"

	authenticatableServiceAccountName  = "my-service-account"
	authenticatableServiceAccountName1 = "my-service-account-1"
	authenticatableServiceAccountName2 = "my-service-account-2"
	authenticatable2ServiceAccountName = "service-account-of-2nd-authenticatable"
)

func init() {
	// Add types to scheme
	duckv1alpha1.AddToScheme(scheme.Scheme)
	duckv1beta1.AddToScheme(scheme.Scheme)

	scheme.Scheme.AddKnownTypeWithName(
		schema.FromAPIVersionAndKind(unauthenticatableAPIVersion, unauthenticatableKind),
		&unstructured.Unstructured{},
	)
	scheme.Scheme.AddKnownTypeWithName(
		schema.FromAPIVersionAndKind(unauthenticatableAPIVersion, unauthenticatableKind+"List"),
		&unstructured.UnstructuredList{},
	)
	scheme.Scheme.AddKnownTypeWithName(
		schema.FromAPIVersionAndKind(authenticatableAPIVersion, authenticatableKind),
		&unstructured.Unstructured{},
	)
	scheme.Scheme.AddKnownTypeWithName(
		schema.FromAPIVersionAndKind(authenticatableAPIVersion, authenticatableKind+"List"),
		&unstructured.UnstructuredList{},
	)
}

func TestAuthenticatableResolver_AuthStatusFromObjectReference(t *testing.T) {
	tests := []struct {
		name      string
		objects   []runtime.Object
		objectRef *corev1.ObjectReference
		want      *duckv1.AuthStatus
		wantErr   string
	}{
		{
			name:    "nil everything",
			wantErr: "ref is nil",
		}, {
			name: "Valid authenticatable",
			objects: []runtime.Object{
				getAuthenticatable(),
			},
			objectRef: authenticatableRef(),
			want: &duckv1.AuthStatus{
				ServiceAccountName: ptr.String(authenticatableServiceAccountName),
			},
		}, {
			name: "Valid authenticatable in multiple objects",
			objects: []runtime.Object{
				getUnauthenticatable(),
				getAuthenticatable(),
				getAuthenticatable2(),
			},
			objectRef: authenticatable2Ref(),
			want: &duckv1.AuthStatus{
				ServiceAccountName: ptr.String(authenticatable2ServiceAccountName),
			},
		}, {
			name: "Valid authenticatable multiple SAs",
			objects: []runtime.Object{
				getAuthenticatableWithMultipleSAs(),
			},
			objectRef: authenticatableRef(),
			want: &duckv1.AuthStatus{
				ServiceAccountNames: []string{
					authenticatableServiceAccountName1,
					authenticatableServiceAccountName2,
				},
			},
		}, {
			name: "Unauthenticatable",
			objects: []runtime.Object{
				getUnauthenticatable(),
			},
			objectRef: unauthenticatableRef(),
			wantErr:   fmt.Sprintf(".status.auth is missing in object %s/%s", testNS, unauthenticatableName),
		}, {
			name: "Authenticatable not found",
			objects: []runtime.Object{
				getUnauthenticatable(),
			},
			objectRef: authenticatableRef(),
			wantErr:   fmt.Sprintf("failed to get authenticatable %s/%s: failed to get object %s/%s: %s %q not found", testNS, authenticatableName, testNS, authenticatableName, authenticatableResource, authenticatableName),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, tt.objects...)
			ctx = authstatus.WithDuck(ctx)
			r := resolver.NewAuthenticatableResolverFromTracker(ctx, tracker.New(func(types.NamespacedName) {}, 0))

			// Run it twice since this should be idempotent. AuthenticatableResolver should
			// not modify the cache's copy.
			_, _ = r.AuthStatusFromObjectReference(tt.objectRef, getAuthenticatable())
			authStatus, gotErr := r.AuthStatusFromObjectReference(tt.objectRef, getAuthenticatable())

			if gotErr != nil {
				if tt.wantErr != "" {
					if got, want := gotErr.Error(), tt.wantErr; got != want {
						t.Errorf("Unexpected error (-want, +got) =\n%s", cmp.Diff(want, got))
					}
				} else {
					t.Error("Unexpected error:", gotErr)
				}
				return
			}

			if got, want := authStatus, tt.want; !cmp.Equal(got, want) {
				t.Errorf("Unexpected object (-want, +got) =\n%s", cmp.Diff(got, want))
			}
		})
	}
}

func getAuthenticatable() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": authenticatableAPIVersion,
			"kind":       authenticatableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      authenticatableName,
			},
			"status": map[string]interface{}{
				"auth": map[string]interface{}{
					"serviceAccountName": authenticatableServiceAccountName,
				},
			},
		},
	}
}

func getAuthenticatable2() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": authenticatable2APIVersion,
			"kind":       authenticatable2Kind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      authenticatable2Name,
			},
			"status": map[string]interface{}{
				"auth": map[string]interface{}{
					"serviceAccountName": authenticatable2ServiceAccountName,
				},
			},
		},
	}
}

func getAuthenticatableWithMultipleSAs() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": authenticatableAPIVersion,
			"kind":       authenticatableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      authenticatableName,
			},
			"status": map[string]interface{}{
				"auth": map[string]interface{}{
					"serviceAccountNames": []interface{}{
						authenticatableServiceAccountName1,
						authenticatableServiceAccountName2,
					},
				},
			},
		},
	}
}

func getUnauthenticatable() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": unauthenticatableAPIVersion,
			"kind":       unauthenticatableKind,
			"metadata": map[string]interface{}{
				"namespace": testNS,
				"name":      unauthenticatableName,
			},
			"status": map[string]interface{}{
				"something": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
	}
}

func authenticatableRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       authenticatableKind,
		Name:       authenticatableName,
		APIVersion: authenticatableAPIVersion,
		Namespace:  testNS,
	}
}

func authenticatable2Ref() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       authenticatable2Kind,
		Name:       authenticatable2Name,
		APIVersion: authenticatable2APIVersion,
		Namespace:  testNS,
	}
}

func unauthenticatableRef() *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       unauthenticatableKind,
		Name:       unauthenticatableName,
		APIVersion: unauthenticatableAPIVersion,
		Namespace:  testNS,
	}
}
