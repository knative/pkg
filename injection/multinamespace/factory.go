/*
Copyright 2025 The Knative Authors

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

package multinamespace

import (
	"reflect"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	admissionregistration "k8s.io/client-go/informers/admissionregistration"
	apiserverinternal "k8s.io/client-go/informers/apiserverinternal"
	apps "k8s.io/client-go/informers/apps"
	autoscaling "k8s.io/client-go/informers/autoscaling"
	batch "k8s.io/client-go/informers/batch"
	certificates "k8s.io/client-go/informers/certificates"
	coordination "k8s.io/client-go/informers/coordination"
	core "k8s.io/client-go/informers/core"
	discovery "k8s.io/client-go/informers/discovery"
	events "k8s.io/client-go/informers/events"
	extensions "k8s.io/client-go/informers/extensions"
	flowcontrol "k8s.io/client-go/informers/flowcontrol"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	networking "k8s.io/client-go/informers/networking"
	node "k8s.io/client-go/informers/node"
	policy "k8s.io/client-go/informers/policy"
	rbac "k8s.io/client-go/informers/rbac"
	resource "k8s.io/client-go/informers/resource"
	scheduling "k8s.io/client-go/informers/scheduling"
	storage "k8s.io/client-go/informers/storage"
	storagemigration "k8s.io/client-go/informers/storagemigration"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// scopedFactory is a self-contained SharedInformerFactory. It intercepts only
// the secret informer, replacing it with a merged view over per-namespace
// sub-factories, and delegates every other informer type to an internal default
// factory (cluster-wide, or single-namespace when defaultNamespace is set).
type scopedFactory struct {
	defaultFactory informers.SharedInformerFactory

	namespaces   []string
	subFactories []informers.SharedInformerFactory

	mu           sync.Mutex
	cachedSecret cache.SharedIndexInformer
}

// NewScopedFactory creates a scopedFactory that restricts the secret informer
// to the given namespaces. A separate SharedInformerFactory scoped to each
// namespace is created from client with the provided resync period.
//
// defaultNamespace, when non-empty, scopes the internal default factory used
// for non-secret types (matching injection.WithNamespaceScope). When empty,
// non-secret informers are cluster-wide.
func NewScopedFactory(
	client kubernetes.Interface,
	resync time.Duration,
	namespaces []string,
	defaultNamespace string,
) informers.SharedInformerFactory {
	opts := make([]informers.SharedInformerOption, 0, 1)
	if defaultNamespace != "" {
		opts = append(opts, informers.WithNamespace(defaultNamespace))
	}
	defaultFactory := informers.NewSharedInformerFactoryWithOptions(client, resync, opts...)

	subs := make([]informers.SharedInformerFactory, 0, len(namespaces))
	for _, ns := range namespaces {
		subs = append(subs, informers.NewSharedInformerFactoryWithOptions(
			client, resync, informers.WithNamespace(ns),
		))
	}
	return &scopedFactory{
		defaultFactory: defaultFactory,
		namespaces:     namespaces,
		subFactories:   subs,
	}
}

func (f *scopedFactory) InformerFor(obj runtime.Object, newFunc internalinterfaces.NewInformerFunc) cache.SharedIndexInformer {
	if _, isSecret := obj.(*corev1.Secret); isSecret {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.cachedSecret != nil {
			return f.cachedSecret
		}
		nsInformers := make([]cache.SharedIndexInformer, 0, len(f.subFactories))
		for _, sf := range f.subFactories {
			nsInformers = append(nsInformers, sf.Core().V1().Secrets().Informer())
		}
		f.cachedSecret = newMergedInformer(f.namespaces, nsInformers)
		return f.cachedSecret
	}
	return f.defaultFactory.InformerFor(obj, newFunc)
}

func (f *scopedFactory) Start(stopCh <-chan struct{}) {
	f.defaultFactory.Start(stopCh)
	for _, sf := range f.subFactories {
		sf.Start(stopCh)
	}
}

func (f *scopedFactory) Shutdown() {
	f.defaultFactory.Shutdown()
	for _, sf := range f.subFactories {
		sf.Shutdown()
	}
}

func (f *scopedFactory) WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool {
	result := f.defaultFactory.WaitForCacheSync(stopCh)
	for _, sf := range f.subFactories {
		for k, v := range sf.WaitForCacheSync(stopCh) {
			if existing, ok := result[k]; ok {
				result[k] = existing && v
			} else {
				result[k] = v
			}
		}
	}
	return result
}

func (f *scopedFactory) ForResource(gvr schema.GroupVersionResource) (informers.GenericInformer, error) {
	return f.defaultFactory.ForResource(gvr)
}

func (f *scopedFactory) Core() core.Interface {
	return core.New(f, "", nil)
}

func (f *scopedFactory) Admissionregistration() admissionregistration.Interface {
	return f.defaultFactory.Admissionregistration()
}

func (f *scopedFactory) Internal() apiserverinternal.Interface {
	return f.defaultFactory.Internal()
}

func (f *scopedFactory) Apps() apps.Interface {
	return f.defaultFactory.Apps()
}

func (f *scopedFactory) Autoscaling() autoscaling.Interface {
	return f.defaultFactory.Autoscaling()
}

func (f *scopedFactory) Batch() batch.Interface {
	return f.defaultFactory.Batch()
}

func (f *scopedFactory) Certificates() certificates.Interface {
	return f.defaultFactory.Certificates()
}

func (f *scopedFactory) Coordination() coordination.Interface {
	return f.defaultFactory.Coordination()
}

func (f *scopedFactory) Discovery() discovery.Interface {
	return f.defaultFactory.Discovery()
}

func (f *scopedFactory) Events() events.Interface {
	return f.defaultFactory.Events()
}

func (f *scopedFactory) Extensions() extensions.Interface {
	return f.defaultFactory.Extensions()
}

func (f *scopedFactory) Flowcontrol() flowcontrol.Interface {
	return f.defaultFactory.Flowcontrol()
}

func (f *scopedFactory) Networking() networking.Interface {
	return f.defaultFactory.Networking()
}

func (f *scopedFactory) Node() node.Interface {
	return f.defaultFactory.Node()
}

func (f *scopedFactory) Policy() policy.Interface {
	return f.defaultFactory.Policy()
}

func (f *scopedFactory) Rbac() rbac.Interface {
	return f.defaultFactory.Rbac()
}

func (f *scopedFactory) Resource() resource.Interface {
	return f.defaultFactory.Resource()
}

func (f *scopedFactory) Scheduling() scheduling.Interface {
	return f.defaultFactory.Scheduling()
}

func (f *scopedFactory) Storage() storage.Interface {
	return f.defaultFactory.Storage()
}

func (f *scopedFactory) Storagemigration() storagemigration.Interface {
	return f.defaultFactory.Storagemigration()
}
