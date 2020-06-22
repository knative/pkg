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

package propagation

import (
	"net/http"

	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/plugin/ochttp/propagation/tracecontext"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

// TraceContextB3 is a propagation.HTTPFormat that reads both TraceContext and B3 tracing
// formats, preferring TraceContext. It always writes both formats.
var TraceContextB3 = &HTTPFormatSequence{
	&tracecontext.HTTPFormat{},
	&b3.HTTPFormat{},
}

// HTTPFormatSequence is a propagation.HTTPFormat that applies multiple other propagation formats.
// For incoming requests, it will use the first SpanContext it can find, checked in the order of
// HTTPFormatSequence.Formats.
// For outgoing requests, it will apply all the formats to the outgoing request, in the order of
// HTTPFormatSequence.Formats.
type HTTPFormatSequence []propagation.HTTPFormat

var _ propagation.HTTPFormat = (*HTTPFormatSequence)(nil)

// SpanContextFromRequest satisfies the propagation.HTTPFormat interface.
func (h *HTTPFormatSequence) SpanContextFromRequest(req *http.Request) (trace.SpanContext, bool) {
	for _, format := range *h {
		if sc, ok := format.SpanContextFromRequest(req); ok {
			return sc, true
		}
	}
	return trace.SpanContext{}, false
}

// SpanContextToRequest satisfies the propagation.HTTPFormat interface.
func (h *HTTPFormatSequence) SpanContextToRequest(sc trace.SpanContext, req *http.Request) {
	for _, format := range *h {
		format.SpanContextToRequest(sc, req)
	}
}
