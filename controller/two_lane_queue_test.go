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

package controller

import (
	"strconv"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

type chanRateLimiter struct {
	t *testing.T
	// Called when this ratelimiter is consulted for when to process a value.
	whenCalled chan interface{}
	retryCount map[interface{}]int
}

func (r chanRateLimiter) When(item interface{}) time.Duration {
	r.whenCalled <- item
	return 0 * time.Second
}
func (r chanRateLimiter) Forget(item interface{}) {
	r.t.Fatalf("Forgetting item %+v, we should not be forgetting any items.", item)
}
func (r chanRateLimiter) NumRequeues(item interface{}) int {
	return r.retryCount[item]
}

var _ workqueue.RateLimiter = &chanRateLimiter{}

func TestRateLimit(t *testing.T) {
	whenCalled := make(chan interface{}, 2)
	rl := chanRateLimiter{
		t,
		whenCalled,
		make(map[interface{}]int),
	}
	q := newTwoLaneWorkQueue("live-in-the-limited-lane", rl)
	q.SlowLane().Add("1")
	q.Add("2")

	k, done := q.Get()
	q.Done(k)
	q.AddRateLimited(k)
	rlK := <-whenCalled
	if got, want := k.(string), rlK.(string); got != want {
		t.Errorf(`Got = %q, want: "%q"`, got, want)
	}
	if done {
		t.Error("The queue is unexpectedly shutdown")
	}

	k2, done := q.Get()
	q.Done(k2)
	q.AddRateLimited(k2)
	rlK2 := <-whenCalled
	if got, want := k2.(string), rlK2.(string); got != want {
		t.Errorf(`Got = %q, want: "%q"`, got, want)
	}
	if s1, s2 := k.(string), k2.(string); (s1 != "1" || s2 != "2") && (s1 != "2" || s2 != "1") {
		t.Errorf("Expected to see 1 and 2, instead saw %q and %q", s1, s2)
	}
	if done {
		t.Error("The queue is unexpectedly shutdown")
	}

	// Queue has async moving parts so if we check at the wrong moment, this might still be 0 or 1.
	if wait.PollImmediate(10*time.Millisecond, 250*time.Millisecond, func() (bool, error) {
		return q.Len() == 2, nil
	}) != nil {
		t.Error("Queue length was never 2")
	}
}

func TestSlowQueue(t *testing.T) {
	q := newTwoLaneWorkQueue("live-in-the-fast-lane", workqueue.DefaultControllerRateLimiter())
	q.SlowLane().Add("1")
	// Queue has async moving parts so if we check at the wrong moment, this might still be 0.
	if wait.PollImmediate(10*time.Millisecond, 250*time.Millisecond, func() (bool, error) {
		return q.Len() == 1, nil
	}) != nil {
		t.Error("Queue length was never 1")
	}

	k, done := q.Get()
	if got, want := k.(string), "1"; got != want {
		t.Errorf(`Got = %q, want: "1"`, got)
	}
	if done {
		t.Error("The queue is unexpectedly shutdown")
	}
	q.Done(k)
	q.ShutDown()
	if !q.SlowLane().ShuttingDown() {
		t.Error("ShutDown did not propagate to the slow queue")
	}
	if _, done := q.Get(); !done {
		t.Error("Get did not return positive shutdown signal")
	}
}

func TestDoubleKey(t *testing.T) {
	// Verifies that we don't get double concurrent processing of the same key.
	q := newTwoLaneWorkQueue("live-in-the-fast-lane", workqueue.DefaultControllerRateLimiter())
	q.Add("1")
	t.Cleanup(q.ShutDown)

	k, done := q.Get()
	if got, want := k.(string), "1"; got != want {
		t.Errorf(`Got = %q, want: "1"`, got)
	}
	if done {
		t.Error("The queue is unexpectedly shutdown")
	}

	// This should not be read from the queue until we actually call `Done`.
	q.SlowLane().Add("1")
	sentinel := make(chan struct{})
	go func() {
		defer close(sentinel)
		k, done := q.Get()
		if got, want := k.(string), "1"; got != want {
			t.Errorf(`2nd time got = %q, want: "1"`, got)
		}
		if done {
			t.Error("The queue is unexpectedly shutdown")
		}
		q.Done(k)
	}()
	select {
	case <-sentinel:
		t.Error("The sentinel should not have fired")
	case <-time.After(600 * time.Millisecond):
		// Expected.
	}
	// This should permit the re-reading of the same key.
	q.Done(k)
	select {
	case <-sentinel:
		// Expected.
	case <-time.After(200 * time.Millisecond):
		t.Error("The item was not processed as expected")
	}
}

func TestOrder(t *testing.T) {
	// Verifies that we read from the fast queue first.
	q := newTwoLaneWorkQueue("live-in-the-fast-lane", workqueue.DefaultControllerRateLimiter())
	stop := make(chan struct{})
	t.Cleanup(func() {
		close(stop)
		q.ShutDown()
		// Drain the rest.
		for q.Len() > 0 {
			q.Get()
		}
	})

	go func() {
		for i := 1; ; i++ {
			q.Add(strconv.Itoa(i))
			// Get fewer of those, to ensure the first priority select wins.
			if i%2 == 0 {
				q.SlowLane().Add("slow" + strconv.Itoa(i))
			}
			select {
			case <-stop:
				return
			default:
			}
		}
	}()
	done := time.After(300 * time.Millisecond)
	for {
		select {
		case <-done:
			return
		default:
		}
		v, sd := q.Get()
		if sd {
			t.Error("Got shutdown signal")
		} else if v.(string) == "slow" {
			t.Error("Got item from the slow queue")
		}
		q.Done(v)
	}
}
