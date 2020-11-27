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

package psbinding

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	. "knative.dev/pkg/testing/duck"
)

func TestExact(t *testing.T) {
	em := make(exactMatcher, 1)

	want := &TestBindable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "blah",
			Name:      "asdf",
		},
	}

	gvk := want.GetGroupVersionKind()
	key := exactKey{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: want.GetNamespace(),
		Name:      want.GetName(),
	}

	// Before we add something, we shouldn't be able to get anything.
	if got := em.get(key); len(got) != 0 {
		t.Errorf("Get(%+v) = %v; wanted empty list", key, got)
	}

	// Now add it.
	em.add(key, want)

	// After we add something, we should be able to get it.
	wanted := []Bindable{want}
	if got := em.get(key); !cmp.Equal(got, wanted) {
		t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
	}

	otherKey := exactKey{
		Group:     "apps",
		Kind:      "Deployment",
		Namespace: "foo",
		Name:      "bar",
	}

	// After we add something, we still shouldn't return things for other keys.
	if got := em.get(otherKey); len(got) != 0 {
		t.Errorf("Get(%+v) = %v; wanted empty list", key, got)
	}

	// Add another Binding with the same coordinates.
	alsoWant := &TestBindable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "blah",
			Name:      "asdf",
		},
	}
	em.add(key, want)

	// We should now get both Bindings.
	wanted = []Bindable{want, alsoWant}
	if got := em.get(key); !cmp.Equal(got, wanted) {
		t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
	}
}

func TestInexact(t *testing.T) {
	want := &TestBindable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "blah",
			Name:      "asdf",
			Labels: map[string]string{
				"foo": "bar",
				"baz": "blah",
			},
		},
	}
	ls := labels.Set(want.Labels)

	gvk := want.GetGroupVersionKind()
	key := inexactKey{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: want.GetNamespace(),
	}

	t.Run("empty matcher doesn't match", func(t *testing.T) {
		im := make(inexactMatcher, 1)
		if got := im.get(key, ls); len(got) != 0 {
			t.Errorf("Get(%+v) = %v; wanted empty list", key, got)
		}
	})

	t.Run("matcher with exact labels matches", func(t *testing.T) {
		im := make(inexactMatcher, 1)

		// Use exactly the labels from the resource.
		selector := ls.AsSelector()

		im.add(key, selector, want)

		// With an appropriate selector, we match and get the binding.
		wanted := []Bindable{want}
		if got := im.get(key, ls); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})

	t.Run("matcher for everything matches", func(t *testing.T) {
		im := make(inexactMatcher, 1)

		// Match everything.
		selector := labels.Everything()

		im.add(key, selector, want)

		// With an appropriate selector, we match and get the binding.
		wanted := []Bindable{want}
		if got := im.get(key, ls); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})

	t.Run("matcher for nothing does not match", func(t *testing.T) {
		im := make(inexactMatcher, 1)

		// Match nothing.
		selector := labels.Nothing()

		im.add(key, selector, want)

		if got := im.get(key, ls); len(got) != 0 {
			t.Errorf("Get(%+v) = %v; wanted empty list", key, got)
		}
	})

	t.Run("matcher with a subset of labels matches", func(t *testing.T) {
		im := make(inexactMatcher, 1)

		// Use a subset of the resources labels.
		selector := labels.Set(map[string]string{
			"foo": "bar",
		}).AsSelector()

		im.add(key, selector, want)

		// With an appropriate selector, we match and get the binding.
		wanted := []Bindable{want}
		if got := im.get(key, ls); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})

	t.Run("matcher with overlapping labels does not match", func(t *testing.T) {
		im := make(inexactMatcher, 1)

		// Use a subset of the resources labels.
		selector := labels.Set(map[string]string{
			"foo": "bar",
			"not": "found",
		}).AsSelector()

		im.add(key, selector, want)

		// We shouldn't match because the second labels shouldn't match.
		if got := im.get(key, ls); len(got) != 0 {
			t.Errorf("Get(%+v) = %v; wanted empty list", key, got)
		}
	})

	t.Run("matcher with exact labels doesn't match a different namespace", func(t *testing.T) {
		im := make(inexactMatcher, 1)

		// Use exactly the labels from the resource.
		selector := ls.AsSelector()

		im.add(key, selector, want)

		otherKey := key
		otherKey.Namespace = "another"

		// We shouldn't match because the second labels shouldn't match.
		if got := im.get(otherKey, ls); len(got) != 0 {
			t.Errorf("Get(%+v) = %v; wanted empty list", key, got)
		}
	})

	t.Run("multiple Bindings match", func(t *testing.T) {
		alsoWant := &TestBindable{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "blah",
				Name:      "asdf",
				Labels: map[string]string{
					"foo": "bar",
					"baz": "blah",
				},
			},
		}

		im := make(inexactMatcher, 1)

		// Use a subset of the resources labels.
		selector := labels.Set(map[string]string{
			"foo": "bar",
		}).AsSelector()

		im.add(key, selector, want)
		im.add(key, selector, alsoWant)

		// With an appropriate selector, we match and get both bindings.
		wanted := []Bindable{want, alsoWant}
		if got := im.get(key, ls); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})
}

