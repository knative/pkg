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
package metricskey_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/resource"
	"knative.dev/pkg/metrics/metricskey"
)

func TestResourceContext(t *testing.T) {
	ctx := context.Background()

	if r := metricskey.GetResource(ctx); r != nil {
		t.Errorf("Got Resource %+v from context, expected nil", r)
	}

	orig := resource.Resource{
		Type:   "foo",
		Labels: map[string]string{"a": "1", "b": "2"},
	}

	ctx = metricskey.WithResource(ctx, orig)

	r := metricskey.GetResource(ctx)
	if r == nil {
		t.Fatal("Expected non-nil Resource from context, got nil")
	}

	if diff := cmp.Diff(orig, *r); diff != "" {
		t.Errorf("Expected same Resource: diff(-want,+got)\n%s", diff)
	}
}
