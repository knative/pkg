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

package v1alpha1

import (
	"context"
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	"knative.dev/eventing/pkg/reconciler/names"
	"knative.dev/pkg/apis"
	pkgapisduck "knative.dev/pkg/apis/duck"
	duckv1alpha1 "knative.dev/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/tracker"

	"knative.dev/pkg/injection/clients/dynamicclient"
)

// AddressableResolver resolves a Destination into a URI.
// This is a helper component for resources that references other resources' addresses.
type AddressableResolver struct {
	tracker         tracker.Interface
	informerFactory pkgapisduck.InformerFactory
}

// NewAddressableResolver creates and initializes a new AddressableResolver.
func NewAddressableResolver(ctx context.Context, callback func(string)) *AddressableResolver {
	ret := &AddressableResolver{}

	ret.tracker = tracker.New(callback, controller.GetTrackerLease(ctx))
	ret.informerFactory = &pkgapisduck.CachedInformerFactory{
		Delegate: &pkgapisduck.EnqueueInformerFactory{
			Delegate: &pkgapisduck.TypedInformerFactory{
				Client:       dynamicclient.Get(ctx),
				Type:         &duckv1alpha1.AddressableType{},
				ResyncPeriod: controller.GetResyncPeriod(ctx),
				StopChannel:  ctx.Done(),
			},
			EventHandler: controller.HandleAll(ret.tracker.OnChanged),
		},
	}

	return ret
}

// GetURI resolves a Destination into a URI string.
func (r *AddressableResolver) GetURI(dest Destination, parent interface{}) (string, error) {
	// Prefer resolved object reference + path, then try URI + path, honoring the Destination documentation
	if dest.ObjectReference != nil {
		url, err := r.resolveObjectReference(dest.ObjectReference, parent)
		if err != nil {
			return "", err
		}
		return extendPath(url, dest.Path).String(), nil
	} else if dest.URI != nil {
		return extendPath(dest.URI, dest.Path).String(), nil
	} else {
		return "", fmt.Errorf("destination missing ObjectReference and URI, expected at least one")
	}
}

func (r *AddressableResolver) resolveObjectReference(ref *corev1.ObjectReference, parent interface{}) (*apis.URL, error) {
	if ref == nil {
		return nil, fmt.Errorf("ref is nil")
	}

	if err := r.tracker.Track(*ref, parent); err != nil {
		return nil, fmt.Errorf("failed to track %+v: %+v", ref, err)
	}

	// K8s Services are special cased. They can be called, even though they do not satisfy the
	// Callable interface.
	if ref.APIVersion == "v1" && ref.Kind == "Service" {
		url := &apis.URL{
			Scheme: "http",
			Host:   names.ServiceHostName(ref.Name, ref.Namespace),
			Path:   "/",
		}
		return url, nil
	}

	gvr, _ := meta.UnsafeGuessKindToResource(ref.GroupVersionKind())
	_, lister, err := r.informerFactory.Get(gvr)
	if err != nil {
		return nil, fmt.Errorf("failed to get lister for %+v: %+v", gvr, err)
	}

	obj, err := lister.ByNamespace(ref.Namespace).Get(ref.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get ref %+v: %+v", ref, err)
	}

	addressable, ok := obj.(*duckv1alpha1.AddressableType)
	if !ok {
		return nil, fmt.Errorf("%+v is not an AddressableType", ref)
	}
	if addressable.Status.Address == nil {
		return nil, fmt.Errorf("address not set for %+v", ref)
	}
	url := addressable.Status.Address.GetURL()
	if url.Host == "" {
		return nil, fmt.Errorf("missing hostname in address of %+v", ref)
	}
	return &url, nil
}

// extendPath is a convenience wrapper to add a destination's path.
func extendPath(url *apis.URL, extrapath *string) *apis.URL {
	if extrapath == nil {
		return url
	}

	url.Path = path.Join(url.Path, *extrapath)
	return url
}
