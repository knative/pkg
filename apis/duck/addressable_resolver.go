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

package duck

import (
	"context"
	"fmt"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	duckv1beta1 "github.com/knative/pkg/apis/duck/v1beta1"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/system"
	"github.com/knative/pkg/tracker"

	"github.com/knative/pkg/injection/clients/dynamicclient"
)

// AddressableResolver is a helper for Sources. It triggers
// reconciliation on creation, updates to or deletion of the source's sink.
type AddressableResolver struct {
	tracker             tracker.Interface
	addressableInformer InformerFactory
}

// NewAddressableResolver creates and initializes a new AddressableResolver
func NewAddressableResolver(ctx context.Context, callback func(string)) *AddressableResolver {
	ret := &AddressableResolver{}

	ret.tracker = tracker.New(callback, controller.GetTrackerLease(ctx))
	ret.addressableInformer = &CachedInformerFactory{
		Delegate: &EnqueueInformerFactory{
			Delegate: &TypedInformerFactory{
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

// Resolve tracks the given addressable reference and if possible, retrieves
// the addressables URI.
func (r *AddressableResolver) Resolve(addressableRef *corev1.ObjectReference, observer interface{}, observerDesc string) (string, error) {
	if addressableRef == nil {
		return "", fmt.Errorf("addressable ref is nil")
	}

	if err := r.tracker.Track(*addressableRef, observer); err != nil {
		return "", fmt.Errorf("failed to tracking addressable %q for observer %q: %+v", addressableRef.String(), observerDesc, err)
	}

	// K8s Services are special cased. They can be called, even though they do not satisfy the
	// Callable interface.
	if addressableRef.APIVersion == "v1" && addressableRef.Kind == "Service" {
		return DomainToURL(ServiceHostName(addressableRef.Name, addressableRef.Namespace)), nil
	}

	gvr, _ := meta.UnsafeGuessKindToResource(addressableRef.GroupVersionKind())
	_, lister, err := r.addressableInformer.Get(gvr)
	if err != nil {
		return "", fmt.Errorf("failed to get lister for an addressable resource '%+v': %+v", gvr, err)
	}

	addressableObj, err := lister.ByNamespace(addressableRef.Namespace).Get(addressableRef.Name)
	if err != nil {
		return "", fmt.Errorf("failed to fetch addressable %q for observer %q: %v", refName(addressableRef), observerDesc, err)
	}

	var resolved apis.URL

	switch obj := addressableObj.(type) {
	case *duckv1alpha1.AddressableType:
		if obj.Status.Address == nil {
			return "", fmt.Errorf("object %q does not contain address", refName(addressableRef))
		}
		resolved = obj.Status.Address.GetURL()

	case *duckv1beta1.AddressableType:
		if obj.Status.Address == nil || obj.Status.Address.URL == nil {
			return "", fmt.Errorf("object %q does not contain address", refName(addressableRef))
		}
		resolved = *obj.Status.Address.URL

	default:
		return "", fmt.Errorf("object %q is of an unknown kind", refName(addressableRef))
	}

	if resolved.Host == "" {
		return "", fmt.Errorf("object %q contains an empty hostname", refName(addressableRef))
	}

	return resolved.String(), nil
}

func refName(ref *corev1.ObjectReference) string {
	return fmt.Sprintf("%s/%s %s.%s", ref.Name, ref.Namespace, ref.Kind, ref.APIVersion)
}

// ServiceHostName creates the hostname for a Kubernetes Service.
func ServiceHostName(serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.%s", serviceName, namespace, system.GetClusterDomainName())
}

// DomainToURL converts a domain into an HTTP URL.
func DomainToURL(domain string) string {
	u := url.URL{
		Scheme: "http",
		Host:   domain,
		Path:   "/",
	}
	return u.String()
}
