/*
Copyright 2025 The Knative Authors

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

package parser

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type testConfig struct {
	str    string
	toggle bool
	i16    int16
	i32    int32
	i64    int64
	u16    uint16
	u32    uint32
	u64    uint64
	i      int
	ui     uint
	f32    float32
	f64    float64
	dur    time.Duration
}

func TestParse(t *testing.T) {
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
			"test-uint64":   "6",
			"test-int":      "4",
			"test-uint":     "5",
			"test-float32":  "1.2",
			"test-float64":  "1.0",
			"test-duration": "1m",
			"test-set":      "a,b,c, d",
			"test-quantity": "500m",

			"test-namespaced-name":          "some-namespace/some-name",
			"test-optional-namespaced-name": "some-other-namespace/some-other-name",

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
			u64:    6,
			f32:    1.2,
			f64:    1.0,
			i:      4,
			ui:     5,
			dur:    time.Minute,
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
		},
		want: testConfig{
			str:    "foo.bar",
			toggle: true,
			i32:    1,
			i64:    2,
			f64:    1.0,
			i:      4,
			dur:    time.Minute,
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
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Parse(test.data,
				As("test-string", &test.conf.str),
				As("test-bool", &test.conf.toggle),
				As("test-int16", &test.conf.i16),
				As("test-int32", &test.conf.i32),
				As("test-int64", &test.conf.i64),
				As("test-uint16", &test.conf.u16),
				As("test-uint32", &test.conf.u32),
				As("test-uint64", &test.conf.u64),
				As("test-int", &test.conf.i),
				As("test-uint", &test.conf.ui),
				As("test-float32", &test.conf.f32),
				As("test-float64", &test.conf.f64),
				As("test-duration", &test.conf.dur),
			); (err == nil) == test.expectErr {
				t.Fatal("Failed to parse data:", err)
			}

			if diff := cmp.Diff(test.want, test.conf, cmp.AllowUnexported(testConfig{})); diff != "" {
				t.Fatal("(-want, +got)", diff)
			}
		})
	}
}
