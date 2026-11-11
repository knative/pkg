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

package leaderelection

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
)

func testPod(name string, ready bool) *corev1.Pod {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: system.Namespace(), Labels: map[string]string{"app": "ctrl"}},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: status}}},
	}
}

func TestBalancerReserve(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	terminating := testPod("terminating", true)
	terminating.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	kc := fakekube.NewSimpleClientset(testPod("a", true), testPod("b", true), testPod("c", true),
		testPod("not-ready", false), terminating)
	rc := &replicaCounter{kc: kc, selector: "app=ctrl", ttl: time.Hour, logger: logtesting.TestLogger(t)}
	cc := ComponentConfig{LeaseDuration: time.Minute, RetryPeriod: 10 * time.Second}
	b := newBalancer(logtesting.TestLogger(t), &reconciler.LeaderAwareFuncs{}, rc, 10, cc)
	now := time.Now()
	b.now = func() time.Time { return now }

	// R=3 -> cap=ceil(10/3)=4.
	if got := b.capacity(ctx); got != 4 {
		t.Fatalf("capacity = %d, want 4", got)
	}
	for i := range 4 {
		if ok, _, _ := b.reserve(ctx, strconv.Itoa(i)); !ok {
			t.Fatalf("reserve(%d) refused under cap", i)
		}
	}
	// Over cap: refused until refused continuously for LeaseDuration.
	for i := range 6 {
		if ok, _, _ := b.reserve(ctx, "orphan"); ok {
			t.Fatalf("reserve(orphan) accepted after %v", time.Duration(i)*cc.RetryPeriod)
		}
		now = now.Add(cc.RetryPeriod)
	}
	if ok, _, _ := b.reserve(ctx, "orphan"); !ok {
		t.Fatal("reserve(orphan) refused after being unowned for LeaseDuration")
	}

	// A gap longer than 3 RetryPeriods restarts the orphan window.
	if ok, _, _ := b.reserve(ctx, "gap"); ok {
		t.Fatal("reserve(gap) accepted over cap")
	}
	now = now.Add(2 * cc.LeaseDuration)
	if ok, _, _ := b.reserve(ctx, "gap"); ok {
		t.Fatal("reserve(gap) accepted although refusals were not continuous")
	}

	// Releasing a bucket frees a slot.
	b.unmark("orphan")
	b.unmark("0")
	if ok, _, _ := b.reserve(ctx, "gap"); !ok {
		t.Fatal("reserve(gap) refused under cap")
	}

	// Unknown R (no matching pods) keeps default behavior: cap = all buckets.
	rc0 := &replicaCounter{kc: fakekube.NewSimpleClientset(), selector: "app=ctrl", ttl: time.Hour, logger: logtesting.TestLogger(t)}
	if got := newBalancer(logtesting.TestLogger(t), nil, rc0, 10, cc).capacity(ctx); got != 10 {
		t.Fatalf("capacity with unknown R = %d, want 10", got)
	}
}

