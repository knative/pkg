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

package webhook

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	// Make system.Namespace() work in tests.
	_ "knative.dev/pkg/system/testing"

	. "knative.dev/pkg/reconciler/testing"
)

func newDefaultOptions() Options {
	return Options{
		ServiceName: "webhook",
		Port:        8443,
		SecretName:  "webhook-certs",
	}
}

const (
	testResourceName = "test-resource"
	user1            = "brutto@knative.dev"
)

func newNonRunningTestWebhook(t *testing.T, options Options, acs ...interface{}) (
	ctx context.Context, ac *Webhook, cancel context.CancelFunc,
) {
	t.Helper()

	// override the grace period so it drains quickly
	options.GracePeriod = 100 * time.Millisecond

	// Create fake clients
	ctx, ctxCancel, informers := SetupFakeContextWithCancel(t)
	ctx = WithOptions(ctx, options)

	stopCb, err := RunAndSyncInformers(ctx, informers...)
	if err != nil {
		t.Fatal("StartInformers() =", err)
	}
	cancel = func() {
		ctxCancel()
		stopCb()
	}

	ac, err = New(ctx, acs)
	if err != nil {
		t.Fatal("Failed to create new admission controller:", err)
	}
	return ctx, ac, cancel
}

func TestRegistrationStopChanFire(t *testing.T) {
	test := testSetup(t, withNoTLS())
	defer test.cancel()

	stopCh := make(chan struct{})

	var g errgroup.Group
	g.Go(func() error {
		return test.webhook.Run(stopCh)
	})
	close(stopCh)

	if err := g.Wait(); err != nil {
		t.Fatal("Error during run: ", err)
	}
	conn, err := net.Dial("tcp", test.addr)
	if err == nil {
		conn.Close()
		t.Error("Unexpected success to dial to ", test.addr)
	}
}

func newAdmissionControllerWebhook(t *testing.T, options Options, acs ...interface{}) (*Webhook, error) {
	ctx, cancel, _ := SetupFakeContextWithCancel(t)
	defer cancel()
	ctx = WithOptions(ctx, options)
	return New(ctx, acs)
}

func TestTLSMinVersionWebhookOption(t *testing.T) {
	opts := newDefaultOptions()
	t.Run("when TLSMinVersion is not configured, and the default is used", func(t *testing.T) {
		_, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}
	})
	t.Run("when the TLS minimum version configured is supported", func(t *testing.T) {
		opts.TLSMinVersion = tls.VersionTLS12
		_, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}
	})
	t.Run("when the TLS minimum version configured is not supported", func(t *testing.T) {
		opts.TLSMinVersion = tls.VersionTLS11
		_, err := newAdmissionControllerWebhook(t, opts)
		if err == nil {
			t.Fatal("Admission Controller Webhook creation expected to fail due to unsupported TLS version")
		}
	})
}

func TestTLSMaxVersionWebhookOption(t *testing.T) {
	opts := newDefaultOptions()
	t.Run("when TLSMaxVersion is not configured, default is used", func(t *testing.T) {
		wh, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}

		if wh.tlsConfig != nil && wh.tlsConfig.MaxVersion != 0 {
			t.Errorf("Expected MaxVersion to be 0 (default), got %d", wh.tlsConfig.MaxVersion)
		}
	})
	t.Run("when TLSMaxVersion is configured to TLS 1.3", func(t *testing.T) {
		opts.TLSMaxVersion = tls.VersionTLS13
		wh, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}
		if wh.tlsConfig == nil {
			t.Fatal("Expected tlsConfig to be set")
		}
		if wh.tlsConfig.MaxVersion != tls.VersionTLS13 {
			t.Errorf("Expected MaxVersion to be TLS 1.3, got %d", wh.tlsConfig.MaxVersion)
		}
	})
	t.Run("when both TLSMinVersion and TLSMaxVersion are TLS 1.3 (Modern profile)", func(t *testing.T) {
		opts.TLSMinVersion = tls.VersionTLS13
		opts.TLSMaxVersion = tls.VersionTLS13
		wh, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}
		if wh.tlsConfig == nil {
			t.Fatal("Expected tlsConfig to be set")
		}
		if wh.tlsConfig.MinVersion != tls.VersionTLS13 {
			t.Errorf("Expected MinVersion to be TLS 1.3, got %d", wh.tlsConfig.MinVersion)
		}
		if wh.tlsConfig.MaxVersion != tls.VersionTLS13 {
			t.Errorf("Expected MaxVersion to be TLS 1.3, got %d", wh.tlsConfig.MaxVersion)
		}
	})
}

