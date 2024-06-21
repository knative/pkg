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
	"strings"
	"testing"

	"knative.dev/pkg/depcheck"
)

// TestExample doesn't follow the Go Example style because it isn't well
// suited for libraries needing *testing.T
func TestExample(t *testing.T) {
	// For larger packages, it can make the most sense to simply avoid
	// known "heavy" packages, which pull in large amount of code or data.
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

	// Sample failure case, duck clearly relies on corev1 for all assortment of things.
	if err := depcheck.CheckNoDependency("knative.dev/pkg/apis/duck", []string{"k8s.io/api/core/v1"}); err == nil {
		t.Error("CheckNoDependency() = nil, wanted error")
	} else if !strings.Contains(err.Error(), "knative.dev/pkg/tracker") {
		t.Errorf("CheckNoDependency() = %v, expected to contain: %v", err, "knative.dev/pkg/tracker")
	} else {
		t.Log("CheckNoDependency() =", err)
	}

	// For small packages, it can make the most sense to curate
	// the external dependencies very carefully.
	depcheck.AssertOnlyDependencies(t, map[string][]string{
		// Example libraries with very limited dependencies.
		"knative.dev/pkg/pool": {
			"context",
			"sync",
			"golang.org/x/sync/errgroup",
		},
		"knative.dev/pkg/ptr": {
			"time",
		},
	})

	// Sample failure case, doesn't include transitive dependencies!
	if err := depcheck.CheckOnlyDependencies("knative.dev/pkg/depcheck", map[string]struct{}{
		"fmt":                            {},
		"sort":                           {},
		"strings":                        {},
		"testing":                        {},
		"golang.org/x/tools/go/packages": {},
	}); err == nil {
		t.Error("CheckOnlyDependencies() = nil, wanted error")
	} else if !strings.Contains(err.Error(), "golang.org/x/tools/go/gcexportdata") {
		t.Errorf("CheckOnlyDependencies() = %v, expected to contain: %v", err, "golang.org/x/tools/go/gcexportdata")
	}
}
