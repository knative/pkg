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
package apis

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseURL(t *testing.T) {
	testCases := map[string]struct {
		t         string
		want      *URL
		wantEmpty bool
		wantErr   bool
	}{
		"empty string": {
			want:      nil,
			wantEmpty: true,
		},
		"invalid format": {
			t:         "💩://error",
			want:      nil,
			wantEmpty: true,
			wantErr:   true,
		},
		"relative": {
			t: "/path/to/something",
			want: &URL{
				Path: "/path/to/something",
			},
		},
		"url": {
			t: "http://path/to/something",
			want: &URL{
				Scheme: "http",
				Host:   "path",
				Path:   "/to/something",
			},
		},
		"simplehttp": {
			t:    "http://foo",
			want: HTTP("foo"),
		},
		"simplehttps": {
			t:    "https://foo",
			want: HTTPS("foo"),
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			got, err := ParseURL(tc.t)
			if err != nil {
				if !tc.wantErr {
					t.Fatalf("ParseURL() = %v", err)
				}
				return
			} else if tc.wantErr {
				t.Fatalf("ParseURL() = %v, wanted error", got)
			}

			if tc.wantEmpty {
				if !got.IsEmpty() {
					t.Errorf("Expected empty for %q, got %v", tc.t, got)
				}
			} else {
				if got.IsEmpty() {
					t.Errorf("Expected non-empty for %q", tc.t)
				}
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected object (-want, +got) = %v", diff)
			}
		})
	}
}

func TestJsonMarshalURL(t *testing.T) {
	testCases := map[string]struct {
		t    string
		want []byte
	}{
		"empty": {},
		"empty string": {
			t: "",
		},
		"invalid url": {
			t:    "not a url",
			want: []byte(`"not%20a%20url"`),
		},
		"relative format": {
			t:    "/path/to/something",
			want: []byte(`"/path/to/something"`),
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			var got []byte
			tt, err := ParseURL(tc.t)
			if err != nil {
				t.Fatalf("ParseURL() = %v", err)
			}
			if tt != nil {
				got, _ = tt.MarshalJSON()
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Logf("got: %s", string(got))
				t.Errorf("unexpected object (-want, +got) = %v", diff)
			}
		})
	}
}

func TestJsonUnmarshalURL(t *testing.T) {
	testCases := map[string]struct {
		b       []byte
		want    *URL
		wantErr string
	}{
		"empty": {
			wantErr: "unexpected end of JSON input",
		},
		"invalid format": {
			b:       []byte("%"),
			wantErr: "invalid character '%' looking for beginning of value",
		},
		"relative": {
			b: []byte(`"/path/to/something"`),
			want: &URL{
				Path: "/path/to/something",
			},
		},
		"url": {
			b: []byte(`"http://path/to/something"`),
			want: &URL{
				Scheme: "http",
				Host:   "path",
				Path:   "/to/something",
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			got := &URL{}
			err := got.UnmarshalJSON(tc.b)

			if tc.wantErr != "" || err != nil {
				var gotErr string
				if err != nil {
					gotErr = err.Error()
				}
				if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
					t.Errorf("unexpected error (-want, +got) = %v", diff)
				}
				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected object (-want, +got) = %v", diff)
			}
		})
	}
}

func TestJsonMarshalURLAsMember(t *testing.T) {

	type objectType struct {
		URL URL `json:"url,omitempty"`
	}

	testCases := map[string]struct {
		obj     *objectType
		want    []byte
		wantErr string
	}{
		"nil": {
			want: []byte(`null`),
		},
		"empty": {
			obj:  &objectType{},
			want: []byte(`{"url":""}`),
		},
		"relative": {
			obj:  &objectType{URL: URL{Path: "/path/to/something"}},
			want: []byte(`{"url":"/path/to/something"}`),
		},
		"url": {
			obj: &objectType{URL: URL{
				Scheme: "http",
				Host:   "path",
				Path:   "/to/something",
			}},
			want: []byte(`{"url":"http://path/to/something"}`),
		},
		"empty url": {
			obj:  &objectType{URL: URL{}},
			want: []byte(`{"url":""}`),
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			got, err := json.Marshal(tc.obj)

			if tc.wantErr != "" || err != nil {
				var gotErr string
				if err != nil {
					gotErr = err.Error()
				}
				if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
					t.Errorf("unexpected error (-want, +got) = %v", diff)
				}
				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected object (-want, +got) = %v", diff)
				t.Logf("got: %s", string(got))
			}
		})
	}
}

