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

package apis_test

import (
	"testing"

	"knative.dev/pkg/depcheck"
)

func TestNoDeps(t *testing.T) {
	depcheck.AssertNoDependency(t, map[string][]string{
		"knative.dev/pkg/apis":                depcheck.KnownHeavyDependencies,
		"knative.dev/pkg/apis/duck":           depcheck.KnownHeavyDependencies,
		"knative.dev/pkg/apis/duck/ducktypes": depcheck.KnownHeavyDependencies,
		"knative.dev/pkg/apis/duck/v1alpha1":  depcheck.KnownHeavyDependencies,
		"knative.dev/pkg/apis/duck/v1beta1":   depcheck.KnownHeavyDependencies,
		"knative.dev/pkg/apis/duck/v1":        depcheck.KnownHeavyDependencies,
	})
}
