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

package profiling

import (
	"net/http"
	"net/http/pprof"
)

// ProfilingPort is the port where we expose profiling information if profiling is enabled
const ProfilingPort = ":8008"

// Handler holds the main HTTP handler and a flag indicating
// whether the handler is active
type Handler struct {
	Enabled bool
	Handler http.Handler
}

// NewHandler create a new ProfilingHandler which serves runtime profiling data
// according to the given context path
func NewHandler() *Handler {
	const pprofPrefix = "/debug/pprof/"

	mux := http.NewServeMux()
	mux.HandleFunc(pprofPrefix, pprof.Index)
	mux.HandleFunc(pprofPrefix+"cmdline", pprof.Cmdline)
	mux.HandleFunc(pprofPrefix+"profile", pprof.Profile)
	mux.HandleFunc(pprofPrefix+"symbol", pprof.Symbol)
	mux.HandleFunc(pprofPrefix+"trace", pprof.Trace)

	return &Handler{
		Enabled: false,
		Handler: mux,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Enabled {
		h.Handler.ServeHTTP(w, r)
	} else {
		http.NotFoundHandler().ServeHTTP(w, r)
	}
}

// NewServer creates a new http server that exposes profiling data using the
// HTTP handler that is passed as an argument
func NewServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    ProfilingPort,
		Handler: handler,
	}
}
