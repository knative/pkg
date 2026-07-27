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
	"context"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

// newMergedInformer returns a SharedIndexInformer that aggregates N
// per-namespace sub-informers into a single logical informer.
//
// namespaces and subInformers are parallel slices: subInformers[i] watches
// namespaces[i]. This association lets the merged indexer route per-namespace
// lookups to the owning sub-indexer in O(1).
//
// Event handlers are fanned out to all sub-informers. The merged indexer
// provides a unified read-only view: writes panic because the sub-informers
// own their own stores.
func newMergedInformer(namespaces []string, subInformers []cache.SharedIndexInformer) cache.SharedIndexInformer {
	return &mergedInformer{namespaces: namespaces, subs: subInformers}
}

// mergedInformer implements cache.SharedIndexInformer by delegating to N
// per-namespace sub-informers. namespaces[i] is the namespace watched by
// subs[i].
type mergedInformer struct {
	namespaces []string
	subs       []cache.SharedIndexInformer
}

// multiRegistration fans out RemoveEventHandler to all N registrations.
type multiRegistration struct {
	regs []cache.ResourceEventHandlerRegistration
}

func (m *multiRegistration) HasSynced() bool {
	for _, r := range m.regs {
		if !r.HasSynced() {
			return false
		}
	}
	return true
}

func (inf *mergedInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	regs := make([]cache.ResourceEventHandlerRegistration, 0, len(inf.subs))
	for _, si := range inf.subs {
		reg, err := si.AddEventHandler(handler)
		if err != nil {
			return nil, err
		}
		regs = append(regs, reg)
	}
	return &multiRegistration{regs: regs}, nil
}

func (inf *mergedInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) (cache.ResourceEventHandlerRegistration, error) {
	regs := make([]cache.ResourceEventHandlerRegistration, 0, len(inf.subs))
	for _, si := range inf.subs {
		reg, err := si.AddEventHandlerWithResyncPeriod(handler, resyncPeriod)
		if err != nil {
			return nil, err
		}
		regs = append(regs, reg)
	}
	return &multiRegistration{regs: regs}, nil
}

func (inf *mergedInformer) AddEventHandlerWithOptions(handler cache.ResourceEventHandler, opts cache.HandlerOptions) (cache.ResourceEventHandlerRegistration, error) {
	regs := make([]cache.ResourceEventHandlerRegistration, 0, len(inf.subs))
	for _, si := range inf.subs {
		reg, err := si.AddEventHandlerWithOptions(handler, opts)
		if err != nil {
			return nil, err
		}
		regs = append(regs, reg)
	}
	return &multiRegistration{regs: regs}, nil
}

