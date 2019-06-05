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

package injection

import (
	"context"
	"testing"

	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/controller"
)

func injectFooController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return nil
}

func injectBarController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return nil
}

func TestRegisterController(t *testing.T) {
	i := &impl{}

	if want, got := 0, len(i.GetControllers()); got != want {
		t.Errorf("GetControllers() = %d, wanted %d", want, got)
	}

	i.RegisterController(injectFooController)

	if want, got := 1, len(i.GetControllers()); got != want {
		t.Errorf("GetControllers() = %d, wanted %d", want, got)
	}

	i.RegisterController(injectBarController)

	if want, got := 2, len(i.GetControllers()); got != want {
		t.Errorf("GetControllers() = %d, wanted %d", want, got)
	}
}
