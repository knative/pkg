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

package webhook

import (
	"context"
	"github.com/knative/pkg/apis"
	"reflect"
	"strings"
)

// strictValidate validates the following rule sets:
// - On Create, Spec and Status fields that are marked `Deprecated:` are not
//   allowed to be set.
// - On Update, Spec and Status fields that are marked `Deprecated:` are not
//   allowed to be updated, unless deleting.
//
// Note: nil values denote absence of `old` (create) or `new` (delete) objects.
func strictValidate(ctx context.Context, old GenericCRD, new GenericCRD) error {
	// Check if it is a delete, if it is we don't validate.
	if new == nil {
		return nil
	}

	var errs *apis.FieldError

	// If old is nil, then we are creating. Not not allowed to set deprecated
	// fields.
	if old == nil {
		errs = errs.Also(
			strictValidateCreateReflectedByName(new, "Spec").ViaField("Spec"),
		)

		errs = errs.Also(
			strictValidateCreateReflectedByName(new, "Status").ViaField("Status"),
		)
		return errs
	}

	// old and new are both non-nil, so it is an update. Not allowed to update a deprecated field unless it is a delete.
	errs = errs.Also(
		strictValidateUpdateReflectedByName(old, new, "Spec").ViaField("Spec"),
	)

	errs = errs.Also(
		strictValidateUpdateReflectedByName(old, new, "Status").ViaField("Status"),
	)

	return errs
}

func strictValidateCreateReflectedByName(res interface{}, name string) *apis.FieldError {
	resValue := reflect.Indirect(reflect.ValueOf(res))

	// If res is not a struct, don't even try to use it.
	if resValue.Kind() != reflect.Struct {
		return nil
	}

	namedField := resValue.FieldByName(name)

	return strictValidateCreateReflected(namedField)
}

func strictValidateCreateReflected(r reflect.Value) *apis.FieldError {
	// If the field is nil, just move on.
	if !r.IsValid() || r.Kind() != reflect.Struct {
		return &apis.FieldError{
			Message: "failed to inspect object %v",
			Paths:   []string{apis.CurrentField},
		}
	}

	var errs *apis.FieldError

	for i := 0; i < r.NumField(); i++ {
		f := r.Type().Field(i)
		if v := r.Field(i); v.IsValid() {
			if strings.HasPrefix(f.Name, "Deprecated") {
				if nonZero(v) {
					// TODO: add a field error.
					errs = errs.Also(&apis.FieldError{
						Message: "deprecated field set",
						Paths:   []string{f.Name},
					})
					continue
				}
			}

			switch v.Kind() {
			case reflect.Ptr:
				if v.IsNil() {
					continue
				}
				errs = errs.Also(
					strictValidateCreateReflected(v.Elem()).ViaField(f.Name),
				)

			case reflect.Struct:
				errs = errs.Also(
					strictValidateCreateReflected(v).ViaField(f.Name),
				)

			case reflect.Slice, reflect.Array:
				for index := 0; index < v.Len(); index++ {
					value := v.Index(index)
					errs = errs.Also(
						strictValidateCreateReflected(value).ViaFieldIndex(f.Name, index),
					)
				}

			case reflect.Map:
				it := v.MapRange()
				for it.Next() {
					key := it.Key()
					value := it.Value()
					errs = errs.Also(
						strictValidateCreateReflected(value).ViaFieldKey(f.Name, key.String()),
					)
				}
			}
		}
	}
	return errs
}

