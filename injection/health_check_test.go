/*
Copyright 2023 The Knative Authors

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

package injection

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHealthCheckHandler(t *testing.T) {
	tests := []struct {
		name              string
		ctx               context.Context
		expectedReadiness int
		expectedLiveness  int
	}{{
		name:              "user provided no handlers, default health check handlers are used",
		ctx:               context.Background(),
		expectedReadiness: http.StatusOK,
		expectedLiveness:  http.StatusOK,
	}, {
		name:              "user provided readiness health check handler, liveness default handler is used",
		ctx:               AddReadiness(context.Background(), testHandler()),
		expectedReadiness: http.StatusBadGateway,
		expectedLiveness:  http.StatusOK,
	}, {
		name:              "user provided liveness health check handler, readiness default handler is used",
		ctx:               AddLiveness(context.Background(), testHandler()),
		expectedReadiness: http.StatusOK,
		expectedLiveness:  http.StatusBadGateway,
	}, {
		name:              "user provided custom probes",
		ctx:               AddReadiness(AddLiveness(context.Background(), testHandler()), testHandler()),
		expectedReadiness: http.StatusOK,
		expectedLiveness:  http.StatusBadGateway,
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mux := muxWithHandles(tc.ctx)
			reqReadiness := http.Request{
				URL: &url.URL{
					Path: "/readiness",
				},
			}
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, &reqReadiness)
			if got, want := resp.Code, tc.expectedReadiness; got != want {
				t.Errorf("Probe status = %d, wanted %d", got, want)
			}
			reqLiveness := http.Request{
				URL: &url.URL{
					Path: "/health",
				},
			}
			resp = httptest.NewRecorder()
			mux.ServeHTTP(resp, &reqLiveness)
			if got, want := resp.Code, tc.expectedLiveness; got != want {
				t.Errorf("Probe status = %d, wanted %d", got, want)
			}
		})
	}
}

func testHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "test", http.StatusBadGateway)
	}
}
