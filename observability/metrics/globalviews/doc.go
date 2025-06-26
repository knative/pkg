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

// OpenTelemetry views allow you to override the default behaviour
// of the SDK. It defines how data should be collected for certain
// instruments.
//
// OpenCensus allowed views to be registered globally to simplify
// setup. In contrast, OpenTelemetry requires views to be provided
// when creating a [MeterProvider] using the [WithView] option. This
// was done to provide flexibility of having multiple metrics pipelines.
//
// To provide a similar UX experience the globalviews provides a similar
// way to register OTel views globally. This allows Knative maintainers
// to author packages with metrics and include views that can be consumed
// by packages such as sharedmain.
//
// [MeterProvider]: https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric#NewMeterProvider
// [WithView]: https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric#WithView
package globalviews
