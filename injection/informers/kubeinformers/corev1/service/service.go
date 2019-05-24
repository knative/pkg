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

package service

import (
	"context"

	corev1 "k8s.io/client-go/informers/core/v1"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/injection"
	"github.com/knative/pkg/injection/informers/kubeinformers/factory"
)

func init() {
	injection.Default.RegisterInformer(withServiceInformer)
}

// serviceInformerKey is used as the key for associating information
// with a context.Context.
type serviceInformerKey struct{}

func withServiceInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := factory.Get(ctx)
	inf := f.Core().V1().Services()
	return context.WithValue(ctx, serviceInformerKey{}, inf), inf.Informer()
}

// Get extracts the Kubernetes Service informer from the context.
func Get(ctx context.Context) corev1.ServiceInformer {
	untyped := ctx.Value(serviceInformerKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(corev1.ServiceInformer)
}
