/*
Copyright 2025 The Knative Authors

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

package sharedmain

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/metric"
)

func TestOTelViewOption(t *testing.T) {
	ctx := context.Background()
	views := OTelViews(ctx)

	if len(views) != 0 {
		t.Error("expected no views")
	}

	view := func(metric.Instrument) (metric.Stream, bool) {
		return metric.Stream{}, false
	}

	ctx = WithOTelViews(ctx, view)
	views = OTelViews(ctx)

	if len(views) != 1 {
		t.Error("expecte a single view")
	}
}
