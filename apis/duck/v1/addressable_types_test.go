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

package v1

import (
	"testing"

	"knative.dev/pkg/apis"
)

func TestBetaConversion(t *testing.T) {
	tests := []struct {
		name string
		addr Addressable
		want apis.URL
	}{{
		name: "just url",
		addr: Addressable{
			URL: &apis.URL{
				Scheme: "https",
				Host:   "bar.com",
			},
		},
		want: apis.URL{
			Scheme: "https",
			Host:   "bar.com",
		},
	}, {
		name: "url with path",
		addr: Addressable{
			URL: &apis.URL{
				Scheme: "https",
				Host:   "bar.com",
				Path:   "/v1",
			},
		},
		want: apis.URL{
			Scheme: "https",
			Host:   "bar.com",
			Path:   "/v1",
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beta := test.addr.ToBeta()
			gotBeta := beta.URL
			gotRoundtrip := FromBeta(beta).URL

			if test.want.String() != gotBeta.String() {
				t.Errorf("v1beta1.URL = %v, wanted %v", gotBeta, test.want)
			}
			if test.want.String() != gotRoundtrip.String() {
				t.Errorf("rountrip v1.URL = %v, wanted %v", gotRoundtrip, test.want)
			}
		})
	}
}

func TestAlphaConversion(t *testing.T) {
	tests := []struct {
		name string
		addr Addressable
		want apis.URL
	}{{
		name: "just url",
		addr: Addressable{
			URL: &apis.URL{
				Scheme: "https",
				Host:   "bar.com",
			},
		},
		want: apis.URL{
			Scheme: "https",
			Host:   "bar.com",
		},
	}, {
		name: "url with path",
		addr: Addressable{
			URL: &apis.URL{
				Scheme: "https",
				Host:   "bar.com",
				Path:   "/v1",
			},
		},
		want: apis.URL{
			Scheme: "https",
			Host:   "bar.com",
			Path:   "/v1",
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			alpha := test.addr.ToAlpha()
			gotAlpha := alpha.GetURL()
			gotRoundtrip := FromAlpha(alpha).URL

			if test.want.String() != gotAlpha.String() {
				t.Errorf("v1alpha1.GetURL() = %v, wanted %v", gotAlpha, test.want)
			}
			if test.want.String() != gotRoundtrip.String() {
				t.Errorf("roundtrip v1.GetURL() = %v, wanted %v", gotRoundtrip, test.want)
			}
		})
	}
}
