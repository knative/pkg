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

package metricskey

import (
	"context"

	"go.opencensus.io/resource"
)

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// userKey is the key for opencensus Resource values in Contexts. It is
// unexported; clients use metricskey.NewContext and metricskey.FromContext
// instead of using this key directly.
var resourceKey key

// NewContext returns a new Context that carries value u.
func NewContext(ctx context.Context, r *resource.Resource) context.Context {
	return context.WithValue(ctx, resourceKey, r)
}

// FromContext returns the User value stored in ctx, if any.
func FromContext(ctx context.Context) (*resource.Resource, bool) {
	r, ok := ctx.Value(resourceKey).(*resource.Resource)
	return r, ok
}