func TestTLSCipherSuitesWebhookOption(t *testing.T) {
	opts := newDefaultOptions()
	t.Run("when TLSCipherSuites is not configured", func(t *testing.T) {
		wh, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}

		if wh.tlsConfig != nil && wh.tlsConfig.CipherSuites != nil {
			t.Errorf("Expected CipherSuites to be nil (default), got %v", wh.tlsConfig.CipherSuites)
		}
	})
	t.Run("when TLSCipherSuites is configured with specific ciphers", func(t *testing.T) {
		expectedCiphers := []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		}
		opts.TLSCipherSuites = expectedCiphers
		wh, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}
		if wh.tlsConfig == nil {
			t.Fatal("Expected tlsConfig to be set")
		}
		if len(wh.tlsConfig.CipherSuites) != len(expectedCiphers) {
			t.Errorf("Expected %d cipher suites, got %d", len(expectedCiphers), len(wh.tlsConfig.CipherSuites))
		}
		for i, cipher := range expectedCiphers {
			if wh.tlsConfig.CipherSuites[i] != cipher {
				t.Errorf("Expected cipher suite at index %d to be %d, got %d", i, cipher, wh.tlsConfig.CipherSuites[i])
			}
		}
	})
}

func TestTLSCurvePreferencesWebhookOption(t *testing.T) {
	opts := newDefaultOptions()
	t.Run("when TLSCurvePreferences is not configured", func(t *testing.T) {
		wh, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}

		if wh.tlsConfig != nil && wh.tlsConfig.CurvePreferences != nil {
			t.Errorf("Expected CurvePreferences to be nil (default), got %v", wh.tlsConfig.CurvePreferences)
		}
	})
	t.Run("when TLSCurvePreferences is configured with specific curves", func(t *testing.T) {
		expectedCurves := []tls.CurveID{
			tls.CurveP256,
			tls.CurveP384,
			tls.X25519,
		}
		opts.TLSCurvePreferences = expectedCurves
		wh, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}
		if wh.tlsConfig == nil {
			t.Fatal("Expected tlsConfig to be set")
		}
		if len(wh.tlsConfig.CurvePreferences) != len(expectedCurves) {
			t.Errorf("Expected %d curve preferences, got %d", len(expectedCurves), len(wh.tlsConfig.CurvePreferences))
		}
		for i, curve := range expectedCurves {
			if wh.tlsConfig.CurvePreferences[i] != curve {
				t.Errorf("Expected curve at index %d to be %d, got %d", i, curve, wh.tlsConfig.CurvePreferences[i])
			}
		}
	})
}

func TestTLSConfigCombinedOptions(t *testing.T) {
	opts := newDefaultOptions()
	t.Run("when all TLS options are configured together", func(t *testing.T) {
		opts.TLSMinVersion = tls.VersionTLS12
		opts.TLSMaxVersion = tls.VersionTLS13
		opts.TLSCipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		}
		opts.TLSCurvePreferences = []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		}

		wh, err := newAdmissionControllerWebhook(t, opts)
		if err != nil {
			t.Fatal("Unexpected error", err)
		}

		if wh.tlsConfig == nil {
			t.Fatal("Expected tlsConfig to be set")
		}

		if wh.tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("Expected MinVersion to be TLS 1.2, got %d", wh.tlsConfig.MinVersion)
		}
		if wh.tlsConfig.MaxVersion != tls.VersionTLS13 {
			t.Errorf("Expected MaxVersion to be TLS 1.3, got %d", wh.tlsConfig.MaxVersion)
		}
		if len(wh.tlsConfig.CipherSuites) != 2 {
			t.Errorf("Expected 2 cipher suites, got %d", len(wh.tlsConfig.CipherSuites))
		}
		if len(wh.tlsConfig.CurvePreferences) != 2 {
			t.Errorf("Expected 2 curve preferences, got %d", len(wh.tlsConfig.CurvePreferences))
		}
	})
}
