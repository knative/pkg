/*
Copyright 2021 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kmap

// Accessor is a utility struct for getting values from a map
// given a list of keys
//
// This is to help the migration/renaming of annotations & labels
type Accessor struct {
	// Access will be done in order
	Keys []string
}

// Key returns the default key that should be used for
// accessing the map
func (a *Accessor) Key() string {
	return a.Keys[0]
}

// Value returns maps value for the Accessor's keys
func (a *Accessor) Value(m map[string]string) string {
	_, v, _ := a.Get(m)
	return v
}

func (a *Accessor) Get(m map[string]string) (string, string, bool) {
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

func NewAccessor(keys ...string) *Accessor {
	if len(keys) == 0 {
		panic("expected to have at least a single key")
	}
	return &Accessor{Keys: keys}
}
