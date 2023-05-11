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

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/webhook/resourcesemantics"
)

type options struct {
	path                  string
	types                 map[schema.GroupVersionKind]resourcesemantics.GenericCRD
	wc                    func(context.Context) context.Context
	disallowUnknownFields bool
	callbacks             map[schema.GroupVersionKind]Callback
	kinds                 map[schema.GroupKind]GroupKindConversion
}

func NewOptions() *options {
	return &options{}
}

type OptionFunc func(*options)

func WithCallbacks(callbacks map[schema.GroupVersionKind]Callback) OptionFunc {
	return func(o *options) {
		o.callbacks = callbacks
	}
}

func (o *options) GetCallbacks() map[schema.GroupVersionKind]Callback {
	return o.callbacks
}

func WithKinds(kinds map[schema.GroupKind]GroupKindConversion) OptionFunc {
	return func(o *options) {
		o.kinds = kinds
	}
}

func (o *options) GetKinds() map[schema.GroupKind]GroupKindConversion {
	return o.kinds
}

func WithPath(path string) OptionFunc {
	return func(o *options) {
		o.path = path
	}
}

func (o *options) GetPath() string {
	return o.path
}

func WithTypes(types map[schema.GroupVersionKind]resourcesemantics.GenericCRD) OptionFunc {
	return func(o *options) {
		o.types = types
	}
}

func (o *options) GetTypes() map[schema.GroupVersionKind]resourcesemantics.GenericCRD {
	return o.types
}

func WithWrapContext(f func(context.Context) context.Context) OptionFunc {
	return func(o *options) {
		o.wc = f
	}
}

func (o *options) GetWrapContext() func(context.Context) context.Context {
	return o.wc
}

func WithDisallowUnknownFields() OptionFunc {
	return func(o *options) {
		o.disallowUnknownFields = true
	}
}

func (o *options) GetDisallowUnknownFields() bool {
	return o.disallowUnknownFields
}
