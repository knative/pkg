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

type mockTimer struct {
	now        time.Time // our current time.
	deadline   time.Time // when we're supposed to fire
	c          chan time.Time
	resetCalls int
	stopped    bool
}

func (mt *mockTimer) advance(d time.Duration) {
	mt.now = mt.now.Add(d)
	if !mt.now.Before(mt.deadline) {
		mt.stopped = true
		mt.c <- mt.now
	}
}

func (mt *mockTimer) Reset(d time.Duration) bool {
	mt.resetCalls++
	if mt.stopped {
		mt.now = time.Now()
		mt.deadline = mt.now.Add(d)
		mt.stopped = false
	}
	return !mt.stopped
}

func (mt *mockTimer) Stop() bool {
	if mt.stopped {
		return false
	}
	mt.stopped = true
	return true
}

func (mt *mockTimer) tickChan() <-chan time.Time {
	return mt.c
}

func TestDrainMechanics(t *testing.T) {
	var (
		w     http.ResponseWriter
		req   = &http.Request{}
		probe = &http.Request{
			Header: http.Header{
				"User-Agent": []string{network.KubeProbeUAPrefix},
			},
		}
		cnt   = 0
		inner = http.HandlerFunc(func(http.ResponseWriter, *http.Request) { cnt++ })
	)

	const (
		timeout = 100 * time.Millisecond
		epsilon = time.Nanosecond
	)

	// We need init channel to signal the main thread that the drain
	// has been initialized in the background thread.
	init := make(chan struct{})
	nt := newTimer
	t.Cleanup(func() {
		newTimer = nt
	})
	// The mock timer will only fire when we advance it past timeout.
	mt := &mockTimer{
		c: make(chan time.Time),
	}
	newTimer = func(d time.Duration) timer {
		// When we close the init channel, we know that first drain has been called, and the test can progress.
		defer close(init)
		mt.now = time.Now()
		mt.deadline = mt.now.Add(d)
		return mt
	}
	drainer := &Drainer{
		Inner:       inner,
		QuietPeriod: timeout,
	}

	// Works before Drain is called.
	drainer.ServeHTTP(w, req)
	drainer.ServeHTTP(w, req)
	drainer.ServeHTTP(w, req)
	if cnt != 3 {
		t.Error("Inner handler was not properly invoked")
	}

	// Check for 200 OK.
	resp := httptest.NewRecorder()
	drainer.ServeHTTP(resp, probe)
	if got, want := resp.Code, http.StatusOK; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}

	// Start to drain, and close the channel when it returns.
	done := make(chan struct{})
	go func() {
		defer close(done)
		drainer.Drain()
	}()

	select {
	case <-done:
		t.Error("Drain terminated prematurely.")
	case <-init:
		// OK.
	}
	mt.advance(timeout - epsilon)

	// Now send a request to reset things.
	rc := mt.resetCalls
	drainer.ServeHTTP(w, req)
	if mt.resetCalls != rc+1 {
		t.Errorf("ResetCalls = %d, want: %d", mt.resetCalls, rc+1)
	}

	// Check for 503 as a probe response when shutting down.
	resp = httptest.NewRecorder()
	drainer.ServeHTTP(resp, probe)
	if got, want := resp.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}
	// Verify no reset was called.
	if got, want := mt.resetCalls, rc+1; got != want {
		t.Errorf("ResetCalls = %d, want: %d", got, want)
	}
	rc++

	for i := 0; i < 3; i++ {
		mt.advance(timeout - epsilon)
		select {
		case <-done:
			t.Error("Drain terminated prematurely.")
		default:
			// OK
		}
		// For the last one we don't want to reset the drain timer.
		if i < 2 {
			drainer.ServeHTTP(w, req)

			// Two more drains should have been called.
			if got, want := mt.resetCalls, rc+1; got != want {
				t.Errorf("ResetCalls = %d, want: %d", got, want)
			}
			rc++
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
	case <-done:
	case <-done1:
	case <-done2:
	case <-done3:
	default:
		// Expected.
	}

	// Finally we made it there!
	mt.advance(epsilon)
	select {
	case <-done:
	case <-done1:
	case <-done2:
	case <-done3:
	case <-time.After(time.Second): // We can't use default here, since it will race the tick in the drainer.
		t.Error("Drains should have happened!")
	}

	// Check that a 4th and final one after things complete finishes instantly.
	done4 := make(chan struct{})
	go func() {
		defer close(done4)
		drainer.Drain()
	}()

	// We need to ensure all the go routines complete, so give them ample time.
	for idx, dch := range []chan struct{}{done, done1, done2, done3, done4} {
		select {
		case <-dch:
			// Should be done.
		case <-time.After(time.Second):
			t.Errorf("Drain[%d] did not complete.", idx)
		}
	}
}

func TestDrainerKProbe(t *testing.T) {
	var (
		w          http.ResponseWriter
		req        = &http.Request{}
		kprobehash = "hash"
		kprobe     = &http.Request{
			Header: http.Header{
				"K-Network-Probe": []string{"probe"},
				"K-Network-Hash":  []string{kprobehash},
			},
		}
		kprobeerr = &http.Request{
			Header: http.Header{
				"K-Network-Probe": []string{"probe"},
			},
		}
		cnt   = 0
		inner = http.HandlerFunc(func(http.ResponseWriter, *http.Request) { cnt++ })
	)
	drainer := &Drainer{
		Inner: inner,
	}

	// Works before Drain is called.
	drainer.ServeHTTP(w, req)
	drainer.ServeHTTP(w, req)
	drainer.ServeHTTP(w, req)
	if cnt != 3 {
		t.Error("Inner handler was not properly invoked")
	}

	resp := httptest.NewRecorder()
	drainer.ServeHTTP(resp, kprobe)
	if got, want := resp.Code, http.StatusOK; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}

	if got, want := resp.Header().Get("K-Network-Hash"), kprobehash; got != want {
		t.Errorf("KProbe hash = %s, wanted %s", got, want)
	}

	resp = httptest.NewRecorder()
	drainer.ServeHTTP(resp, kprobeerr)
	if got, want := resp.Code, http.StatusBadRequest; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}

	if cnt != 3 {
		t.Error("Inner handler was not properly invoked")
	}
}

func TestDefaultQuietPeriod(t *testing.T) {
	nt := newTimer
	t.Cleanup(func() {
		newTimer = nt
	})
	mt := &mockTimer{
		c: make(chan time.Time),
	}
	init := make(chan struct{})
	newTimer = func(d time.Duration) timer {
		defer close(init)
		mt.now = time.Now()
		mt.deadline = mt.now.Add(d)
		return mt
	}
	drainer := &Drainer{
		Inner: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}
	go drainer.Drain()
	select {
	case <-init:
		if got, want := mt.deadline.Sub(mt.now), network.DefaultDrainTimeout; got != want {
			t.Errorf("DefaultDrainTimeout = %v, want: %v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("Failed to call drain in 1s")
	}
	mt.advance(network.DefaultDrainTimeout)
}