func TestOrphanProtectionRequiresSuccessfulCurrentMembershipWrite(t *testing.T) {
	for _, scenario := range []string{"success", "write fails", "membership changes during write"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			kc := fakekube.NewSimpleClientset(testPod("a", true))
			cc := ComponentConfig{Buckets: 3, LeaseDuration: time.Second, RetryPeriod: 100 * time.Millisecond}
			b := newBalancer(logtesting.TestLogger(t), &reconciler.LeaderAwareFuncs{},
				&replicaCounter{kc: kc, selector: "app=ctrl", logger: logtesting.TestLogger(t)}, 3, cc)
			now := time.Now()
			b.now = func() time.Time { return now }
			bkts := newStandardBuckets("queue", cc)
			for _, bkt := range bkts[:2] {
				if ok, _, _ := b.reserve(ctx, bkt.Name()); !ok {
					t.Fatal("initial reservation failed")
				}
			}
			e := b.elector(bkts[0])
			e.cancel = func() {}
			if err := e.promote(ctx, func(reconciler.Bucket, types.NamespacedName) {}); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"b", "c"} {
				if _, err := kc.CoreV1().Pods(system.Namespace()).Create(ctx, testPod(name, true), metav1.CreateOptions{}); err != nil {
					t.Fatal(err)
				}
			}
			writes := 0
			writeErr := errors.New("lease write failed")
			write := func(context.Context, resourcelock.LeaderElectionRecord) error {
				writes++
				if scenario == "write fails" {
					return writeErr
				}
				if scenario == "membership changes during write" {
					pod := testPod("b", true)
					pod.UID = "replacement"
					if _, err := kc.CoreV1().Pods(system.Namespace()).Update(ctx, pod, metav1.UpdateOptions{}); err != nil {
						t.Fatal(err)
					}
					b.capacity(ctx)
				}
				return nil
			}
			lock := &balancedLock{b: b, name: bkts[2].Name()}
			for range 10 {
				if err := lock.acquire(ctx, resourcelock.LeaderElectionRecord{}, write); err == nil || writes != 0 {
					t.Fatal("acquired orphan before grace elapsed")
				}
				now = now.Add(cc.RetryPeriod)
			}
			err := lock.acquire(ctx, resourcelock.LeaderElectionRecord{}, write)
			if writes != 1 || (scenario == "write fails" && !errors.Is(err, writeErr)) || (scenario != "write fails" && err != nil) {
				t.Fatalf("orphan acquisition: writes=%d err=%v", writes, err)
			}
			if yielded := e.yield(1) != nil; yielded != (scenario != "success") {
				t.Fatalf("yielded=%v after %s", yielded, scenario)
			}
		})
	}
}

func TestRebalanceWaitsForPromotion(t *testing.T) {
	started, allowPromote := make(chan struct{}, 2), make(chan struct{})
	demoted := make(chan string, 4)
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(io.Discard), zap.InfoLevel)
	logger := zap.New(core, zap.Hooks(func(entry zapcore.Entry) error {
		// Pause the real client-go callback immediately before Promote.
		if strings.Contains(entry.Message, "has started leading") {
			started <- struct{}{}
			<-allowPromote
		}
		return nil
	})).Sugar()
	ctx, cancel := context.WithCancel(logging.WithLogger(context.Background(), logger))
	cc := ComponentConfig{
		Component: "component", Buckets: 2, LeaseDuration: time.Minute,
		RenewDeadline: 40 * time.Second, RetryPeriod: 10 * time.Millisecond, RebalancePodSelector: "app=ctrl",
	}
	kc := fakekube.NewSimpleClientset(testPod("a", true))
	h := &heldSet{bkts: map[string]struct{}{}}
	la := h.leaderAware()
	onDemote := la.DemoteFunc
	la.DemoteFunc = func(b reconciler.Bucket) {
		onDemote(b)
		demoted <- b.Name()
	}
	el, err := BuildElector(WithStandardLeaderElectorBuilder(ctx, kc, cc), la, "queue",
		func(reconciler.Bucket, types.NamespacedName) {})
	if err != nil {
		t.Fatal(err)
	}
	ra := el.(*runAll)
	bal := ra.les[len(ra.les)-1].(*balancer)
	done := make(chan struct{})
	go func() { el.Run(ctx); close(done) }()
	defer func() {
		select {
		case <-allowPromote:
		default:
			close(allowPromote)
		}
		cancel()
		<-done
	}()
	for range 2 {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out acquiring leases")
		}
	}
	// A new replica lowers the cap while both Promote callbacks are pending.
	bal.rc.mu.Lock()
	bal.rc.count, bal.rc.fetched, bal.rc.ttl = 2, time.Now(), time.Hour
	bal.rc.mu.Unlock()
	bal.maybeYield(ctx)
	select {
	case name := <-demoted:
		t.Fatalf("demoted %q before Promote completed", name)
	default:
	}
	close(allowPromote)
	for deadline := time.Now().Add(5 * time.Second); ; {
		bal.mu.Lock()
		ready := len(bal.leaders) == 2
		bal.mu.Unlock()
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out promoting buckets")
		}
		time.Sleep(time.Millisecond)
	}
	bal.maybeYield(ctx)
	var yielded string
	select {
	case yielded = <-demoted:
	case <-time.After(5 * time.Second):
		t.Fatal("did not yield a promoted bucket")
	}
	for deadline := time.Now().Add(5 * time.Second); bal.isHeld(yielded); {
		if time.Now().After(deadline) {
			t.Fatal("timed out releasing yielded bucket")
		}
		time.Sleep(time.Millisecond)
	}
	lease, err := kc.CoordinationV1().Leases(system.Namespace()).Get(ctx, yielded, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if lease.Spec.HolderIdentity != nil && *lease.Spec.HolderIdentity != "" {
		t.Fatal("yielded lease still has a holder")
	}
	if names := h.names(); len(names) != 1 || names[0] == yielded {
		t.Fatalf("leaders after yield = %v, want only the other bucket", names)
	}
}

