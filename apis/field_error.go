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

// FieldError is a collection of field errors.
// +k8s:deepcopy-gen=false
type FieldError struct {
	Message string
	Paths   []string
	// Details contains an optional longer payload.
	Details string
	errors  []fieldError
}

// FieldError implements error
var _ error = (*FieldError)(nil)

// ViaField is used to propagate a validation error along a field access.
func (fe *FieldError) ViaField(prefix ...string) *FieldError {
	if fe == nil {
		return nil
	}
	var newErrs []fieldError
	for _, e := range fe.getNormalizedErrors() {
		if newErr := e.ViaField(prefix...); newErr != nil {
			newErrs = append(newErrs, *newErr)
		}
	}
	fe.errors = newErrs
	return fe
}

// ViaIndex is used to attach an index to the next ViaField provided.
func (fe *FieldError) ViaIndex(index int) *FieldError {
	if fe == nil {
		return nil
	}
	var newErrs []fieldError
	for _, e := range fe.getNormalizedErrors() {
		if newErr := e.ViaIndex(index); newErr != nil {
			newErrs = append(newErrs, *newErr)
		}
	}
	fe.errors = newErrs
	return fe
}

// ViaFieldIndex is the short way to chain: err.ViaIndex(bar).ViaField(foo)
func (fe *FieldError) ViaFieldIndex(field string, index int) *FieldError {
	if fe == nil {
		return nil
	}
	var newErrs []fieldError
	for _, e := range fe.getNormalizedErrors() {
		if newErr := e.ViaFieldIndex(field, index); newErr != nil {
			newErrs = append(newErrs, *newErr)
		}
	}
	fe.errors = newErrs
	return fe
}

// ViaKey is used to attach a key to the next ViaField provided.
func (fe *FieldError) ViaKey(key string) *FieldError {
	if fe == nil {
		return nil
	}
	var newErrs []fieldError
	for _, e := range fe.getNormalizedErrors() {
		if newErr := e.ViaKey(key); newErr != nil {
			newErrs = append(newErrs, *newErr)
		}
	}
	fe.errors = newErrs
	return fe
}

// ViaFieldKey is the short way to chain: err.ViaKey(bar).ViaField(foo)
func (fe *FieldError) ViaFieldKey(field string, key string) *FieldError {
	if fe == nil {
		return nil
	}
	var newErrs []fieldError
	for _, e := range fe.getNormalizedErrors() {
		if newErr := e.ViaFieldKey(field, key); newErr != nil {
			newErrs = append(newErrs, *newErr)
		}
	}
	fe.errors = newErrs
	return fe
}

func (fe *FieldError) getNormalizedErrors() []fieldError {
	if fe.Message != "" {
		err := fieldError{
			Message: fe.Message,
			Paths:   fe.Paths,
			Details: fe.Details,
		}
		fe.Message = ""
		fe.Paths = []string(nil)
		fe.Details = ""
		fe.errors = append(fe.errors, err)
	}
	return fe.errors
}

// Also collects errors, returns a new collection of existing errors and new errors.
func (fe *FieldError) Also(errs ...*FieldError) *FieldError {
	var newErrs []fieldError
	if fe != nil {
		newErrs = append(newErrs, fe.getNormalizedErrors()...)
	}

	for _, e := range errs {
		newErrs = append(newErrs, e.getNormalizedErrors()...)
	}

	fe.errors = newErrs
	return fe
}

// fieldError implements error
func (fe *FieldError) Error() string {
	var errs []string
	for _, e := range fe.getNormalizedErrors() {
		if e.Details == "" {
			errs = append(errs, fmt.Sprintf("%v: %v", e.Message, strings.Join(e.Paths, ", ")))
		} else {
			errs = append(errs, fmt.Sprintf("%v: %v\n%v", e.Message, strings.Join(e.Paths, ", "), e.Details))
		}
	}
	return strings.Join(errs, "\n")
}
