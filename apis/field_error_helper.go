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
	"fmt"
	"strings"
)

// Wrap converts a Error into a *FieldError for easy integration.
func (e Error) Wrap() *FieldError {
	return &FieldError{
		&e,
	}
}

// ErrMissingField is a variadic helper method for constructing a Error for
// a set of missing fields.
func ErrMissingField(fieldPaths ...string) *FieldError {
	return Error{
		Message: "missing field(s)",
		Paths:   fieldPaths,
	}.Wrap()
}

// ErrDisallowedFields is a variadic helper method for constructing a Error
// for a set of disallowed fields.
func ErrDisallowedFields(fieldPaths ...string) *FieldError {
	return Error{
		Message: "must not set the field(s)",
		Paths:   fieldPaths,
	}.Wrap()
}

// ErrInvalidValue constructs a Error for a field that has received an
// invalid string value.
func ErrInvalidValue(value, fieldPath string) *FieldError {
	return Error{
		Message: fmt.Sprintf("invalid value %q", value),
		Paths:   []string{fieldPath},
	}.Wrap()
}

// ErrMissingOneOf is a variadic helper method for constructing a Error for
// not having at least one field in a mutually exclusive field group.
func ErrMissingOneOf(fieldPaths ...string) *FieldError {
	return Error{
		Message: "expected exactly one, got neither",
		Paths:   fieldPaths,
	}.Wrap()
}

// ErrMultipleOneOf is a variadic helper method for constructing a Error
// for having more than one field set in a mutually exclusive field group.
func ErrMultipleOneOf(fieldPaths ...string) *FieldError {
	return Error{
		Message: "expected exactly one, got both",
		Paths:   fieldPaths,
	}.Wrap()
}

// ErrInvalidKeyName is a variadic helper method for constructing a
// Error that specifies a key name that is invalid.
func ErrInvalidKeyName(value, fieldPath string, details ...string) *FieldError {
	return Error{
		Message: fmt.Sprintf("invalid key name %q", value),
		Paths:   []string{fieldPath},
		Details: strings.Join(details, ", "),
	}.Wrap()
}
