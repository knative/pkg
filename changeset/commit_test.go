/*
Copyright 2022 The Knative Authors

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

package changeset

import (
	"runtime/debug"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGet(t *testing.T) {

	cases := []struct {
		name    string
		info    *debug.BuildInfo
		ok      bool
		result  string
		wantErr string
	}{{
		name:    "info fails",
		ok:      false,
		wantErr: "unable to read build info",
	}, {
		name:   "missing revision",
		ok:     true,
		info:   &debug.BuildInfo{},
		result: Unknown,
	}, {
		name: "SHA1 revision is truncated",
		ok:   true,
		info: &debug.BuildInfo{
			Settings: []debug.BuildSetting{{
				Key: "vcs.revision", Value: "3666ce749d32abe7be0528380c8c05a4282cb733",
			}},
		},
		result: "3666ce7",
	}, {
		name: "SHA256 revision is truncated",
		ok:   true,
		info: &debug.BuildInfo{
			Settings: []debug.BuildSetting{{
				Key: "vcs.revision", Value: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			}},
		},
		result: "2cf24db",
	}, {
		name: "modified workspace results in -dirty suffix",
		ok:   true,
		info: &debug.BuildInfo{
			Settings: []debug.BuildSetting{{
				Key: "vcs.revision", Value: "3666ce749d32abe7be0528380c8c05a4282cb733",
			}, {
				Key: "vcs.modified", Value: "true",
			}},
		},
		result: "3666ce7-dirty",
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			once = sync.Once{}
			readBuildInfo = func() (info *debug.BuildInfo, ok bool) {
				return c.info, c.ok
			}

			val, err := Get()
			if c.wantErr == "" && err != nil {
				t.Fatal("unexpected error", err)
			} else if c.wantErr != "" && err != nil {
				if diff := cmp.Diff(c.wantErr, err.Error()); diff != "" {
					t.Fatalf("error doesn't match expected: %s", diff)
				}
			} else if c.wantErr != "" && err == nil {
				t.Fatalf("expected error %q but was nil", c.wantErr)
			}

			if diff := cmp.Diff(c.result, val); diff != "" {
				t.Errorf("result doesn't match expected: %s", diff)
			}
		})
	}
}
