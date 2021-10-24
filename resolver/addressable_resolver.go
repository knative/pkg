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

package resolver

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"knative.dev/pkg/apis"
	pkgapisduck "knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/client/injection/ducks/duck/v1/addressable"
	"knative.dev/pkg/client/injection/kube/informers/core/v1/service"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/network"
	"knative.dev/pkg/tracker"
)

// ServicePortAnnotation specifies which port should be used as a target in v1.Service destination.
const ServicePortAnnotation = "knative.dev/destination-port"

// RefResolverFunc resolves ObjectReferences into a URI.
// It returns true when it handled the reference, in which case it also returns the resolved URI or an error.
type RefResolverFunc func(ctx context.Context, ref *corev1.ObjectReference) (bool, *apis.URL, error)

// URIResolver resolves Destinations and ObjectReferences into a URI.
type URIResolver struct {
	tracker       tracker.Interface
	listerFactory func(schema.GroupVersionResource) (cache.GenericLister, error)
	serviceLister corev1listers.ServiceLister
	resolvers     []RefResolverFunc
}

// NewURIResolverFromTracker constructs a new URIResolver with context, a tracker and an optional list of custom resolvers.
func NewURIResolverFromTracker(ctx context.Context, t tracker.Interface, resolvers ...RefResolverFunc) *URIResolver {
	ret := &URIResolver{
		tracker:       t,
		resolvers:     resolvers,
		serviceLister: service.Get(ctx).Lister(),
	}

	informerFactory := &pkgapisduck.CachedInformerFactory{
		Delegate: &pkgapisduck.EnqueueInformerFactory{
			Delegate:     addressable.Get(ctx),
			EventHandler: controller.HandleAll(ret.tracker.OnChanged),
		},
	}

	ret.listerFactory = func(gvr schema.GroupVersionResource) (cache.GenericLister, error) {
		_, l, err := informerFactory.Get(ctx, gvr)
		return l, err
	}

	return ret
}

// URIFromDestination resolves a v1beta1.Destination into a URI string.
func (r *URIResolver) URIFromDestination(ctx context.Context, dest duckv1beta1.Destination, parent interface{}) (string, error) {
	var deprecatedObjectReference *corev1.ObjectReference
	if !(dest.DeprecatedAPIVersion == "" && dest.DeprecatedKind == "" && dest.DeprecatedName == "" && dest.DeprecatedNamespace == "") {
		deprecatedObjectReference = &corev1.ObjectReference{
			Kind:       dest.DeprecatedKind,
			APIVersion: dest.DeprecatedAPIVersion,
			Name:       dest.DeprecatedName,
			Namespace:  dest.DeprecatedNamespace,
		}
	}
	if dest.Ref != nil && deprecatedObjectReference != nil {
		return "", errors.New("ref and [apiVersion, kind, name] can't be both present")
	}
	var ref *corev1.ObjectReference
	if dest.Ref != nil {
		ref = dest.Ref
	} else {
		ref = deprecatedObjectReference
	}
	if ref != nil {
		url, err := r.URIFromObjectReference(ctx, ref, parent)
		if err != nil {
			return "", err
		}
		if dest.URI != nil {
			if dest.URI.URL().IsAbs() {
				return "", errors.New("absolute URI is not allowed when Ref or [apiVersion, kind, name] exists")
			}
			return url.ResolveReference(dest.URI).String(), nil
		}
		return url.URL().String(), nil
	}

	if dest.URI != nil {
		// IsAbs check whether the URL has a non-empty scheme. Besides the non non-empty scheme, we also require dest.URI has a non-empty host
		if !dest.URI.URL().IsAbs() || dest.URI.Host == "" {
			return "", fmt.Errorf("URI is not absolute (both scheme and host should be non-empty): %q", dest.URI.String())
		}
		return dest.URI.String(), nil
	}

	return "", errors.New("destination missing Ref, [apiVersion, kind, name] and URI, expected at least one")
}

// URIFromDestinationV1 resolves a v1.Destination into a URL.
func (r *URIResolver) URIFromDestinationV1(ctx context.Context, dest duckv1.Destination, parent interface{}) (*apis.URL, error) {
	if dest.Ref != nil {
		url, err := r.URIFromKReference(ctx, dest.Ref, parent)
		if err != nil {
			return nil, err
		}
		if dest.URI != nil {
			if dest.URI.URL().IsAbs() {
				return nil, errors.New("absolute URI is not allowed when Ref or [apiVersion, kind, name] exists")
			}
			return url.ResolveReference(dest.URI), nil
		}
		return url, nil
	}

	if dest.URI != nil {
		// IsAbs check whether the URL has a non-empty scheme. Besides the non non-empty scheme, we also require dest.URI has a non-empty host
		if !dest.URI.URL().IsAbs() || dest.URI.Host == "" {
			return nil, fmt.Errorf("URI is not absolute(both scheme and host should be non-empty): %q", dest.URI.String())
		}
		return dest.URI, nil
	}

	return nil, errors.New("destination missing Ref and URI, expected at least one")
}

