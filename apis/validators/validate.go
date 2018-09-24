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

	"github.com/knative/pkg/apis"
)

const (
	jsonTagName       = "json"
	validationTagName = "validate"
)

// Returns validator struct corresponding to validation type
func getValidatorFromTag(tag string) Validator {
	v, vOpts := parseTag(tag)

	switch v {
	case QualifiedName:
		return NewK8sQualifiedNameValidator(vOpts)
	}

	return NewDefaultValidator()
}

// getName will inspect the struct field for a json annotation and return the
// specified name, otherwise default to the name of the struct field.
func getName(field reflect.StructField) string {
	tag := field.Tag.Get(jsonTagName)
	if tag == "" || tag == "-" {
		// no json name defined, use the struct name.
		return field.Name
	}
	name, _ := parseTag(tag)
	return name
}

func Validate(obj interface{}) *apis.FieldError {

	v := reflect.ValueOf(obj)
	if !v.IsValid() {
		return nil
	}

	var errs *apis.FieldError
	for i := 0; i < v.NumField(); i++ {
		// Get the field name
		name := getName(v.Type().Field(i))

		// Get the validation tag
		tag := v.Type().Field(i).Tag.Get(validationTagName)

		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}

		// Get a validator that corresponds to a tag
		validator := getValidatorFromTag(tag)

		// Perform validation
		if validator.OnParent() {
			errs = errs.Also(validator.Validate(obj).ViaField(name))
		}
		if validator.OnField() {
			if field := v.Field(i); field.CanInterface() {
				errs = errs.Also(validator.Validate(field.Interface()).ViaField(name))
			}
		}
	}
	return errs
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
	Validate(value interface{}) *apis.FieldError
}
