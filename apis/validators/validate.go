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

const tagName = "validate"

// Returns validator struct corresponding to validation type
func getValidatorFromTag(tag string) Validator {
	args := strings.Split(tag, ",")

	switch args[0] {
	case QualifiedName:
		return NewK8sQualifiedNameValidator(strings.Join(args[1:], ","))
	}

	return NewDefaultValidator()
}

func Validate(obj interface{}) (bool, []*apis.FieldError) {

	v := reflect.ValueOf(obj)

	valid := true
	errs := []*apis.FieldError(nil)

	for i := 0; i < v.NumField(); i++ {
		// Get the field tag value
		tag := v.Type().Field(i).Tag.Get(tagName)

		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}

		// Get a validator that corresponds to a tag
		validator := getValidatorFromTag(tag)

		// Perform validation
		if validator.OnParent() {
			ok, err := validator.Validate(v.Type().Field(i).Name, obj)
			if !ok {
				valid = false
			}
			if err != nil {
				errs = append(errs, err)
			}
		}
		if validator.OnField() {
			ok, err := validator.Validate(v.Type().Field(i).Name, v.Field(i).Interface())
			if !ok {
				valid = false
			}
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return valid, errs
}

// TODO(n3wscott): OnParent and OnField might be strange. Not sure there is
// a case where I want to be able to send both parent and field.

type Validator interface {
	// OnParent if true, Validate will will pass the containing object down to
	// the validators validate method. This is intended to be used for OneOf
	// sets.
	OnParent() bool
	// OnField if true, Validate will pass the field value to the validators
	// validate method.
	OnField() bool
	// Validate( will perform the validation for the given field based on
	// OnField and OnParent.
	Validate(fieldName string, value interface{}) (bool, *apis.FieldError)
}
