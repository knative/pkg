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
	"encoding/json"
	"testing"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

type testConfig struct {
	str    string
	toggle bool
	i16    int16
	i32    int32
	i64    int64
	u16    uint16
	u32    uint32
	i      int
	f64    float64
	dur    time.Duration
	set    sets.String
	qua    *resource.Quantity

	nsn  types.NamespacedName
	onsn *types.NamespacedName

	overlay jsonpatch.Patch

	dict map[string]string
}

func TestParse(t *testing.T) {
	op := []byte("\"add\"")
	opMsg := json.RawMessage(op)
	path := []byte("\"/metadata/labels\"")
	pathMsg := json.RawMessage(path)
	value := []byte("{ \"over1\": \"yes1\" }")
	valueMsg := json.RawMessage(value)
	patch := jsonpatch.Operation{"op": &opMsg, "path": &pathMsg, "value": &valueMsg}

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
			"test-string":   "foo.bar",
			"test-bool":     "true",
			"test-int16":    "6",
			"test-int32":    "1",
			"test-int64":    "2",
			"test-uint16":   "5",
			"test-uint32":   "3",
			"test-int":      "4",
			"test-float64":  "1.0",
			"test-duration": "1m",
			"test-set":      "a,b,c, d",
			"test-quantity": "500m",

			"test-namespaced-name":          "some-namespace/some-name",
			"test-optional-namespaced-name": "some-other-namespace/some-other-name",

			"test-overlay": "[{\"op\": \"add\", \"path\": \"/metadata/labels\", \"value\": { \"over1\": \"yes1\" }}]",

			"test-dict.k":  "v",
			"test-dict.k1": "v1",
		},
		want: testConfig{
			str:    "foo.bar",
			toggle: true,
			i16:    6,
			i32:    1,
			i64:    2,
			u16:    5,
			u32:    3,
			f64:    1.0,
			i:      4,
			dur:    time.Minute,
			set:    sets.NewString("a", "b", "c", "d"),
			qua:    &fiveHundredM,
			nsn: types.NamespacedName{
				Name:      "some-name",
				Namespace: "some-namespace",
			},
			onsn: &types.NamespacedName{
				Name:      "some-other-name",
				Namespace: "some-other-namespace",
			},
			//	overlay: jsonpatch.Patch{{"op": json.RawMessage([]byte("add")), "path": "/metadata/labels", "value": "{ \"over1\": \"yes1\" }"}},
			overlay: jsonpatch.Patch{patch},
			dict: map[string]string{
				"k":  "v",
				"k1": "v1",
			},
		},
	}, {
		name: "respect defaults",
		conf: testConfig{
			str:    "foo.bar",
			toggle: true,
			i32:    1,
			i64:    2,
			f64:    1.0,
			i:      4,
			dur:    time.Minute,
			qua:    &fiveHundredM,
		},
		want: testConfig{
			str:    "foo.bar",
			toggle: true,
			i32:    1,
			i64:    2,
			f64:    1.0,
			i:      4,
			dur:    time.Minute,
			qua:    &fiveHundredM,
		},
	}, {
		name: "junk bool fails",
		data: map[string]string{
			"test-bool": "foo",
		},
		expectErr: true,
	}, {
		name: "int32 error",
		data: map[string]string{
			"test-int32": "foo",
		},
		expectErr: true,
	}, {
		name: "int64 error",
		data: map[string]string{
			"test-int64": "foo",
		},
		expectErr: true,
	}, {
		name: "uint32 error",
		data: map[string]string{
			"test-uint32": "foo",
		},
		expectErr: true,
	}, {
		name: "int error",
		data: map[string]string{
			"test-int": "foo",
		},
		expectErr: true,
	}, {
		name: "float64 error",
		data: map[string]string{
			"test-float64": "foo",
		},
		expectErr: true,
	}, {
		name: "duration error",
		data: map[string]string{
			"test-duration": "foo",
		},
		expectErr: true,
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
		name: "types.Overlay",
		data: map[string]string{
			"test-overlay": "",
		},
		expectErr: false,
	}, {
		name: "types.Overlay",
		data: map[string]string{
			"test-overlay": "   ",
		},
		expectErr: false,
	}, {
		name: "types.Overlay",
		data: map[string]string{
			"test-overlay": "  aaa ",
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
				AsString("test-string", &test.conf.str),
				AsBool("test-bool", &test.conf.toggle),
				AsInt16("test-int16", &test.conf.i16),
				AsInt32("test-int32", &test.conf.i32),
				AsInt64("test-int64", &test.conf.i64),
				AsUint16("test-uint16", &test.conf.u16),
				AsUint32("test-uint32", &test.conf.u32),
				AsInt("test-int", &test.conf.i),
				AsFloat64("test-float64", &test.conf.f64),
				AsDuration("test-duration", &test.conf.dur),
				AsStringSet("test-set", &test.conf.set),
				AsQuantity("test-quantity", &test.conf.qua),
				AsNamespacedName("test-namespaced-name", &test.conf.nsn),
				AsOptionalNamespacedName("test-optional-namespaced-name", &test.conf.onsn),
				AsJSONPatch("test-overlay", &test.conf.overlay),
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
