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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

const (
	kind       = "SomeKind"
	apiVersion = "v1mega1"
	name       = "a-name"
	namespace  = "b-namespace"
)

var (
	testCert = `-----BEGIN CERTIFICATE-----
MIIDmjCCAoKgAwIBAgIUYzA4bTMXevuk3pl2Mn8hpCYL2C0wDQYJKoZIhvcNAQEL
BQAwLzELMAkGA1UEBhMCVVMxIDAeBgNVBAMMF0tuYXRpdmUtRXhhbXBsZS1Sb290
LUNBMB4XDTIzMDQwNTEzMTUyNFoXDTI2MDEyMzEzMTUyNFowbTELMAkGA1UEBhMC
VVMxEjAQBgNVBAgMCVlvdXJTdGF0ZTERMA8GA1UEBwwIWW91ckNpdHkxHTAbBgNV
BAoMFEV4YW1wbGUtQ2VydGlmaWNhdGVzMRgwFgYDVQQDDA9sb2NhbGhvc3QubG9j
YWwwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC5teo+En6U5nhqn7Sc
uanqswUmPlgs9j/8l21Rhb4T+ezlYKGQGhbJyFFMuiCE1Rjn8bpCwi7Nnv12Y2nz
FhEv2Jx0yL3Tqx0Q593myqKDq7326EtbO7wmDT0XD03twH5i9XZ0L0ihPWn1mjUy
WxhnHhoFpXrsnQECJorZY6aTrFbGVYelIaj5AriwiqyL0fET8pueI2GwLjgWHFSH
X8XsGAlcLUhkQG0Z+VO9usy4M1Wpt+cL6cnTiQ+sRmZ6uvaj8fKOT1Slk/oUeAi4
WqFkChGzGzLik0QrhKGTdw3uUvI1F2sdQj0GYzXaWqRz+tP9qnXdzk1GrszKKSlm
WBTLAgMBAAGjcDBuMB8GA1UdIwQYMBaAFJJcCftus4vj98N0zQQautsjEu82MAkG
A1UdEwQCMAAwCwYDVR0PBAQDAgTwMBQGA1UdEQQNMAuCCWxvY2FsaG9zdDAdBgNV
HQ4EFgQUnu/3vqA3VEzm128x/hLyZzR9JlgwDQYJKoZIhvcNAQELBQADggEBAFc+
1cKt/CNjHXUsirgEhry2Mm96R6Yxuq//mP2+SEjdab+FaXPZkjHx118u3PPX5uTh
gTT7rMfka6J5xzzQNqJbRMgNpdEFH1bbc11aYuhi0khOAe0cpQDtktyuDJQMMv3/
3wu6rLr6fmENo0gdcyUY9EiYrglWGtdXhlo4ySRY8UZkUScG2upvyOhHTxVCAjhP
efbMkNjmDuZOMK+wqanqr5YV6zMPzkQK7DspfRgasMAQmugQu7r2MZpXg8Ilhro1
s/wImGnMVk5RzpBVrq2VB9SkX/ThTVYEC/Sd9BQM364MCR+TA1l8/ptaLFLuwyw8
O2dgzikq8iSy1BlRsVw=
-----END CERTIFICATE-----`

	csr = `-----BEGIN CERTIFICATE REQUEST-----
MIICvDCCAaQCAQAwdzELMAkGA1UEBhMCVVMxDTALBgNVBAgMBFV0YWgxDzANBgNV
BAcMBkxpbmRvbjEWMBQGA1UECgwNRGlnaUNlcnQgSW5jLjERMA8GA1UECwwIRGln
aUNlcnQxHTAbBgNVBAMMFGV4YW1wbGUuZGlnaWNlcnQuY29tMIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEA8+To7d+2kPWeBv/orU3LVbJwDrSQbeKamCmo
wp5bqDxIwV20zqRb7APUOKYoVEFFOEQs6T6gImnIolhbiH6m4zgZ/CPvWBOkZc+c
1Po2EmvBz+AD5sBdT5kzGQA6NbWyZGldxRthNLOs1efOhdnWFuhI162qmcflgpiI
WDuwq4C9f+YkeJhNn9dF5+owm8cOQmDrV8NNdiTqin8q3qYAHHJRW28glJUCZkTZ
wIaSR6crBQ8TbYNE0dc+Caa3DOIkz1EOsHWzTx+n0zKfqcbgXi4DJx+C1bjptYPR
BPZL8DAeWuA8ebudVT44yEp82G96/Ggcf7F33xMxe0yc+Xa6owIDAQABoAAwDQYJ
KoZIhvcNAQEFBQADggEBAB0kcrFccSmFDmxox0Ne01UIqSsDqHgL+XmHTXJwre6D
hJSZwbvEtOK0G3+dr4Fs11WuUNt5qcLsx5a8uk4G6AKHMzuhLsJ7XZjgmQXGECpY
Q4mC3yT3ZoCGpIXbw+iP3lmEEXgaQL0Tx5LFl/okKbKYwIqNiyKWOMj7ZR/wxWg/
ZDGRs55xuoeLDJ/ZRFf9bI+IaCUd1YrfYcHIl3G87Av+r49YVwqRDT0VDV7uLgqn
29XI1PpVUNCPQGn9p/eX6Qo7vpDaPybRtA2R7XLKjQaF9oXWeCUqy1hvJac9QFO2
97Ob1alpHPoZ7mWiEuJwjBPii6a9M9G30nUo39lBi1w=
-----END CERTIFICATE REQUEST-----`

	invaidCert = "certificate"
)

