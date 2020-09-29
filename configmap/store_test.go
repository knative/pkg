/*
Copyright 2018 The Knative Authors.

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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "knative.dev/pkg/logging/testing"
)

const (
	config1 = "config-name-1"
	config2 = "config-name-2"
)

var constructor = func(c *corev1.ConfigMap) (interface{}, error) {
	return c.Name, nil
}

func TestStoreBadConstructors(t *testing.T) {
	tests := []struct {
		name        string
		constructor interface{}
	}{{
		name:        "not a function",
		constructor: "i'm pretending to be a function",
	}, {
		name:        "no function arguments",
		constructor: func() (bool, error) { return true, nil },
	}, {
		name:        "single argument is not a configmap",
		constructor: func(bool) (bool, error) { return true, nil },
	}, {
		name:        "single return",
		constructor: func(*corev1.ConfigMap) error { return nil },
	}, {
		name:        "wrong second return",
		constructor: func(*corev1.ConfigMap) (bool, bool) { return true, true },
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected NewUntypedStore to panic")
				}
			}()

			NewUntypedStore("store", nil, Constructors{
				"test": test.constructor,
			})
		})
	}
}

func TestStoreWatchConfigs(t *testing.T) {
	store := NewUntypedStore(
		"name",
		TestLogger(t),
		Constructors{
			config1: constructor,
			config2: constructor,
		},
	)

	watcher := &mockWatcher{}
	store.WatchConfigs(watcher)

	want := []string{
		config1,
		config2,
	}

	got := watcher.watches

	if diff := cmp.Diff(want, got, sortStrings); diff != "" {
		t.Errorf("Unexpected configmap watches (-want, +got):\n%s", diff)
	}
}

func TestOnAfterStore(t *testing.T) {
	var calledFor string
	store := NewUntypedStore(
		"name",
		TestLogger(t),
		Constructors{
			config1: constructor,
			config2: constructor,
		},
		func(name string, value interface{}) {
			calledFor = name
		},
	)

	store.OnConfigChanged(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: config1,
		},
	})

	if calledFor != config1 {
		t.Fatalf("calledFor = %s, want %s", calledFor, config1)
	}

	store.OnConfigChanged(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: config2,
		},
	})

	if calledFor != config2 {
		t.Fatalf("calledFor = %s, want %s", calledFor, config2)
	}
}

func TestStoreConfigChange(t *testing.T) {
	store := NewUntypedStore(
		"name",
		TestLogger(t),
		Constructors{
			config1: constructor,
			config2: constructor,
		},
	)

	store.OnConfigChanged(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: config1,
		},
	})

	result := store.UntypedLoad(config1)

	if diff := cmp.Diff(result, config1); diff != "" {
		t.Error("Expected loaded value diff:", diff)
	}

	result = store.UntypedLoad(config2)

	if diff := cmp.Diff(result, nil); diff != "" {
		t.Error("Unexpected loaded value diff:", diff)
	}

	store.OnConfigChanged(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: config2,
		},
	})

	result = store.UntypedLoad(config2)

	if diff := cmp.Diff(result, config2); diff != "" {
		t.Error("Expected loaded value diff:", diff)
	}
}

func TestStoreFailedFirstConversionCrashes(t *testing.T) {
	if os.Getenv("CRASH") == "1" {
		constructor := func(c *corev1.ConfigMap) (interface{}, error) {
			return nil, errors.New("failure")
		}

		store := NewUntypedStore("name", TestLogger(t),
			Constructors{config1: constructor},
		)

		store.OnConfigChanged(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: config1,
			},
		})
		return
	}

	cmd := exec.Command(os.Args[0], fmt.Sprintf("-test.run=%s", t.Name()))
	cmd.Env = append(os.Environ(), "CRASH=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatal("process should have exited with status 1 - err", err)
}

func TestStoreFailedUpdate(t *testing.T) {
	induceConstructorFailure := false

	constructor := func(c *corev1.ConfigMap) (interface{}, error) {
		if induceConstructorFailure {
			return nil, errors.New("failure")
		}

		return time.Now().String(), nil
	}

	store := NewUntypedStore("name", TestLogger(t),
		Constructors{config1: constructor},
	)

	store.OnConfigChanged(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: config1,
		},
	})

	firstLoad := store.UntypedLoad(config1)

	induceConstructorFailure = true
	store.OnConfigChanged(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: config1,
		},
	})

	secondLoad := store.UntypedLoad(config1)

	if diff := cmp.Diff(firstLoad, secondLoad); diff != "" {
		t.Error("Expected loaded value to remain the same dff:", diff)
	}
}

type mockWatcher struct {
	watches []string
}

func (w *mockWatcher) Watch(config string, o ...Observer) {
	w.watches = append(w.watches, config)
}

func (*mockWatcher) Start(<-chan struct{}) error { return nil }

var _ Watcher = (*mockWatcher)(nil)

var sortStrings = cmpopts.SortSlices(func(x, y string) bool {
	return x < y
})
