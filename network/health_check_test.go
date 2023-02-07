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

package network

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
		header            http.Header
		expectedReadiness int
		expectedLiveness  int
	}{{
		name: "default health check handler",
		ctx:  context.Background(),
		header: http.Header{
			UserAgentKey: []string{KubeProbeUAPrefix},
		},
		expectedReadiness: http.StatusOK,
		expectedLiveness:  http.StatusOK,
	}, {
		name:              "default health check handler, no kubelet probe",
		ctx:               context.Background(),
		header:            http.Header{},
		expectedReadiness: http.StatusBadRequest,
		expectedLiveness:  http.StatusBadRequest,
	}, {
		name: "serve only readiness probes",
		ctx:  WithUserReadinessProbe(context.Background()),
		header: http.Header{
			UserAgentKey: []string{KubeProbeUAPrefix},
		},
		expectedReadiness: http.StatusOK,
		expectedLiveness:  http.StatusNotFound,
	}, {
		name: "serve only liveness probes",
		ctx:  WithUserLivenessProbe(context.Background()),
		header: http.Header{
			UserAgentKey: []string{KubeProbeUAPrefix},
		},
		expectedReadiness: http.StatusNotFound,
		expectedLiveness:  http.StatusOK,
	}, {
		name: "user provided health check handler",
		ctx: WithOverrideHealthCheckHandler(context.Background(), &HealthCheckHandler{
			ReadinessProbeRequestHandle: testHandler(),
			LivenessProbeRequestHandle:  testHandler(),
		}),
		header: http.Header{
			UserAgentKey: []string{KubeProbeUAPrefix},
		},
		expectedReadiness: http.StatusBadGateway,
		expectedLiveness:  http.StatusBadGateway,
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mux := muxWithHandles(tc.ctx)
			reqReadiness := http.Request{
				URL: &url.URL{
					Path: "/readiness",
				},
				Header: tc.header,
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
				Header: tc.header,
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