func TestBucketElectorIgnoresPreviousTerm(t *testing.T) {
	promotes, demotes := 0, 0
	la := &reconciler.LeaderAwareFuncs{
		PromoteFunc: func(reconciler.Bucket, func(reconciler.Bucket, types.NamespacedName)) error { promotes++; return nil },
		DemoteFunc:  func(reconciler.Bucket) { demotes++ },
	}
	b := newBalancer(logtesting.TestLogger(t), la, nil, 1, ComponentConfig{})
	e := b.elector(reconciler.UniversalBucket())
	enq := func(reconciler.Bucket, types.NamespacedName) {}
	oldCtx, oldCancel := context.WithCancel(context.Background())
	defer oldCancel()
	e.cancel = oldCancel
	b.held[e.bkt.Name()] = struct{}{}
	if err := e.promote(oldCtx, enq); err != nil {
		t.Fatal(err)
	}
	finishYield := e.yield(0)
	if finishYield == nil {
		t.Fatal("did not yield")
	}
	// Renewal fails during the drain wait; OnStoppedLeading finishes the old
	// term and Run starts a new term before the old yield calls cancel.
	oldCancel()
	e.stopped()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.cancel = cancel
	b.held[e.bkt.Name()] = struct{}{}
	if err := e.promote(ctx, enq); err != nil {
		t.Fatal(err)
	}
	finishYield.cancel()
	if err := e.promote(oldCtx, enq); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() != nil || !la.IsLeaderFor(types.NamespacedName{}) || promotes != 2 || demotes != 1 {
		t.Fatalf("old term affected new leader: ctx=%v promotes=%d demotes=%d", ctx.Err(), promotes, demotes)
	}
	cancel()
	e.stopped()
	if err := e.promote(ctx, enq); err != nil {
		t.Fatal(err)
	}
	if la.IsLeaderFor(types.NamespacedName{}) || b.isHeld(e.bkt.Name()) || promotes != 2 || demotes != 2 {
		t.Fatalf("stopped bucket was promoted again: promotes=%d demotes=%d", promotes, demotes)
	}
}

type blockingDrainer struct {
	reconciler.LeaderAwareFuncs
	started chan struct{}
	finish  chan struct{}
}

func (d *blockingDrainer) Drain(ctx context.Context, _ reconciler.Bucket) error {
	close(d.started)
	<-ctx.Done()
	<-d.finish
	return ctx.Err()
}

func TestDrainFailureRestoresOnlyCurrentTerm(t *testing.T) {
	for _, scenario := range []string{"timeout", "lease lost", "reelected", "shutdown"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			promotes := 0
			la := &blockingDrainer{
				LeaderAwareFuncs: reconciler.LeaderAwareFuncs{
					PromoteFunc: func(reconciler.Bucket, func(reconciler.Bucket, types.NamespacedName)) error {
						promotes++
						return nil
					},
				},
				started: make(chan struct{}), finish: make(chan struct{}),
			}
			logger := logtesting.TestLogger(t)
			rc := &replicaCounter{count: 2, fetched: time.Now(), ttl: time.Hour}
			b := newBalancer(logger, la, rc, 2, ComponentConfig{LeaseDuration: 30 * time.Millisecond})
			now := time.Now()
			b.now = func() time.Time { return now }
			e := b.elector(reconciler.UniversalBucket())
			oldCtx, oldCancel := context.WithCancel(ctx)
			defer oldCancel()
			e.cancel = oldCancel
			b.held[""] = struct{}{}
			b.held["other"] = struct{}{}
			enq := func(reconciler.Bucket, types.NamespacedName) {}
			if err := e.promote(oldCtx, enq); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			go func() { b.maybeYield(ctx); close(done) }()
			<-la.started
			var newCtx context.Context
			switch scenario {
			case "lease lost", "reelected":
				oldCancel()
				e.stopped()
				if scenario == "reelected" {
					var newCancel context.CancelFunc
					newCtx, newCancel = context.WithCancel(ctx)
					defer newCancel()
					e.cancel = newCancel
					b.held[""] = struct{}{}
					if err := e.promote(newCtx, enq); err != nil {
						t.Fatal(err)
					}
				}
			case "shutdown":
				cancel()
				e.stopped()
			}
			close(la.finish)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("failed drain did not return")
			}
			wantLeader := scenario == "timeout" || scenario == "reelected"
			wantPromotes := 1
			if wantLeader {
				wantPromotes = 2
			}
			if la.IsLeaderFor(types.NamespacedName{}) != wantLeader || promotes != wantPromotes {
				t.Fatalf("leadership after drain: leader=%v promotes=%d", la.IsLeaderFor(types.NamespacedName{}), promotes)
			}
			if scenario == "timeout" && (oldCtx.Err() != nil || !b.isHeld("") || e.yield(1) != nil) {
				t.Fatal("timeout released the lease or failed to cool down the bucket")
			}
			if newCtx != nil && newCtx.Err() != nil {
				t.Fatal("old drain canceled the new term")
			}
		})
	}
}

