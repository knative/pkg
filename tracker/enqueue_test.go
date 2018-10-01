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
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/knative/pkg/testing"
)

// Ensure our resource satisfies the interface.
var _ accessor = (*Resource)(nil)

func TestFoo(t *testing.T) {
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
