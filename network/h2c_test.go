/*
Copyright 2026 The Knative Authors

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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewH2CTransportRoundTrip verifies NewH2CTransport negotiates cleartext
// HTTP/2 (h2c) end-to-end against a NewServer.
func TestNewH2CTransportRoundTrip(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Unable to create listener:", err)
	}
	defer l.Close()

	s := NewServer(l.Addr().String(), okHandler(t, 2))
	go s.Serve(l) //nolint:errcheck // Serve always returns a non-nil error on close.
	defer s.Close()

	//goland:noinspection HttpUrlsUsage
	assertRoundTripOK(t, NewH2CTransport(), "http://"+l.Addr().String(), "HTTP/2.0")
}

// TestNewServerH2OverTLS verifies NewServer advertises h2 via ALPN and
// negotiates HTTP/2 over TLS end-to-end.
func TestNewServerH2OverTLS(t *testing.T) {
	addr, pool := startTLSServer(t, okHandler(t, 2))
	assertRoundTripOK(t, newH2Transport(false, h2DialerTLSContext(pool)), "https://"+addr, "HTTP/2.0")
}

// h2 must negotiate even when the caller's tls.Config omits NextProtos
// (serving's activator/queue-proxy wiring).
func TestH2BackoffDialerNegotiatesH2WithoutExplicitNextProtos(t *testing.T) {
	addr, pool := startTLSServer(t, okHandler(t, 2))
	assertRoundTripOK(t, newH2Transport(false, backoffTLSContext(pool)), "https://"+addr, "HTTP/2.0")
}

// h2 ALPN must not leak into the HTTP/1.1 branch sharing the same dialer.
func TestProxyAutoTLSTransportHTTP1StillWorks(t *testing.T) {
	addr, pool := startTLSServer(t, okHandler(t, 1))
	assertRoundTripOK(t, NewProxyAutoTLSTransport(100, 100, backoffTLSContext(pool)), "https://"+addr, "HTTP/1.1")
}

// TestNewH2TransportRoundTrip verifies newH2Transport negotiates HTTP/2 over
// TLS against a standard-library h2 server.
func TestNewH2TransportRoundTrip(t *testing.T) {
	s := httptest.NewUnstartedServer(okHandler(t, 2))
	s.EnableHTTP2 = true
	s.StartTLS()
	defer s.Close()

	pool := x509.NewCertPool()
	pool.AddCert(s.Certificate())

	assertRoundTripOK(t, newH2Transport(false, h2DialerTLSContext(pool)), s.URL, "HTTP/2.0")
}

// okHandler responds "ok" and asserts the server saw the expected HTTP version.
func okHandler(t *testing.T, wantProtoMajor int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != wantProtoMajor {
			t.Errorf("server saw ProtoMajor = %d, want %d", r.ProtoMajor, wantProtoMajor)
		}
		io.WriteString(w, "ok")
	})
}

// startTLSServer starts a TLS NewServer serving h and returns its address and a
// pool trusting its certificate.
func startTLSServer(t *testing.T, h http.Handler) (addr string, pool *x509.CertPool) {
	t.Helper()

	cert, pool := selfSignedCert(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Unable to create listener:", err)
	}

	s := NewServer(l.Addr().String(), h)
	s.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	go s.ServeTLS(l, "", "") //nolint:errcheck // Serve always returns a non-nil error on close.
	t.Cleanup(func() {
		s.Close()
		l.Close()
	})

	return l.Addr().String(), pool
}

// backoffTLSContext dials via DialTLSWithBackOff without setting NextProtos,
// mirroring serving.
func backoffTLSContext(pool *x509.CertPool) DialTLSContextFunc {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return DialTLSWithBackOff(ctx, network, addr, &tls.Config{
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
		})
	}
}

// h2DialerTLSContext dials with NextProtos ["h2"] set explicitly.
func h2DialerTLSContext(pool *x509.CertPool) DialTLSContextFunc {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := &tls.Dialer{Config: &tls.Config{
			RootCAs:    pool,
			NextProtos: []string{"h2"},
			MinVersion: tls.VersionTLS12,
		}}
		return d.DialContext(ctx, network, addr)
	}
}

// assertRoundTripOK GETs url on rt and asserts the protocol and "ok" body.
func assertRoundTripOK(t *testing.T, rt http.RoundTripper, url, wantProto string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal("NewRequest() =", err)
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal("RoundTrip() =", err)
	}
	defer resp.Body.Close()

	if got := resp.Proto; got != wantProto {
		t.Errorf("resp.Proto = %q, want %q", got, wantProto)
	}
	if body, _ := io.ReadAll(resp.Body); string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

// selfSignedCert returns a self-signed cert for 127.0.0.1 and a pool trusting it.
func selfSignedCert(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal("GenerateKey() =", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal("CreateCertificate() =", err)
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal("ParseCertificate() =", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(leaf)

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv, Leaf: leaf}, pool
}
