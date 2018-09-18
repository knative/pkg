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

// FieldErrors is a collection of field errors.
type FieldErrors []*FieldError

// FieldError implements error
var _ error = (*FieldErrors)(nil)

// ViaField is used to propagate a validation error along a field access.
func (fe *FieldErrors) ViaField(prefix ...string) *FieldErrors {
	if fe == nil {
		return nil
	}
	newErrs := FieldErrors(nil)
	for _, e := range *fe {
		if newErr := e.ViaField(prefix...); newErr != nil {
			newErrs = append(newErrs, newErr)
		}
	}
	return &newErrs
}

// ViaIndex is used to attach an index to the next ViaField provided.
func (fe *FieldErrors) ViaIndex(index int) *FieldErrors {
	if fe == nil {
		return nil
	}
	newErrs := FieldErrors(nil)
	for _, e := range *fe {
		if newErr := e.ViaIndex(index); newErr != nil {
			newErrs = append(newErrs, newErr)
		}
	}
	return &newErrs
}

// ViaFieldIndex is the short way to chain: err.ViaIndex(bar).ViaField(foo)
func (fe *FieldErrors) ViaFieldIndex(field string, index int) *FieldErrors {
	if fe == nil {
		return nil
	}
	newErrs := FieldErrors(nil)
	for _, e := range *fe {
		if newErr := e.ViaFieldIndex(field, index); newErr != nil {
			newErrs = append(newErrs, newErr)
		}
	}
	return &newErrs
}

// ViaKey is used to attach a key to the next ViaField provided.
func (fe *FieldErrors) ViaKey(key string) *FieldErrors {
	if fe == nil {
		return nil
	}
	newErrs := FieldErrors(nil)
	for _, e := range *fe {
		if newErr := e.ViaKey(key); newErr != nil {
			newErrs = append(newErrs, newErr)
		}
	}
	return &newErrs
}

// ViaFieldKey is the short way to chain: err.ViaKey(bar).ViaField(foo)
func (fe *FieldErrors) ViaFieldKey(field string, key string) *FieldErrors {
	if fe == nil {
		return nil
	}
	newErrs := FieldErrors(nil)
	for _, e := range *fe {
		if newErr := e.ViaFieldKey(field, key); newErr != nil {
			newErrs = append(newErrs, newErr)
		}
	}
	return &newErrs
}

// Also collects errors, returns a new collection of existing errors and new errors.
func (fe *FieldErrors) Also(errs ...*FieldError) *FieldErrors {
	newErrs := FieldErrors(nil)
	if fe != nil {
		newErrs = append(newErrs, *fe...)
	}
	newErrs = append(newErrs, errs...)
	return &newErrs
}

// Error implements error
func (fe *FieldErrors) Error() string {
	var errs []string
	for _, e := range *fe {
		if e.Details == "" {
			errs = append(errs, fmt.Sprintf("%v: %v", e.Message, strings.Join(e.Paths, ", ")))
		} else {
			errs = append(errs, fmt.Sprintf("%v: %v\n%v", e.Message, strings.Join(e.Paths, ", "), e.Details))
		}
	}
	return strings.Join(errs, "\n")
}
