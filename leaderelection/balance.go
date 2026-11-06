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

// Bucket rebalancing for standardBuilder.
//
// Without rebalancing, whichever replica comes up first can grab every bucket
// and never release excess buckets. When RebalancePodSelector is set, each
// replica:
//   - refuses to acquire a bucket while it already holds ceil(B/R) buckets
//     (gated in the resource lock, so parallel electors cannot overshoot);
//   - yields at most one excess bucket per LeaseDuration: Demote first, drain
//     in-flight reconciles while renewing, then release the lease;
//   - aborts a drain after two LeaseDurations, restoring local leadership only
//     in the same live term, and waits one LeaseDuration before retrying it;
//   - still takes a bucket over the cap if it has stayed unowned for
//     LeaseDuration, and retains the required capacity until Ready pod
//     membership changes, so a wrong R does not repeatedly orphan buckets.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
)

// replicaCounter counts Ready, non-terminating pods matching selector in the
// system namespace. The result is cached for ttl and shared by all queues of
// a process. 0 means unknown.
type replicaCounter struct {
	kc       kubernetes.Interface
	selector string
	ttl      time.Duration
	logger   *zap.SugaredLogger

	mu         sync.Mutex
	count      int
	fetched    time.Time
	members    sets.Set[string]
	generation uint64
}

func (rc *replicaCounter) get(ctx context.Context) (int, uint64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if time.Since(rc.fetched) < rc.ttl {
		return rc.count, rc.generation
	}
	rc.fetched = time.Now()
	pods, err := rc.kc.CoreV1().Pods(system.Namespace()).List(ctx, metav1.ListOptions{LabelSelector: rc.selector})
	if err != nil {
		rc.logger.Warnw("Failed to count replicas, keeping last value", "count", rc.count, zap.Error(err))
		return rc.count, rc.generation
	}
	members := sets.New[string]()
	for i := range pods.Items {
		if podReady(&pods.Items[i]) {
			p := &pods.Items[i]
			members.Insert(p.Name + "/" + string(p.UID))
		}
	}
	if !members.Equal(rc.members) {
		rc.members = members
		rc.generation++
	}
	rc.count = len(members)
	return rc.count, rc.generation
}

func podReady(p *corev1.Pod) bool {
	if p.DeletionTimestamp != nil {
		return false
	}
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

// balancer tracks the buckets of one queue held by this replica.
type balancer struct {
	logger   *zap.SugaredLogger
	la       reconciler.LeaderAware
	rc       *replicaCounter
	total    int
	interval time.Duration // LeaseDuration: yield cooldown and orphan grace.
	retry    time.Duration // RetryPeriod: cadence of orphan acquisition attempts.
	now      func() time.Time

	mu           sync.Mutex
	leaders      map[string]*bucketElector // Only buckets whose Promote has completed.
	held         map[string]struct{}
	refusedSince map[string]time.Time
	lastRefused  map[string]time.Time
	generation   uint64
	minHeld      int // Successful orphan takeover protects this capacity until membership changes.
}

func newBalancer(logger *zap.SugaredLogger, la reconciler.LeaderAware, rc *replicaCounter, total int, cc ComponentConfig) *balancer {
	return &balancer{
		logger:       logger,
		la:           la,
		rc:           rc,
		total:        total,
		interval:     cc.LeaseDuration,
		retry:        cc.RetryPeriod,
		now:          time.Now,
		leaders:      make(map[string]*bucketElector, total),
		held:         make(map[string]struct{}, total),
		refusedSince: make(map[string]time.Time),
		lastRefused:  make(map[string]time.Time),
	}
}

// capacity returns ceil(total/R), or total (default behavior) if R is unknown.
func (b *balancer) capacity(ctx context.Context) int {
	r, generation := b.rc.get(ctx)
	b.mu.Lock()
	if b.generation < generation {
		b.generation = generation
		b.minHeld = 0
	}
	b.mu.Unlock()
	if r < 1 {
		return b.total
	}
	return (b.total + r - 1) / r
}

func (b *balancer) isHeld(name string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.held[name]
	return ok
}

// reserve claims a slot for acquiring bucket name. client-go only tries to
// acquire a lease that is free or expired, so continuous refusals mean the
// bucket is orphaned; after interval we take it regardless of the cap.
func (b *balancer) reserve(ctx context.Context, name string) (reserved, orphan bool, generation uint64) {
	c := b.capacity(ctx)
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.held) < c {
		b.held[name] = struct{}{}
		delete(b.refusedSince, name)
		delete(b.lastRefused, name)
		return true, false, 0
	}
	now := b.now()
	// Refusals are continuous if they come at least every ~2.2 RetryPeriod
	// (client-go jitter factor 1.2); a longer gap means someone else owned it.
	if now.Sub(b.lastRefused[name]) > 3*b.retry {
		b.refusedSince[name] = now
	}
	b.lastRefused[name] = now
	if now.Sub(b.refusedSince[name]) < b.interval {
		return false, false, 0
	}
	b.logger.Warnf("Bucket %q unowned for %v, acquiring over cap (held=%d cap=%d)", name, b.interval, len(b.held), c)
	b.held[name] = struct{}{}
	delete(b.refusedSince, name)
	delete(b.lastRefused, name)
	return true, true, b.generation
}

func (b *balancer) unmark(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.held, name)
	if b.minHeld > len(b.held) {
		b.minHeld = len(b.held)
	}
}

// Run implements Elector: it yields one excess bucket per interval.
func (b *balancer) Run(ctx context.Context) {
	t := time.NewTicker(b.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.maybeYield(ctx)
		}
	}
}

