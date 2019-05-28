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

// Code generated by injection-gen. DO NOT EDIT.

package endpoints

import (
	"context"

	factory "github.com/knative/pkg/client/injection/informers/core/factory"
	controller "github.com/knative/pkg/controller"
	injection "github.com/knative/pkg/injection"
	v1 "k8s.io/client-go/informers/core/v1"
)

func init() {
	injection.Default.RegisterInformer(withInformer)
}

// key is used for associating the Informer inside the context.Context.
type Key struct{}

func withInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := factory.Get(ctx)
	inf := f.Core().V1().Endpoints()
	return context.WithValue(ctx, Key{}, inf), inf.Informer()
}

// Get extracts the typed informer from the context.
func Get(ctx context.Context) v1.EndpointsInformer {
	untyped := ctx.Value(Key{})
	if untyped == nil {
		return nil
	}
	return untyped.(v1.EndpointsInformer)
}
