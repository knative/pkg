/*
Copyright 2023 The Knative Authors

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

package common

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/webhook"
)

// CallbackFunc is the function to be invoked.
type CallbackFunc func(ctx context.Context, unstructured *unstructured.Unstructured) error

// Callback is a generic function to be called by a consumer of defaulting.
type Callback struct {
	// Function is the callback to be invoked.
	Function CallbackFunc

	// SupportedVerbs are the verbs supported for the callback.
	// The function will only be called on these actions.
	SupportedVerbs map[webhook.Operation]struct{}
}

// ConvertibleObject defines the functionality our API types
// are required to implement in order to be convertible from
// one version to another
//
// Optionally if the object implements apis.Defaultable the
// ConversionController will apply defaults before returning
// the response
type ConvertibleObject interface {
	// ConvertTo(ctx, to)
	// ConvertFrom(ctx, from)
	apis.Convertible

	// DeepCopyObject()
	// GetObjectKind() => SetGroupVersionKind(gvk)
	runtime.Object
}

// GroupKindConversion specifies how a specific Kind for a given
// group should be converted
type GroupKindConversion struct {
	// DefinitionName specifies the CustomResourceDefinition that should
	// be reconciled with by the controller.
	//
	// The conversion webhook configuration will be updated
	// when the CA bundle changes
	DefinitionName string

	// HubVersion specifies which version of the CustomResource supports
	// conversions to and from all types
	//
	// It is expected that the Zygotes map contains an entry for the
	// specified HubVersion
	HubVersion string

	// Zygotes contains a map of version strings (ie. v1, v2) to empty
	// ConvertibleObject objects
	//
	// During a conversion request these zygotes will be deep copied
	// and manipulated using the apis.Convertible interface
	Zygotes map[string]ConvertibleObject
}
