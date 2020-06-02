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

func TestChecksum(t *testing.T) {
	tests := []struct {
		in   string
		want uint32
	}{{
		in:   "",
		want: 0,
	}, {
		in:   "1",
		want: 2212294583,
	}, {
		in:   "a somewhat longer test",
		want: 4278087538,
	}}

	for _, test := range tests {
		if got := Checksum(test.in); got != test.want {
			t.Errorf("Checksum(%q) = %d, want %d", test.in, got, test.want)
		}
	}
}
