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
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/ptr"
)

func TestValidateDestination(t *testing.T) {
	ctx := context.TODO()

	validRef := corev1.ObjectReference{
		Kind:       "SomeKind",
		APIVersion: "v1mega1",
		Name:       "a-name",
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
				ObjectReference: &validRef,
			},
		},
		"invalid ref, missing name": {
			dest: &Destination{
				ObjectReference: &corev1.ObjectReference{
					Kind:       "SomeKind",
					APIVersion: "v1mega1",
				},
			},
			want: "missing field(s): name",
		},
		"invalid ref, missing api version": {
			dest: &Destination{
				ObjectReference: &corev1.ObjectReference{
					Kind: "SomeKind",
					Name: "a-name",
				},
			},
			want: "missing field(s): apiVersion",
		},
		"invalid ref, missing kind": {
			dest: &Destination{
				ObjectReference: &corev1.ObjectReference{
					APIVersion: "v1mega1",
					Name:       "a-name",
				},
			},
			want: "missing field(s): kind",
		},
		"valid ref with path": {
			dest: &Destination{
				ObjectReference: &validRef,
				Path:            ptr.String("/a-path"),
			},
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
			want: "invalid value: http:: uri",
		},
		"invalid, uri has no scheme": {
			dest: &Destination{
				URI: &apis.URL{
					Host: "host",
				},
			},
			want: "invalid value: //host: uri",
		},
		"valid uri with path": {
			dest: &Destination{
				URI:  &validURL,
				Path: ptr.String("/a-path"),
			},
		},
		"invalid, both uri and ref": {
			dest: &Destination{
				URI:             &validURL,
				ObjectReference: &validRef,
			},
			want: "expected exactly one, got both: [apiVersion, kind, name], uri",
		},
		"invalid, just path": {
			dest: &Destination{
				Path: ptr.String("/a-path"),
			},
			want: "expected exactly one, got neither: [apiVersion, kind, name], uri",
		},
		"invalid, path without leading slash": {
			dest: &Destination{
				Path: ptr.String("a-path"),
			},
			want: `expected exactly one, got neither: [apiVersion, kind, name], uri
invalid value: a-path: path`,
		},
		"invalid, ref and path with query": {
			dest: &Destination{
				Path: ptr.String("/path?query"),
			},
			want: `expected exactly one, got neither: [apiVersion, kind, name], uri
invalid value: /path?query: path`,
		},
		"invalid, ref and path as uri": {
			dest: &Destination{
				ObjectReference: &validRef,
				Path:            ptr.String("http://host/path"),
			},
			want: "invalid value: http://host/path: path",
		},
		"invalid, uri and path with query": {
			dest: &Destination{
				URI:  &validURL,
				Path: ptr.String("/path?query"),
			},
			want: "invalid value: /path?query: path",
		},
		"invalid, uri and path as uri": {
			dest: &Destination{
				URI:  &validURL,
				Path: ptr.String("http://host/path"),
			},
			want: "invalid value: http://host/path: path",
		},
		"invalid, path with %": {
			dest: &Destination{
				URI:  &validURL,
				Path: ptr.String("/%"),
			},
			want: "invalid value: /%: path",
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

func TestDestinationWithPath(t *testing.T) {
	t.Run("uri has path", func(t *testing.T) {
		uri, _ := apis.ParseURL("http://example.com/foo")
		dest := NewDestinationURI(*uri)
		if got, want := dest.Path, "/foo"; *got != want {
			t.Errorf("Path was %q, wanted %q", *got, want)
		}
	})

	t.Run("multipart path", func(t *testing.T) {
		uri, _ := apis.ParseURL("http://example.com/foo")
		dest := NewDestinationURI(*uri).WithPath("bar")
		if got, want := *dest.Path, "/foo/bar"; got != want {
			t.Errorf("Path was %q, wanted %q", got, want)
		}
	})
}
