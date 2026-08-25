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
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestInformerForSecretReturnsMergedInformer(t *testing.T) {
	client := fakekube.NewSimpleClientset()
	namespaces := []string{"ns-a", "ns-b"}

	factory := NewScopedFactory(client, 0, namespaces, "")

	inf1 := factory.InformerFor(&corev1.Secret{}, nil)
	inf2 := factory.InformerFor(&corev1.Secret{}, nil)

	if inf1 == nil {
		t.Fatal("InformerFor(Secret) returned nil")
	}
	if inf1 != inf2 {
		t.Error("InformerFor(Secret) should return the same cached instance on repeated calls")
	}
}

func TestInformerForNonSecretDelegatesToDefaultFactory(t *testing.T) {
	client := fakekube.NewSimpleClientset()
	namespaces := []string{"ns-a"}

	factory := NewScopedFactory(client, 0, namespaces, "")

	inf := factory.Core().V1().ConfigMaps().Informer()
	if inf == nil {
		t.Fatal("Core().V1().ConfigMaps().Informer() returned nil")
	}

	secretInf := factory.InformerFor(&corev1.Secret{}, nil)
	if inf == secretInf {
		t.Error("ConfigMap informer should not be the same as the secret informer")
	}
}

func TestWaitForCacheSyncIncludesSecretType(t *testing.T) {
	client := fakekube.NewSimpleClientset()
	namespaces := []string{"ns-a"}

	factory := NewScopedFactory(client, 0, namespaces, "")

	factory.InformerFor(&corev1.Secret{}, nil)

	stopCh := make(chan struct{})
	defer close(stopCh)
	factory.Start(stopCh)

	result := factory.WaitForCacheSync(stopCh)
	secretType := reflect.TypeOf(&corev1.Secret{})
	if _, ok := result[secretType]; !ok {
		t.Errorf("WaitForCacheSync result missing *corev1.Secret key; got keys: %v", result)
	}
}

func TestNewScopedFactoryCreatesOneSubFactoryPerNamespace(t *testing.T) {
	client := fakekube.NewSimpleClientset()
	namespaces := []string{"ns-a", "ns-b", "ns-c"}

	f := NewScopedFactory(client, 0, namespaces, "").(*scopedFactory)

	if len(f.subFactories) != len(namespaces) {
		t.Errorf("expected %d sub-factories, got %d", len(namespaces), len(f.subFactories))
	}
	if !reflect.DeepEqual(f.namespaces, namespaces) {
		t.Errorf("namespaces = %v, want %v", f.namespaces, namespaces)
	}
	if f.defaultFactory == nil {
		t.Error("defaultFactory should be non-nil")
	}
}

func TestScopedFactoryShutdown(t *testing.T) {
	client := fakekube.NewSimpleClientset()
	factory := NewScopedFactory(client, 0, []string{"ns-a"}, "")

	stopCh := make(chan struct{})
	factory.Start(stopCh)
	close(stopCh)
	factory.Shutdown()
}

func TestScopedFactoryResync(t *testing.T) {
	client := fakekube.NewSimpleClientset()
	resync := 30 * time.Second

	f := NewScopedFactory(client, resync, []string{"ns-a"}, "").(*scopedFactory)

	if len(f.subFactories) != 1 {
		t.Fatalf("expected 1 sub-factory, got %d", len(f.subFactories))
	}
}

func makeSecretInIndexer(t *testing.T, namespace, name string) cache.Indexer {
	t.Helper()
	idx := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
	if err := idx.Add(secret); err != nil {
		t.Fatalf("failed to add secret to indexer: %v", err)
	}
	return idx
}

func TestMergedIndexerList(t *testing.T) {
	idxA := makeSecretInIndexer(t, "ns-a", "secret-a")
	idxB := makeSecretInIndexer(t, "ns-b", "secret-b")

	m := &mergedIndexer{
		subs:        []cache.Indexer{idxA, idxB},
		byNamespace: map[string]cache.Indexer{"ns-a": idxA, "ns-b": idxB},
	}

	items := m.List()
	if len(items) != 2 {
		t.Errorf("List() returned %d items, want 2", len(items))
	}
}

