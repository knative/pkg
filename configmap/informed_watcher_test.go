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

package configmap

import (
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
)

type counter struct {
	name string
	cfg  []*corev1.ConfigMap
	wg   *sync.WaitGroup
}

func (c *counter) callback(cm *corev1.ConfigMap) {
	c.cfg = append(c.cfg, cm)
	if c.wg != nil {
		c.wg.Done()
	}
}

func (c *counter) count() int {
	return len(c.cfg)
}

func TestInformedWatcher(t *testing.T) {
	fooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
	}
	barCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "bar",
		},
	}
	kc := fakekubeclientset.NewSimpleClientset(fooCM, barCM)
	cm := NewInformedWatcher(kc, "default")

	foo1 := &counter{name: "foo1"}
	foo2 := &counter{name: "foo2"}
	bar := &counter{name: "bar"}
	cm.Watch("foo", foo1.callback)
	cm.Watch("foo", foo2.callback)
	cm.Watch("bar", bar.callback)

	stopCh := make(chan struct{})
	defer close(stopCh)
	err := cm.Start(stopCh)
	if err != nil {
		t.Fatalf("cm.Start() = %v", err)
	}

	// When Start returns the callbacks should have been called with the
	// version of the objects that is available.
	for _, obj := range []*counter{foo1, foo2, bar} {
		if got, want := obj.count(), 1; got != want {
			t.Errorf("%v.count = %v, want %v", obj, got, want)
		}
	}

	// After a "foo" event, the "foo" watchers should have 2,
	// and the "bar" watchers should still have 1
	cm.updateConfigMapEvent(nil, fooCM)
	for _, obj := range []*counter{foo1, foo2} {
		if got, want := obj.count(), 2; got != want {
			t.Errorf("%v.count = %v, want %v", obj, got, want)
		}
	}

	for _, obj := range []*counter{bar} {
		if got, want := obj.count(), 1; got != want {
			t.Errorf("%v.count = %v, want %v", obj, got, want)
		}
	}

	// After a "foo" and "bar" event, the "foo" watchers should have 3,
	// and the "bar" watchers should still have 2
	cm.updateConfigMapEvent(nil, fooCM)
	cm.updateConfigMapEvent(nil, barCM)
	for _, obj := range []*counter{foo1, foo2} {
		if got, want := obj.count(), 3; got != want {
			t.Errorf("%v.count = %v, want %v", obj, got, want)
		}
	}
	for _, obj := range []*counter{bar} {
		if got, want := obj.count(), 2; got != want {
			t.Errorf("%v.count = %v, want %v", obj, got, want)
		}
	}

	// After a "bar" event, all watchers should have 3
	cm.updateConfigMapEvent(nil, barCM)
	for _, obj := range []*counter{foo1, foo2, bar} {
		if got, want := obj.count(), 3; got != want {
			t.Errorf("%v.count = %v, want %v", obj, got, want)
		}
	}

	// After an unwatched ConfigMap update, no change.

	cm.updateConfigMapEvent(nil, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "not-watched",
		},
	})
	for _, obj := range []*counter{foo1, foo2, bar} {
		if got, want := obj.count(), 3; got != want {
			t.Errorf("%v.count = %v, want %v", obj, got, want)
		}
	}

	// After a change in an unrelated namespace, no change.
	cm.updateConfigMapEvent(nil, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "not-default",
			Name:      "foo",
		},
	})
	for _, obj := range []*counter{foo1, foo2, bar} {
		if got, want := obj.count(), 3; got != want {
			t.Errorf("%v.count = %v, want %v", obj, got, want)
		}
	}
}

func TestWatchMissingFailsOnStart(t *testing.T) {
	kc := fakekubeclientset.NewSimpleClientset()
	cm := NewInformedWatcher(kc, "default")

	foo1 := &counter{name: "foo1"}
	cm.Watch("foo", foo1.callback)

	stopCh := make(chan struct{})
	defer close(stopCh)

	// This should error because we don't have a ConfigMap named "foo".
	err := cm.Start(stopCh)
	if err == nil {
		t.Fatal("cm.Start() succeeded, wanted error")
	}
}

func TestWatchMissingOKWithDefaultOnStart(t *testing.T) {
	fooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
	}

	kc := fakekubeclientset.NewSimpleClientset()
	cm := NewInformedWatcher(kc, "default")

	foo1 := &counter{name: "foo1"}
	cm.WatchWithDefault(*fooCM, foo1.callback)

	stopCh := make(chan struct{})
	defer close(stopCh)

	// This shouldn't error because we don't have a ConfigMap named "foo", but we do have a default.
	err := cm.Start(stopCh)
	if err != nil {
		t.Fatalf("cm.Start() failed, %v", err)
	}

	if foo1.count() != 1 {
		t.Errorf("foo1.count = %v, want 1", foo1.count())
	}
}