func (b *balancer) maybeYield(ctx context.Context) {
	c := b.capacity(ctx)
	b.mu.Lock()
	held := len(b.held)
	if held <= c || held <= b.minHeld {
		b.mu.Unlock()
		return
	}
	leaders := make([]*bucketElector, 0, len(b.leaders))
	for _, e := range b.leaders {
		leaders = append(leaders, e)
	}
	b.mu.Unlock()

	for _, e := range leaders {
		if term := e.yield(c); term != nil {
			b.logger.Infof("Yielding bucket %q (held=%d cap=%d)", e.bkt.Name(), held, c)
			drainCtx, stop := context.WithTimeout(term.ctx, 2*b.interval)
			err := b.la.(reconciler.BucketDrainer).Drain(drainCtx, e.bkt)
			stop()
			if err != nil {
				b.logger.Warnw("Aborting bucket yield without releasing the lease", "bucket", e.bkt.Name(), zap.Error(err))
				if err := e.restore(term); err != nil {
					b.logger.Fatalf("Failed to restore bucket %q after drain: %v", e.bkt.Name(), err)
				}
			} else {
				term.cancel() // Release only the term that was demoted.
			}
			return
		}
	}
}

// wrap gates lease acquisition of bkt on the balancer's capacity.
func (b *balancer) wrap(bkt reconciler.Bucket, rl resourcelock.Interface) resourcelock.Interface {
	return &balancedLock{Interface: rl, b: b, name: bkt.Name()}
}

// elector runs le in a loop with a per-bucket context that maybeYield can cancel.
func (b *balancer) elector(bkt reconciler.Bucket) *bucketElector {
	return &bucketElector{b: b, bkt: bkt}
}

type bucketTerm struct {
	ctx    context.Context
	cancel context.CancelFunc
	enq    func(reconciler.Bucket, types.NamespacedName)
}

type bucketElector struct {
	b   *balancer
	bkt reconciler.Bucket
	le  Elector

	// Callbacks are serialized per bucket; never invoke them under b.mu.
	mu         sync.Mutex
	cancel     context.CancelFunc
	term       *bucketTerm
	yieldAfter time.Time
}

func (e *bucketElector) promote(ctx context.Context, enq func(reconciler.Bucket, types.NamespacedName)) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.promoteLocked(&bucketTerm{ctx: ctx, cancel: e.cancel, enq: enq})
}

// promoteLocked requires e.mu so restoration cannot race a stopped/new term.
func (e *bucketElector) promoteLocked(term *bucketTerm) error {
	// client-go starts this callback asynchronously. A previous term's callback
	// can arrive after OnStoppedLeading, even after another term has started.
	select {
	case <-term.ctx.Done():
		return nil
	default:
	}
	if err := e.b.la.Promote(e.bkt, term.enq); err != nil {
		return err
	}
	e.term = term
	e.b.mu.Lock()
	e.b.leaders[e.bkt.Name()] = e
	e.b.mu.Unlock()
	return nil
}

func (e *bucketElector) restore(term *bucketTerm) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.term != term || term.ctx.Err() != nil {
		return nil
	}
	e.yieldAfter = e.b.now().Add(e.b.interval)
	return e.promoteLocked(term)
}

// demote requires e.mu. Remove the leader only after its callback completes.
func (e *bucketElector) demote() bool {
	e.b.mu.Lock()
	_, leading := e.b.leaders[e.bkt.Name()]
	e.b.mu.Unlock()
	if leading {
		e.b.la.Demote(e.bkt)
		e.b.mu.Lock()
		delete(e.b.leaders, e.bkt.Name())
		e.b.mu.Unlock()
	}
	return leading
}

func (e *bucketElector) stopped() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.demote()
	e.term = nil
	e.b.unmark(e.bkt.Name())
}

func (e *bucketElector) yield(capacity int) *bucketTerm {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.b.now().Before(e.yieldAfter) {
		return nil
	}
	e.b.mu.Lock()
	excess := len(e.b.held) > capacity && len(e.b.held) > e.b.minHeld
	e.b.mu.Unlock()
	if excess && e.demote() {
		return e.term
	}
	return nil
}

// Run implements Elector.
func (e *bucketElector) Run(ctx context.Context) {
	for ctx.Err() == nil {
		bctx, cancel := context.WithCancel(ctx)
		e.mu.Lock()
		e.cancel = cancel
		e.mu.Unlock()
		e.le.Run(bctx)
		cancel()
	}
}

type balancedLock struct {
	resourcelock.Interface
	b    *balancer
	name string
}

func (l *balancedLock) Create(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	return l.acquire(ctx, ler, l.Interface.Create)
}

func (l *balancedLock) Update(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	return l.acquire(ctx, ler, l.Interface.Update)
}

// acquire passes renewals and releases of a held bucket through, and gates
// new acquisitions on capacity.
func (l *balancedLock) acquire(ctx context.Context, ler resourcelock.LeaderElectionRecord,
	write func(context.Context, resourcelock.LeaderElectionRecord) error,
) error {
	if l.b.isHeld(l.name) {
		return write(ctx, ler)
	}
	reserved, orphan, generation := l.b.reserve(ctx, l.name)
	if !reserved {
		return fmt.Errorf("bucket cap reached, not acquiring %s", l.name)
	}
	if err := write(ctx, ler); err != nil {
		l.b.unmark(l.name)
		return err
	}
	if orphan {
		l.b.mu.Lock()
		if l.b.generation == generation {
			l.b.minHeld = len(l.b.held)
		}
		l.b.mu.Unlock()
	}
	return nil
}
