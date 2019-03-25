package webhook

import (
	"context"
	"reflect"
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

	// If old is nil, then we are creating. Not not allowed to set deprecated
	// fields.
	if old == nil {

		return nil
	}

	// old and new are both non-nil, so it is an update. Not allowed to update a deprecated field unless it is a delete.

	return nil
}

func reflectedSpec(res interface{}) interface{} {
	resValue := reflect.Indirect(reflect.ValueOf(res))

	// If res is not a struct, don't even try to use it.
	if resValue.Kind() != reflect.Struct {
		return nil
	}

	specField := resValue.FieldByName("Spec")

	if specField.IsValid() && specField.CanInterface() {

		for i := 0; i < specField.NumField(); i++ {
			v := specField.Field(i)

			switch v.Kind() {
			case reflect.Slice:
			case reflect.Map:
			case reflect.Array:
			case reflect.Interface:
			default:
				// normal type that does not have any data below it.
			}

		}
	}
	return nil
}
