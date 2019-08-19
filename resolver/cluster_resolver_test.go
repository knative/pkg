/*
Copyright 2019 The Knative Authors

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

package resolver

import (
	"strings"
	"testing"
)

func TestReadClusterDomainName(t *testing.T) {
	testCases := map[string]struct {
		resolvConf string
		want       string
	}{
		"all good": {
			resolvConf: `
nameserver 1.1.1.1
search default.svc.abc.com svc.abc.com abc.com
options ndots:5
`,
			want: "abc.com",
		},
		"missing search line": {
			resolvConf: `
nameserver 1.1.1.1
options ndots:5
`,
			want: defaultDomainName,
		},
		"non k8s resolv.conf format": {
			resolvConf: `
nameserver 1.1.1.1
search  abc.com xyz.com
options ndots:5
`,
			want: defaultDomainName,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			got := readClusterDomainName(strings.NewReader(tc.resolvConf))
			if got != tc.want {
				t.Errorf("Expected: %s but got: %s", tc.want, got)
			}
		})
	}
}

func TestNames(t *testing.T) {
	testCases := []struct {
		Name string
		F    func() string
		Want string
	}{{
		Name: "ServiceHostName",
		F: func() string {
			return ServiceHostName("foo", "namespace")
		},
		Want: "foo.namespace.svc." + ClusterDomainName(),
	}}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if got := tc.F(); got != tc.Want {
				t.Errorf("want %v, got %v", tc.Want, got)
			}
		})
	}
}
