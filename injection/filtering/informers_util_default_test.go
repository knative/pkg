/*
Copyright 2022 The Knative Authors

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

package filtering

import (
	"context"
	"testing"

	"k8s.io/client-go/rest"
	secretfilteredfakeinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret/filtered/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/factory/filtered/fake"
	"knative.dev/pkg/injection"
)

func TestWithoutFilteringInformersSetup(t *testing.T) {
	t.Setenv("SYSTEM_NAMESPACE", "system")
	ctx := InformersFilterByLabel(context.Background())
	ctx, infs := injection.Fake.SetupInformers(ctx, &rest.Config{})
	if want, got := 1, len(infs); got != want {
		t.Errorf("SetupInformers() = %d, wanted %d", got, want)
	}
	_ = secretfilteredfakeinformer.Get(ctx, "")
}