func (r *URIResolver) URIFromKReference(ctx context.Context, ref *duckv1.KReference, parent interface{}) (*apis.URL, error) {
	return r.URIFromObjectReference(ctx, &corev1.ObjectReference{Name: ref.Name, Namespace: ref.Namespace, APIVersion: ref.APIVersion, Kind: ref.Kind}, parent)
}

// URIFromObjectReference resolves an ObjectReference to a URI string.
func (r *URIResolver) URIFromObjectReference(ctx context.Context, ref *corev1.ObjectReference, parent interface{}) (*apis.URL, error) {
	if ref == nil {
		return nil, apierrs.NewBadRequest("ref is nil")
	}

	// try custom resolvers first
	for _, resolver := range r.resolvers {
		handled, url, err := resolver(ctx, ref)
		if handled {
			return url, err
		}

		// when handled is false, both url and err are ignored.
	}

	gvr, _ := meta.UnsafeGuessKindToResource(ref.GroupVersionKind())
	if err := r.tracker.TrackReference(tracker.Reference{
		APIVersion: ref.APIVersion,
		Kind:       ref.Kind,
		Namespace:  ref.Namespace,
		Name:       ref.Name,
	}, parent); err != nil {
		return nil, fmt.Errorf("failed to track reference %s %s/%s: %w", gvr.String(), ref.Namespace, ref.Name, err)
	}

	lister, err := r.listerFactory(gvr)
	if err != nil {
		return nil, fmt.Errorf("failed to get lister for %s: %w", gvr.String(), err)
	}

	obj, err := lister.ByNamespace(ref.Namespace).Get(ref.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s/%s: %w", ref.Namespace, ref.Name, err)
	}

	// K8s Services are special cased. They can be called, even though they do not satisfy the
	// Callable interface.
	if ref.APIVersion == "v1" && ref.Kind == "Service" {
		svc, err := r.serviceLister.Services(ref.Namespace).Get(ref.Name)
		if err != nil {
			return nil, fmt.Errorf("looking up Service in namespace %q: %w", ref.Namespace, err)
		}

		url := &apis.URL{
			Scheme: "http",
			Host:   network.GetServiceHostname(ref.Name, ref.Namespace),
		}

		// destination port is explicitly set in Service annotation
		if destinationPort, ok := svc.Annotations[ServicePortAnnotation]; ok {
			for _, port := range svc.Spec.Ports {
				if port.Name == destinationPort {
					return sanitizeURL(url, port.Port), nil
				}
			}
			return nil, fmt.Errorf("port %q not found in Service \"%s/%s\"", destinationPort, ref.Namespace, ref.Name)
		}

		// if service has only one port, use it
		if len(svc.Spec.Ports) == 1 {
			return sanitizeURL(url, svc.Spec.Ports[0].Port), nil
		}

		// port 80, if exposed, used for backward compatibility
		for _, port := range svc.Spec.Ports {
			if port.Port == 80 {
				return url, nil
			}
		}

		return nil, fmt.Errorf("service \"%s/%s\" does not have target port annotation %q", ref.Namespace, ref.Name, ServicePortAnnotation)
	}

	addressable, ok := obj.(*duckv1.AddressableType)
	if !ok {
		return nil, apierrs.NewBadRequest(fmt.Sprintf("%+v (%T) is not an AddressableType", ref, ref))
	}
	if addressable.Status.Address == nil {
		return nil, apierrs.NewBadRequest(fmt.Sprintf("address not set for %+v", ref))
	}
	url := addressable.Status.Address.URL
	if url == nil {
		return nil, apierrs.NewBadRequest(fmt.Sprintf("URL missing in address of %+v", ref))
	}
	if url.Host == "" {
		return nil, apierrs.NewBadRequest(fmt.Sprintf("hostname missing in address of %+v", ref))
	}
	return url, nil
}

func sanitizeURL(url *apis.URL, port int32) *apis.URL {
	switch port {
	case 80:
	case 443:
		url.Scheme = "https"
	default:
		url.Host = fmt.Sprintf("%s:%d", url.Host, port)
	}
	return url
}
