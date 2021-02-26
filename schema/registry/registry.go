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

package registry

import (
	"reflect"
)

type Registry struct {
	// easy for now
	kinds map[string]reflect.Type
}

var r = &Registry{
	kinds: map[string]reflect.Type{},
}

func Register(kind string, obj interface{}) {
	t := reflect.TypeOf(obj)
	r.kinds[kind] = t
}

func Kinds() []string {
	kinds := make([]string, 0)
	for k, _ := range r.kinds {
		kinds = append(kinds, k)
	}
	return kinds
}

func TypeFor(kind string) reflect.Type {
	return r.kinds[kind]
}