func TestIndex(t *testing.T) {
	wantExact1 := &TestBindable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "a",
			Name:      "b",
			Labels: map[string]string{
				"colour": "red",
			},
		},
	}

	wantExact2 := &TestBindable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "a",
			Name:      "b",
			Labels: map[string]string{
				"colour": "blue",
			},
		},
	}

	wantInexact1 := &TestBindable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "a",
			Name:      "b",
			Labels: map[string]string{
				"colour": "red",
			},
		},
	}

	wantInexact2 := &TestBindable{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "a",
			Name:      "b",
			Labels: map[string]string{
				"colour": "blue",
			},
		},
	}

	gvk := wantExact1.GetGroupVersionKind()
	eKey := exactKey{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: wantExact1.GetNamespace(),
		Name:      wantExact1.GetName(),
	}
	iKey := inexactKey{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: wantExact1.GetNamespace(),
	}
	redLabels := labels.Set(wantInexact1.Labels)
	redSelector := redLabels.AsSelector()
	blueLabels := labels.Set(wantInexact2.Labels)
	blueSelector := blueLabels.AsSelector()
	none := make(labels.Set)
	allSelector := none.AsSelector()

	t.Run("single exact match is found", func(t *testing.T) {
		var index index
		newIndexBuilder().
			associate(eKey, wantExact1).
			build(&index)

		wanted := []Bindable{wantExact1}
		if got := index.lookUp(eKey, none); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})

	t.Run("single inexact match is found", func(t *testing.T) {
		var index index
		newIndexBuilder().
			associateSelection(iKey, redSelector, wantInexact1).
			build(&index)

		wanted := []Bindable{wantInexact1}
		if got := index.lookUp(eKey, redLabels); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})

	t.Run("multiple exact matches are found", func(t *testing.T) {
		var index index
		newIndexBuilder().
			associate(eKey, wantExact1).
			associate(eKey, wantExact2).
			build(&index)

		wanted := []Bindable{wantExact1, wantExact2}
		if got := index.lookUp(eKey, none); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})

	t.Run("multiple inexact matches are found", func(t *testing.T) {
		var index index
		newIndexBuilder().
			associateSelection(iKey, allSelector, wantInexact1).
			associateSelection(iKey, allSelector, wantInexact2).
			build(&index)

		wanted := []Bindable{wantInexact1, wantInexact2}
		if got := index.lookUp(eKey, redLabels); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})

	t.Run("multiple exact and inexact matches are found", func(t *testing.T) {
		var index index
		newIndexBuilder().
			associate(eKey, wantExact1).
			associate(eKey, wantExact2).
			associateSelection(iKey, allSelector, wantInexact1).
			associateSelection(iKey, allSelector, wantInexact2).
			build(&index)

		wanted := []Bindable{wantExact1, wantExact2, wantInexact1, wantInexact2}
		if got := index.lookUp(eKey, none); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})

	t.Run("the correct exact and inexact matches are found", func(t *testing.T) {
		var index index
		newIndexBuilder().
			associate(eKey, wantExact1).
			associate(eKey, wantExact2).
			associateSelection(iKey, redSelector, wantInexact1).
			associateSelection(iKey, blueSelector, wantInexact2).
			build(&index)

		wanted := []Bindable{wantExact1, wantExact2, wantInexact1}
		if got := index.lookUp(eKey, redLabels); !cmp.Equal(got, wanted) {
			t.Error("Get (-want, +got):", cmp.Diff(wanted, got))
		}
	})
}
