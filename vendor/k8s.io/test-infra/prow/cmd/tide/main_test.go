/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"flag"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/flagutil"
)

func Test_gatherOptions(t *testing.T) {
	cases := []struct {
		name     string
		args     map[string]string
		del      sets.String
		expected func(*options)
		err      bool
	}{
		{
			name: "minimal flags work",
		},
		{
			name: "explicitly set --config-path",
			args: map[string]string{
				"--config-path": "/random/value",
			},
			expected: func(o *options) {
				o.configPath = "/random/value"
			},
		},
		{
			name: "empty config-path defaults to old value",
			args: map[string]string{
				"--config-path": "",
			},
			expected: func(o *options) {
				o.configPath = config.DefaultConfigPath
			},
		},
		{
			name: "expicitly set --dry-run=false",
			args: map[string]string{
				"--dry-run": "false",
			},
			expected: func(o *options) {
				o.dryRun = false
			},
		},
		{
			name: "--dry-run=true requires --deck-url",
			args: map[string]string{
				"--dry-run":  "true",
				"--deck-url": "",
			},
			err: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expected := &options{
				port:              8888,
				configPath:        "yo",
				jobConfigPath:     "",
				dryRun:            true,
				syncThrottle:      800,
				statusThrottle:    400,
				maxRecordsPerPool: 1000,
				github:            flagutil.GitHubOptions{},
				kubernetes:        flagutil.ExperimentalKubernetesOptions{DeckURI: "http://whatever"},
			}
			expectedfs := flag.NewFlagSet("fake-flags", flag.PanicOnError)
			expected.github.AddFlags(expectedfs)
			if tc.expected != nil {
				tc.expected(expected)
			}

			argMap := map[string]string{
				"--config-path": "yo",
				"--deck-url":    "http://whatever",
			}
			for k, v := range tc.args {
				argMap[k] = v
			}
			for k := range tc.del {
				delete(argMap, k)
			}

			var args []string
			for k, v := range argMap {
				args = append(args, k+"="+v)
			}
			fs := flag.NewFlagSet("fake-flags", flag.PanicOnError)
			actual := gatherOptions(fs, args...)
			switch err := actual.Validate(); {
			case err != nil:
				if !tc.err {
					t.Errorf("unexpected error: %v", err)
				}
			case tc.err:
				t.Errorf("failed to receive expected error")
			case !reflect.DeepEqual(*expected, actual):
				t.Errorf("%#v != expected %#v", actual, *expected)
			}
		})
	}
}