func TestMergedIndexerGetByKey(t *testing.T) {
	idxA := makeSecretInIndexer(t, "ns-a", "secret-a")
	idxB := makeSecretInIndexer(t, "ns-b", "secret-b")

	m := &mergedIndexer{
		subs:        []cache.Indexer{idxA, idxB},
		byNamespace: map[string]cache.Indexer{"ns-a": idxA, "ns-b": idxB},
	}

	tests := []struct {
		key       string
		wantFound bool
	}{
		{"ns-a/secret-a", true},
		{"ns-b/secret-b", true},
		{"ns-a/secret-b", false},
		{"ns-c/secret-x", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			obj, found, err := m.GetByKey(tt.key)
			if err != nil {
				t.Fatalf("GetByKey(%q) error: %v", tt.key, err)
			}
			if found != tt.wantFound {
				t.Errorf("GetByKey(%q) found=%v, want %v", tt.key, found, tt.wantFound)
			}
			if tt.wantFound && obj == nil {
				t.Errorf("GetByKey(%q) returned nil object", tt.key)
			}
		})
	}
}

func TestMergedIndexerGet(t *testing.T) {
	idxA := makeSecretInIndexer(t, "ns-a", "secret-a")
	idxB := makeSecretInIndexer(t, "ns-b", "secret-b")

	m := &mergedIndexer{
		subs:        []cache.Indexer{idxA, idxB},
		byNamespace: map[string]cache.Indexer{"ns-a": idxA, "ns-b": idxB},
	}

	secretA := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "ns-a", Name: "secret-a"}}
	_, found, err := m.Get(secretA)
	if err != nil || !found {
		t.Errorf("Get(secret-a) found=%v, err=%v; want found=true, err=nil", found, err)
	}

	missing := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "ns-a", Name: "missing"}}
	_, found, err = m.Get(missing)
	if err != nil || found {
		t.Errorf("Get(missing) found=%v, err=%v; want found=false, err=nil", found, err)
	}
}

func TestMergedIndexerListKeys(t *testing.T) {
	idxA := makeSecretInIndexer(t, "ns-a", "secret-a")
	idxB := makeSecretInIndexer(t, "ns-b", "secret-b")

	m := &mergedIndexer{
		subs:        []cache.Indexer{idxA, idxB},
		byNamespace: map[string]cache.Indexer{"ns-a": idxA, "ns-b": idxB},
	}

	keys := m.ListKeys()
	if len(keys) != 2 {
		t.Errorf("ListKeys() returned %d keys, want 2: %v", len(keys), keys)
	}
}

func TestMergedIndexerByIndex(t *testing.T) {
	idxA := makeSecretInIndexer(t, "ns-a", "secret-a")
	idxB := makeSecretInIndexer(t, "ns-b", "secret-b")

	m := &mergedIndexer{
		subs:        []cache.Indexer{idxA, idxB},
		byNamespace: map[string]cache.Indexer{"ns-a": idxA, "ns-b": idxB},
	}

	items, err := m.ByIndex(cache.NamespaceIndex, "ns-a")
	if err != nil {
		t.Fatalf("ByIndex error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("ByIndex(ns-a) returned %d items, want 1", len(items))
	}
}

func TestMergedIndexerWritesPanic(t *testing.T) {
	m := &mergedIndexer{}

	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("%s: expected panic but did not panic", name)
			}
		}()
		fn()
	}

	assertPanics("Add", func() { _ = m.Add(nil) })
	assertPanics("Update", func() { _ = m.Update(nil) })
	assertPanics("Delete", func() { _ = m.Delete(nil) })
	assertPanics("Replace", func() { _ = m.Replace(nil, "") })
}

