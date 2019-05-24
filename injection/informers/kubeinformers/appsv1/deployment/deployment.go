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

package deployment

import (
	"context"

	appsv1 "k8s.io/client-go/informers/apps/v1"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/injection"
	"github.com/knative/pkg/injection/informers/kubeinformers/factory"
)

func init() {
	injection.Default.RegisterInformer(withDeployInformer)
}

// deployInformerKey is used as the key for associating information
// with a context.Context.
type deployInformerKey struct{}

func withDeployInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := factory.Get(ctx)
	inf := f.Apps().V1().Deployments()
	return context.WithValue(ctx, deployInformerKey{}, inf), inf.Informer()
}

// Get extracts the Kubernetes Deployment informer from the context.
func Get(ctx context.Context) appsv1.DeploymentInformer {
	untyped := ctx.Value(deployInformerKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(appsv1.DeploymentInformer)
}