func TestErrorOnMultipleStarts(t *testing.T) {
	fooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
	}
	kc := fakekubeclientset.NewSimpleClientset(fooCM)
	cm := NewInformedWatcher(kc, "default")

	foo1 := &counter{name: "foo1"}
	cm.Watch("foo", foo1.callback)

	stopCh := make(chan struct{})
	defer close(stopCh)

	// This should succeed because the watched resource exists.
	if err := cm.Start(stopCh); err != nil {
		t.Fatalf("cm.Start() = %v", err)
	}

	// This should error because we already called Start()
	if err := cm.Start(stopCh); err == nil {
		t.Fatal("cm.Start() succeeded, wanted error")
	}
}

func TestDefaultObserved(t *testing.T) {
	defaultFooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
		Data: map[string]string{
			"default": "from code",
		},
	}
	fooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
		Data: map[string]string{
			"from": "k8s",
		},
	}

	kc := fakekubeclientset.NewSimpleClientset(fooCM)
	cm := NewInformedWatcher(kc, "default")

	foo1 := &counter{name: "foo1"}
	cm.WatchWithDefault(*defaultFooCM, foo1.callback)

	stopCh := make(chan struct{})
	defer close(stopCh)
	err := cm.Start(stopCh)
	if err != nil {
		t.Fatalf("cm.Start() = %v", err)
	}
	// We expect:
	// 1. The default to be seen once during startup.
	// 2. The real K8s version during the initial pass.
	expected := []*corev1.ConfigMap{defaultFooCM, fooCM}
	if foo1.count() != len(expected) {
		t.Fatalf("foo1.count = %v, want %d", len(foo1.cfg), len(expected))
	}
	for i, cfg := range expected {
		if got, want := foo1.cfg[i].Data, cfg.Data; !equality.Semantic.DeepEqual(want, got) {
			t.Errorf("%d config seen should have been '%v', actually '%v'", i, want, got)
		}
	}
}

func TestDefaultConfigMapDeleted(t *testing.T) {
	defaultFooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
		Data: map[string]string{
			"default": "from code",
		},
	}
	fooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
		Data: map[string]string{
			"from": "k8s",
		},
	}

	kc := fakekubeclientset.NewSimpleClientset(fooCM)
	cm := NewInformedWatcher(kc, "default")

	foo1 := &counter{name: "foo1"}
	cm.WatchWithDefault(*defaultFooCM, foo1.callback)

	stopCh := make(chan struct{})
	defer close(stopCh)
	err := cm.Start(stopCh)
	if err != nil {
		t.Fatalf("cm.Start() = %v", err)
	}

	// Delete the real ConfigMap in K8s, which should cause the default to be processed again.
	// Because this happens asynchronously via a watcher, use a sync.WaitGroup to wait until it has
	// occurred.
	foo1.wg = &sync.WaitGroup{}
	foo1.wg.Add(1)
	err = kc.CoreV1().ConfigMaps(fooCM.Namespace).Delete(fooCM.Name, nil)
	if err != nil {
		t.Fatalf("Error deleting fooCM: %v", err)
	}
	foo1.wg.Wait()

	// We expect:
	// 1. The default to be seen once during startup.
	// 2. The real K8s version during the initial pass.
	// 3. The default again, when the real K8s version is deleted.
	expected := []*corev1.ConfigMap{defaultFooCM, fooCM, defaultFooCM}
	if foo1.count() != len(expected) {
		t.Fatalf("foo1.count = %v, want %d", len(foo1.cfg), len(expected))
	}
	for i, cfg := range expected {
		if got, want := foo1.cfg[i].Data, cfg.Data; !equality.Semantic.DeepEqual(want, got) {
			t.Errorf("%d config seen should have been '%v', actually '%v'", i, want, got)
		}
	}
}

func TestWatchWithDefaultAfterStart(t *testing.T) {
	defaultFooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
		Data: map[string]string{
			"default": "from code",
		},
	}
	fooCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
		Data: map[string]string{
			"from": "k8s",
		},
	}

	kc := fakekubeclientset.NewSimpleClientset(fooCM)
	cm := NewInformedWatcher(kc, "default")

	stopCh := make(chan struct{})
	defer close(stopCh)
	// Start before adding the WatchWithDefault.
	err := cm.Start(stopCh)
	if err != nil {
		t.Fatalf("cm.Start() = %v", err)
	}

	foo1 := &counter{name: "foo1"}

	// Add the WatchWithDefault. This should panic because the InformedWatcher has already started.
	func() {
		defer func() {
			recover()
		}()
		cm.WatchWithDefault(*defaultFooCM, foo1.callback)
		t.Fatal("WatchWithDefault should have panicked")
	}()

	// We expect nothing.
	var expected []*corev1.ConfigMap
	if foo1.count() != len(expected) {
		t.Fatalf("foo1.count = %v, want %d", len(foo1.cfg), len(expected))
	}
}
