/*
Copyright 2020 The Knative Authors

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

package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"knative.dev/pkg/network"
)

func TestDrainMechanics(t *testing.T) {
	var (
		w     http.ResponseWriter
		req   = &http.Request{}
		probe = &http.Request{
			Header: http.Header{
				"User-Agent": []string{network.KubeProbeUAPrefix},
			},
		}
	)

	inner := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	drainer := &Drainer{
		Inner:       inner,
		QuietPeriod: 100 * time.Millisecond,
	}

	// Works before Drain is called.
	drainer.ServeHTTP(w, req)
	drainer.ServeHTTP(w, req)
	drainer.ServeHTTP(w, req)

	// Check for 200 OK.
	resp := httptest.NewRecorder()
	drainer.ServeHTTP(resp, probe)
	if got, want := resp.Code, http.StatusOK; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}

	// Start to drain, and cancel the context when it returns.
	done := make(chan struct{})
	go func() {
		defer close(done)
		drainer.Drain()
	}()

	select {
	case <-time.After(40 * time.Millisecond):
		// Drain is blocking.
	case <-done:
		t.Error("Drain terminated prematurely.")
	}
	// Now send a request to reset things.
	drainer.ServeHTTP(w, req)

	// Check for 503 as a probe response when shutting down.
	resp = httptest.NewRecorder()
	drainer.ServeHTTP(resp, probe)
	if got, want := resp.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}

	for i := 0; i < 3; i++ {
		select {
		case <-time.After(40 * time.Millisecond):
			// Drain is blocking.
		case <-done:
			t.Error("Drain terminated prematurely.")
		}
		// For the last one we don't want to reset the drain timer.
		if i < 2 {
			drainer.ServeHTTP(w, req)
		}
	}
	// Probing does not reset the clock.
	// Check for 503 on a probe when shutting down.
	resp = httptest.NewRecorder()
	drainer.ServeHTTP(resp, probe)
	if got, want := resp.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}

	// Big finish, test that multiple invocations of Drain all block.
	done1 := make(chan struct{})
	go func() {
		defer close(done1)
		drainer.Drain()
	}()
	done2 := make(chan struct{})
	go func() {
		defer close(done2)
		drainer.Drain()
	}()
	done3 := make(chan struct{})
	go func() {
		defer close(done3)
		drainer.Drain()
	}()

	select {
	case <-time.After(80 * time.Millisecond):
		t.Error("Timed out waiting for Drain to return.")
	case <-done:
	case <-done1:
	case <-done2:
	case <-done3:
		// Once the first context is cancelled, check that all of them are cancelled.
	}

	// Check that a 4th and final one after things complete finishes instantly.
	done4 := make(chan struct{})
	go func() {
		defer close(done4)
		drainer.Drain()
	}()

	// Give the test a short window to launch and execute the go routine.
	time.Sleep(5 * time.Millisecond)

	for idx, dch := range []chan struct{}{done, done1, done2, done3, done4} {
		select {
		case <-dch:
			// Should be done.
		default:
			t.Errorf("Drain[%d] did not complete.", idx)
		}
	}
}
