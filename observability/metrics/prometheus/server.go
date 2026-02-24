/*
Copyright 2025 The Knative Authors

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

package prometheus

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultPrometheusPort            = "9090"
	defaultPrometheusReportingPeriod = 5
	maxPrometheusPort                = 65535
	minPrometheusPort                = 1024
	defaultPrometheusHost            = "" // IPv4 and IPv6
	prometheusPortEnvName            = "METRICS_PROMETHEUS_PORT"
	prometheusHostEnvName            = "METRICS_PROMETHEUS_HOST"
	prometheusTLSCertEnvName         = "METRICS_TLS_CERT"
	prometheusTLSKeyEnvName          = "METRICS_TLS_KEY"
)

type ServerOption func(*options)

type Server struct {
	http     *http.Server
	certFile string
	keyFile  string
}

func NewServer(opts ...ServerOption) (*Server, error) {
	o := options{
		host: defaultPrometheusHost,
		port: defaultPrometheusPort,
	}

	for _, opt := range opts {
		opt(&o)
	}

	envOverride(&o.host, prometheusHostEnvName)
	envOverride(&o.port, prometheusPortEnvName)
	envOverride(&o.certFile, prometheusTLSCertEnvName)
	envOverride(&o.keyFile, prometheusTLSKeyEnvName)

	if err := validate(&o); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())

	addr := net.JoinHostPort(o.host, o.port)

	return &Server{
		http: &http.Server{
			Addr:              addr,
			Handler:           mux,
			TLSConfig:         o.tlsConfig,
			ReadHeaderTimeout: 5 * time.Second,
		},
		certFile: o.certFile,
		keyFile:  o.keyFile,
	}, nil
}

func (s *Server) ListenAndServe() error {
	if s.http.TLSConfig != nil || (s.certFile != "" && s.keyFile != "") {
		return s.http.ListenAndServeTLS(s.certFile, s.keyFile)
	}
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

type options struct {
	host      string
	port      string
	tlsConfig *tls.Config
	certFile  string
	keyFile   string
}

func WithHost(host string) ServerOption {
	return func(o *options) {
		o.host = host
	}
}

func WithPort(port string) ServerOption {
	return func(o *options) {
		o.port = port
	}
}

// WithTLSConfig configures the server to use the provided TLS configuration.
// This allows programmatic control over TLS settings like MinVersion, CipherSuites, etc.
func WithTLSConfig(cfg *tls.Config) ServerOption {
	return func(o *options) {
		o.tlsConfig = cfg
	}
}

// WithTLSCertFiles configures the server to use TLS with the provided certificate and key files.
func WithTLSCertFiles(certFile, keyFile string) ServerOption {
	return func(o *options) {
		o.certFile = certFile
		o.keyFile = keyFile
	}
}

func validate(o *options) error {
	port, err := strconv.ParseUint(o.port, 10, 16)
	if err != nil {
		return fmt.Errorf("prometheus port %q could not be parsed as a port number: %w",
			o.port, err)
	}

	if port < minPrometheusPort || port > maxPrometheusPort {
		return fmt.Errorf("prometheus port %d, should be between %d and %d",
			port, minPrometheusPort, maxPrometheusPort)
	}

	return nil
}

func envOverride(target *string, envName string) {
	val := os.Getenv(envName)
	if val != "" {
		*target = val
	}
}

type tlsConfigKey struct{}

// ContextWithTLSConfig adds a TLS configuration to the context.
// This allows programmatic configuration of TLS settings like MinVersion, CipherSuites, ClientAuth, etc.
// when using frameworks like sharedmain where direct NewServer() options aren't accessible.
func ContextWithTLSConfig(ctx context.Context, cfg *tls.Config) context.Context {
	return context.WithValue(ctx, tlsConfigKey{}, cfg)
}

// TLSConfigFromContext retrieves the TLS configuration from the context.
// Returns nil if no TLS configuration was set.
func TLSConfigFromContext(ctx context.Context) *tls.Config {
	if cfg, ok := ctx.Value(tlsConfigKey{}).(*tls.Config); ok {
		return cfg
	}
	return nil
}
