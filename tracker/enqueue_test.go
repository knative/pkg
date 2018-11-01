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

package tracker

import (
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/knative/pkg/testing"
)

// Ensure our resource satisfies the interface.
var _ accessor = (*Resource)(nil)

func TestHappyPaths(t *testing.T) {
	calls := 0
	f := func(key string) {
		calls = calls + 1
	}

	trk := New(f, 10*time.Millisecond)

	thing1 := &Resource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "ref.knative.dev/v1alpha1",
			Kind:       "Thing1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "foo",
		},
	}
	objRef := objectReference(thing1)

	thing2 := &Resource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "reffer.knative.dev/v1alpha1",
			Kind:       "Thing2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "bar",
		},
	}

	t.Run("Not tracked yet", func(t *testing.T) {
		trk.OnChanged(thing1)
		if got, want := calls, 0; got != want {
			t.Errorf("OnChanged() = %v, wanted %v", got, want)
		}
	})

	t.Run("Tracked gets called", func(t *testing.T) {
		if err := trk.Track(objRef, thing2); err != nil {
			t.Errorf("Track() = %v", err)
		}

		trk.OnChanged(thing1)
		if got, want := calls, 1; got != want {
			t.Errorf("OnChanged() = %v, wanted %v", got, want)
		}
	})

	t.Run("Still gets called", func(t *testing.T) {
		trk.OnChanged(thing1)
		if got, want := calls, 2; got != want {
			t.Errorf("OnChanged() = %v, wanted %v", got, want)
		}
	})

	// Check that after the sleep duration, we stop getting called.
	time.Sleep(20 * time.Millisecond)
	t.Run("Stops getting called", func(t *testing.T) {
		trk.OnChanged(thing1)
		if got, want := calls, 2; got != want {
			t.Errorf("OnChanged() = %v, wanted %v", got, want)
		}
		if _, stillThere := trk.(*impl).mapping[objRef]; stillThere {
			t.Errorf("Timeout passed, but mapping for objectReference is still there")
		}
	})

	t.Run("Starts getting called again", func(t *testing.T) {
		if err := trk.Track(objRef, thing2); err != nil {
			t.Errorf("Track() = %v", err)
		}

		trk.OnChanged(thing1)
		if got, want := calls, 3; got != want {
			t.Errorf("OnChanged() = %v, wanted %v", got, want)
		}
	})

	t.Run("OnChanged non-accessor", func(t *testing.T) {
		// Check that passing in a resource that doesn't implement
		// accessor won't panic.
		trk.OnChanged("not an accessor")

		if got, want := calls, 3; got != want {
			t.Errorf("OnChanged() = %v, wanted %v", got, want)
		}
	})

	t.Run("Track bad object", func(t *testing.T) {
		if err := trk.Track(objRef, struct{}{}); err == nil {
			t.Error("Track() = nil, wanted error")
		}
	})
}

func TestBadObjectReferences(t *testing.T) {
	trk := New(func(key string) {}, 10*time.Millisecond)
	thing1 := &Resource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "ref.knative.dev/v1alpha1",
			Kind:       "Thing1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "foo",
		},
	}

	tests := []struct {
		name      string
		objRef    corev1.ObjectReference
		substring string
	}{{
		name: "Missing APIVersion",
		objRef: corev1.ObjectReference{
			// APIVersion: "build.knative.dev/v1alpha1",
			Kind:      "Build",
			Namespace: "default",
			Name:      "kaniko",
		},
		substring: "APIVersion",
	}, {
		name: "Missing Kind",
		objRef: corev1.ObjectReference{
			APIVersion: "build.knative.dev/v1alpha1",
			// Kind:      "Build",
			Namespace: "default",
			Name:      "kaniko",
		},
		substring: "Kind",
	}, {
		name: "Missing Namespace",
		objRef: corev1.ObjectReference{
			APIVersion: "build.knative.dev/v1alpha1",
			Kind:       "Build",
			// Namespace: "default",
			Name: "kaniko",
		},
		substring: "Namespace",
	}, {
		name: "Missing Name",
		objRef: corev1.ObjectReference{
			APIVersion: "build.knative.dev/v1alpha1",
			Kind:       "Build",
			Namespace:  "default",
			// Name:      "kaniko",
		},
		substring: "Name",
	}, {
		name:   "Missing All",
		objRef: corev1.ObjectReference{
			// APIVersion: "build.knative.dev/v1alpha1",
			// Kind:       "Build",
			// Namespace:  "default",
			// Name:      "kaniko",
		},
		substring: "APIVersion Kind Namespace Name",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := trk.Track(test.objRef, thing1); err == nil {
				t.Error("Track() = nil, wanted error")
			} else if !strings.Contains(err.Error(), test.substring) {
				t.Errorf("Track() = %v, wanted substring: %s", err, test.substring)
			}
		})
	}
}
