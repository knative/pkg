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

package validators

import (
	"testing"

	"github.com/knative/pkg/apis"
)

type oneOfNoGroup struct {
	Foo string `validate:"OneOf"`
	Bar string `validate:"OneOf"`
}

type oneOfFoo struct {
	Foo string `validate:"OneOf,foo" json:"foo"`
	Bar string `validate:"OneOf,foo"`
}

type oneOfThree struct {
	Foo string `validate:"OneOf,foo" json:"foo"`
	Bar string `validate:"OneOf,foo"`
	Baz string `validate:"OneOf,foo" json:"baz"`
}

type oneOfTwoGroups struct {
	Bar string `validate:"OneOf,foo"`
	Foo string `validate:"OneOf,foo"`
	Cow string `validate:"OneOf,animal"`
	Dog string `validate:"OneOf,animal"`
}

func TestOneOfValidator_oneOfNoGroup(t *testing.T) {
	tests := []ValidateTest{{
		name: "just foo",
		obj: oneOfNoGroup{
			Foo: "foo",
		},
		want: nil,
	}, {
		name: "just bar",
		obj: oneOfNoGroup{
			Bar: "bar",
		},
		want: nil,
	}, {
		name: "both",
		obj: oneOfNoGroup{
			Foo: "foo",
			Bar: "bar",
		},
		want: apis.ErrMultipleOneOf("Foo", "Bar"),
	}, {
		name: "none",
		obj:  oneOfNoGroup{},
		want: apis.ErrMissingOneOf("Foo", "Bar"),
	}}
	doTestValidate(t, tests)
}

func TestOneOfValidator_oneOfFoo(t *testing.T) {
	tests := []ValidateTest{{
		name: "just foo",
		obj: oneOfFoo{
			Foo: "foo",
		},
		want: nil,
	}, {
		name: "just bar",
		obj: oneOfFoo{
			Bar: "bar",
		},
		want: nil,
	}, {
		name: "both",
		obj: oneOfFoo{
			Foo: "foo",
			Bar: "bar",
		},
		want: apis.ErrMultipleOneOf("foo", "Bar"),
	}, {
		name: "none",
		obj:  oneOfFoo{},
		want: apis.ErrMissingOneOf("foo", "Bar"),
	}}
	doTestValidate(t, tests)
}

func TestOneOfValidator_oneOfThree(t *testing.T) {
	tests := []ValidateTest{{
		name: "just foo",
		obj: oneOfThree{
			Foo: "foo",
		},
		want: nil,
	}, {
		name: "just bar",
		obj: oneOfThree{
			Bar: "bar",
		},
		want: nil,
	}, {
		name: "both",
		obj: oneOfThree{
			Foo: "foo",
			Bar: "bar",
		},
		want: apis.ErrMultipleOneOf("foo", "Bar", "baz"),
	}, {
		name: "none",
		obj:  oneOfThree{},
		want: apis.ErrMissingOneOf("foo", "Bar", "baz"),
	}}
	doTestValidate(t, tests)
}

func TestOneOfValidator_oneOfTwoGroups(t *testing.T) {
	tests := []ValidateTest{{
		name: "valid",
		obj: oneOfTwoGroups{
			Foo: "Foo",
			Cow: "Moo",
		},
		want: nil,
	}, {
		name: "just a Bar",
		obj: oneOfTwoGroups{
			Bar: "bar",
		},
		want: apis.ErrMissingOneOf("Cow", "Dog"),
	}, {
		name: "just Cow",
		obj: oneOfTwoGroups{
			Cow: "moo",
		},
		want: apis.ErrMissingOneOf("Bar", "Foo"),
	}, {
		name: "none",
		obj:  oneOfTwoGroups{},
		want: apis.ErrMissingOneOf("Bar", "Foo").
			Also(apis.ErrMissingOneOf("Cow", "Dog")),
	}}
	doTestValidate(t, tests)
}
