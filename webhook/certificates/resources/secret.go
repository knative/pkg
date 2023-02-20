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

package resources

import (
	"context"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ServerKey is the name of the key associated with the secret's private key.
	ServerKey = "server-key.pem"
	// ServerKeyEnv is the env var name for the webhook secret's key eg. `tls.key`.
	ServerKeyEnv = "KNATIVE_WEBHOOK_SERVER_KEY"
	// ServerCert is the name of the key associated with the secret's public key.
	ServerCert = "server-cert.pem"
	// ServerCertEnv is the env var name for the webhook secret's ca data key eg. `tls.crt`.
	ServerCertEnv = "KNATIVE_WEBHOOK_SERVER_CERT"
	// CACert is the name of the key associated with the certificate of the CA for
	// the keypair.
	CACert = "ca-cert.pem"

	oneWeek = 7 * 24 * time.Hour
)

// MakeSecret synthesizes a Kubernetes Secret object with the keys specified by
// ServerKey, ServerCert, and CACert populated with a fresh certificate.
// This is mutable to make deterministic testing possible.
var MakeSecret = MakeSecretInternal

// MakeSecretInternal is only public so MakeSecret can be restored in testing.  Use MakeSecret.
func MakeSecretInternal(ctx context.Context, name, namespace, serviceName string) (*corev1.Secret, error) {
	serverKey, serverCert, caCert, err := CreateCerts(ctx, serviceName, namespace, time.Now().Add(oneWeek))
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			ServerKey:  serverKey,
			ServerCert: serverCert,
			CACert:     caCert,
		},
	}, nil
}

// GetSecretDataKeyNamesOrDefault gets the names of the keys in the webhook secret's data.
func GetSecretDataKeyNamesOrDefault() (serverKey string, serverCert string) {
	serverKey = ServerKey
	serverCert = ServerCert
	if val, ok := os.LookupEnv(ServerKeyEnv); ok {
		serverKey = val
	}
	if val, ok := os.LookupEnv(ServerCertEnv); ok {
		serverCert = val
	}
	return serverKey, serverCert
}
