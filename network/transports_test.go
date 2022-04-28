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

package network_test

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
	"knative.dev/pkg/network"
)

func TestHTTPRoundTripper(t *testing.T) {
	wants := sets.NewString()
	frt := func(key string) http.RoundTripper {
		return network.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			wants.Insert(key)
			return nil, nil
		})
	}

	rt := network.NewRoundTripperAutoTransport(frt("v1"), frt("v2"))

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
			if _, err := rt.RoundTrip(r); err != nil {
				t.Error(err)
			}

			if !wants.Has(e.want) {
				t.Error("Wrong transport selected for request.")
			}
		})
	}
}

func TestDialWithBackoff(t *testing.T) {
	t.Parallel()
	p := findUnusedPortsOrFail(t, 6)
	t.Logf("Unused ports: %v", p.numbers)
	t.Run("PlainText", func(t *testing.T) {
		t.Run("Success", dialSuccess(plainConf(p)))
		t.Run("Timeout", dialTimeout(plainConf(p)))
		t.Run("ConnectionRefused", dialConnectionRefused(plainConf(p)))
	})
	t.Run("TLS", func(t *testing.T) {
		t.Run("Success", dialSuccess(tlsConf(p)))
		t.Run("Timeout", dialTimeout(tlsConf(p)))
		t.Run("ConnectionRefused", dialConnectionRefused(tlsConf(p)))
	})
}

func dialConnectionRefused(c dialWithBackoffTestConfig) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s port: %v", t.Name(), c.port)
		ctx := context.TODO()
		addr := fmt.Sprintf("127.0.0.1:%d", c.port)
		dial := newDialer(ctx, c.tlsConf)
		conn, err := dial(addr)
		closeOrFail(t, conn)
		if !errors.Is(err, syscall.ECONNREFUSED) {
			t.Fatalf("Unexpected error: %+v", err)
		}
	}
}

func dialTimeout(c dialWithBackoffTestConfig) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("%s port: %v", t.Name(), c.port)
		ctx := context.TODO()
		closer, addr, err := listenOne(c)
		if err != nil {
			t.Fatal("Unable to create listener:", err)
		}
		defer closer()
		c1, err := net.Dial("tcp4", addr.String())
		if err != nil {
			t.Fatalf("Unable to connect to server on %s: %s", addr, err)
		}
		defer closeOrFail(t, c1)

		// Since the backlog is full, the next request must time out.
		dial := newDialer(ctx, c.tlsConf)
		conn, err := dial(addr.String())
		if err == nil {
			closeOrFail(t, conn)
			t.Fatal("Unexpected success dialing")
		}
		if !errors.Is(err, network.ErrTimeoutDialing) {
			t.Fatalf("Unexpected error: %+v", err)
		}
	}
}

func dialSuccess(c dialWithBackoffTestConfig) func(t *testing.T) {
	//goland:noinspection HttpUrlsUsage
	const (
		prefixHTTP  = "http://"
		prefixHTTPS = "https://"
	)
	ctx := context.TODO()
	return func(t *testing.T) {
		t.Logf("%s port: %v", t.Name(), c.port)
		var s *httptest.Server
		servFn := newServerFn(c, t)
		s = servFn(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer s.Close()
		prefix := prefixHTTP
		if c.tlsConf != nil {
			prefix = prefixHTTPS
			rootCAs := x509.NewCertPool()
			rootCAs.AddCert(s.Certificate())
			c.tlsConf.RootCAs = rootCAs
		}
		addr := strings.TrimPrefix(s.URL, prefix)

		dial := newDialer(ctx, c.tlsConf)
		conn, err := dial(addr)
		if err != nil {
			t.Fatal("Dial error =", err)
		}
		closeOrFail(t, conn)
	}
}

func newServerFn(c dialWithBackoffTestConfig, t testingT) func(handler http.Handler) *httptest.Server {
	return func(handler http.Handler) *httptest.Server {
		server := httptest.NewUnstartedServer(handler)
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", c.port))
		if err != nil {
			t.Fatal(err)
		}
		closeOrFail(t, server.Listener)
		server.Listener = listener
		if c.tlsConf != nil {
			server.StartTLS()
		} else {
			server.Start()
		}
		return server
	}
}

type dialWithBackoffTestConfig struct {
	tlsConf *tls.Config
	port    int
}

func plainConf(p *ports) dialWithBackoffTestConfig {
	return dialWithBackoffTestConfig{
		tlsConf: nil,
		port:    p.pop(),
	}
}

func tlsConf(p *ports) dialWithBackoffTestConfig {
	return dialWithBackoffTestConfig{
		tlsConf: &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         "example.com",
			MinVersion:         tls.VersionTLS12,
		},
		port: p.pop(),
	}
}

func newDialer(ctx context.Context, tlsConf *tls.Config) func(addr string) (net.Conn, error) {
	// Make the test short.
	bo := wait.Backoff{
		Duration: time.Millisecond,
		Factor:   1.4,
		Jitter:   0.1, // At most 10% jitter.
		Steps:    2,
	}

	dialFn := func(addr string) (net.Conn, error) {
		return network.NewBackoffDialer(bo)(ctx, "tcp4", addr)
	}
	if tlsConf != nil {
		dialFn = func(addr string) (net.Conn, error) {
			bo.Duration = 10 * time.Millisecond
			bo.Steps = 6
			return network.NewTLSBackoffDialer(bo)(ctx, "tcp4", addr, tlsConf)
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

func findUnusedPortsOrFail(t testingT, num int) *ports {
	p := make([]int, num)
	for i := 0; i < num; i++ {
		func(i int) {
			l, err := net.Listen("tcp", "localhost:0")
			if err != nil {
				t.Fatal(err)
			}
			defer closeOrFail(t, l)
			p[i] = l.Addr().(*net.TCPAddr).Port
		}(i)
	}
	return &ports{numbers: p}
}

type ports struct {
	numbers []int
}

func (p *ports) pop() int {
	var port int
	port, p.numbers = p.numbers[0], p.numbers[1:]
	return port
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
func listenOne(c dialWithBackoffTestConfig) (func(), *net.TCPAddr, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, nil, newTestErr("Couldn't get socket", err)
	}
	sa := &syscall.SockaddrInet4{
		Port: c.port,
		Addr: [4]byte{127, 0, 0, 1},
	}
	if err = syscall.Bind(fd, sa); err != nil {
		return nil, nil, newTestErr("Unable to bind", err)
	}
	if err = syscall.Listen(fd, 0); err != nil {
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
