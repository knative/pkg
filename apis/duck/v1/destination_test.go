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
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

const (
	kind       = "SomeKind"
	apiVersion = "v1mega1"
	name       = "a-name"
)

func TestValidateDestination(t *testing.T) {
	ctx := context.TODO()

	validRef := corev1.ObjectReference{
		Kind:       kind,
		APIVersion: apiVersion,
		Name:       name,
	}

	validURL := apis.URL{
		Scheme: "http",
		Host:   "host",
	}

	tests := map[string]struct {
		dest *Destination
		want string
	}{
		"nil valid": {
			dest: nil,
			want: "",
		},
		"valid ref": {
			dest: &Destination{
				Ref: &validRef,
			},
			want: "",
		},
		"invalid ref, missing name": {
			dest: &Destination{
				Ref: &corev1.ObjectReference{
					Kind:       kind,
					APIVersion: apiVersion,
				},
			},
			want: "missing field(s): ref.name",
		},
		"invalid ref, missing api version": {
			dest: &Destination{
				Ref: &corev1.ObjectReference{
					Kind: kind,
					Name: apiVersion,
				},
			},
			want: "missing field(s): ref.apiVersion",
		},
		"invalid ref, missing kind": {
			dest: &Destination{
				Ref: &corev1.ObjectReference{
					APIVersion: apiVersion,
					Name:       name,
				},
			},
			want: "missing field(s): ref.kind",
		},
		"valid uri": {
			dest: &Destination{
				URI: &validURL,
			},
		},
		"invalid, uri has no host": {
			dest: &Destination{
				URI: &apis.URL{
					Scheme: "http",
				},
			},
			want: "invalid value: Relative URI is not allowed when Ref and [apiVersion, kind, name] is absent: uri",
		},
		"invalid, uri is not absolute url": {
			dest: &Destination{
				URI: &apis.URL{
					Host: "host",
				},
			},
			want: "invalid value: Relative URI is not allowed when Ref and [apiVersion, kind, name] is absent: uri",
		},
		"invalid, both uri and ref, uri is absolute URL": {
			dest: &Destination{
				URI: &validURL,
				Ref: &validRef,
			},
			want: "Absolute URI is not allowed when Ref or [apiVersion, kind, name] is present: [apiVersion, kind, name], ref, uri",
		},
		"invalid, both ref, [apiVersion, kind, name] and uri  are nil": {
			dest: &Destination{},
			want: "expected at least one, got none: ref, uri",
		},
		"valid, both uri and ref, uri is not a absolute URL": {
			dest: &Destination{
				URI: &apis.URL{
					Path: "/handler",
				},
				Ref: &validRef,
			},
		},
	}

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

func TestDestination_GetRef(t *testing.T) {
	ref := &corev1.ObjectReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       name,
	}
	tests := map[string]struct {
		dest *Destination
		want *corev1.ObjectReference
	}{
		"nil destination": {
			dest: nil,
			want: nil,
		},
		"uri": {
			dest: &Destination{
				URI: &apis.URL{
					Host: "foo",
				},
			},
			want: nil,
		},
		"ref": {
			dest: &Destination{
				Ref: ref,
			},
			want: ref,
		},
	}

	for n, tc := range tests {
		t.Run(n, func(t *testing.T) {
			got := tc.dest.GetRef()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Unexpected result (-want +got): %s", diff)
			}
		})
	}
}
