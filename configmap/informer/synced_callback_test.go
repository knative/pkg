/*
Copyright 2021 The Knative Authors

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

package informer

import (
	"testing"
	"time"
)

func TestNamedWaitGroup(t *testing.T) {
	nwg := newNamedWaitGroup()

	// nothing has been added so wait returns immediately
	initiallyDone := make(chan struct{})
	go func() {
		defer close(initiallyDone)
		nwg.Wait()
	}()
	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("Wait should have returned immediately but still hadn't after timeout elapsed")
	case <-initiallyDone:
		// the Wait returned as expected since nothing was tracked
	}

	// Add some keys to track
	nwg.Add("foo")
	nwg.Add("bar")
	// Adding keys multiple times shouldn't increment the counter again
	nwg.Add("bar")

	// Now that we've added keys, when we Wait, it should block
	done := make(chan struct{})
	go func() {
		defer close(done)
		nwg.Wait()
	}()

	// Indicate that this key is done
	nwg.Done("foo")
	// Indicating done on a key that doesn't exist should do nothing
	nwg.Done("doesnt exist")

	// Only one of the tracked keys has completed, so the channel should not yet have closed
	select {
	case <-done:
		t.Fatalf("Wait returned before all keys were done")
	default:
		// as expected, the channel is still open (waiting for the final key to be done)
	}

	// Indicate the final key is done
	nwg.Done("bar")

	// Now that all keys are done, the Wait should return
	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("Wait should have returned immediately but still hadn't after timeout elapsed")
	case <-done:
		// completed successfully
	}
}

func TestSyncedCallback(t *testing.T) {
	keys := []string{"foo", "bar"}
	objs := []interface{}{"fooobj", "barobj"}
	var seen []interface{}
	callback := func(obj interface{}) {
		seen = append(seen, obj)
	}
	sc := newSyncedCallback(keys, callback)

	// Wait for the callback to be called for all of the keys
	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc.WaitForAllKeys(stopCh)
	}()

	// Call the callback for one of the keys
	sc.Call(objs[0], "foo")

	// Only one of the tracked keys has been synced so we should still be waiting
	select {
	case <-done:
		t.Fatalf("Wait returned before all keys were done")
	default:
		// as expected, the channel is still open (waiting for the final key to be done)
	}

	// Call the callback for the other key
	sc.Call(objs[1], "bar")

	// Now that all keys are done, the Wait should return
	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("WaitForAllKeys should have returned but still hadn't after timeout elapsed")
	case <-done:
		// completed successfully
	}

	if len(seen) != 2 || seen[0] != objs[0] || seen[1] != objs[1] {
		t.Errorf("callback wasn't called as expected, expected to see %v but saw %v", objs, seen)
	}
}

func TestSyncedCallbackStops(t *testing.T) {
	sc := newSyncedCallback([]string{"somekey"}, func(obj interface{}) {})

	// Wait for the callback to be called - which it won't be!
	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc.WaitForAllKeys(stopCh)
	}()

	// Nothing has been synced so we should still be waiting
	select {
	case <-done:
		t.Fatalf("Wait returned before all keys were done")
	default:
		// as expected, the channel is still open
	}

	// signal to stop via the stop channel
	close(stopCh)

	// Even though the callback wasn't called, the Wait should return b/c of the stop channel
	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("WaitForAllKeys should have returned because of the stop channel but still hadn't after timeout elapsed")
	case <-done:
		// stopped successfully
	}
}