func TestBucketElectorStopWaitsForPromotion(t *testing.T) {
	started, finishPromote := make(chan struct{}), make(chan struct{})
	la := &reconciler.LeaderAwareFuncs{PromoteFunc: func(reconciler.Bucket, func(reconciler.Bucket, types.NamespacedName)) error {
		close(started)
		<-finishPromote
		return nil
	}}
	b := newBalancer(logtesting.TestLogger(t), la, nil, 1, ComponentConfig{})
	e := b.elector(reconciler.UniversalBucket())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	promoteDone := make(chan error, 1)
	go func() { promoteDone <- e.promote(ctx, func(reconciler.Bucket, types.NamespacedName) {}) }()
	<-started
	defer func() {
		select {
		case <-finishPromote:
		default:
			close(finishPromote)
		}
	}()
	if !b.mu.TryLock() {
		t.Fatal("Promote holds the balancer lock, blocking other buckets")
	}
	b.mu.Unlock()
	cancel()
	stopped := make(chan struct{})
	go func() { e.stopped(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("stop returned before Promote completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(finishPromote)
	if err := <-promoteDone; err != nil {
		t.Fatal(err)
	}
	<-stopped
	if la.IsLeaderFor(types.NamespacedName{}) {
		t.Fatal("stopped bucket still reports leadership")
	}
}

// heldSet records the buckets a replica's reconciler is leader for.
type heldSet struct {
	mu   sync.Mutex
	bkts map[string]struct{}
}

// These elector tests do no work outside the synchronous leadership callbacks.
type idleLeaderAware struct {
	reconciler.LeaderAwareFuncs
}

func (*idleLeaderAware) Drain(context.Context, reconciler.Bucket) error { return nil }

func (h *heldSet) leaderAware() *idleLeaderAware {
	return &idleLeaderAware{LeaderAwareFuncs: reconciler.LeaderAwareFuncs{
		PromoteFunc: func(b reconciler.Bucket, _ func(reconciler.Bucket, types.NamespacedName)) error {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.bkts[b.Name()] = struct{}{}
			return nil
		},
		DemoteFunc: func(b reconciler.Bucket) {
			h.mu.Lock()
			defer h.mu.Unlock()
			delete(h.bkts, b.Name())
		},
	}}
}

func (h *heldSet) names() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.bkts))
	for n := range h.bkts {
		out = append(out, n)
	}
	return out
}

