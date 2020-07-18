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
	"context"
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

	// Check for 200 OK
	resp := httptest.NewRecorder()
	drainer.ServeHTTP(resp, probe)
	if got, want := resp.Code, http.StatusOK; got != want {
		t.Errorf("probe status = %d, wanted %d", got, want)
	}

	// Start to drain, and cancel the context when it returns.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		drainer.Drain()
		cancel()
	}()

	select {
	case <-time.After(40 * time.Millisecond):
		// Drain is blocking.
	case <-ctx.Done():
		t.Error("Drain terminated prematurely.")
	}
	// Now send a request to reset things.
	drainer.ServeHTTP(w, req)

	// Check for 400 shutting down
	resp = httptest.NewRecorder()
	drainer.ServeHTTP(resp, probe)
	if got, want := resp.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("probe status = %d, wanted %d", got, want)
	}

	select {
	case <-time.After(40 * time.Millisecond):
		// Drain is blocking.
	case <-ctx.Done():
		t.Error("Drain terminated prematurely.")
	}
	// Now send a request to reset things.
	drainer.ServeHTTP(w, req)

	select {
	case <-time.After(40 * time.Millisecond):
		// Drain is blocking.
	case <-ctx.Done():
		t.Error("Drain terminated prematurely.")
	}
	// Now send a request to reset things.
	drainer.ServeHTTP(w, req)

	select {
	case <-time.After(40 * time.Millisecond):
		// Drain is blocking.
	case <-ctx.Done():
		t.Error("Drain terminated prematurely.")
	}
	// Probing does not reset the clock.
	// Check for 500 shutting down
	resp = httptest.NewRecorder()
	drainer.ServeHTTP(resp, probe)
	if got, want := resp.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("probe status = %d, wanted %d", got, want)
	}

	// Big finish, test that multiple invocations of Drain all block.
	ctx1, cancel1 := context.WithCancel(context.Background())
	go func() {
		drainer.Drain()
		cancel1()
	}()
	ctx2, cancel2 := context.WithCancel(context.Background())
	go func() {
		drainer.Drain()
		cancel2()
	}()
	ctx3, cancel3 := context.WithCancel(context.Background())
	go func() {
		drainer.Drain()
		cancel3()
	}()

	select {
	case <-time.After(70 * time.Millisecond):
		t.Error("Timed out waiting for Drain to return.")

	case <-ctx.Done():
	case <-ctx1.Done():
	case <-ctx2.Done():
	case <-ctx3.Done():
		// Once the first context is cancelled, check that all of them are cancelled.
	}

	// Check that a 4th and final one after things complete finishes instantly.
	ctx4, cancel4 := context.WithCancel(context.Background())
	go func() {
		drainer.Drain()
		cancel4()
	}()

	// Give the rest a short window to complete.
	time.Sleep(time.Millisecond)

	for idx, ictx := range []context.Context{ctx, ctx1, ctx2, ctx3, ctx4} {
		select {
		case <-ictx.Done():
			// Should be done.
		default:
			t.Errorf("Drain[%d] did not complete.", idx)
		}
	}
}
