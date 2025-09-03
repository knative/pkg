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

package metrics

import (
	"strings"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func temporalitySelectorFor(val string) metric.TemporalitySelector {
	switch strings.ToLower(val) {
	case "cumulative":
		return cumulativeTemporality
	case "delta":
		return deltaTemporality
	case "lowmemory":
		return lowMemory
	}
	return nil
}

// Copied from https://github.com/open-telemetry/opentelemetry-go/blob/8c8cd0a9caa0c7c90f929b4992f50475b5e1a303/exporters/otlp/otlpmetric/otlpmetrichttp/internal/oconf/envconfig.go#L184
func cumulativeTemporality(metric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func deltaTemporality(ik metric.InstrumentKind) metricdata.Temporality {
	switch ik {
	case metric.InstrumentKindCounter, metric.InstrumentKindHistogram, metric.InstrumentKindObservableCounter:
		return metricdata.DeltaTemporality
	default:
		return metricdata.CumulativeTemporality
	}
}

func lowMemory(ik metric.InstrumentKind) metricdata.Temporality {
	switch ik {
	case metric.InstrumentKindCounter, metric.InstrumentKindHistogram:
		return metricdata.DeltaTemporality
	default:
		return metricdata.CumulativeTemporality
	}
}
