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

package v1alpha1

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

// Destination represents a target of an invocation over HTTP.
type Destination struct {
	// Ref points to an Addressable.
	// +optional
	Ref *corev1.ObjectReference `json:"ref,omitempty"`

	// +optional
	DeprecatedAPIVersion string `json:"apiVersion,omitempty"`

	// +optional
	DeprecatedKind string `json:"kind,omitempty"`

	// +optional
	DeprecatedName string `json:"name,omitempty"`

	// +optional
	DeprecatedNamespace string `json:"namespace,omitempty"`

	// URI can be an absolute URL(non-empty scheme and non-empty host) pointing to the target or a relative URI. Relative URIs will be resolved using the base URI retrieved from Ref.
	// +optional
	URI *apis.URL `json:"uri,omitempty"`
}

func (current *Destination) Validate(ctx context.Context) *apis.FieldError {
	if current != nil {
		return ValidateDestination(*current).ViaField(apis.CurrentField)
	} else {
		return nil
	}
}

func ValidateDestination(dest Destination) *apis.FieldError {
	var deprecatedObjectReference *corev1.ObjectReference
	if dest.DeprecatedAPIVersion == "" && dest.DeprecatedKind == "" && dest.DeprecatedName == "" && dest.DeprecatedNamespace == "" {
		deprecatedObjectReference = nil
	} else {
		deprecatedObjectReference = &corev1.ObjectReference{
			Kind:            dest.DeprecatedKind,
			APIVersion:       dest.DeprecatedAPIVersion,
			Name:            dest.DeprecatedName,
			Namespace:       dest.DeprecatedNamespace,
		}
	}
	if dest.Ref != nil && deprecatedObjectReference != nil {
		return apis.ErrGeneric("Ref and [apiVersion, kind, name] can't be both present", "[apiVersion, kind, name]", "ref")
	}

	var ref *corev1.ObjectReference
	if dest.Ref != nil {
		ref = dest.Ref
	} else {
		ref = deprecatedObjectReference
	}
	if ref == nil && dest.URI == nil {
		return apis.ErrGeneric("expected at least one, got none", "[apiVersion, kind, name]", "ref", "uri")
	}

	if ref != nil && dest.URI != nil && dest.URI.URL().IsAbs() {
		return apis.ErrGeneric("Absolute URI is not allowed when Ref or [apiVersion, kind, name] is present", "[apiVersion, kind, name]", "ref", "uri")
	}
	// IsAbs() check whether the URL has a non-empty scheme. Besides the non-empty scheme, we also require dest.URI has a non-empty host
	if ref == nil && dest.URI != nil && (!dest.URI.URL().IsAbs() || dest.URI.Host == "") {
			return apis.ErrInvalidValue("Relative URI is not allowed when Ref and [apiVersion, kind, name] is absent",  "uri")
		}
	if ref != nil && dest.URI == nil{
		if dest.Ref != nil {
			return validateDestinationRef(*ref).ViaField("ref")
		} else {
			return validateDestinationRef(*ref)
		}
	}
	return nil
}


func validateDestinationRef(ref corev1.ObjectReference) *apis.FieldError {
	// Check the object.
	var errs *apis.FieldError
	// Required Fields
	if ref.Name == "" {
		errs = errs.Also(apis.ErrMissingField("name"))
	}
	if ref.APIVersion == "" {
		errs = errs.Also(apis.ErrMissingField("apiVersion"))
	}
	if ref.Kind == "" {
		errs = errs.Also(apis.ErrMissingField("kind"))
	}

	return errs
}