type fakeSharedIndexInformer struct {
	cache.SharedIndexInformer
	synced   bool
	indexer  cache.Indexer
	handlers []cache.ResourceEventHandler
}

func newFakeInformer(synced bool, idx cache.Indexer) *fakeSharedIndexInformer {
	return &fakeSharedIndexInformer{synced: synced, indexer: idx}
}

func (f *fakeSharedIndexInformer) HasSynced() bool                  { return f.synced }
func (f *fakeSharedIndexInformer) GetIndexer() cache.Indexer        { return f.indexer }
func (f *fakeSharedIndexInformer) Run(_ <-chan struct{})            {}
func (f *fakeSharedIndexInformer) RunWithContext(_ context.Context) {}
func (f *fakeSharedIndexInformer) AddEventHandler(h cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	f.handlers = append(f.handlers, h)
	return &fakeRegistration{}, nil
}
func (f *fakeSharedIndexInformer) AddEventHandlerWithResyncPeriod(h cache.ResourceEventHandler, _ time.Duration) (cache.ResourceEventHandlerRegistration, error) {
	return f.AddEventHandler(h)
}
func (f *fakeSharedIndexInformer) RemoveEventHandler(_ cache.ResourceEventHandlerRegistration) error {
	return nil
}
func (f *fakeSharedIndexInformer) AddIndexers(_ cache.Indexers) error                   { return nil }
func (f *fakeSharedIndexInformer) SetWatchErrorHandler(_ cache.WatchErrorHandler) error { return nil }
func (f *fakeSharedIndexInformer) IsStopped() bool                                      { return false }
func (f *fakeSharedIndexInformer) LastSyncResourceVersion() string                      { return "" }

type fakeRegistration struct{}

func (r *fakeRegistration) HasSynced() bool { return true }

func TestMergedInformerHasSyncedAllTrue(t *testing.T) {
	inf := &mergedInformer{
		subs: []cache.SharedIndexInformer{
			newFakeInformer(true, nil),
			newFakeInformer(true, nil),
		},
	}
	if !inf.HasSynced() {
		t.Error("HasSynced() = false, want true when all sub-informers are synced")
	}
}

func TestMergedInformerHasSyncedOneFalse(t *testing.T) {
	inf := &mergedInformer{
		subs: []cache.SharedIndexInformer{
			newFakeInformer(true, nil),
			newFakeInformer(false, nil),
		},
	}
	if inf.HasSynced() {
		t.Error("HasSynced() = true, want false when any sub-informer is not synced")
	}
}

func TestMergedInformerAddEventHandlerFansOut(t *testing.T) {
	subA := newFakeInformer(true, nil)
	subB := newFakeInformer(true, nil)

	inf := &mergedInformer{
		subs: []cache.SharedIndexInformer{subA, subB},
	}

	handler := cache.ResourceEventHandlerFuncs{}
	reg, err := inf.AddEventHandler(handler)
	if err != nil {
		t.Fatalf("AddEventHandler error: %v", err)
	}
	if reg == nil {
		t.Fatal("AddEventHandler returned nil registration")
	}
	if len(subA.handlers) != 1 || len(subB.handlers) != 1 {
		t.Errorf("handlers not fanned out: subA=%d subB=%d", len(subA.handlers), len(subB.handlers))
	}
}

func TestMergedInformerGetIndexer(t *testing.T) {
	idxA := makeSecretInIndexer(t, "ns-a", "secret-a")
	idxB := makeSecretInIndexer(t, "ns-b", "secret-b")

	inf := &mergedInformer{
		namespaces: []string{"ns-a", "ns-b"},
		subs: []cache.SharedIndexInformer{
			newFakeInformer(true, idxA),
			newFakeInformer(true, idxB),
		},
	}

	merged := inf.GetIndexer()
	if merged == nil {
		t.Fatal("GetIndexer() returned nil")
	}

	items := merged.List()
	if len(items) != 2 {
		t.Errorf("GetIndexer().List() = %d items, want 2", len(items))
	}
}

