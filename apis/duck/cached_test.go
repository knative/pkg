/*
Copyright 2018 The Knative Authors

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

package duck

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type BlockingInformerFactory struct {
	block  chan struct{}
	nCalls atomic.Int32
}

var _ InformerFactory = (*BlockingInformerFactory)(nil)

func (bif *BlockingInformerFactory) Get(ctx context.Context, gvr schema.GroupVersionResource) (cache.SharedIndexInformer, cache.GenericLister, error) {
	bif.nCalls.Inc()
	// Wait here until we can acquire the lock
	<-bif.block

	// return dummies to avoid subsequent calls to informerCache.init
	inf := &fakeSharedIndexInformer{}
	lister := fakeGenericLister(gvr.GroupResource())

	return inf, lister, nil
}

func TestSameGVR(t *testing.T) {
	bif := &BlockingInformerFactory{
		block: make(chan struct{}),
	}

	cif := &CachedInformerFactory{
		Delegate: bif,
	}

	// counts the number of calls to cif.Get that returned
	retGetCount := atomic.NewInt32(0)

	errGrp, ctx := errgroup.WithContext(context.Background())

	// Use the same GVR each iteration to ensure we hit the cache and don't
	// initialize the informerCache for that GVR multiple times through our
	// Delegate.
	gvr := schema.GroupVersionResource{
		Group:    "testing.knative.dev",
		Version:  "v3",
		Resource: "caches",
	}

	const iter = 10
	for i := 0; i < iter; i++ {
		errGrp.Go(func() error {
			_, _, err := cif.Get(ctx, gvr)
			retGetCount.Inc()
			return err
		})
	}

	// Give the goroutines time to make progress.
	time.Sleep(100 * time.Millisecond)

	// Check that no call to cif.Get have returned and bif.Get was called
	// only once.
	if got, want := retGetCount.Load(), int32(0); got != want {
		t.Errorf("Got %d returned call(s) to cif.Get, wanted %d", got, want)
	}
	if got, want := bif.nCalls.Load(), int32(1); got != want {
		t.Errorf("Got %d call(s) to bif.Get, wanted %d", got, want)
	}

	// Allow the Get calls to proceed.
	close(bif.block)

	if err := errGrp.Wait(); err != nil {
		t.Fatal("Error while calling cif.Get:", err)
	}

	// Check that all calls to cif.Get have returned and calls to bif.Get
	// didn't increase.
	if got, want := retGetCount.Load(), int32(iter); got != want {
		t.Errorf("Got %d returned call(s) to cif.Get, wanted %d", got, want)
	}
	if got, want := bif.nCalls.Load(), int32(1); got != want {
		t.Errorf("Got %d call(s) to bif.Get, wanted %d", got, want)
	}
}

func TestDifferentGVRs(t *testing.T) {
	bif := &BlockingInformerFactory{
		block: make(chan struct{}),
	}

	cif := &CachedInformerFactory{
		Delegate: bif,
	}

	// counts the number of calls to cif.Get that returned
	retGetCount := atomic.NewInt32(0)

	errGrp, ctx := errgroup.WithContext(context.Background())

	const iter = 10
	for i := 0; i < iter; i++ {
		// Use a different GVR each iteration to check that calls
		// to bif.Get can proceed even if a call is in progress
		// for another GVR.
		gvr := schema.GroupVersionResource{
			Group:    "testing.knative.dev",
			Version:  fmt.Sprint("v", i),
			Resource: "caches",
		}

		errGrp.Go(func() error {
			_, _, err := cif.Get(ctx, gvr)
			retGetCount.Inc()
			return err
		})
	}

	// Give the goroutines time to make progress.
	time.Sleep(100 * time.Millisecond)

	// Check that no call to cif.Get have returned and bif.Get was called
	// once per iteration.
	if got, want := retGetCount.Load(), int32(0); got != want {
		t.Errorf("Got %d returned call(s) to cif.Get, wanted %d", got, want)
	}
	if got, want := bif.nCalls.Load(), int32(iter); got != want {
		t.Errorf("Got %d call(s) to bif.Get, wanted %d", got, want)
	}

	// Allow the Get calls to proceed.
	close(bif.block)

	if err := errGrp.Wait(); err != nil {
		t.Fatal("Error while calling cif.Get:", err)
	}

	// Check that all calls to cif.Get have returned and the number of
	// calls to bif.Get didn't increase.
	if got, want := retGetCount.Load(), int32(iter); got != want {
		t.Errorf("Got %d returned call(s) to cif.Get, wanted %d", got, want)
	}
	if got, want := bif.nCalls.Load(), int32(iter); got != want {
		t.Errorf("Got %d call(s) to bif.Get, wanted %d", got, want)
	}
}

// fakeGenericLister returns a dummy cache.GenericLister.
func fakeGenericLister(gr schema.GroupResource) cache.GenericLister {
	var dummyKeyFunc cache.KeyFunc = func(interface{}) (string, error) {
		return "", nil
	}

	dummyIndexer := cache.NewIndexer(dummyKeyFunc, cache.Indexers{})
	return cache.NewGenericLister(dummyIndexer, gr)
}
