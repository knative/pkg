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

	// URI can be a absolute URI which points to the URI designation or it can be a relative URI will be merged to the URI retrieved from Ref.
	// +optional
	URI *apis.URL `json:"uri,omitempty"`
}

// NewDestination constructs a Destination from an object reference as a convenience.
func NewDestination(obj *corev1.ObjectReference) (*Destination, error) {
	return &Destination{
		Ref: obj,
	}, nil
}

// NewDestinationURI constructs a Destination from a URI.
func NewDestinationURI(uri *apis.URL) (*Destination, error) {
	return &Destination{
		URI: uri,
	}, nil
}


func (current *Destination) Validate(ctx context.Context) *apis.FieldError {
	if current != nil {
		return validateDestination(*current).ViaField(apis.CurrentField)
	} else {
		return nil
	}
}

func validateDestination(dest Destination) *apis.FieldError {
	if dest.Ref == nil && dest.URI == nil {
		return apis.ErrGeneric("expected at least one, got neither", "uri", "[apiVersion, kind, name]")
	}
	if dest.Ref != nil && dest.URI != nil && dest.URI.URL().IsAbs() {
		return apis.ErrGeneric("URI with absolute URL is not allowed when Ref exists", "uri", "[apiVersion, kind, name]")
	}
	if dest.Ref != nil && dest.URI == nil{
		return validateDestinationRef(*dest.Ref)
	}
	if dest.Ref == nil && dest.URI != nil && (!dest.URI.URL().IsAbs() || dest.URI.Host == "") {
			return apis.ErrInvalidValue(dest.URI.String(), "uri")
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
