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

/*
Package monitoring provides common methods for all the monitoring components used in the tests

This package exposes following methods:

	SetupZipkinTracing(*kubernetes.Clientset) error
		SetupZipkinTracing sets up zipkin tracing by setting up port-forwarding from
		localhost to zipkin pod on the cluster. On successful setup this method sets
		an internal flag zipkinTracingEnabled to true.
*/
package monitoring
