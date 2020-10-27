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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type counter struct {
	name string
	mu   sync.RWMutex
	cfg  []*corev1.ConfigMap
	wg   *sync.WaitGroup
}

func (c *counter) callback(cm *corev1.ConfigMap) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg = append(c.cfg, cm)
	if c.wg != nil {
		c.wg.Done()
	}
}

func (c *counter) count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cfg)
}

func TestManualStartNOOP(t *testing.T) {
	watcher := ManualWatcher{
		Namespace: "default",
	}
	if err := watcher.Start(nil); err != nil {
		t.Error("Unexpected error watcher.Start() =", err)
	}
}

func TestCallbackInvoked(t *testing.T) {
	watcher := ManualWatcher{
		Namespace: "default",
	}

	// Verify empty works as designed.
	watcher.OnChange(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
	})
	observer := counter{}

	watcher.Watch("foo", observer.callback)
	watcher.OnChange(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
	})

	if observer.count() == 0 {
		t.Error("Expected callback to be invoked - got invocations", observer.count())
	}
}

func TestDifferentNamespace(t *testing.T) {
	watcher := ManualWatcher{
		Namespace: "default",
	}

	observer := counter{}

	watcher.Watch("foo", observer.callback)
	watcher.OnChange(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "not-default",
			Name:      "foo",
		},
	})

	if observer.count() != 0 {
		t.Error("Expected callback to be not be invoked - got invocations", observer.count())
	}
}

func TestDifferentConfigName(t *testing.T) {
	watcher := ManualWatcher{
		Namespace: "default",
	}

	observer := counter{}

	watcher.OnChange(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "foo",
		},
	})

	watcher.Watch("bar", observer.callback)

	if observer.count() != 0 {
		t.Error("Expected callback to be not be invoked - got invocations", observer.count())
	}
}
