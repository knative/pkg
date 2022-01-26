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

package network

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestHTTPRoundTripper(t *testing.T) {
	wants := sets.NewString()
	frt := func(key string) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			wants.Insert(key)
			return nil, nil
		})
	}

	rt := newAutoTransport(frt("v1"), frt("v2"))

	examples := []struct {
		label      string
		protoMajor int
		want       string
	}{{
		label:      "use default transport for HTTP1",
		protoMajor: 1,
		want:       "v1",
	}, {
		label:      "use h2c transport for HTTP2",
		protoMajor: 2,
		want:       "v2",
	}, {
		label:      "use default transport for all others",
		protoMajor: 99,
		want:       "v1",
	}}

	for _, e := range examples {
		t.Run(e.label, func(t *testing.T) {
			wants.Delete(e.want)
			r := &http.Request{ProtoMajor: e.protoMajor}
			rt.RoundTrip(r)

			if !wants.Has(e.want) {
				t.Error("Wrong transport selected for request.")
			}
		})
	}
}

func TestDialWithBackoff(t *testing.T) {
	var tlsConf *tls.Config
	t.Parallel()
	t.Run("ConnectionRefused", testDialWithBackoffConnectionRefused(tlsConf))
	t.Run("Timeout", testDialWithBackoffTimeout(tlsConf))
	t.Run("Success", testDialWithBackoffSuccess(tlsConf))
}

func TestDialTLSWithBackoff(t *testing.T) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         "example.com",
		MinVersion:         tls.VersionTLS12,
	}
	t.Parallel()
	t.Run("ConnectionRefused", testDialWithBackoffConnectionRefused(tlsConf))
	t.Run("Timeout", testDialWithBackoffTimeout(tlsConf))
	t.Run("Success", testDialWithBackoffSuccess(tlsConf))
}

func testDialWithBackoffConnectionRefused(tlsConf *tls.Config) func(t *testing.T) {
	ctx := context.TODO()
	return func(t *testing.T) {
		port := findUnusedPortOrFail(t)
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		c, err := dialer(ctx, tlsConf)(addr)
		closeOrFail(t, c)
		if !errors.Is(err, syscall.ECONNREFUSED) {
			t.Errorf("Unexpected error: %+v", err)
		}
	}
}

func testDialWithBackoffTimeout(tlsConf *tls.Config) func(t *testing.T) {
	return func(t *testing.T) {
		// Timeout. Use non-routable IP. See: https://stackoverflow.com/a/31581323/844449
		c, err := dialer(context.TODO(), tlsConf)("10.0.0.0:81")
		if err == nil {
			closeOrFail(t, c)
			t.Error("Unexpected success dialing")
		}
		if !errors.Is(err, ErrTimeoutDialing) {
			t.Errorf("Unexpected error: %+v", err)
		}
	}
}

func testDialWithBackoffSuccess(tlsConf *tls.Config) func(t *testing.T) {
	//goland:noinspection HttpUrlsUsage
	const (
		prefixHTTP  = "http://"
		prefixHTTPS = "https://"
	)
	ctx := context.TODO()
	return func(t *testing.T) {
		var s *httptest.Server
		servFn := httptest.NewServer
		if tlsConf != nil {
			servFn = httptest.NewTLSServer
		}
		s = servFn(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer s.Close()
		prefix := prefixHTTP
		if tlsConf != nil {
			servFn = httptest.NewTLSServer
			prefix = prefixHTTPS
			rootCAs := x509.NewCertPool()
			rootCAs.AddCert(s.Certificate())
			tlsConf.RootCAs = rootCAs
		}
		addr := strings.TrimPrefix(s.URL, prefix)

		c, err := dialer(ctx, tlsConf)(addr)
		if err != nil {
			t.Fatal("Dial error =", err)
		}
		closeOrFail(t, c)
	}
}

func dialer(ctx context.Context, tlsConf *tls.Config) func(addr string) (net.Conn, error) {
	// Make the test short.
	bo := backOffTemplate
	bo.Steps = 1

	dialFn := func(addr string) (net.Conn, error) {
		bo.Duration = time.Millisecond
		return NewBackoffDialer(bo)(ctx, "tcp4", addr)
	}
	if tlsConf != nil {
		dialFn = func(addr string) (net.Conn, error) {
			bo.Duration = 10 * time.Millisecond
			return NewTLSBackoffDialer(bo)(ctx, "tcp4", addr, tlsConf)
		}
	}
	return dialFn
}

func closeOrFail(tb testing.TB, con io.Closer) {
	tb.Helper()
	if con == nil {
		return
	}
	if err := con.Close(); err != nil {
		tb.Fatal(err)
	}
}

func findUnusedPortOrFail(tb testing.TB) int {
	tb.Helper()
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		tb.Fatal(err)
	}
	defer closeOrFail(tb, l)
	return l.Addr().(*net.TCPAddr).Port
}
