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

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

type testConfig struct {
	set sets.Set[string]
	qua *resource.Quantity

	nsn  types.NamespacedName
	onsn *types.NamespacedName

	dict map[string]string
}

func TestParse(t *testing.T) {
	fiveHundredM := resource.MustParse("500m")
	tests := []struct {
		name      string
		conf      testConfig
		data      map[string]string
		want      testConfig
		expectErr bool
	}{{
		name: "all good",
		data: map[string]string{
			"test-set":      "a,b,c, d",
			"test-quantity": "500m",

			"test-namespaced-name":          "some-namespace/some-name",
			"test-optional-namespaced-name": "some-other-namespace/some-other-name",

			"test-dict.k":  "v",
			"test-dict.k1": "v1",
		},
		want: testConfig{
			set: sets.New("a", "b", "c", "d"),
			qua: &fiveHundredM,
			nsn: types.NamespacedName{
				Name:      "some-name",
				Namespace: "some-namespace",
			},
			onsn: &types.NamespacedName{
				Name:      "some-other-name",
				Namespace: "some-other-namespace",
			},
			dict: map[string]string{
				"k":  "v",
				"k1": "v1",
			},
		},
	}, {
		name: "respect defaults",
		conf: testConfig{
			qua: &fiveHundredM,
		},
		want: testConfig{
			qua: &fiveHundredM,
		},
	}, {
		name: "quantity error",
		data: map[string]string{
			"test-quantity": "foo",
		},
		expectErr: true,
	}, {
		name: "types.NamespacedName bad dns name error",
		data: map[string]string{
			"test-namespaced-name": "some.bad.name/blah.bad.name",
		},
		expectErr: true,
	}, {
		name: "types.NamespacedName bad segment count error",
		data: map[string]string{
			"test-namespaced-name": "default/resource/whut",
		},
		expectErr: true,
	}, {
		name: "dict without key and dot",
		data: map[string]string{
			"test-dict": "v",
		},
		expectErr: false,
	}, {
		name: "dict without key",
		data: map[string]string{
			"test-dict.": "v",
		},
		expectErr: false,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Parse(test.data,
				AsStringSet("test-set", &test.conf.set),
				AsQuantity("test-quantity", &test.conf.qua),
				AsNamespacedName("test-namespaced-name", &test.conf.nsn),
				AsOptionalNamespacedName("test-optional-namespaced-name", &test.conf.onsn),
				CollectMapEntriesWithPrefix("test-dict", &test.conf.dict),
			); (err == nil) == test.expectErr {
				t.Fatal("Failed to parse data:", err)
			}

			if diff := cmp.Diff(test.want, test.conf, cmp.AllowUnexported(testConfig{})); diff != "" {
				t.Fatal("(-want, +got)", diff)
			}
		})
	}
}
