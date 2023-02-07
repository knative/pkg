/*
Copyright 2023 The Knative Authors

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
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"knative.dev/pkg/logging"
)

// ServeHealthProbes sets up liveness and readiness probes.
// If user sets no probes explicitly via the context then defaults are added.
func ServeHealthProbes(ctx context.Context) {
	logger := logging.FromContext(ctx)
	port := os.Getenv("KNATIVE_HEALTH_PROBES_PORT")
	if port == "" {
		port = "8080"
	}

	server := http.Server{ReadHeaderTimeout: time.Minute, Handler: muxWithHandles(ctx), Addr: ":" + port}

	go func() {
		go func() {
			<-ctx.Done()
			_ = server.Shutdown(ctx)
		}()

		// start the web server on port and accept requests
		logger.Infof("Probes server listening on port %s", port)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal(err)
		}
	}()
}

func muxWithHandles(ctx context.Context) *http.ServeMux {
	var handler HealthCheckHandler
	if h := getOverrideHealthCheckHandler(ctx); h != nil {
		handler = *h
	} else {
		defaultHandle := newDefaultProbesHandle(ctx)
		handler = HealthCheckHandler{
			ReadinessProbeRequestHandle: defaultHandle,
			LivenessProbeRequestHandle:  defaultHandle,
		}
	}
	mux := http.NewServeMux()
	if UserReadinessProbe(ctx) {
		mux.HandleFunc("/readiness", handler.ReadinessProbeRequestHandle)
	}
	if UserLivenessProbe(ctx) {
		mux.HandleFunc("/health", handler.LivenessProbeRequestHandle)
	}
	// Set both probes if user does not want to explicitly set only one.
	if !UserReadinessProbe(ctx) && !UserLivenessProbe(ctx) {
		mux.HandleFunc("/readiness", handler.ReadinessProbeRequestHandle)
		mux.HandleFunc("/health", handler.LivenessProbeRequestHandle)
	}
	return mux
}

// HealthCheckHandler allows to set up a handler for probes. User can override the default handler
// with a custom one.
type HealthCheckHandler struct {
	ReadinessProbeRequestHandle http.HandlerFunc
	LivenessProbeRequestHandle  http.HandlerFunc
}

func newDefaultProbesHandle(sigCtx context.Context) http.HandlerFunc {
	logger := logging.FromContext(sigCtx)
	once := sync.Once{}
	return func(w http.ResponseWriter, r *http.Request) {
		f := func() error {
			select {
			// When we get SIGTERM (sigCtx done), let readiness probes start failing.
			case <-sigCtx.Done():
				once.Do(func() {
					logger.Info("Signal context canceled")
				})
				return errors.New("received SIGTERM from kubelet")
			default:
				return nil
			}
		}

		if IsKubeletProbe(r) {
			if err := f(); err != nil {
				logger.Errorf("Healthcheck failed: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			return
		}
		http.Error(w, "Unexpected request", http.StatusBadRequest)
	}
}

type addUserReadinessProbeKey struct{}

// WithUserReadinessProbe signals to ServeHealthProbes that it should set explicitly a readiness probe.
func WithUserReadinessProbe(ctx context.Context) context.Context {
	return context.WithValue(ctx, addUserReadinessProbeKey{}, struct{}{})
}

// UserReadinessProbe checks if user has explicitly requested to set a readiness probe in the related context.
func UserReadinessProbe(ctx context.Context) bool {
	return ctx.Value(addUserReadinessProbeKey{}) != nil
}

type addUserLivenessProbeKey struct{}

// WithUserLivenessProbe signals to ServeHealthProbes that it should set explicitly a liveness probe.
func WithUserLivenessProbe(ctx context.Context) context.Context {
	return context.WithValue(ctx, addUserLivenessProbeKey{}, struct{}{})
}

// UserLivenessProbe checks if user has explicitly requested to set a liveness probe in the related context.
func UserLivenessProbe(ctx context.Context) bool {
	return ctx.Value(addUserLivenessProbeKey{}) != nil
}

type overrideHealthCheckHandlerKey struct{}

// WithOverrideHealthCheckHandler signals to ServeHealthProbes that it should override the default handler with the one passed.
func WithOverrideHealthCheckHandler(ctx context.Context, handler *HealthCheckHandler) context.Context {
	return context.WithValue(ctx, overrideHealthCheckHandlerKey{}, handler)
}

func getOverrideHealthCheckHandler(ctx context.Context) *HealthCheckHandler {
	if ctx.Value(overrideHealthCheckHandlerKey{}) != nil {
		return ctx.Value(overrideHealthCheckHandlerKey{}).(*HealthCheckHandler)
	}
	return nil
}
