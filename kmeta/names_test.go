/*
copyright 2019 the knative authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/

package kmeta

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestChildName(t *testing.T) {
	tests := []struct {
		parent string
		suffix string
		want   string
	}{{
		parent: "asdf",
		suffix: "-deployment",
		want:   "asdf-deployment",
	}, {
		parent: strings.Repeat("f", 63),
		suffix: "-deployment",
		want:   "ffffffffffffffffffff105d7597f637e83cc711605ac3ea4957-deployment",
	}, {
		parent: strings.Repeat("f", 63),
		suffix: "-deploy",
		want:   "ffffffffffffffffffffffff105d7597f637e83cc711605ac3ea4957-deploy",
	}}

	for _, test := range tests {
		if got, want := ChildName(test.parent, test.suffix), test.want; got != want {
			t.Errorf("%s-%s: got: %63s want: %63s\ndiff:%s", test.parent, test.suffix, got, want, cmp.Diff(got, want))
		}
	}
}
