/*
Copyright 2026 The Knative Authors

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
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/reconciler"
)

// bucketDrainer observes the worker calls without changing the reconciler's
// leadership checks. The workqueue already serializes calls for the same key.
type bucketDrainer struct {
	reconciler.LeaderAware
	mu     sync.Mutex
	active map[types.NamespacedName]chan struct{}
}

func (d *bucketDrainer) start(key types.NamespacedName) func() {
	done := make(chan struct{})
	d.mu.Lock()
	d.active[key] = done
	d.mu.Unlock()
	return func() {
		d.mu.Lock()
		delete(d.active, key)
		close(done)
		d.mu.Unlock()
	}
}

// Drain is called only for a voluntary handoff, after Demote. Calls that start
// after this snapshot observe the demoted state in their leadership check.
func (d *bucketDrainer) Drain(ctx context.Context, bkt reconciler.Bucket) error {
	d.mu.Lock()
	var pending []chan struct{}
	for key, done := range d.active {
		if bkt.Has(key) {
			pending = append(pending, done)
		}
	}
	d.mu.Unlock()
	for _, done := range pending {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
