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

package fake

import (
	"context"
	"testing"

	"k8s.io/client-go/rest"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/injection"
)

func TestRegistration(t *testing.T) {
	ctx := context.Background()

	// Get before registration
	if empty := Get(ctx); empty != nil {
		t.Errorf("Unexpected informer factory: %v", empty)
	}

	// Check how many informer factories have registered.
	inffs := injection.Fake.GetInformerFactories()
	if want, got := 1, len(inffs); want != got {
		t.Errorf("GetInformerFactories() = %d, wanted %d", want, got)
	}

	// Setup the informers.
	var infs []controller.Informer
	ctx, infs = injection.Fake.SetupInformers(ctx, &rest.Config{})

	// We should see that a single informer was set up.
	if want, got := 0, len(infs); want != got {
		t.Errorf("SetupInformers() = %d, wanted %d", want, got)
	}

	// Get our informer from the context.
	if inf := Get(ctx); inf == nil {
		t.Error("Get() = nil, wanted non-nil")
	}
}
