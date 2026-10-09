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
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewH2CTransportRoundTrip verifies that the transport returned by
// NewH2CTransport negotiates cleartext HTTP/2 (h2c) end-to-end against a
// server created with NewServer.
func TestNewH2CTransportRoundTrip(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("server saw ProtoMajor = %d, want 2", r.ProtoMajor)
		}
		io.WriteString(w, "ok")
	})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("Unable to create listener:", err)
	}
	defer l.Close()

	s := NewServer(l.Addr().String(), handler)
	go s.Serve(l) //nolint:errcheck // Serve always returns a non-nil error on close.
	defer s.Close()

	//goland:noinspection HttpUrlsUsage
	req, err := http.NewRequest(http.MethodGet, "http://"+l.Addr().String(), nil)
	if err != nil {
		t.Fatal("NewRequest() =", err)
	}

	rt := NewH2CTransport()
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal("RoundTrip() =", err)
	}
	defer resp.Body.Close()

	if got := resp.Proto; got != "HTTP/2.0" {
		t.Errorf("resp.Proto = %q, want HTTP/2.0", got)
	}
	if body, _ := io.ReadAll(resp.Body); string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

// TestNewH2TransportRoundTrip verifies that the transport returned by
// newH2Transport negotiates HTTP/2 over TLS end-to-end.
func TestNewH2TransportRoundTrip(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("server saw ProtoMajor = %d, want 2", r.ProtoMajor)
		}
		io.WriteString(w, "ok")
	})

	s := httptest.NewUnstartedServer(handler)
	s.EnableHTTP2 = true
	s.StartTLS()
	defer s.Close()

	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(s.Certificate())

	tlsContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := &tls.Dialer{Config: &tls.Config{
			RootCAs:    rootCAs,
			NextProtos: []string{"h2"},
			MinVersion: tls.VersionTLS12,
		}}
		return d.DialContext(ctx, network, addr)
	}

	addr := strings.TrimPrefix(s.URL, "https://")
	req, err := http.NewRequest(http.MethodGet, "https://"+addr, nil)
	if err != nil {
		t.Fatal("NewRequest() =", err)
	}

	rt := newH2Transport(false /*disableCompression*/, tlsContext)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal("RoundTrip() =", err)
	}
	defer resp.Body.Close()

	if got := resp.Proto; got != "HTTP/2.0" {
		t.Errorf("resp.Proto = %q, want HTTP/2.0", got)
	}
	if body, _ := io.ReadAll(resp.Body); string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}
