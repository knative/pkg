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

package v1alpha1

import (
	"testing"

	"github.com/knative/pkg/apis"
	"github.com/knative/pkg/apis/duck/v1beta1"
)

func TestGetURL(t *testing.T) {
	tests := []struct {
		name string
		addr Addressable
		want apis.URL
	}{{
		name: "just hostname",
		addr: Addressable{
			Hostname: "foo.com",
		},
		want: apis.URL{
			Scheme: "http",
			Host:   "foo.com",
		},
	}, {
		name: "just url",
		addr: Addressable{
			Addressable: v1beta1.Addressable{
				URL: &apis.URL{
					Scheme: "https",
					Host:   "bar.com",
				},
			},
		},
		want: apis.URL{
			Scheme: "https",
			Host:   "bar.com",
		},
	}, {
		name: "both fields",
		addr: Addressable{
			Hostname: "foo.bar.svc.cluster.local",
			Addressable: v1beta1.Addressable{
				URL: &apis.URL{
					Scheme: "https",
					Host:   "baz.com",
				},
			},
		},
		want: apis.URL{
			Scheme: "https",
			Host:   "baz.com",
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.addr.GetURL()
			if got.String() != test.want.String() {
				t.Errorf("GetURL() = %v, wanted %v", test.want, got)
			}
		})
	}
}
