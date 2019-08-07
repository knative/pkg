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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

func TestUpdateProfilingFromConfigMap(t *testing.T) {
	observabilityConfigTests := []struct {
		name        string
		wantEnabled bool
		config      *corev1.ConfigMap
	}{{
		name:        "observability with profiling disabled",
		wantEnabled: false,
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
		name:        "observability config with profiling enabled",
		wantEnabled: true,
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
		name:        "observability config with unparseable value",
		wantEnabled: false,
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
			logger, _ := logging.NewLogger("", "debug")
			handler := NewHandler(false)

			updateFunc := UpdateProfilingFromConfigMap(handler, logger)
			updateFunc(tt.config)

			if handler.Enabled != tt.wantEnabled {
				t.Fatalf("Test: %q; want %v, but got %v", tt.name, tt.wantEnabled, handler.Enabled)
			}
		})
	}
}