func TestValidateDestination(t *testing.T) {
	ctx := context.Background()

	validRef := KReference{
		Kind:       kind,
		APIVersion: apiVersion,
		Name:       name,
		Namespace:  namespace,
	}

	validURL := apis.URL{
		Scheme: "http",
		Host:   "host",
	}

	tests := map[string]struct {
		dest *Destination
		want string
	}{"nil valid": {
		dest: nil,
	}, "valid ref": {
		dest: &Destination{
			Ref: &validRef,
		},
	}, "invalid ref, missing name": {
		dest: &Destination{
			Ref: &KReference{
				Namespace:  namespace,
				Kind:       kind,
				APIVersion: apiVersion,
			},
		},
		want: "missing field(s): ref.name",
	}, "invalid ref, missing api version": {
		dest: &Destination{
			Ref: &KReference{
				Namespace: namespace,
				Kind:      kind,
				Name:      name,
			},
		},
		want: "missing field(s): ref.apiVersion",
	}, "invalid ref, missing kind": {
		dest: &Destination{
			Ref: &KReference{
				Namespace:  namespace,
				APIVersion: apiVersion,
				Name:       name,
			},
		},
		want: "missing field(s): ref.kind",
	}, "valid uri": {
		dest: &Destination{
			URI: &validURL,
		},
	}, "invalid, uri has no host": {
		dest: &Destination{
			URI: &apis.URL{
				Scheme: "http",
			},
		},
		want: "invalid value: Relative URI is not allowed when Ref and [apiVersion, kind, name] is absent: uri",
	}, "invalid, uri is not absolute url": {
		dest: &Destination{
			URI: &apis.URL{
				Host: "host",
			},
		},
		want: "invalid value: Relative URI is not allowed when Ref and [apiVersion, kind, name] is absent: uri",
	}, "invalid, both uri and ref, uri is absolute URL": {
		dest: &Destination{
			URI: &validURL,
			Ref: &validRef,
		},
		want: "Absolute URI is not allowed when Ref or [apiVersion, kind, name] is present: [apiVersion, kind, name], ref, uri",
	}, "invalid, both ref, [apiVersion, kind, name] and uri  are nil": {
		dest: &Destination{},
		want: "expected at least one, got none: ref, uri",
	}, "valid, both uri and ref, uri is not a absolute URL": {
		dest: &Destination{
			URI: &apis.URL{
				Path: "/handler",
			},
			Ref: &validRef,
		},
	}, "valid, CACert is valid": {
		dest: &Destination{
			URI: &apis.URL{
				Path: "/handler",
			},
			Ref:     &validRef,
			CACerts: &testCert,
		},
	}, "invalid,CACert is invalid": {
		dest: &Destination{
			URI: &apis.URL{
				Path: "/handler",
			},
			Ref:     &validRef,
			CACerts: &invaidCert,
		},
		want: "invalid value: CA Cert provided is invalid: caCert",
	}, "invalid,CSR provided not CA Cert": {
		dest: &Destination{
			URI: &apis.URL{
				Path: "/handler",
			},
			Ref:     &validRef,
			CACerts: &csr,
		},
		want: "invalid value: CA Cert provided is not a certificate: caCert",
	}}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotErr := tc.dest.Validate(ctx)

			if tc.want != "" {
				if got, want := gotErr.Error(), tc.want; got != want {
					t.Errorf("%s: Error() = %v, wanted %v", name, got, want)
				}
			} else if gotErr != nil {
				t.Errorf("%s: Validate() = %v, wanted nil", name, gotErr)
			}
		})
	}
}