func TestOrphanFallbackKeepsCoverageUntilMembershipChanges(t *testing.T) {
	ctx, cancel := context.WithCancel(logtesting.TestContextWithLogger(t))
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	// Both pods are Ready, but initially only A participates in this queue.
	aPod, bPod := testPod("a", true), testPod("b", true)
	bPod.UID = "old-b"
	kc := fakekube.NewSimpleClientset(aPod, bPod)
	cc := ComponentConfig{
		Component: "component", Buckets: 2,
		LeaseDuration: time.Second, RenewDeadline: 800 * time.Millisecond,
		RetryPeriod: 100 * time.Millisecond, RebalancePodSelector: "app=ctrl",
	}
	start := func() *heldSet {
		h := &heldSet{bkts: map[string]struct{}{}}
		el, err := BuildElector(WithStandardLeaderElectorBuilder(ctx, kc, cc), h.leaderAware(), "queue",
			func(reconciler.Bucket, types.NamespacedName) {})
		if err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func() { defer wg.Done(); el.Run(ctx) }()
		return h
	}
	a := start()
	for deadline := time.Now().Add(5 * time.Second); len(a.names()) != 2; {
		if time.Now().After(deadline) {
			t.Fatal("orphan fallback did not acquire both buckets")
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Exercise several yield ticks after fallback without changing membership.
	for deadline := time.Now().Add(3 * cc.LeaseDuration); time.Now().Before(deadline); {
		if names := a.names(); len(names) != 2 {
			t.Fatalf("orphan fallback lost coverage with unchanged membership: %v", names)
		}
		time.Sleep(5 * time.Millisecond)
	}
	// Replacing a pod must reset protection even when its name and R stay the same.
	if err := kc.CoreV1().Pods(system.Namespace()).Delete(ctx, "b", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	bPod.UID = "new-b"
	if _, err := kc.CoreV1().Pods(system.Namespace()).Create(ctx, bPod, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	b := start()
	for deadline := time.Now().Add(5 * time.Second); ; {
		an, bn := a.names(), b.names()
		if len(an) == 1 && len(bn) == 1 && an[0] != bn[0] {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("membership change did not restore balance: a=%v b=%v", an, bn)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRebalanceConverges(t *testing.T) {
	ctx, cancel := context.WithCancel(logtesting.TestContextWithLogger(t))
	// Electors log via the test logger, so wait for them before returning.
	var wg sync.WaitGroup
	defer func() {
		cancel()
		wg.Wait()
	}()
	kc := fakekube.NewSimpleClientset(testPod("a", true))
	cc := ComponentConfig{
		Component:            "component",
		Buckets:              10,
		LeaseDuration:        time.Second,
		RenewDeadline:        800 * time.Millisecond,
		RetryPeriod:          100 * time.Millisecond,
		RebalancePodSelector: "app=ctrl",
	}
	start := func(ctx context.Context) *heldSet {
		h := &heldSet{bkts: map[string]struct{}{}}
		el, err := BuildElector(WithStandardLeaderElectorBuilder(ctx, kc, cc), h.leaderAware(), "queue",
			func(reconciler.Bucket, types.NamespacedName) {})
		if err != nil {
			t.Fatal("BuildElector() =", err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			el.Run(ctx)
		}()
		return h
	}
	// waitBalanced waits until every bucket has exactly one owner and no
	// replica holds more than maxHeld buckets.
	waitBalanced := func(maxHeld int, timeout time.Duration, replicas ...*heldSet) {
		t.Helper()
		var state [][]string
		for deadline := time.Now().Add(timeout); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			state = state[:0]
			owners := map[string]int{}
			ok := true
			for _, r := range replicas {
				names := r.names()
				state = append(state, names)
				ok = ok && len(names) <= maxHeld
				for _, n := range names {
					owners[n]++
				}
			}
			for _, c := range owners {
				ok = ok && c == 1
			}
			if ok && len(owners) == int(cc.Buckets) {
				return
			}
		}
		t.Fatalf("not balanced (max %d per replica) after %v: %v", maxHeld, timeout, state)
	}

	// A single replica takes every bucket.
	a := start(ctx)
	waitBalanced(10, 5*time.Second, a)

	// Two more replicas join: R=3, cap=4. A yields one bucket per LeaseDuration.
	for _, n := range []string{"b", "c"} {
		if _, err := kc.CoreV1().Pods(system.Namespace()).Create(ctx, testPod(n, true), metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	cctx, cancelC := context.WithCancel(ctx)
	b, c := start(ctx), start(cctx)
	waitBalanced(4, 20*time.Second, a, b, c)

	// C goes away: R=2, cap=5, its buckets are taken over by A and B.
	cancelC()
	if err := kc.CoreV1().Pods(system.Namespace()).Delete(ctx, "c", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	waitBalanced(5, 10*time.Second, a, b)
}
