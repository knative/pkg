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

package upgrade_test

import (
	"reflect"
	"strings"
	"testing"
)

type assertions struct {
	t *testing.T
}

func (a assertions) textContains(haystack string, needles texts) {
	for _, needle := range needles.elms {
		if !strings.Contains(haystack, needle) {
			a.t.Errorf(
				"expected %q is not in: %q",
				needle, haystack,
			)
		}
	}
}

func (a assertions) arraysEqual(actual []string, expected []string) {
	if !reflect.DeepEqual(actual, expected) {
		a.t.Errorf("arrays differ:\n  actual: %#v\nexpected: %#v", actual, expected)
	}
}