func TestDestinationGetRef(t *testing.T) {
	ref := &KReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
	}
	tests := map[string]struct {
		dest *Destination
		want *KReference
	}{"nil destination": {
		dest: nil,
		want: nil,
	}, "uri": {
		dest: &Destination{
			URI: &apis.URL{
				Host: "foo",
			},
		},
		want: nil,
	}, "ref": {
		dest: &Destination{
			Ref: ref,
		},
		want: ref,
	}}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			got := tc.dest.GetRef()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Error("Unexpected result (-want +got):", diff)
			}
		})
	}
}

func TestDestinationSetDefaults(t *testing.T) {
	ctx := context.Background()

	const parentNamespace = "parentNamespace"

	tests := map[string]struct {
		d    *Destination
		ctx  context.Context
		want string
	}{"destination nil ": {
		d:   nil,
		ctx: ctx,
	}, "uri set, nothing in ref, not modified ": {
		d:   &Destination{URI: apis.HTTP("example.com")},
		ctx: ctx,
	}, "namespace set, nothing in context, not modified ": {
		d:    &Destination{Ref: &KReference{Namespace: namespace}},
		ctx:  ctx,
		want: namespace,
	}, "namespace set, context set, not modified ": {
		d:    &Destination{Ref: &KReference{Namespace: namespace}},
		ctx:  apis.WithinParent(ctx, metav1.ObjectMeta{Namespace: parentNamespace}),
		want: namespace,
	}, "namespace set, uri set, context set, not modified ": {
		d:    &Destination{Ref: &KReference{Namespace: namespace}, URI: apis.HTTP("example.com")},
		ctx:  apis.WithinParent(ctx, metav1.ObjectMeta{Namespace: parentNamespace}),
		want: namespace,
	}, "namespace not set, context set, defaulted": {
		d:    &Destination{Ref: &KReference{}},
		ctx:  apis.WithinParent(ctx, metav1.ObjectMeta{Namespace: parentNamespace}),
		want: parentNamespace,
	}, "namespace not set, uri set, context set, defaulted": {
		d:    &Destination{Ref: &KReference{}, URI: apis.HTTP("example.com")},
		ctx:  apis.WithinParent(ctx, metav1.ObjectMeta{Namespace: parentNamespace}),
		want: parentNamespace,
	}}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.d.SetDefaults(tc.ctx)
			if tc.d != nil {
				if tc.d.Ref != nil && tc.d.Ref.Namespace != tc.want {
					t.Errorf("Got: %s wanted %s", tc.d.Ref.Namespace, tc.want)
				}
				if tc.d.Ref == nil && tc.want != "" {
					t.Error("Got: nil Ref wanted", tc.want)
				}
			}
		})
	}
}
