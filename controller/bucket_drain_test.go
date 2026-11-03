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
	"fmt"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/hash"
	"knative.dev/pkg/leaderelection"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
)

type slowLeaderReconciler struct {
	reconciler.LeaderAwareFuncs
	run func(string)
}

func (r *slowLeaderReconciler) Reconcile(_ context.Context, key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	if r.IsLeaderFor(types.NamespacedName{Namespace: ns, Name: name}) {
		r.run(key)
	}
	return nil
}

func TestRebalanceDrainsInflightReconcile(t *testing.T) {
	testRebalanceInflight(t, "app=ctrl", false)
}

func TestRebalanceDrainTimeoutRestoresLeadership(t *testing.T) {
	testRebalanceInflight(t, "app=ctrl", true)
}

func TestWithoutRebalanceKeepsInflightReconcile(t *testing.T) {
	testRebalanceInflight(t, "", false)
}

func testRebalanceInflight(t *testing.T, selector string, timeout bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(logtesting.TestContextWithLogger(t))
	var wg sync.WaitGroup
	unblock := map[string]chan struct{}{}
	defer func() {
		for _, done := range unblock {
			select {
			case <-done:
			default:
				close(done)
			}
		}
		cancel()
		wg.Wait()
	}()
	pod := func(name string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: system.Namespace(), Labels: map[string]string{"app": "ctrl"}},
			Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}},
		}
	}
	kc := fakekube.NewSimpleClientset(pod("a"))
	cc := leaderelection.ComponentConfig{
		Buckets: 2, LeaseDuration: time.Second, RenewDeadline: 800 * time.Millisecond,
		RetryPeriod: 100 * time.Millisecond, RebalancePodSelector: selector,
		LeaseName: func(i uint32) string { return fmt.Sprintf("review-%d", i) },
	}
	var keys []types.NamespacedName
	bucketKeys := map[string]string{}
	for _, bucket := range hash.NewBucketSet(sets.New[string]("review-0", "review-1")).Buckets() {
		for i := 0; ; i++ {
			key := types.NamespacedName{Namespace: "default", Name: fmt.Sprintf("task-%d", i)}
			if bucket.Has(key) {
				keys = append(keys, key)
				bucketKeys[bucket.Name()] = key.String()
				unblock[key.String()] = make(chan struct{})
				break
			}
		}
	}
	started := make(chan string, 2)
	newOwner := make(chan string, 2)
	demoted := make(chan string, 8)
	restored := make(chan string, 8)
	recovered := make(chan string, 1)
	var mu sync.Mutex
	active := map[string]bool{}
	start := func(id string) *Impl {
		r := &slowLeaderReconciler{}
		promoted := map[string]bool{}
		var promoteMu sync.Mutex
		r.PromoteFunc = func(bucket reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
			promoteMu.Lock()
			if id == "a" && promoted[bucket.Name()] {
				restored <- bucket.Name()
			}
			promoted[bucket.Name()] = true
			promoteMu.Unlock()
			for _, key := range keys {
				enq(bucket, key)
			}
			return nil
		}
		if id == "a" {
			r.DemoteFunc = func(bucket reconciler.Bucket) { demoted <- bucket.Name() }
		}
		r.run = func(key string) {
			mu.Lock()
			if id == "a" {
				if _, ok := unblock[key]; !ok {
					mu.Unlock()
					recovered <- key
					return
				}
				active[key] = true
				mu.Unlock()
				select {
				case started <- key:
				default:
				}
				<-unblock[key]
				mu.Lock()
				delete(active, key)
				mu.Unlock()
				return
			}
			stillActive := active[key]
			mu.Unlock()
			if stillActive {
				t.Errorf("new replica reconciled %s while the old replica was still executing it", key)
			}
			newOwner <- key
		}
		cc.Identity = id
		cctx := leaderelection.WithStandardLeaderElectorBuilder(ctx, kc, cc)
		impl := NewContext(cctx, r, ControllerOptions{
			WorkQueueName: "drain-queue",
			Logger:        logtesting.TestLogger(t),
			Concurrency:   3,
		})
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := impl.Run(cctx); err != nil {
				t.Errorf("controller Run: %v", err)
			}
		}()
		return impl
	}
	a := start("a")
	for range 2 {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("old replica did not start both reconciles")
		}
	}
	if _, err := kc.CoreV1().Pods(system.Namespace()).Create(ctx, pod("b"), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	start("b")
	if selector == "" {
		select {
		case bucket := <-demoted:
			t.Fatalf("demoted %s with rebalancing disabled", bucket)
		case <-time.After(2 * cc.LeaseDuration):
		}
		return
	}
	var bucket string
	select {
	case bucket = <-demoted:
	case <-time.After(5 * time.Second):
		t.Fatal("did not start yielding a bucket")
	}
	before, err := kc.CoordinationV1().Leases(system.Namespace()).Get(ctx, bucket, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Hold the old reconcile beyond both RetryPeriod and LeaseDuration. Renewal
	// must continue until it exits, rather than releasing after a fixed delay.
	select {
	case key := <-newOwner:
		t.Fatalf("transferred %s before its reconcile completed", key)
	case <-time.After(cc.LeaseDuration + 2*cc.RetryPeriod):
	}
	after, err := kc.CoordinationV1().Leases(system.Namespace()).Get(ctx, bucket, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if after.Spec.HolderIdentity == nil || *after.Spec.HolderIdentity != "a" ||
		!after.Spec.RenewTime.After(before.Spec.RenewTime.Time) {
		t.Fatalf("lease was not renewed by the old owner while draining: %+v", after.Spec)
	}
	key := bucketKeys[bucket]
	if timeout {
		select {
		case got := <-restored:
			if got != bucket {
				t.Fatalf("restored bucket = %s, want %s", got, bucket)
			}
		case <-time.After(2 * cc.LeaseDuration):
			t.Fatal("timed-out drain did not restore local leadership")
		}
		select {
		case got := <-newOwner:
			t.Fatalf("transferred %s after a drain timeout", got)
		default:
		}
		lease, err := kc.CoordinationV1().Leases(system.Namespace()).Get(ctx, bucket, metav1.GetOptions{})
		if err != nil || lease.Spec.HolderIdentity == nil || *lease.Spec.HolderIdentity != "a" {
			t.Fatalf("timed-out drain lost its lease: lease=%v err=%v", lease, err)
		}
		// Both original keys remain blocked. A third worker must be able to
		// handle a new key in the restored bucket without releasing its lease.
		for _, bkt := range hash.NewBucketSet(sets.New[string]("review-0", "review-1")).Buckets() {
			if bkt.Name() != bucket {
				continue
			}
			for i := 0; ; i++ {
				fresh := types.NamespacedName{Namespace: "default", Name: fmt.Sprintf("fresh-%d", i)}
				if bkt.Has(fresh) {
					a.EnqueueKey(fresh)
					break
				}
			}
		}
		select {
		case <-recovered:
		case <-time.After(cc.LeaseDuration):
			t.Fatal("restored bucket did not process new work")
		}
	}
	close(unblock[key])
	select {
	case got := <-newOwner:
		if got != key {
			t.Fatalf("transferred key = %s, want %s", got, key)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("did not transfer the drained bucket")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(active) != 1 {
		t.Fatalf("other bucket did not remain in flight: %v", active)
	}
}
