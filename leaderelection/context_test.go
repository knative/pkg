// +build !race
// TODO(https://github.com/kubernetes/kubernetes/issues/90952): Remove the above.

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

package leaderelection

import (
	"context"
	"fmt"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	fakekube "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	"knative.dev/pkg/reconciler"
	_ "knative.dev/pkg/system/testing"
)

func TestWithBuilder(t *testing.T) {
	const buckets = 3
	cc := ComponentConfig{
		Component:     "the-component",
		Buckets:       buckets,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
	}
	kc := fakekube.NewSimpleClientset()
	ctx := context.Background()

	gotNames := make(sets.String, buckets)
	promoted := make(chan string)
	demoted := make(chan struct{})
	laf := &reconciler.LeaderAwareFuncs{
		PromoteFunc: func(bkt reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
			promoted <- bkt.Name()
			return nil
		},
		DemoteFunc: func(bkt reconciler.Bucket) {
			demoted <- struct{}{}
		},
	}
	enq := func(reconciler.Bucket, types.NamespacedName) {}

	created := make(chan struct{})
	kc.PrependReactor("create", "leases",
		func(action ktesting.Action) (bool, runtime.Object, error) {
			created <- struct{}{}
			return false, nil, nil
		},
	)

	updated := make(chan struct{})
	kc.PrependReactor("update", "leases",
		func(action ktesting.Action) (bool, runtime.Object, error) {
			// Only close updated once.
			select {
			case <-updated:
			default:
				close(updated)
			}
			return false, nil, nil
		},
	)

	if HasLeaderElection(ctx) {
		t.Error("HasLeaderElection() = true, wanted false")
	}
	if le, err := BuildElector(ctx, laf, "name", enq); err != nil {
		t.Errorf("BuildElector() = %v, wanted an unopposedElector", err)
	} else if _, ok := le.(*unopposedElector); !ok {
		t.Errorf("BuildElector() = %T, wanted an unopposedElector", le)
	}

	ctx = WithStandardLeaderElectorBuilder(ctx, kc, cc)
	if !HasLeaderElection(ctx) {
		t.Error("HasLeaderElection() = false, wanted true")
	}

	le, err := BuildElector(ctx, laf, "name", enq)
	if err != nil {
		t.Fatal("BuildElector() =", err)
	}

	// We shouldn't see leases until we Run the elector.
	select {
	case s := <-promoted:
		gotNames.Insert(s)
		t.Error("Got promoted, want no actions.")
	case <-demoted:
		t.Error("Got demoted, want no actions.")
	case <-created:
		t.Error("Got created, want no actions.")
	case <-updated:
		t.Error("Got updated, want no actions.")
	default:
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go le.Run(ctx)

	// We expect 3 lease to be created.
	for i := 0; i < buckets; i++ {
		select {
		case <-created:
		case <-time.After(1 * time.Second):
			t.Fatal("Timed out waiting for lease creation.")
		}
	}
	// We expect to have been promoted 3 times.
	for i := 0; i < buckets; i++ {
		select {
		case s := <-promoted:
			gotNames.Insert(s)
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for promotion.")
		}
	}

	// Cancelling the context should case us to give up leadership.
	cancel()

	select {
	case <-updated:
		// We expect the lease to be updated.
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for lease update.")
	}
	// We expect to have been demoted 3 times.
	for i := 0; i < buckets; i++ {
		select {
		case <-demoted:
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for demotion.")
		}
	}

	want := sets.NewString(
		"the-component.name.00-of-03",
		"the-component.name.01-of-03",
		"the-component.name.02-of-03",
	)
	if !gotNames.Equal(want) {
		t.Errorf("BucketSet.BucketList() = %q, want: %q", gotNames, want)
	}
}

func TestBuilderWithCustomizedLeaseName(t *testing.T) {
	const buckets = 3
	cc := ComponentConfig{
		Component:     "the-component",
		Buckets:       buckets,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		LeaseName: func(i uint32) string {
			return fmt.Sprintf("bucket-%02d", i)
		},
	}
	kc := fakekube.NewSimpleClientset()
	ctx := context.Background()

	gotNames := make(sets.String, buckets)
	promoted := make(chan string)
	laf := &reconciler.LeaderAwareFuncs{
		PromoteFunc: func(bkt reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
			promoted <- bkt.Name()
			return nil
		},
	}
	enq := func(reconciler.Bucket, types.NamespacedName) {}
	ctx = WithStandardLeaderElectorBuilder(ctx, kc, cc)
	le, err := BuildElector(ctx, laf, "name", enq)
	if err != nil {
		t.Fatal("BuildElector() =", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go le.Run(ctx)

	// We expect to have been promoted 3 times.
	for i := 0; i < buckets; i++ {
		select {
		case s := <-promoted:
			gotNames.Insert(s)
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for promotion.")
		}
	}

	want := sets.NewString(
		"bucket-00",
		"bucket-01",
		"bucket-02",
	)
	if !gotNames.Equal(want) {
		t.Errorf("BucketSet.BucketList() = %q, want: %q", gotNames, want)
	}
}
