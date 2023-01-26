/*
Copyright 2022 The Knative Authors

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
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// ServeHealthProbes sets up liveness and readiness probes.
func ServeHealthProbes(ctx context.Context) {
	port := os.Getenv("KNATIVE_HEALTH_PROBES_PORT")
	if port == "" {
		port = "8080"
	}
	handler := healthHandler{HealthCheck: newHealthCheck(ctx)}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.handle)
	mux.HandleFunc("/health", handler.handle)
	mux.HandleFunc("/readiness", handler.handle)

	server := http.Server{ReadHeaderTimeout: time.Minute, Handler: mux, Addr: ":" + port}

	go func() {
		go func() {
			<-ctx.Done()
			_ = server.Shutdown(ctx)
		}()

		// start the web server on port and accept requests
		log.Printf("Probes server listening on port %s", port)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
}

func newHealthCheck(sigCtx context.Context) func() error {
	once := sync.Once{}
	return func() error {
		select {
		// When we get SIGTERM (sigCtx done), let readiness probes start failing.
		case <-sigCtx.Done():
			once.Do(func() {
				log.Println("Signal context canceled")
			})
			return errors.New("received SIGTERM from kubelet")
		default:
			return nil
		}
	}
}

// healthHandler handles responding to kubelet probes with a provided health check.
type healthHandler struct {
	HealthCheck func() error
}

func (h *healthHandler) handle(w http.ResponseWriter, r *http.Request) {
	if IsKubeletProbe(r) {
		if err := h.HealthCheck(); err != nil {
			log.Println("Healthcheck failed: ", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		return
	}
	http.Error(w, "Unexpected request", http.StatusBadRequest)
}
