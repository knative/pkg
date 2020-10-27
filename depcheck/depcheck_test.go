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

// Package depcheck_test demonstrates the usage of depcheck.
// Tests should be put into the special `_test` package to avoid
// polluting the dependencies of the package they are testing.
package depcheck_test

import (
	"testing"

	"knative.dev/pkg/depcheck"
)

// TestExample doesn't follow the Go Example style because it isn't well
// suited for libraries needing *testing.T
func TestExample(t *testing.T) {
	depcheck.AssertNoDependency(t, map[string][]string{
		// Our duck types shouldn't depend on fuzzers.
		"knative.dev/pkg/apis/duck/v1": {
			"k8s.io/apimachinery/pkg/api/apitesting/fuzzer",
		},
		"knative.dev/pkg/apis/duck/v1beta1": {
			"k8s.io/apimachinery/pkg/api/apitesting/fuzzer",
		},

		// We intentionally avoid using the Kubernetes "sets" package.
		"knative.dev/pkg/depcheck": {
			"k8s.io/apimachinery/pkg/util/sets",
		},
	})
}
