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

package sharedmain

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"knative.dev/pkg/injection"
)

func TestEnabledControllers(t *testing.T) {
	tests := []struct {
		name                string
		disabledControllers []string
		ctors               []injection.NamedControllerConstructor
		wantNames           []string
	}{
		{
			name:                "zero",
			disabledControllers: []string{"foo"},
			ctors:               []injection.NamedControllerConstructor{{Name: "bar"}},
			wantNames:           []string{"bar"},
		},
		{
			name:                "one",
			disabledControllers: []string{"foo"},
			ctors:               []injection.NamedControllerConstructor{{Name: "foo"}},
			wantNames:           []string{},
		},
		{
			name:                "two",
			disabledControllers: []string{"foo"},
			ctors: []injection.NamedControllerConstructor{
				{Name: "foo"},
				{Name: "bar"},
			},
			wantNames: []string{"bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enabledControllers(tt.disabledControllers, tt.ctors)
			if diff := cmp.Diff(tt.wantNames, namesOf(got)); diff != "" {
				t.Error("(-want, +got)", diff)
			}
		})
	}
}

func namesOf(ctors []injection.NamedControllerConstructor) []string {
	names := make([]string, 0, len(ctors))
	for _, x := range ctors {
		names = append(names, x.Name)
	}
	return names
}
