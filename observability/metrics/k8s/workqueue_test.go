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

package k8s

import (
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestNewWorkqueueMetricsProvider(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	p, err := NewWorkqueueMetricsProvider(WithMeterProvider(provider))
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if p == nil {
		t.Fatal("provider returned was nil")
	}
}

func TestNewWorkqueueMetricsProviderGlobal(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	otel.SetMeterProvider(provider)
	p, err := NewWorkqueueMetricsProvider()
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if p == nil {
		t.Fatal("provider returned was nil")
	}
}

func TestWorkqueueMetricsProviderHelpers(t *testing.T) {
	r := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r))
	p, err := NewWorkqueueMetricsProvider(WithMeterProvider(provider))
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if m := p.NewAddsMetric("queue"); m == nil {
		t.Error("NewAddsMetric() returned nil")
	}
	if m := p.NewDepthMetric("queue"); m == nil {
		t.Error("NewDepthMetric() returned nil")
	}
	if m := p.NewLatencyMetric("queue"); m == nil {
		t.Error("NewLatencyMetric() returned nil")
	}
	if m := p.NewWorkDurationMetric("queue"); m == nil {
		t.Error("NewWorkDurationMetric() returned nil")
	}
	if m := p.NewLongestRunningProcessorSecondsMetric("queue"); m == nil {
		t.Error("NewLongestRunningProcessorSecondsMetric() returned nil")
	}
	if m := p.NewUnfinishedWorkSecondsMetric("queue"); m == nil {
		t.Error("NewUnfinishedWorkSecondsMetric() returned nil")
	}
	if m := p.NewRetriesMetric("queue"); m == nil {
		t.Error("NewRetriesMetric() returned nil")
	}
}
