/*
Copyright 2018 The Knative Authors

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

package webhook

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/system"
	certresources "knative.dev/pkg/webhook/certificates/resources"

	. "knative.dev/pkg/logging/testing"
)

func TestEnsureLabelSelectorExpressions(t *testing.T) {
	fooExpression := metav1.LabelSelectorRequirement{
		Key:      "foo.bar/baz",
		Operator: metav1.LabelSelectorOpDoesNotExist,
	}
	knativeExpression := metav1.LabelSelectorRequirement{
		Key:      "knative.dev/foo",
		Operator: metav1.LabelSelectorOpDoesNotExist,
	}

	tests := []struct {
		name    string
		current *metav1.LabelSelector
		want    *metav1.LabelSelector
		expect  *metav1.LabelSelector
	}{{
		name: "all nil",
	}, {
		name:    "current nil",
		current: nil,
		want: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
		expect: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
	}, {
		name:    "current empty",
		current: &metav1.LabelSelector{},
		want: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
		expect: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
	}, {
		name: "want nil",
		current: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
		want: nil,
		expect: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
	}, {
		name: "want empty",
		current: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
		want: &metav1.LabelSelector{},
		expect: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
	}, {
		name: "add new",
		current: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression},
		},
		want: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{knativeExpression},
		},
		expect: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				knativeExpression, fooExpression},
		},
	}, {
		name: "remove obsolete",
		current: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{fooExpression, {
				Key:      "knative.dev/bar",
				Operator: metav1.LabelSelectorOpDoesNotExist,
			}},
		},
		want: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{knativeExpression},
		},
		expect: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				knativeExpression, fooExpression},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EnsureLabelSelectorExpressions(tc.current, tc.want)
			if !cmp.Equal(got, tc.expect) {
				t.Errorf("LabelSelector mismatch: diff(-want,+got):\n%s", cmp.Diff(tc.expect, got))
			}
		})
	}
}
func waitForNonTLSServerAvailable(t *testing.T, serverURL string, timeout time.Duration) error {
	t.Helper()
	var interval = 100 * time.Millisecond

	conditionFunc := func() (done bool, err error) {
		var conn net.Conn
		conn, _ = net.DialTimeout("tcp", serverURL, timeout)
		if conn != nil {
			conn.Close()
			return true, nil
		}
		return false, nil
	}

	return wait.PollImmediate(interval, timeout, conditionFunc)
}

func waitForServerAvailable(t *testing.T, serverURL string, timeout time.Duration) error {
	t.Helper()

	var (
		// if this is too low you'll see TLS handshake EOF warnings
		interval = 500 * time.Millisecond
		tlsConf  = &tls.Config{InsecureSkipVerify: true}
		dialer   = &net.Dialer{
			Timeout:   interval, // Initial duration.
			KeepAlive: 5 * time.Second,
			DualStack: true,
		}
	)

	conditionFunc := func() (done bool, err error) {
		conn, _ := tls.DialWithDialer(dialer, "tcp", serverURL, tlsConf)
		if conn != nil {
			conn.Close()
			return true, nil
		}
		return false, nil
	}

	return wait.PollImmediate(interval, timeout, conditionFunc)
}

func newTestPort() (port int, err error) {
	server, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}

	defer server.Close()

	_, portString, err := net.SplitHostPort(server.Addr().String())
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(portString)
}

func createNamespace(t *testing.T, kubeClient kubernetes.Interface, name string) error {
	t.Helper()
	testns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), testns, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func createTestConfigMap(t *testing.T, kubeClient kubernetes.Interface) error {
	t.Helper()
	configMaps := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "extension-apiserver-authentication",
		},
		Data: map[string]string{"requestheader-client-ca-file": "test-client-file"},
	}
	_, err := kubeClient.CoreV1().ConfigMaps(metav1.NamespaceSystem).Create(context.Background(), configMaps, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func createSecureTLSClient(t *testing.T, kubeClient kubernetes.Interface, acOpts *Options) (*http.Client, error) {
	t.Helper()
	ctx := TestContextWithLogger(t)

	secret, err := kubeClient.CoreV1().Secrets(system.Namespace()).Get(ctx, acOpts.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	serverKey := secret.Data[certresources.ServerKey]
	serverCert := secret.Data[certresources.ServerCert]
	caCert := secret.Data[certresources.CACert]

	// Build cert pool with CA Cert
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)

	// Build key pair
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}

	tlsClientConfig := &tls.Config{
		// Add knative namespace as CN
		ServerName:   "webhook." + system.Namespace(),
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsClientConfig,
		},
	}, nil
}

func createNonTLSClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{},
	}
}
