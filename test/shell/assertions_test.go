/*
Copyright 2020 The Knative Authors

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

package shell_test

import (
	"strings"
	"testing"
)

type assertions struct {
	t *testing.T
}

func (a assertions) NoError(err error) {
	if err != nil {
		a.t.Error(err)
	}
}

func (a assertions) Contains(haystack, needle string) {
	if !strings.Contains(haystack, needle) {
		a.t.Errorf("wanted to \ncontain: %#v\n     in: %#v",
			needle, haystack)
	}
}

func (a assertions) Equal(want, got string) {
	if got != want {
		a.t.Errorf("want: %#v\n got:%#v", want, got)
	}
}
