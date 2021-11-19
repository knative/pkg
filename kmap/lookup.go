/*
Copyright 2021 The Knative Authors

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

package kmap

// OrderedLookup is a utility struct for getting values from a map
// given a list of ordered keys
//
// This is to help the migration/renaming of annotations & labels
type OrderedLookup struct {
	Keys []string
}

// Key returns the default key that should be used for
// accessing the map
func (a *OrderedLookup) Key() string {
	return a.Keys[0]
}

// Value iterates looks up the ordered keys in the map and returns
// a string value. An empty string will be returned if the keys
// are not present in the map
func (a *OrderedLookup) Value(m map[string]string) string {
	_, v, _ := a.Get(m)
	return v
}

// Get iterates over the ordered keys and looks up the corresponding
// values in the map
//
// It returns the key, value, and true|false signaling whether the
// key was present in the map
//
// If no key is present the default key (lowest ordinal) is returned
// with an empty string as the value
func (a *OrderedLookup) Get(m map[string]string) (string, string, bool) {
	var k, v string
	var ok bool
	for _, k = range a.Keys {
		v, ok = m[k]
		if ok {
			return k, v, ok
		}
	}

	return a.Keys[0], "", false
}

// NewOrderedLookup builds a utilty struct for looking up N keys
// in a map in a specific order
//
// If no keys are supplied this method will panic
func NewOrderedLookup(keys ...string) *OrderedLookup {
	if len(keys) == 0 {
		panic("expected to have at least a single key")
	}
	return &OrderedLookup{Keys: keys}
}