func TestJsonMarshalURLAsPointerMember(t *testing.T) {

	type objectType struct {
		URL *URL `json:"url,omitempty"`
	}

	testCases := map[string]struct {
		obj     *objectType
		want    []byte
		wantErr string
	}{
		"nil": {
			want: []byte(`null`),
		},
		"empty": {
			obj:  &objectType{},
			want: []byte(`{}`),
		},
		"relative": {
			obj:  &objectType{URL: &URL{Path: "/path/to/something"}},
			want: []byte(`{"url":"/path/to/something"}`),
		},
		"url": {
			obj: &objectType{URL: &URL{
				Scheme: "http",
				Host:   "path",
				Path:   "/to/something",
			}},
			want: []byte(`{"url":"http://path/to/something"}`),
		},
		"empty url": {
			obj:  &objectType{URL: &URL{}},
			want: []byte(`{"url":""}`),
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			got, err := json.Marshal(tc.obj)

			if tc.wantErr != "" || err != nil {
				var gotErr string
				if err != nil {
					gotErr = err.Error()
				}
				if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
					t.Errorf("unexpected error (-want, +got) = %v", diff)
				}
				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected object (-want, +got) = %v", diff)
				t.Logf("got: %s", string(got))
			}
		})
	}
}

func TestJsonUnmarshalURLAsMember(t *testing.T) {

	type objectType struct {
		URL URL `json:"url,omitempty"`
	}

	testCases := map[string]struct {
		b       []byte
		want    *objectType
		wantErr string
	}{
		"zero": {
			wantErr: "unexpected end of JSON input",
		},
		"empty": {
			b:    []byte(`{}`),
			want: &objectType{},
		},
		"invalid format": {
			b:       []byte(`{"url":"%"}`),
			wantErr: `parse %: invalid URL escape "%"`,
		},
		"relative": {
			b:    []byte(`{"url":"/path/to/something"}`),
			want: &objectType{URL: URL{Path: "/path/to/something"}},
		},
		"url": {
			b: []byte(`{"url":"http://path/to/something"}`),
			want: &objectType{URL: URL{
				Scheme: "http",
				Host:   "path",
				Path:   "/to/something",
			}},
		},
		"empty url": {
			b:    []byte(`{"url":""}`),
			want: &objectType{URL: URL{}},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			got := &objectType{}
			err := json.Unmarshal(tc.b, got)

			if tc.wantErr != "" || err != nil {
				var gotErr string
				if err != nil {
					gotErr = err.Error()
				}
				if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
					t.Errorf("unexpected error (-want, +got) = %v", diff)
				}
				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected object (-want, +got) = %v", diff)
			}
		})
	}
}

func TestJsonUnmarshalURLAsMemberPointer(t *testing.T) {

	type objectType struct {
		URL *URL `json:"url,omitempty"`
	}

	testCases := map[string]struct {
		b       []byte
		want    *objectType
		wantErr string
	}{
		"zero": {
			wantErr: "unexpected end of JSON input",
		},
		"empty": {
			b:    []byte(`{}`),
			want: &objectType{},
		},
		"invalid format": {
			b:       []byte(`{"url":"%"}`),
			wantErr: `parse %: invalid URL escape "%"`,
		},
		"relative": {
			b:    []byte(`{"url":"/path/to/something"}`),
			want: &objectType{URL: &URL{Path: "/path/to/something"}},
		},
		"url": {
			b: []byte(`{"url":"http://path/to/something"}`),
			want: &objectType{URL: &URL{
				Scheme: "http",
				Host:   "path",
				Path:   "/to/something",
			}},
		},
		"empty url": {
			b:    []byte(`{"url":""}`),
			want: &objectType{URL: &URL{}},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			got := &objectType{}
			err := json.Unmarshal(tc.b, got)

			if tc.wantErr != "" || err != nil {
				var gotErr string
				if err != nil {
					gotErr = err.Error()
				}
				if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
					t.Errorf("unexpected error (-want, +got) = %v", diff)
				}
				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("unexpected object (-want, +got) = %v", diff)
			}
		})
	}
}

func TestURLString(t *testing.T) {
	testCases := map[string]struct {
		t    string
		want string
	}{
		"empty": {
			want: "",
		},
		"relative": {
			t:    "/path/to/something",
			want: "/path/to/something",
		},
		"url": {
			t:    "http://path/to/something",
			want: "http://path/to/something",
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			tt, err := ParseURL(tc.t)
			if err != nil {
				t.Fatalf("ParseURL() = %v", err)
			}
			got := tt.String()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Logf("got: %s", string(got))
				t.Errorf("unexpected string (-want, +got) = %v", diff)
			}
		})
	}
}
