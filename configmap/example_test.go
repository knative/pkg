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

package configmap

import "testing"

func TestExampleHash(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{{
		in:   "",
		want: "e3b0c4429",
	}, {
		in:   "1",
		want: "6b86b273f",
	}, {
		in:   "a somewhat longer test",
		want: "2b55bd0be",
	}}

	for _, test := range tests {
		if got := ExampleHash(test.in); got != test.want {
			t.Errorf("ExampleHash(%q) = %s, want %s", test.in, got, test.want)
		}
	}
}
