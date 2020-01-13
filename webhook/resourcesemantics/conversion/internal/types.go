/*
Copyright 2020 The Knative Authors

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

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

const (
	Group = "webhook.pkg.knative.dev"
	Kind  = "Resource"

	ErrorMarshal     = "marshal"
	ErrorUnmarshal   = "unmarshal"
	ErrorConvertUp   = "convertUp"
	ErrorConvertDown = "convertDown"
)

type (
	// V1Resource will never has a prefix or suffix on Spec.Property
	// This type is used for testing conversion webhooks
	//
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	V1Resource struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`
		Spec              Spec `json:"spec"`
	}

	// V2Resource will always have a 'prefix/' in front of it's property
	// This type is used for testing conversion webhooks
	//
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	V2Resource struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`
		Spec              Spec `json:"spec"`
	}

	// V3Resource will always have a '/suffix' in front of it's property
	// This type is used for testing conversion webhooks
	//
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	V3Resource struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`
		Spec              Spec `json:"spec"`
	}

	// ErrorResource explodes in various settings depending on the property
	// set. Use the Error* constants
	//
	//This type is used for testing conversion webhooks
	//
	// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
	ErrorResource struct {
		// We embed the V1Resource as an easy way to still marshal & unmarshal
		// this type without infinite loops - since we override the methods
		// in order to induce failures
		V1Resource `json:",inline"`
	}

	// Spec holds our fancy string property
	Spec struct {
		Property string `json:"prop"`
	}
)

var (
	_ apis.Convertible = (*V1Resource)(nil)
	_ apis.Convertible = (*V2Resource)(nil)
	_ apis.Convertible = (*V3Resource)(nil)
	_ apis.Convertible = (*ErrorResource)(nil)
)

func NewV1(prop string) *V1Resource {
	return &V1Resource{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: Group + "/v1",
		},
		Spec: Spec{
			Property: prop,
		},
	}
}

func NewV2(prop string) *V2Resource {
	return &V2Resource{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: Group + "/v2",
		},
		Spec: Spec{
			Property: fmt.Sprintf("prefix/%s", prop),
		},
	}
}

func NewV3(prop string) *V3Resource {
	return &V3Resource{
		TypeMeta: metav1.TypeMeta{
			Kind:       Kind,
			APIVersion: Group + "/v3",
		},
		Spec: Spec{
			Property: fmt.Sprintf("%s/suffix", prop),
		},
	}
}

func NewErrorResource(failure string) *ErrorResource {
	return &ErrorResource{
		V1Resource: V1Resource{
			TypeMeta: metav1.TypeMeta{
				Kind:       Kind,
				APIVersion: Group + "/error",
			},
			Spec: Spec{
				Property: failure,
			},
		},
	}
}

func (r *V1Resource) ConvertUp(ctx context.Context, to apis.Convertible) error {
	switch sink := to.(type) {
	case *V2Resource:
		sink.Spec.Property = "prefix/" + r.Spec.Property
	case *V3Resource:
		sink.Spec.Property = r.Spec.Property + "/suffix"
	case *ErrorResource:
		sink.Spec.Property = r.Spec.Property
	case *V1Resource:
		sink.Spec.Property = r.Spec.Property
	default:
		return fmt.Errorf("unsupported type %T", sink)
	}
	return nil
}

func (r *V1Resource) ConvertDown(ctx context.Context, from apis.Convertible) error {
	switch source := from.(type) {
	case *V2Resource:
		r.Spec.Property = strings.TrimPrefix(source.Spec.Property, "prefix/")
	case *V3Resource:
		r.Spec.Property = strings.TrimSuffix(source.Spec.Property, "/suffix")
	case *ErrorResource:
		r.Spec.Property = source.Spec.Property
	case *V1Resource:
		r.Spec.Property = source.Spec.Property
	default:
		return fmt.Errorf("unsupported type %T", source)
	}
	return nil
}

func (*V2Resource) ConvertUp(ctx context.Context, to apis.Convertible) error {
	panic("unimplemented")
}
func (*V2Resource) ConvertDown(ctx context.Context, from apis.Convertible) error {
	panic("unimplemented")
}
func (*V3Resource) ConvertUp(ctx context.Context, to apis.Convertible) error {
	panic("unimplemented")
}
func (*V3Resource) ConvertDown(ctx context.Context, from apis.Convertible) error {
	panic("unimplemented")
}
func (r *ErrorResource) ConvertUp(ctx context.Context, to apis.Convertible) error {
	if r.Spec.Property == ErrorConvertUp {
		return errors.New("boooom - convert up!")
	}

	return r.V1Resource.ConvertUp(ctx, to)
}

func (r *ErrorResource) ConvertDown(ctx context.Context, from apis.Convertible) error {
	err := r.V1Resource.ConvertDown(ctx, from)

	if err == nil && r.Spec.Property == ErrorConvertDown {
		err = errors.New("boooom - convert down!")
	}
	return err
}

func (e *ErrorResource) UnmarshalJSON(data []byte) (err error) {
	err = json.Unmarshal(data, &e.V1Resource)
	if err == nil && e.Spec.Property == ErrorUnmarshal {
		err = errors.New("boooom - unmarshal json!")
	}
	return
}

func (e *ErrorResource) MarshalJSON() ([]byte, error) {
	if e.Spec.Property == ErrorMarshal {
		return nil, errors.New("boooom - marshal json!")
	}
	return json.Marshal(e.V1Resource)
}
