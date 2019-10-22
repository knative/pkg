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

package config

import (
	"google.golang.org/api/option"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// ConfigMapNameEnv is the name of the environment variable that contains the name of
	// the ConfigMap for stackdriver configuration.
	ConfigMapNameEnv = "CONFIG_STACKDRIVER_NAME"
	// secretNamespace is the namespace in which secrets for authenticating with Stackdriver
	// should be stored.
	secretNamespace = "knative-serving"
	// secretDataFieldKey is key of the Kubernetes Secret Data field that contains the GCP
	// service account key to use.
	secretDataFieldKey = "key.json"

	// ProjectIDKey is the name of the ConfigMap field for ProjectID.
	ProjectIDKey = "project-id"
	// GcpLocationKey is name of the ConfigMap field for GcpLocation.
	GcpLocationKey = "gcp-location"
	// ClusterNameKey is the name of the ConfigMap field for ClusterName.
	ClusterNameKey = "cluster-name"
	// GcpSecretNameKey is the name of the ConfigMap field for GcpSecretName.
	GcpSecretNameKey = "gcp-secret-name"
)

// Config encapsulates the metadata required to configure a Stackdriver client.
type Config struct {
	// ProjectID is the stackdriver project ID to which data is uploaded.
	// This is not necessarily the GCP project ID where the Kubernetes cluster is hosted.
	// Required when the Kubernetes cluster is not hosted on GCE.
	ProjectID string
	// GcpLocation is the GCP region or zone to which data is uploaded.
	// This is not necessarily the GCP project ID where the Kubernetes cluster is hosted.
	// Required when the Kubernetes cluster is not hosted on GCE.
	GcpLocation string
	// ClusterName is the cluster name with which the data will be associated in Stackdriver.
	// Required when the Kubernetes cluster is not hosted on GCE.
	ClusterName string
	// GcpSecretName is the optional GCP service account key which will be used to
	// authenticate with Stackdriver. If not provided, Google Application Default Credentials
	// will be used (https://cloud.google.com/docs/authentication/production).
	GcpSecretName string
}

// NewStackdriverConfigFromConfigMap returns a Config for the given configmap
func NewStackdriverConfigFromConfigMap(config *corev1.ConfigMap) (*Config, error) {
	return NewStackdriverConfigFromMap(config.Data)
}

// NewStackdriverConfigFromMap returns a Config for the given map
func NewStackdriverConfigFromMap(config map[string]string) (*Config, error) {
	sc := &Config{}

	if pi, ok := config[ProjectIDKey]; ok {
		sc.ProjectID = pi
	}

	if gl, ok := config[GcpLocationKey]; ok {
		sc.GcpLocation = gl
	}

	if cn, ok := config[ClusterNameKey]; ok {
		sc.ClusterName = cn
	}

	if gsn, ok := config[GcpSecretNameKey]; ok {
		sc.GcpSecretName = gsn
	}

	return sc, nil
}

var (
	// kubeclient is the in-cluster Kubernetes kubeclient.
	kubeclient *kubernetes.Clientset
)

// GetStackdriverSecret returns the Kubernetes Secret specified in the given config.
func GetStackdriverSecret(config *Config) (*v1.Secret, error) {
	ensureKubeclient()
	return kubeclient.CoreV1().Secrets(secretNamespace).Get(config.GcpSecretName, metav1.GetOptions{})
}

// ConvertSecretToExporterOption converts a Kubernetes Secret to an OpenCensus Stackdriver Exporter Option.
func ConvertSecretToExporterOption(secret *v1.Secret) option.ClientOption {
	return option.WithCredentialsJSON(secret.Data[secretDataFieldKey])
}

func ensureKubeclient() {
	if kubeclient != nil {
		return
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	kubeclient = cs
}