func (inf *mergedInformer) RemoveEventHandler(handle cache.ResourceEventHandlerRegistration) error {
	mr, ok := handle.(*multiRegistration)
	if !ok {
		for _, si := range inf.subs {
			_ = si.RemoveEventHandler(handle)
		}
		return nil
	}
	for i, si := range inf.subs {
		if i < len(mr.regs) {
			if err := si.RemoveEventHandler(mr.regs[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (inf *mergedInformer) GetStore() cache.Store {
	return inf.GetIndexer()
}

func (inf *mergedInformer) GetController() cache.Controller {
	return nil
}

func (inf *mergedInformer) Run(stopCh <-chan struct{}) {
	for _, si := range inf.subs {
		go si.Run(stopCh)
	}
	<-stopCh
}

func (inf *mergedInformer) RunWithContext(ctx context.Context) {
	for _, si := range inf.subs {
		go si.RunWithContext(ctx)
	}
	<-ctx.Done()
}

func (inf *mergedInformer) HasSynced() bool {
	for _, si := range inf.subs {
		if !si.HasSynced() {
			return false
		}
	}
	return true
}

func (inf *mergedInformer) LastSyncResourceVersion() string {
	if len(inf.subs) > 0 {
		return inf.subs[0].LastSyncResourceVersion()
	}
	return ""
}

func (inf *mergedInformer) SetWatchErrorHandler(handler cache.WatchErrorHandler) error {
	for _, si := range inf.subs {
		if err := si.SetWatchErrorHandler(handler); err != nil {
			return err
		}
	}
	return nil
}

func (inf *mergedInformer) SetWatchErrorHandlerWithContext(handler cache.WatchErrorHandlerWithContext) error {
	for _, si := range inf.subs {
		if err := si.SetWatchErrorHandlerWithContext(handler); err != nil {
			return err
		}
	}
	return nil
}

func (inf *mergedInformer) SetTransform(fn cache.TransformFunc) error {
	for _, si := range inf.subs {
		if err := si.SetTransform(fn); err != nil {
			return err
		}
	}
	return nil
}

func (inf *mergedInformer) IsStopped() bool {
	for _, si := range inf.subs {
		if !si.IsStopped() {
			return false
		}
	}
	return true
}

func (inf *mergedInformer) AddIndexers(indexers cache.Indexers) error {
	for _, si := range inf.subs {
		if err := si.AddIndexers(indexers); err != nil {
			return err
		}
	}
	return nil
}

func (inf *mergedInformer) GetIndexer() cache.Indexer {
	indexers := make([]cache.Indexer, 0, len(inf.subs))
	byNamespace := make(map[string]cache.Indexer, len(inf.subs))
	for i, si := range inf.subs {
		idx := si.GetIndexer()
		indexers = append(indexers, idx)
		if i < len(inf.namespaces) {
			byNamespace[inf.namespaces[i]] = idx
		}
	}
	return &mergedIndexer{subs: indexers, byNamespace: byNamespace}
}

// mergedIndexer implements cache.Indexer over N per-namespace sub-indexers.
// It is read-only: the per-namespace informers own their underlying stores.
type mergedIndexer struct {
	subs        []cache.Indexer
	byNamespace map[string]cache.Indexer
}

func (m *mergedIndexer) Add(obj interface{}) error {
	panic("multinamespace: mergedIndexer is read-only")
}

func (m *mergedIndexer) Update(obj interface{}) error {
	panic("multinamespace: mergedIndexer is read-only")
}

func (m *mergedIndexer) Delete(obj interface{}) error {
	panic("multinamespace: mergedIndexer is read-only")
}

func (m *mergedIndexer) Replace(objs []interface{}, resourceVersion string) error {
	panic("multinamespace: mergedIndexer is read-only")
}

func (m *mergedIndexer) Resync() error { return nil }

func (m *mergedIndexer) List() []interface{} {
	var all []interface{}
	for _, s := range m.subs {
		all = append(all, s.List()...)
	}
	return all
}

func (m *mergedIndexer) ListKeys() []string {
	var all []string
	for _, s := range m.subs {
		all = append(all, s.ListKeys()...)
	}
	return all
}

func (m *mergedIndexer) Get(obj interface{}) (interface{}, bool, error) {
	ns := namespaceOf(obj)
	sub := m.subForNamespace(ns)
	if sub != nil {
		return sub.Get(obj)
	}
	for _, s := range m.subs {
		item, exists, err := s.Get(obj)
		if err != nil || exists {
			return item, exists, err
		}
	}
	return nil, false, nil
}

func (m *mergedIndexer) GetByKey(key string) (interface{}, bool, error) {
	ns := namespaceFromKey(key)
	sub := m.subForNamespace(ns)
	if sub != nil {
		return sub.GetByKey(key)
	}
	for _, s := range m.subs {
		item, exists, err := s.GetByKey(key)
		if err != nil || exists {
			return item, exists, err
		}
	}
	return nil, false, nil
}

func (m *mergedIndexer) Index(indexName string, obj interface{}) ([]interface{}, error) {
	var all []interface{}
	for _, s := range m.subs {
		items, err := s.Index(indexName, obj)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	return all, nil
}

func (m *mergedIndexer) IndexKeys(indexName, indexedValue string) ([]string, error) {
	var all []string
	for _, s := range m.subs {
		keys, err := s.IndexKeys(indexName, indexedValue)
		if err != nil {
			return nil, err
		}
		all = append(all, keys...)
	}
	return all, nil
}

func (m *mergedIndexer) ListIndexFuncValues(indexName string) []string {
	seen := make(map[string]struct{})
	var all []string
	for _, s := range m.subs {
		for _, v := range s.ListIndexFuncValues(indexName) {
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				all = append(all, v)
			}
		}
	}
	return all
}

func (m *mergedIndexer) ByIndex(indexName, indexedValue string) ([]interface{}, error) {
	var all []interface{}
	for _, s := range m.subs {
		items, err := s.ByIndex(indexName, indexedValue)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	return all, nil
}

func (m *mergedIndexer) GetIndexers() cache.Indexers {
	if len(m.subs) > 0 {
		return m.subs[0].GetIndexers()
	}
	return cache.Indexers{}
}

func (m *mergedIndexer) AddIndexers(newIndexers cache.Indexers) error {
	for _, s := range m.subs {
		if err := s.AddIndexers(newIndexers); err != nil {
			return err
		}
	}
	return nil
}

func (m *mergedIndexer) subForNamespace(ns string) cache.Indexer {
	if ns == "" {
		return nil
	}
	return m.byNamespace[ns]
}

func namespaceOf(obj interface{}) string {
	if obj == nil {
		return ""
	}
	acc, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}
	return acc.GetNamespace()
}

func namespaceFromKey(key string) string {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ""
}
