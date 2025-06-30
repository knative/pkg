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

package webhook

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"knative.dev/pkg/observability/attributekey"
)

const (
	scopeName = "knative.dev/pkg/webhook"

	WebhookTypeAdmission  = "admission"
	WebhookTypeDefaulting = "defaulting"
	WebhookTypeValidation = "validation"
	WebhookTypeConversion = "conversion"
)

var (
	// WebhookType is an attribute that specifies whether the type of webhook is an admission
	// eg. (defaulting/validation) or conversion
	WebhookType = attributekey.String("kn.webhook.type")

	AdmissionOperation   = attributekey.String("kn.webhook.admission.operation")
	AdmissionGroup       = attributekey.String("kn.webhook.admission.group")
	AdmissionVersion     = attributekey.String("kn.webhook.admission.version")
	AdmissionKind        = attributekey.String("kn.webhook.admission.kind")
	AdmissionSubresource = attributekey.String("kn.webhook.admission.subresource")
	AdmissionAllowed     = attributekey.Bool("kn.webhook.admission.result.allowed")

	ConversionDesiredAPIVersion = attributekey.String("kn.webhook.convert.desired_api.version")
	ConversionResultStatus      = attributekey.String("kn.webhook.conversion.result.status")
)

type metrics struct {
	handlerDuration metric.Float64Histogram
}

func newMetrics(o Options) *metrics {
	var (
		m        metrics
		err      error
		provider = o.MeterProvider
	)

	if provider == nil {
		provider = otel.GetMeterProvider()
	}

	meter := provider.Meter(scopeName)

	m.handlerDuration, err = meter.Float64Histogram(
		"kn.webhook.handler.duration",
		metric.WithDescription("The duration of task execution."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10),
	)
	if err != nil {
		panic(err)
	}

	return &m
}

func (m *metrics) recordHandlerDuration(ctx context.Context, d time.Duration, ro ...metric.RecordOption) {
	elapsedTime := float64(d) / float64(time.Second)
	m.handlerDuration.Record(ctx, elapsedTime, ro...)
}
