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
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseURL(t *testing.T) {
	testCases := map[string]struct {
		t       string
		want    *URL
		wantErr bool
	}{
		"empty": {
			want: nil,
		},
		"empty string": {
			t:    "",
			want: nil,
		},
		"invalid format": {
			t:       "💩://error",
			want:    nil,
			wantErr: true,
		},
		"relative": {
			t: "/path/to/something",
			want: func() *URL {
				uu, _ := url.Parse("/path/to/something")
				u := URL(*uu)
				return &u
			}(),
		},
		"url": {
			t: "http://path/to/something",
			want: func() *URL {
				uu, _ := url.Parse("http://path/to/something")
				u := URL(*uu)
				return &u
			}(),
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
			want: func() *URL {
				uu, _ := url.Parse("/path/to/something")
				u := URL(*uu)
				return &u
			}(),
		},
		"url": {
			b: []byte(`"http://path/to/something"`),
			want: func() *URL {
				uu, _ := url.Parse("http://path/to/something")
				u := URL(*uu)
				return &u
			}(),
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
