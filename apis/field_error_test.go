/*
Copyright 2017 The Knative Authors

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
	"strconv"
	"strings"
	"testing"
)

func TestFieldError(t *testing.T) {
	tests := []struct {
		name     string
		err      *FieldError
		prefixes [][]string
		want     string
	}{{
		name: "simple single no propagation",
		err: &FieldError{
			Message: "hear me roar",
			Paths:   []string{"foo.bar"},
		},
		want: "hear me roar: foo.bar",
	}, {
		name: "simple single propagation",
		err: &FieldError{
			Message: `invalid value "blah"`,
			Paths:   []string{"foo"},
		},
		prefixes: [][]string{{"bar"}, {"baz", "ugh"}, {"hoola"}},
		want:     `invalid value "blah": hoola.baz.ugh.bar.foo`,
	}, {
		name: "simple multiple propagation",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo", "bar"},
		},
		prefixes: [][]string{{"baz", "ugh"}},
		want:     "invalid field(s): baz.ugh.foo, baz.ugh.bar",
	}, {
		name: "multiple propagation with details",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo", "bar"},
			Details: `I am a long
long
loooong
Body.`,
		},
		prefixes: [][]string{{"baz", "ugh"}},
		want: `invalid field(s): baz.ugh.foo, baz.ugh.bar
I am a long
long
loooong
Body.`,
	}, {
		name: "single propagation, empty start",
		err: &FieldError{
			Message: "invalid field(s)",
			// We might see this validating a scalar leaf.
			Paths: []string{CurrentField},
		},
		prefixes: [][]string{{"baz", "ugh"}},
		want:     "invalid field(s): baz.ugh",
	}, {
		name: "single propagation, no paths",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   nil,
		},
		prefixes: [][]string{{"baz", "ugh"}},
		want:     "invalid field(s): ",
	}, {
		name:     "nil propagation",
		err:      nil,
		prefixes: [][]string{{"baz", "ugh"}},
	}, {
		name:     "missing field propagation",
		err:      ErrMissingField("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     "missing field(s): baz.foo, baz.bar",
	}, {
		name:     "missing disallowed propagation",
		err:      ErrDisallowedFields("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     "must not set the field(s): baz.foo, baz.bar",
	}, {
		name:     "invalid value propagation",
		err:      ErrInvalidValue("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     `invalid value "foo": baz.bar`,
	}, {
		name:     "missing mutually exclusive fields",
		err:      ErrMissingOneOf("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     `expected exactly one, got neither: baz.foo, baz.bar`,
	}, {
		name:     "multiple mutually exclusive fields",
		err:      ErrMultipleOneOf("foo", "bar"),
		prefixes: [][]string{{"baz"}},
		want:     `expected exactly one, got both: baz.foo, baz.bar`,
	}, {
		name: "invalid key name",
		err: ErrInvalidKeyName("b@r", "foo[0].name",
			"can not use @", "do not try"),
		prefixes: [][]string{{"baz"}},
		want: `invalid key name "b@r": baz.foo[0].name
can not use @, do not try`,
	}, {
		name: "invalid key name with details array",
		err: ErrInvalidKeyName("b@r", "foo[0].name",
			[]string{"can not use @", "do not try"}...),
		prefixes: [][]string{{"baz"}},
		want: `invalid key name "b@r": baz.foo[0].name
can not use @, do not try`,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fe := test.err
			// Simulate propagation up a call stack.
			for _, prefix := range test.prefixes {
				fe = fe.ViaField(prefix...)
			}
			if test.want != "" {
				got := fe.Error()
				if got != test.want {
					t.Errorf("Error() = %v, wanted %v", got, test.want)
				}
			} else if fe != nil {
				t.Errorf("ViaField() = %v, wanted nil", fe)
			}
		})
	}
}

func TestViaIndexFieldError(t *testing.T) {
	tests := []struct {
		name     string
		err      *FieldError
		prefixes [][]string
		want     string
	}{{
		name: "simple single no propagation",
		err: &FieldError{
			Message: "hear me roar",
			Paths:   []string{"bar"},
		},
		prefixes: [][]string{{"foo", "INDEX:1,2,3"}},
		want:     "hear me roar: foo[1][2][3].bar",
	}, {
		name:     "missing field propagation",
		err:      ErrMissingField("foo", "bar"),
		prefixes: [][]string{{"baz", "INDEX:2"}},
		want:     "missing field(s): baz[2].foo, baz[2].bar",
	}, {
		name: "invalid key name",
		err: ErrInvalidKeyName("b@r", "name",
			"can not use @", "do not try"),
		prefixes: [][]string{{"baz", "INDEX:0", "foo"}},
		want: `invalid key name "b@r": foo.baz[0].name
can not use @, do not try`,
	}, {
		name: "multi prefixes provided",
		err: &FieldError{
			Message: "invalid field(s)",
			Paths:   []string{"foo"},
		},
		prefixes: [][]string{{"bee"}, {"INDEX:0"}, {"baa", "baz", "ugh"}, {"INDEX:2"}},
		want:     "invalid field(s): ugh[2].baz.baa.bee[0].foo",
	}, {
		name: "manual call",
		err: func() *FieldError {
			err := &FieldError{
				Message: "invalid field(s)",
				Paths:   []string{"foo"},
			}
			err = err.ViaIndex(-1)
			err = err.ViaField("bar").ViaIndex(0)
			err = err.ViaField("baz").ViaIndex(1, 2)
			err = err.ViaField("boof").ViaIndex(3).ViaIndex(4)
			return err
		}(),
		want: "invalid field(s): boof[3][4].baz[1][2].bar[0].foo[-1]",
	}, {
		name:     "nil propagation",
		err:      nil,
		prefixes: [][]string{{"baz", "ugh", "INDEX:0"}},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fe := test.err
			// Simulate propagation up a call stack.
			for _, prefix := range test.prefixes {
				for _, p := range prefix {
					if strings.Contains(p, "INDEX") {
						index := strings.Split(p, ":")
						fe = fe.ViaIndex(makeIndex(index[1])...)
					} else {
						fe = fe.ViaField(p)
					}
				}
			}

			if test.want != "" {
				got := fe.Error()
				if got != test.want {
					t.Errorf("Error() = %v, wanted %v", got, test.want)
				}
			} else if fe != nil {
				t.Errorf("ViaField() = %v, wanted nil", fe)
			}
		})
	}
}

func makeIndex(index string) []int {
	indexes := []int(nil)

	all := strings.Split(index, ",")
	for _, index := range all {
		if i, err := strconv.Atoi(index); err == nil {
			indexes = append(indexes, i)
		}
	}

	return indexes
}
