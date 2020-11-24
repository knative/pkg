/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless requ ired by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package psbinding

import (
	"sync"

	"k8s.io/apimachinery/pkg/labels"
)

// exactKey is the type for keys that match exactly.
type exactKey struct {
	Group     string
	Kind      string
	Namespace string
	Name      string
}

// exactMatcher is our reverse index from subjects to the Bindings that apply to
// them.
type exactMatcher map[exactKey]Bindable

// add writes a key into the reverse index.
func (em exactMatcher) add(key exactKey, b Bindable) {
	em[key] = b
}

// get fetches the key from the reverse index, if present.
func (em exactMatcher) get(key exactKey) (bindable Bindable, present bool) {
	b, ok := em[key]
	return b, ok
}

// inexactKey is the type for keys that match inexactly (via selector)
type inexactKey struct {
	Group     string
	Kind      string
	Namespace string
}

// pair holds selectors and bindables for a particular inexactKey.
type pair struct {
	selector labels.Selector
	sb       Bindable
}

// inexactMatcher is our reverse index from subjects to the Bindings that apply to
// them.
type inexactMatcher map[inexactKey][]pair

// add writes a key into the reverse index.
func (im inexactMatcher) add(key inexactKey, selector labels.Selector, b Bindable) {
	pl := im[key]
	pl = append(pl, pair{
		selector: selector,
		sb:       b,
	})
	im[key] = pl
}

// get fetches the key from the reverse index, if present.
func (im inexactMatcher) get(key inexactKey, ls labels.Set) (bindable Bindable, present bool) {
	// Iterate over the list of pairs matched for this GK + namespace and return the first
	// Bindable that matches our selector.
	for _, p := range im[key] {
		if p.selector.Matches(ls) {
			return p.sb, true
		}
	}
	return nil, false
}

// index is a collection of Bindables indexed by their subject resources.
type index struct {
	// lock protects access to exact and inexact
	lock    sync.RWMutex
	exact   exactMatcher
	inexact inexactMatcher
}

// indexBuilder allows an index to be built atomically
type indexBuilder struct {
	exact   exactMatcher
	inexact inexactMatcher
}

// newIndexBuilder constructs a new IndexBuilder.
func newIndexBuilder() *indexBuilder {
	return &indexBuilder{
		exact:   make(exactMatcher),
		inexact: make(inexactMatcher),
	}
}

// associate associates a resource with a given exact key with a given Bindable.
// TODO: allow multiple Bindables to be associated with the same exact key.
func (ib *indexBuilder) associate(key exactKey, fb Bindable) {
	ib.exact.add(key, fb)
}

// associateSelection associates resources with the given inexact key and labels matching the given selector with a given Bindable.
func (ib *indexBuilder) associateSelection(inexactKey inexactKey, selector labels.Selector, fb Bindable) {
	ib.inexact.add(inexactKey, selector, fb)
}

// build sets the given index to the built value.
func (ib *indexBuilder) build(index *index) {
	index.setIndex(ib.exact, ib.inexact)
}

func (idx *index) setIndex(exact exactMatcher, inexact inexactMatcher) {
	idx.lock.Lock()
	defer idx.lock.Unlock()
	idx.exact = exact
	idx.inexact = inexact
}

// lookUp returns the Bindables associated with a resource with the given group, kind, namespace, name, and labels.
// TODO: find all the matching Bindables instead of just one.
func (idx *index) lookUp(key exactKey, labels labels.Set) []Bindable {
	idx.lock.RLock()
	defer idx.lock.RUnlock()

	// Always try to find an exact match first.
	if sb, ok := idx.exact.get(key); ok {
		return []Bindable{sb}
	}

	// Next look for inexact matches.
	if sb, ok := idx.inexact.get(inexactKey{
		Group:     key.Group,
		Kind:      key.Kind,
		Namespace: key.Namespace,
	}, labels); ok {
		return []Bindable{sb}
	}

	return nil
}
