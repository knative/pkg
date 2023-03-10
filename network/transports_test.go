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
	"k8s.io/apimachinery/pkg/util/wait"
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

func TestDialWithBackoffConnectionRefused(t *testing.T) {
	testDialWithBackoffConnectionRefused(nil, t)
}

func TestDialWithBackoffTimeout(t *testing.T) {
	testDialWithBackoffTimeout(nil, t)
}

func TestDialWithBackoffSuccess(t *testing.T) {
	testDialWithBackoffSuccess(nil, t)
}

func TestDialTLSWithBackoffConnectionRefused(t *testing.T) {
	testDialWithBackoffConnectionRefused(exampleTLSConf(), t)
}

func TestDialTLSWithBackoffTimeout(t *testing.T) {
	testDialWithBackoffTimeout(exampleTLSConf(), t)
}

func TestDialTLSWithBackoffSuccess(t *testing.T) {
	testDialWithBackoffSuccess(exampleTLSConf(), t)
}

func testDialWithBackoffConnectionRefused(tlsConf *tls.Config, t testingT) {
	ctx := context.TODO()
	port := findUnusedPortOrFail(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	dialer := newDialer(ctx, tlsConf)
	c, err := dialer(addr)
	closeOrFail(t, c)
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatalf("Unexpected error: %+v", err)
	}
}

func testDialWithBackoffTimeout(tlsConf *tls.Config, t testingT) {
	ctx := context.TODO()
	closer, addr, err := listenOne()
	if err != nil {
		t.Fatal("Unable to create listener:", err)
	}
	defer closer()

	for {
		// This seems really strange... we're listening with a backlog of one
		// connection, and we keep creating connections and holding onto them
		// until we get a connection timeout.
		//
		// It turns out that darwin (MacOS) and Linux implement the Listen
		// backlog argument slightly differently, and MacOS needs one more
		// connection than Linux to saturate the backlog. Rather than sniffing
		// the OS, we simply ensure that the backlog is saturated.
		c1, err := net.DialTimeout("tcp4", addr.String(), 10*time.Millisecond)
		if err != nil {
			var neterr net.Error
			if errors.As(err, &neterr) && neterr.Timeout() {
				// Waiting for a timeout
				break
			}
			t.Fatalf("Unable to connect to server on %s: %s", addr, err)
		}
		defer closeOrFail(t, c1)
	}

	// Since the backlog is full, the next request must time out.
	dialer := newDialer(ctx, tlsConf)
	c, err := dialer(addr.String())
	if err == nil {
		closeOrFail(t, c)
		t.Fatal("Unexpected success dialing")
	}
	if !errors.Is(err, ErrTimeoutDialing) {
		t.Fatalf("Unexpected error: %+v", err)
	}
}

func testDialWithBackoffSuccess(tlsConf *tls.Config, t testingT) {
	//goland:noinspection HttpUrlsUsage
	const (
		prefixHTTP  = "http://"
		prefixHTTPS = "https://"
	)
	ctx := context.TODO()
	var s *httptest.Server
	servFn := httptest.NewServer
	if tlsConf != nil {
		servFn = httptest.NewTLSServer
	}
	s = servFn(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer s.Close()
	prefix := prefixHTTP
	if tlsConf != nil {
		prefix = prefixHTTPS
		rootCAs := x509.NewCertPool()
		rootCAs.AddCert(s.Certificate())
		tlsConf.RootCAs = rootCAs
	}
	addr := strings.TrimPrefix(s.URL, prefix)

	dialer := newDialer(ctx, tlsConf)
	c, err := dialer(addr)
	if err != nil {
		t.Fatal("Dial error =", err)
	}
	closeOrFail(t, c)
}

func exampleTLSConf() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         "example.com",
		MinVersion:         tls.VersionTLS12,
	}
}

func newDialer(ctx context.Context, tlsConf *tls.Config) func(addr string) (net.Conn, error) {
	// Make the test short.
	bo := wait.Backoff{
		Duration: time.Millisecond,
		Factor:   1.4,
		Jitter:   0.1, // At most 10% jitter.
		Steps:    1,
	}

	dialFn := func(addr string) (net.Conn, error) {
		return NewBackoffDialer(bo)(ctx, "tcp4", addr)
	}
	if tlsConf != nil {
		dialFn = func(addr string) (net.Conn, error) {
			bo.Duration = 50 * time.Millisecond
			bo.Steps = 3
			return NewTLSBackoffDialer(bo)(ctx, "tcp4", addr, tlsConf)
		}
	}
	return dialFn
}

func closeOrFail(t testingT, con io.Closer) {
	if con == nil {
		return
	}
	if err := con.Close(); err != nil {
		t.Fatal(err)
	}
}

func findUnusedPortOrFail(t testingT) int {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer closeOrFail(t, l)
	return l.Addr().(*net.TCPAddr).Port
}

var errTest = errors.New("testing")

func newTestErr(msg string, err error) error {
	return fmt.Errorf("%w: %s: %v", errTest, msg, err)
}

// listenOne creates a socket with backlog of one, and use that socket, so
// any other connection will guarantee to timeout.
//
// Golang doesn't allow us to set the backlog argument on syscall.Listen from
// net.ListenTCP, so we need to get directly into syscall land.
func listenOne() (func(), *net.TCPAddr, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, newTestErr("Couldn't get socket", err)
	}
	sa := &syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{127, 0, 0, 1},
	}
	if err = syscall.Bind(fd, sa); err != nil {
		return nil, nil, newTestErr("Unable to bind", err)
	}
	if err = syscall.Listen(fd, 1); err != nil {
		return nil, nil, newTestErr("Unable to Listen", err)
	}
	closer := func() { _ = syscall.Close(fd) }
	listenaddr, err := syscall.Getsockname(fd)
	if err != nil {
		closer()
		return nil, nil, newTestErr("Could not get sockname", err)
	}
	sa = listenaddr.(*syscall.SockaddrInet4)
	addr := &net.TCPAddr{
		IP:   sa.Addr[:],
		Port: sa.Port,
	}
	return closer, addr, nil
}

type testingT interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}
