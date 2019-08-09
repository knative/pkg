/*
Copyright 2019 The Knative Authors.

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
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

func TestUpdateFromConfigMap(t *testing.T) {
	observabilityConfigTests := []struct {
		name           string
		wantEnabled    bool
		wantStatusCode int
		config         *corev1.ConfigMap
	}{{
		name:           "observability with profiling disabled",
		wantEnabled:    false,
		wantStatusCode: http.StatusNotFound,
		config: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{
				"profiling.enable": "false",
			},
		},
	}, {
		name:           "observability config with profiling enabled",
		wantEnabled:    true,
		wantStatusCode: http.StatusOK,
		config: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{
				"profiling.enable": "true",
			},
		},
	}, {
		name:           "observability config with unparseable value",
		wantEnabled:    false,
		wantStatusCode: http.StatusNotFound,
		config: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{
				"profiling.enable": "get me some profiles",
			},
		},
	}}

	for _, tt := range observabilityConfigTests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(zap.NewNop().Sugar())
			server := NewServer(handler)
			defer server.Shutdown(context.Background())
			go server.ListenAndServe()

			baseURL := "http://" + server.Addr

			err := waitForServerToBecomeReady(baseURL)
			if err != nil {
				t.Errorf("Timed out waiting for the server to become ready: %v", err)
			}

			handler.UpdateFromConfigMap(tt.config)

			resp, err := sendRequest(baseURL + "/debug/pprof/")
			if err != nil {
				t.Fatal("Error sending request:", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode: %v, want: %v", resp.StatusCode, tt.wantStatusCode)
			}

			if handler.enabled != tt.wantEnabled {
				t.Fatalf("Test: %q; want %v, but got %v", tt.name, tt.wantEnabled, handler.enabled)
			}
		})
	}
}

func waitForServerToBecomeReady(url string) error {
	return wait.PollImmediate(5*time.Millisecond, 2*time.Second, func() (bool, error) {
		if _, err := sendRequest(url); isTCPConnectRefuse(err) {
			return false, nil
		}
		return true, nil
	})
}

func sendRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func isTCPConnectRefuse(err error) bool {
	if err != nil && strings.Contains(err.Error(), "connect: connection refused") {
		return true
	}
	return false
}
