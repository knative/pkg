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
	"github.com/knative/pkg/apis"

	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	QualifiedName            = "QualifiedName"
	K8sQualifiedNameRequired = "Required"
)

// Usage:
//  OptionalName string `validate:"QualifiedName"`
//  RequiredName string `validate:"QualifiedName,Required"`

func NewK8sQualifiedNameValidator(opts tagOptions) *K8sQualifiedNameValidator {
	required := false
	if opts.Contains(K8sQualifiedNameRequired) {
		required = true
	}
	return &K8sQualifiedNameValidator{
		required: required,
	}
}

type K8sQualifiedNameValidator struct {
	required bool
}

var _ Validator = (*K8sQualifiedNameValidator)(nil)

func (v *K8sQualifiedNameValidator) OnParent() bool {
	return false
}

func (v *K8sQualifiedNameValidator) OnField() bool {
	return true
}

func (v *K8sQualifiedNameValidator) Validate(value interface{}) *apis.FieldError {

	name, ok := value.(string)
	if !ok {
		return &apis.FieldError{
			Message: "failed to marshal field",
			Paths:   []string{apis.CurrentField},
		}
	}

	if v.required && len(name) == 0 {
		return apis.ErrMissingField(apis.CurrentField)
	}

	if len(name) != 0 {
		if errs := validation.IsQualifiedName(name); len(errs) > 0 {
			return apis.ErrInvalidKeyName(name, apis.CurrentField, errs...)
		}
	}

	return nil
}
