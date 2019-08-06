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

package apis

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"net/url"
	"strings"
)

// Destination represents a target of an invocation over HTTP.
type Destination struct {
	// ObjectReference points to an Addressable.
	*corev1.ObjectReference `json:",inline"`

	// URI is for direct URI Designations.
	URI *URL `json:"uri,omitempty"`

	// Path is used with the resulting URL from Addressable ObjectReference or
	// URI. Must start with `/`.
	Path *string `json:"path,omitempty"`
}

func (current *Destination) Validate(ctx context.Context) *FieldError {
	if current != nil {
		errs := validateDestination(*current).ViaField(CurrentField)
		if current.Path != nil {
			errs = errs.Also(validateDestinationPath(*current.Path).ViaField("path"))
		}
		return errs
	} else {
		return nil
	}
}

func validateDestination(dest Destination) *FieldError {
	if dest.URI != nil {
		if dest.ObjectReference != nil {
			return ErrMultipleOneOf("uri", "[apiVersion, kind, name]")
		}
		if dest.URI.Host == "" || dest.URI.Scheme == "" {
			return ErrInvalidValue(dest.URI.String(), "uri")
		}
	} else if dest.ObjectReference == nil {
		return ErrMissingOneOf("uri", "[apiVersion, kind, name]")
	} else {
		return validateDestinationRef(*dest.ObjectReference)
	}
	return nil
}

func validateDestinationPath(path string) *FieldError {
	if strings.HasPrefix(path, "/") {
		if pu, err := url.Parse(path); err != nil {
			return ErrInvalidValue(path, CurrentField)
		} else if !equality.Semantic.DeepEqual(pu, &url.URL{Path: pu.Path}) {
			return ErrInvalidValue(path, CurrentField)
		}
	} else {
		return ErrInvalidValue(path, CurrentField)
	}
	return nil
}

func validateDestinationRef(ref corev1.ObjectReference) *FieldError {
	// Check the object.
	var errs *FieldError
	// Required Fields
	if ref.Name == "" {
		errs = errs.Also(ErrMissingField("name"))
	}
	if ref.APIVersion == "" {
		errs = errs.Also(ErrMissingField("apiVersion"))
	}
	if ref.Kind == "" {
		errs = errs.Also(ErrMissingField("kind"))
	}

	return errs
}
