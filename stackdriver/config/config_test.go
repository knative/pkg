package config

import (
	corev1 "k8s.io/api/core/v1"

	"testing"
)

func TestNewStackdriverConfigFromConfigMap(t *testing.T) {
	tests := []struct {
		name           string
		configMap      *corev1.ConfigMap
		expectedConfig Config
	}{
		{
			name: "fullSdConfig",
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"project-id":      "project",
					"gcp-location":    "us-west1",
					"cluster-name":    "cluster",
					"gcp-secret-name": "secret",
				},
			},
			expectedConfig: Config{
				ProjectID:     "project",
				GcpLocation:   "us-west1",
				ClusterName:   "cluster",
				GcpSecretName: "secret",
			},
		},
		{
			name:           "emptySdConfig",
			configMap:      &corev1.ConfigMap{},
			expectedConfig: Config{},
		},
		{
			name: "partialSdConfig",
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"project-id":   "project",
					"gcp-location": "us-west1",
					"cluster-name": "cluster",
				},
			},
			expectedConfig: Config{
				ProjectID:   "project",
				GcpLocation: "us-west1",
				ClusterName: "cluster",
			},
		},
		{
			name: "invalidGcpLocation",
			configMap: &corev1.ConfigMap{
				Data: map[string]string{
					"gcp-location": "narnia",
				},
			},
			expectedConfig: Config{
				GcpLocation: "narnia",
			},
		},
		{
			name:           "nil",
			configMap:      nil,
			expectedConfig: Config{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewStackdriverConfigFromConfigMap(test.configMap)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if test.expectedConfig != *c {
				t.Errorf("Incorrect stackdriver config. Expected: [%v], Got: [%v]", test.expectedConfig, *c)
			}
		})
	}
}

// This test ensures that ensureKubeClient completes. Errors are expected, but ok.
func TestEnsureKubeClientNoDeadlock(t *testing.T) {
	for i := 0; i < 10; i++ {
		go testKubeclient(t)
	}
}

func testKubeclient(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Expected ensureKubeclient to panic due to not being in a Kubernetes cluster. Did the function run?.")
		}
	}()

	ensureKubeclient()
}
