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

package tracing_test

import (
	"testing"

	. "knative.dev/pkg/tracing"
	"knative.dev/pkg/tracing/config"
	. "knative.dev/pkg/tracing/testing"
)

func TestOpenCensusTracerGlobalLifecycle(t *testing.T) {
	reporter, co := FakeZipkinExporter()
	defer reporter.Close()
	oct := NewOpenCensusTracer(co)
	// Apply a config to make us the global OCT
	if err := oct.ApplyConfig(&config.Config{}); err != nil {
		t.Fatalf("Failed to ApplyConfig on tracer: %v", err)
	}

	otherOCT := NewOpenCensusTracer(co)
	if err := otherOCT.ApplyConfig(&config.Config{}); err == nil {
		t.Fatalf("Expected error when applying config to second OCT.")
	}

	if err := oct.Finish(); err != nil {
		t.Fatalf("Failed to finish OCT: %v", err)
	}

	if err := otherOCT.ApplyConfig(&config.Config{}); err != nil {
		t.Fatalf("Failed to ApplyConfig on OtherOCT after finishing OCT: %v", err)
	}
	otherOCT.Finish()
}
