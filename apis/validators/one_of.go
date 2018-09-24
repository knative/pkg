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
	"reflect"
	"strings"

	"github.com/knative/pkg/apis"
)

// Usage:
//  OneOfOne string `validate:"OneOf,groupName"`
//  OneOfTwo string `validate:"OneOf,groupName"`

func NewOneOfValidator(opts tagOptions) *OneOfValidator {
	group, opts := parseTag(string(opts))
	return &OneOfValidator{
		group: group,
	}
}

type OneOfValidator struct {
	group string
}

var _ Validator = (*OneOfValidator)(nil)

func (v *OneOfValidator) OnParent() bool {
	return true
}

func (v *OneOfValidator) OnField() bool {
	return false
}

func (v *OneOfValidator) Validate(value interface{}) *apis.FieldError {
	// We need to take a look at all of the values

	r := reflect.ValueOf(value)
	if !r.IsValid() || r.Kind() != reflect.Struct {
		return &apis.FieldError{
			Message: "failed to inspect object",
			Paths:   []string{apis.CurrentField},
		}
	}

	var fields []reflect.Value
	var names []string
	// for each field,
	for i := 0; i < r.NumField(); i++ {
		tags := r.Type().Field(i).Tag.Get(validationTagName)
		// for each tag on that field
		for _, tag := range strings.Split(tags, ";") {
			t, tOpts := parseTag(tag)
			if t == OneOfValidatorTag {
				group, _ := parseTag(string(tOpts))
				if group == v.group {
					fields = append(fields, r.Field(i))
					names = append(names, getName(r.Type().Field(i)))
				}
			}
		}
	}

	switch countNotNull(fields) {
	case 0:
		// none were set.
		return apis.ErrMissingOneOf(names...)
	case 1:
		// Success!
		return nil
	default:
		// to many set
		return apis.ErrMultipleOneOf(names...)
	}
}

func countNotNull(fields []reflect.Value) int {
	var count int
	for _, f := range fields {
		if f.IsValid() && !reflect.DeepEqual(f.Interface(), reflect.Zero(f.Type()).Interface()) {
			count++
		}
	}
	return count
}

func (v *OneOfValidator) AlreadyIn(validators []Validator) bool {
	for _, validator := range validators {
		oneOf, ok := validator.(*OneOfValidator)
		if ok {
			if oneOf.group == v.group {
				return true
			}
		}
	}
	return false
}
