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
	"time"
)

type testConfig struct {
	str string
	boo bool
	i32 int32
	i64 int64
	f64 float64
	dur time.Duration
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
			"test-int32":    "1",
			"test-int64":    "2",
			"test-float64":  "1.0",
			"test-duration": "1m",
		},
		want: testConfig{
			str: "foo.bar",
			boo: true,
			i32: 1,
			i64: 2,
			f64: 1.0,
			dur: time.Minute,
		},
	}, {
		name: "respect defaults",
		conf: testConfig{
			str: "foo.bar",
			boo: true,
			i32: 1,
			i64: 2,
			f64: 1.0,
			dur: time.Minute,
		},
		want: testConfig{
			str: "foo.bar",
			boo: true,
			i32: 1,
			i64: 2,
			f64: 1.0,
			dur: time.Minute,
		},
	}, {
		name: "bool defaults to false",
		data: map[string]string{
			"test-bool": "foo",
		},
		want: testConfig{
			boo: false,
		},
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
				AsString("test-string", &test.conf.str),
				AsBool("test-bool", &test.conf.boo),
				AsInt32("test-int32", &test.conf.i32),
				AsInt64("test-int64", &test.conf.i64),
				AsFloat64("test-float64", &test.conf.f64),
				AsDuration("test-duration", &test.conf.dur),
			); (err == nil) == test.expectErr {
				t.Fatal("Failed to parse data:", err)
			}

			if test.conf != test.want {
				t.Fatalf("parsed = %v, want %v", test.conf, test.want)
			}
		})
	}
}
