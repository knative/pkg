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
	"sync"

	"google.golang.org/api/option"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (

	// kubeclient is the in-cluster Kubernetes kubeclient.
	kubeclient     *kubernetes.Clientset
	initClientOnce sync.Once
)

const (
	// ConfigMapNameEnv is the name of the environment variable that contains the name of
	// the ConfigMap for stackdriver configuration.
	ConfigMapNameEnv = "CONFIG_STACKDRIVER_NAME"
	// secretNamespaceDefault is the namespace in which secrets for authenticating with Stackdriver
	// should be stored.
	secretNamespaceDefault = "default"
	// secretDataFieldKey is key of the Kubernetes Secret Data field that contains the GCP
	// service account key to use.
	secretDataFieldKey = "key.json"

	// ProjectIDKey is the map key for Config.ProjectID.
	ProjectIDKey = "project-id"
	// GCPLocationKey is the map key for Config.GCPLocation.
	GCPLocationKey = "gcp-location"
	// ClusterNameKey is the map key for Config.ClusterName.
	ClusterNameKey = "cluster-name"
	// GCPSecretNameKey is the map key for GCPSecretName.
	GCPSecretNameKey = "gcp-secret-name"
	// GCPSecretNamespaceKey is the map key for GCPSecretNamespace.
	GCPSecretNamespaceKey = "gcp-secret-namespace"
)

// Config encapsulates the metadata required to configure a Stackdriver client.
type Config struct {
	// ProjectID is the stackdriver project ID to which data is uploaded.
	// This is not necessarily the GCP project ID where the Kubernetes cluster is hosted.
	// Required when the Kubernetes cluster is not hosted on GCE.
	ProjectID string
	// GCPLocation is the GCP region or zone to which data is uploaded.
	// This is not necessarily the GCP location where the Kubernetes cluster is hosted.
	// Required when the Kubernetes cluster is not hosted on GCE.
	GCPLocation string
	// ClusterName is the cluster name with which the data will be associated in Stackdriver.
	// Required when the Kubernetes cluster is not hosted on GCE.
	ClusterName string
	// GCPSecretName is the optional GCP service account key which will be used to
	// authenticate with Stackdriver. If not provided, Google Application Default Credentials
	// will be used (https://cloud.google.com/docs/authentication/production).
	GCPSecretName string
	// GCPSecretNamespace is the Kubernetes namespace where GCPSecretName is located.
	// The Kubernetes ServiceAccount used by the pod that is exporting data to
	// Stackdriver should have access to Secrets in this namespace.
	GCPSecretNamespace string
}

// NewStackdriverConfigFromConfigMap returns a Config for the given configmap
func NewStackdriverConfigFromConfigMap(config *corev1.ConfigMap) (*Config, error) {
	if config == nil {
		return &Config{}, nil
	}
	return NewStackdriverConfigFromMap(config.Data)
}

// NewStackdriverConfigFromMap returns a Config for the given map
func NewStackdriverConfigFromMap(config map[string]string) (*Config, error) {
	sc := &Config{}

	if pi, ok := config[ProjectIDKey]; ok {
		sc.ProjectID = pi
	}

	if gl, ok := config[GCPLocationKey]; ok {
		sc.GCPLocation = gl
	}

	if cn, ok := config[ClusterNameKey]; ok {
		sc.ClusterName = cn
	}

	if gsn, ok := config[GCPSecretNameKey]; ok {
		sc.GCPSecretName = gsn
	}

	if gsns, ok := config[GCPSecretNamespaceKey]; ok {
		sc.GCPSecretNamespace = gsns
	}

	return sc, nil
}

// GetStackdriverSecret returns the Kubernetes Secret specified in the given config.
func GetStackdriverSecret(config *Config) (*v1.Secret, error) {
	ensureKubeclient()
	ns := secretNamespaceDefault
	if config.GCPSecretNamespace != "" {
		ns = config.GCPSecretNamespace
	}

	return kubeclient.CoreV1().Secrets(ns).Get(config.GCPSecretName, metav1.GetOptions{})
}

// ConvertSecretToExporterOption converts a Kubernetes Secret to an OpenCensus Stackdriver Exporter Option.
func ConvertSecretToExporterOption(secret *v1.Secret) option.ClientOption {
	return option.WithCredentialsJSON(secret.Data[secretDataFieldKey])
}

func ensureKubeclient() {
	initClientOnce.Do(func() {
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}

		cs, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		kubeclient = cs

	})
}