func strictValidateUpdateReflectedByName(old, res interface{}, name string) *apis.FieldError {
	oldValue := reflect.Indirect(reflect.ValueOf(old))
	resValue := reflect.Indirect(reflect.ValueOf(res))

	// If res is not a struct, don't even try to use it.
	if resValue.Kind() != reflect.Struct {
		return nil
	}

	// If old is not a struct, don't even try to use it.
	if oldValue.Kind() != reflect.Struct {
		return nil
	}

	resField := resValue.FieldByName(name)
	if !resField.IsValid() || resField.Kind() != reflect.Struct {
		return &apis.FieldError{
			Message: "failed to inspect new object",
			Paths:   []string{apis.CurrentField},
		}
	}

	oldField := oldValue.FieldByName(name)
	if !oldField.IsValid() || oldField.Kind() != reflect.Struct {
		return &apis.FieldError{
			Message: "failed to inspect old object",
			Paths:   []string{apis.CurrentField},
		}
	}

	if oldField.Type() != resField.Type() {
		return &apis.FieldError{
			Message: "failed to match filed type",
			Paths:   []string{apis.CurrentField},
		}
	}

	return strictValidateUpdateReflected(oldField, resField)
}

func strictValidateUpdateReflected(oldField, resField reflect.Value) *apis.FieldError {
	if !resField.IsValid() || resField.Kind() != reflect.Struct {
		return &apis.FieldError{
			Message: "failed to inspect new object",
			Paths:   []string{apis.CurrentField},
		}
	}

	if !oldField.IsValid() || oldField.Kind() != reflect.Struct {
		return &apis.FieldError{
			Message: "failed to inspect old object",
			Paths:   []string{apis.CurrentField},
		}
	}

	if oldField.Type() != resField.Type() {
		return &apis.FieldError{
			Message: "failed to match filed type",
			Paths:   []string{apis.CurrentField},
		}
	}

	var errs *apis.FieldError

	// for each field,
	for i := 0; i < resField.NumField(); i++ {
		f := resField.Type().Field(i)
		if v := resField.Field(i); v.IsValid() {
			if o := oldField.Field(i); o.IsValid() {
				if strings.HasPrefix(f.Name, "Deprecated") {
					if differ(v, o) {
						// TODO: add a field error.
						errs = errs.Also(&apis.FieldError{
							Message: "deprecated field updated",
							Paths:   []string{f.Name},
						})
						continue
					}

					// Can be deleted.
					if nonZero(v) {
						errs = errs.Also(&apis.FieldError{
							Message: "deprecated field set",
							Paths:   []string{f.Name},
						})

					}
				}

				switch v.Kind() {
				case reflect.Ptr:
					if v.IsNil() {
						continue
					}
					if o.IsNil() {
						errs = errs.Also(
							strictValidateCreateReflected(v.Elem()).ViaField(f.Name),
						)
					} else {
						errs = errs.Also(
							strictValidateUpdateReflected(o.Elem(), v.Elem()).ViaField(f.Name),
						)
					}

				case reflect.Struct:
					errs = errs.Also(
						strictValidateUpdateReflected(o, v).ViaField(f.Name),
					)

				// TODO: because slices have no order, I am not sure how to test for this.
				//case reflect.Slice, reflect.Array:

				case reflect.Map:
					it := v.MapRange()
					for it.Next() {
						key := it.Key()
						res := it.Value()
						old := o.MapIndex(key)
						errs = errs.Also(
							strictValidateUpdateReflected(old, res).ViaFieldKey(f.Name, key.String()),
						)
					}
				}

			}
		}
	}
	return errs
}

// nonZero returns true if a != 0
func nonZero(a reflect.Value) bool {
	switch a.Kind() {
	case reflect.Ptr:
		if a.IsNil() {
			return false
		}
		return nonZero(a.Elem())

	case reflect.Map, reflect.Slice, reflect.Array:
		if a.IsNil() {
			return false
		}
		return true

	default:
		if reflect.DeepEqual(a.Interface(), reflect.Zero(a.Type()).Interface()) {
			return false
		}
		return true
	}
}

// differ returns true if a != b
func differ(a, b reflect.Value) bool {
	if a.Kind() != b.Kind() {
		return false
	}

	switch a.Kind() {
	case reflect.Ptr:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() != b.IsNil()
		}
		return differ(a.Elem(), b.Elem())

	default:
		if reflect.DeepEqual(a.Interface(), b.Interface()) {
			return false
		}
		return true
	}
}
