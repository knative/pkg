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

	"k8s.io/client-go/rest"
)

func TestContextNamespace(t *testing.T) {
	ctx := context.Background()

	if HasNamespaceScope(ctx) {
		t.Error("HasNamespaceScope() = true, wanted false")
	}

	want := "this-is-the-best-ns-evar"
	ctx = WithNamespaceScope(ctx, want)

	if !HasNamespaceScope(ctx) {
		t.Error("HasNamespaceScope() = false, wanted true")
	}

	if got := GetNamespaceScope(ctx); got != want {
		t.Errorf("GetNamespaceScope() = %v, wanted %v", got, want)
	}
}

func TestContextNamespaceScopes(t *testing.T) {
	ctx := context.Background()

	if got := GetNamespaceScopes(ctx); got != nil {
		t.Errorf("GetNamespaceScopes() = %v, wanted nil", got)
	}

	want := []string{"ns-a", "ns-b"}
	ctx = WithNamespaceScopes(ctx, want...)

	if got := GetNamespaceScopes(ctx); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("GetNamespaceScopes() = %v, wanted %v", got, want)
	}
}

func TestContextNamespaceScopesFallsBackToSingleScope(t *testing.T) {
	ctx := WithNamespaceScope(context.Background(), "only-ns")

	got := GetNamespaceScopes(ctx)
	if len(got) != 1 || got[0] != "only-ns" {
		t.Errorf("GetNamespaceScopes() = %v, wanted [only-ns]", got)
	}
}

func TestContextNamespaceScopesPreferMultiOverSingle(t *testing.T) {
	ctx := WithNamespaceScope(context.Background(), "single-ns")
	ctx = WithNamespaceScopes(ctx, "multi-a", "multi-b")

	got := GetNamespaceScopes(ctx)
	if len(got) != 2 || got[0] != "multi-a" || got[1] != "multi-b" {
		t.Errorf("GetNamespaceScopes() = %v, wanted [multi-a multi-b]", got)
	}
}

func TestContextConfig(t *testing.T) {
	ctx := context.Background()

	if cfg := GetConfig(ctx); cfg != nil {
		t.Errorf("GetConfig() = %v, wanted nil", cfg)
	}

	want := &rest.Config{}
	ctx = WithConfig(ctx, want)

	if cfg := GetConfig(ctx); cfg != want {
		t.Errorf("GetConfig() = %v, wanted %v", cfg, want)
	}
}

func TestResourceVersion(t *testing.T) {
	ctx := context.Background()

	if got, want := GetResourceVersion(ctx), ""; got != want {
		t.Errorf("GetResourceVersion() = %s, wanted %s", got, want)
	}

	want := "this-is-the-best-version-evar"
	ctx = WithResourceVersion(ctx, want)

	if got := GetResourceVersion(ctx); got != want {
		t.Errorf("GetResourceVersion() = %v, wanted %v", got, want)
	}
}
