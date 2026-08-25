/*
Copyright 2026 The Knative Authors

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
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
)

func TestMetricAttributesDenyFilter(t *testing.T) {
	view := MetricAttributesDenyFilter([]string{"cloudevents.type", "messaging.destination.name"})

	stream, ok := view(metric.Instrument{Name: "some.metric"})
	if !ok {
		t.Fatal("view should match all instruments")
	}
	if stream.AttributeFilter == nil {
		t.Fatal("expected non-nil attribute filter")
	}

	denied := []attribute.KeyValue{
		attribute.String("cloudevents.type", "com.example.event"),
		attribute.String("messaging.destination.name", "my-destination"),
	}
	for _, kv := range denied {
		if stream.AttributeFilter(kv) {
			t.Errorf("attribute %s should be denied", kv.Key)
		}
	}

	allowed := []attribute.KeyValue{
		attribute.String("messaging.system", "knative"),
		attribute.Int("http.response.status_code", 200),
	}
	for _, kv := range allowed {
		if !stream.AttributeFilter(kv) {
			t.Errorf("attribute %s should be allowed", kv.Key)
		}
	}
}

func TestMetricAttributesDenyFilterMatchesAllInstruments(t *testing.T) {
	view := MetricAttributesDenyFilter([]string{"cloudevents.type"})

	instruments := []string{
		"kn.eventing.dispatch.duration",
		"http.server.request.duration",
		"custom.metric",
	}
	for _, name := range instruments {
		if _, ok := view(metric.Instrument{Name: name}); !ok {
			t.Errorf("view should match instrument %s", name)
		}
	}
}
