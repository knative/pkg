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
	"net/url"
	"path"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	"knative.dev/pkg/apis"
)

// Destination represents a target of an invocation over HTTP.
type Destination struct {
	// ObjectReference points to an Addressable.
	*corev1.ObjectReference `json:",inline"`

	// URI is for direct URI Designations.
	URI *apis.URL `json:"uri,omitempty"`

	// Path is used with the resulting URL from Addressable ObjectReference or
	// URI. Must start with `/`. Will be appended to the path of the resulting
	// URL from the Addressable, or URI.
	Path *string `json:"path,omitempty"`
}

// NewDestination constructs a Destination from an object reference as a convenience.
func NewDestination(obj *corev1.ObjectReference) *Destination {
	return &Destination{
		ObjectReference: obj,
	}
}

// NewDestinationURI constructs a Destination from a URI.
func NewDestinationURI(uri apis.URL) *Destination {
	dest := &Destination{
		URI: &uri,
	}

	// Check the URI for a path -- path must be only in the Destination.Path field.
	if uri.Path != "" {
		// Create a new path string on the heap for the destination
		path := uri.Path
		dest.Path = &path

		// Mutate the URI reference to not have a path
		dest.URI.Path = ""
		dest.URI.RawPath = ""
	}

	return dest
}

// WithPath mutates the path set for the Destination; for use with constructors.
func (current *Destination) WithPath(newpath string) *Destination {
	if current.Path != nil {
		newpath = path.Join(*current.Path, newpath)
	}
	current.Path = &newpath
	return current
}

func (current *Destination) Validate(ctx context.Context) *apis.FieldError {
	if current != nil {
		errs := validateDestination(*current).ViaField(apis.CurrentField)
		if current.Path != nil {
			errs = errs.Also(validateDestinationPath(*current.Path).ViaField("path"))
		}
		return errs
	} else {
		return nil
	}
}

func validateDestination(dest Destination) *apis.FieldError {
	if dest.URI != nil {
		if dest.ObjectReference != nil {
			return apis.ErrMultipleOneOf("uri", "[apiVersion, kind, name]")
		}
		if dest.URI.Host == "" || dest.URI.Scheme == "" {
			return apis.ErrInvalidValue(dest.URI.String(), "uri")
		}
	} else if dest.ObjectReference == nil {
		return apis.ErrMissingOneOf("uri", "[apiVersion, kind, name]")
	} else {
		return validateDestinationRef(*dest.ObjectReference)
	}
	return nil
}

func validateDestinationPath(path string) *apis.FieldError {
	if strings.HasPrefix(path, "/") {
		if pu, err := url.Parse(path); err != nil {
			return apis.ErrInvalidValue(path, apis.CurrentField)
		} else if !equality.Semantic.DeepEqual(pu, &url.URL{Path: pu.Path}) {
			return apis.ErrInvalidValue(path, apis.CurrentField)
		}
	} else {
		return apis.ErrInvalidValue(path, apis.CurrentField)
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
