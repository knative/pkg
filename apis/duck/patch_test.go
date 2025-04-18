/*
Copyright 2018 The Knative Authors

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

package duck

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCreateMergePatch(t *testing.T) {
	tests := []struct {
		name    string
		before  interface{}
		after   interface{}
		wantErr bool
		want    []byte
	}{{
		name: "patch single field",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Field1: 12,
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Field1: 13,
				},
			},
		},
		want: []byte(`{"status":{"patchable":{"field1":13}}}`),
	}, {
		name: "patch two fields",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Field1: 12,
					Field2: true,
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Field1: 42,
					Field2: false,
				},
			},
		},
		want: []byte(`{"status":{"patchable":{"field1":42,"field2":null}}}`),
	}, {
		name: "patch slice bigger",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "baz"},
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "bar", "baz"},
				},
			},
		},
		want: []byte(`{"status":{"patchable":{"array":["foo","bar","baz"]}}}`),
	}, {
		name: "patch array smaller",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "bar", "baz", "jimmy"},
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "baz"},
				},
			},
		},
		want: []byte(`{"status":{"patchable":{"array":["foo","baz"]}}}`),
	}, {
		name:    "before doesn't marshal",
		before:  &DoesntMarshal{},
		after:   &Patch{},
		wantErr: true,
	}, {
		name:    "after doesn't marshal",
		before:  &Patch{},
		after:   &DoesntMarshal{},
		wantErr: true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CreateMergePatch(test.before, test.after)
			if err != nil {
				if !test.wantErr {
					t.Error("CreateMergePatch() =", err)
				}
				return
			} else if test.wantErr {
				t.Errorf("CreateMergePatch() = %v, wanted error", got)
				return
			}

			if !cmp.Equal(test.want, got) {
				t.Errorf("CreatePatch = ->%s<-, diff: (-want, +got) =\n%s", string(got), cmp.Diff(string(test.want), string(got)))
			}
		})
	}
}

func TestCreatePatch(t *testing.T) {
	tests := []struct {
		name    string
		before  interface{}
		after   interface{}
		wantErr bool
		want    JSONPatch
	}{{
		name: "patch single field",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Field1: 12,
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Field1: 13,
				},
			},
		},
		want: JSONPatch{{
			Operation: "replace",
			Path:      "/status/patchable/field1",
			Value:     json.Number("13"),
		}},
	}, {
		name: "patch two fields",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Field1: 12,
					Field2: true,
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Field1: 42,
					Field2: false,
				},
			},
		},
		want: JSONPatch{{
			Operation: "replace",
			Path:      "/status/patchable/field1",
			Value:     json.Number("42"),
		}, {
			Operation: "remove",
			Path:      "/status/patchable/field2",
		}},
	}, {
		name: "patch array",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "baz"},
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "bar", "baz"},
				},
			},
		},
		want: JSONPatch{{
			Operation: "add",
			Path:      "/status/patchable/array/2",
			Value:     "baz",
		}, {
			Operation: "replace",
			Path:      "/status/patchable/array/1",
			Value:     "bar",
		}},
	}, {
		name: "patch with remove elements from array",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "bar"},
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"bar"},
				},
			},
		},
		want: JSONPatch{{
			Operation: "remove",
			Path:      "/status/patchable/array/1",
		}, {
			Operation: "replace",
			Path:      "/status/patchable/array/0",
			Value:     "bar",
		}},
	}, {
		name: "patch with remove collection",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "bar"},
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{},
			},
		},
		want: JSONPatch{{
			Operation: "remove",
			Path:      "/status/patchable/array",
		}},
	}, {
		name: "patch with add elements to array",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"bar"},
				},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo", "bar"},
				},
			},
		},
		want: JSONPatch{{
			Operation: "add",
			Path:      "/status/patchable/array/1",
			Value:     "bar",
		}, {
			Operation: "replace",
			Path:      "/status/patchable/array/0",
			Value:     "foo",
		}},
	}, {
		name: "patch with add array",
		before: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{},
			},
		},
		after: &Patch{
			Spec: PatchSpec{
				Patchable: &Patchable{
					Array: []string{"foo"},
				},
			},
		},
		want: JSONPatch{{
			Operation: "add",
			Path:      "/status/patchable/array",
			Value:     []interface{}{string("foo")},
		}},
	}, {
		name:    "before doesn't marshal",
		before:  &DoesntMarshal{},
		after:   &Patch{},
		wantErr: true,
	}, {
		name:    "after doesn't marshal",
		before:  &Patch{},
		after:   &DoesntMarshal{},
		wantErr: true,
	}}

	for _, test := range tests {
		t.Run(test.name+" (CreateBytePatch)", func(t *testing.T) {
			got, err := CreateBytePatch(test.before, test.after)
			if err != nil {
				if !test.wantErr {
					t.Error("CreateBytePatch() =", err)
				}
				return
			} else if test.wantErr {
				t.Errorf("CreateBytePatch() = %v, wanted error", got)
				return
			}

			want, err := test.want.MarshalJSON()
			if err != nil {
				t.Errorf("Error marshaling 'want' condition. want: %v error: %v", want, err)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Error("CreateBytePatch (-want, +got) =", diff)
			}
		})
		t.Run(test.name, func(t *testing.T) {
			got, err := CreatePatch(test.before, test.after)
			if err != nil {
				if !test.wantErr {
					t.Error("CreatePatch() =", err)
				}
				return
			} else if test.wantErr {
				t.Errorf("CreatePatch() = %v, wanted error", got)
				return
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Error("CreatePatch (-want, +got) =", diff)
			}
		})
	}
}

func TestPatchToJSON(t *testing.T) {
	input := JSONPatch{{
		Operation: "replace",
		Path:      "/status/patchable/field1",
		Value:     42.0,
	}, {
		Operation: "remove",
		Path:      "/status/patchable/field2",
	}}

	b, err := input.MarshalJSON()
	if err != nil {
		t.Error("MarshalJSON() =", err)
	}

	want := `[{"op":"replace","path":"/status/patchable/field1","value":42},{"op":"remove","path":"/status/patchable/field2"}]`

	got := string(b)
	if got != want {
		t.Errorf("MarshalJSON() = %v, wanted %v", got, want)
	}
}

type DoesntMarshal struct{}

var _ json.Marshaler = (*DoesntMarshal)(nil)

func (*DoesntMarshal) MarshalJSON() ([]byte, error) {
	return nil, errors.New("what did you expect?")
}

// Define a "Patchable" duck type.
type Patchable struct {
	Field1 int      `json:"field1,omitempty"`
	Field2 bool     `json:"field2,omitempty"`
	Array  []string `json:"array,omitempty"`
}
type Patch struct {
	Spec PatchSpec `json:"status"`
}
type PatchSpec struct {
	Patchable *Patchable `json:"patchable,omitempty"`
}

var (
	_ Implementable = (*Patchable)(nil)
	_ Populatable   = (*Patch)(nil)
)

func (*Patch) GetObjectKind() schema.ObjectKind {
	return nil // not used
}

func (*Patch) DeepCopyObject() runtime.Object {
	return nil // not used
}

func (*Patch) GetListType() runtime.Object {
	return nil // not used
}

func (*Patchable) GetFullType() Populatable {
	return &Patch{}
}

func (f *Patch) Populate() {
	f.Spec.Patchable = &Patchable{
		// Populate ALL fields
		Field1: 42,
		Field2: true,
	}
}