func TestMergedInformerCacheLookup(t *testing.T) {
	idxA := makeSecretInIndexer(t, "ns-a", "auth-secret")
	idxB := makeSecretInIndexer(t, "ns-b", "other-secret")

	inf := &mergedInformer{
		namespaces: []string{"ns-a", "ns-b"},
		subs: []cache.SharedIndexInformer{
			newFakeInformer(true, idxA),
			newFakeInformer(true, idxB),
		},
	}

	indexer := inf.GetIndexer()

	tests := []struct {
		key       string
		wantFound bool
	}{
		{"ns-a/auth-secret", true},
		{"ns-b/other-secret", true},
		{"ns-a/other-secret", false},
		{"ns-b/auth-secret", false},
		{"ns-c/auth-secret", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			obj, found, err := indexer.GetByKey(tt.key)
			if err != nil {
				t.Fatalf("GetByKey(%q) unexpected error: %v", tt.key, err)
			}
			if found != tt.wantFound {
				t.Errorf("GetByKey(%q): found=%v, want %v", tt.key, found, tt.wantFound)
			}
			if tt.wantFound && obj == nil {
				t.Errorf("GetByKey(%q): found=true but obj is nil", tt.key)
			}
		})
	}
}

func TestMergedInformerGetStore(t *testing.T) {
	inf := &mergedInformer{
		namespaces: []string{"ns-a"},
		subs: []cache.SharedIndexInformer{
			newFakeInformer(true, makeSecretInIndexer(t, "ns-a", "s")),
		},
	}
	store := inf.GetStore()
	if store == nil {
		t.Fatal("GetStore() returned nil")
	}
}

func TestMergedInformerRemoveEventHandler(t *testing.T) {
	subA := newFakeInformer(true, nil)
	inf := &mergedInformer{subs: []cache.SharedIndexInformer{subA}}

	reg, _ := inf.AddEventHandler(cache.ResourceEventHandlerFuncs{})
	if err := inf.RemoveEventHandler(reg); err != nil {
		t.Errorf("RemoveEventHandler error: %v", err)
	}
}

func TestNamespaceFromKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"ns-a/secret-a", "ns-a"},
		{"secret-a", ""},
		{"", ""},
		{"ns/name/extra", "ns"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := namespaceFromKey(tt.key); got != tt.want {
				t.Errorf("namespaceFromKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestNamespaceOf(t *testing.T) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "ns-a", Name: "s"}}
	if got := namespaceOf(secret); got != "ns-a" {
		t.Errorf("namespaceOf(secret) = %q, want %q", got, "ns-a")
	}
	if got := namespaceOf(nil); got != "" {
		t.Errorf("namespaceOf(nil) = %q, want empty", got)
	}
	if got := namespaceOf("not-an-object"); got != "" {
		t.Errorf("namespaceOf(string) = %q, want empty", got)
	}
}

func TestSubForNamespace(t *testing.T) {
	idxA := makeSecretInIndexer(t, "ns-a", "s")
	m := &mergedIndexer{
		subs:        []cache.Indexer{idxA},
		byNamespace: map[string]cache.Indexer{"ns-a": idxA},
	}

	if got := m.subForNamespace("ns-a"); got != idxA {
		t.Error("subForNamespace(ns-a) did not return the expected indexer")
	}
	if got := m.subForNamespace("ns-z"); got != nil {
		t.Error("subForNamespace(unknown) should return nil")
	}
	if got := m.subForNamespace(""); got != nil {
		t.Error("subForNamespace('') should return nil")
	}
}

var _ cache.SharedIndexInformer = (*fakeSharedIndexInformer)(nil)
var _ informers.SharedInformerFactory = (*scopedFactory)(nil)
var _ cache.Indexer = (*mergedIndexer)(nil)
var _ cache.SharedIndexInformer = (*mergedInformer)(nil)
var _ cache.ResourceEventHandlerRegistration = (*fakeRegistration)(nil)
var _ runtime.Object = (*corev1.Secret)(nil)
