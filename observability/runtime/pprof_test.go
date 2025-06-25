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

package runtime

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewProfilingHandler(t *testing.T) {
	h := NewProfilingHandler()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	h.ServeHTTP(w, r)

	resp := w.Result()
	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Errorf("unexpected status code %d want: %d", got, want)
	}

	w = httptest.NewRecorder()
	h.SetEnabled(true)
	h.ServeHTTP(w, r)

	resp = w.Result()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("unexpected status code %d want: %d", got, want)
	}
}

func TestNewProfilingServerPort(t *testing.T) {
	s := NewProfilingServer()

	if got, want := s.Server.Addr, ":8008"; got != want {
		t.Errorf("uexpected server port %q, want: %q", got, want)
	}

	t.Setenv(ProfilingPortEnvKey, "8181")
	s = NewProfilingServer()

	if got, want := s.Server.Addr, ":8181"; got != want {
		t.Errorf("uexpected server port %q, want: %q", got, want)
	}
}
