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

package b3tracecontext

import (
	"net/http"

	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/plugin/ochttp/propagation/tracecontext"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

// The propagation.HTTPFormats that this is built on.
var (
	b3Format     = &b3.HTTPFormat{}
	traceContext = &tracecontext.HTTPFormat{}
)

// HTTPFormat is a propagation.HTTPFormat that reads both B3 and TraceContext tracing headers,
// preferring TraceContext. It will write both formats.
type HTTPFormat struct{}

var _ propagation.HTTPFormat = (*HTTPFormat)(nil)

// SpanContextFromRequest satisfies the propagation.HTTPFormat interface.
func (*HTTPFormat) SpanContextFromRequest(req *http.Request) (trace.SpanContext, bool) {
	if sc, ok := traceContext.SpanContextFromRequest(req); ok {
		return sc, true
	}
	if sc, ok := b3Format.SpanContextFromRequest(req); ok {
		return sc, true
	}
	return trace.SpanContext{}, false
}

// SpanContextToRequest satisfies the propagation.HTTPFormat interface.
func (*HTTPFormat) SpanContextToRequest(sc trace.SpanContext, req *http.Request) {
	traceContext.SpanContextToRequest(sc, req)
	b3Format.SpanContextToRequest(sc, req)
}
