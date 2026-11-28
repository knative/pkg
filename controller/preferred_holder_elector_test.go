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
	"errors"
	"sync"
	"testing"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	election "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/utils/ptr"

	"knative.dev/pkg/leaderelection"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
	"knative.dev/pkg/test/ha"
)

// The coordinator precreates and assigns the Lease. An elector can renew its
// own lease while draining, but only the preferred replica can acquire it.
type preferredLock struct{ *resourcelock.LeaseLock }

func newPreferredLock(kc kubernetes.Interface, bucket, identity string) *preferredLock {
	return &preferredLock{&resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: bucket, Namespace: system.Namespace()},
		Client:     kc.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: identity},
	}}
}

func (*preferredLock) Create(context.Context, resourcelock.LeaderElectionRecord) error {
	return errors.New("waiting for the coordinator to create the lease")
}

func (l *preferredLock) Update(ctx context.Context, record resourcelock.LeaderElectionRecord) error {
	current, err := l.Client.Leases(l.LeaseMeta.Namespace).Get(ctx, l.LeaseMeta.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ptr.Deref(current.Spec.HolderIdentity, "") != l.Identity() &&
		(record.HolderIdentity == "" || current.Labels[preferredHolderLabel] != l.Identity()) {
		return errors.New("lease assigned to another candidate")
	}
	// LeaseLock writes its cached resourceVersion. A concurrent reassignment or
	// ownership change conflicts even if it occurs after the read above.
	return l.LeaseLock.Update(ctx, record)
}

func TestPreferredHolderAdmission(t *testing.T) {
	ctx := t.Context()
	kc := fakekube.NewSimpleClientset()
	ha.EnableLeaseResourceVersions(kc)
	leases := kc.CoordinationV1().Leases(system.Namespace())
	lease, err := leases.Create(ctx, &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{
		Name: "bucket", Labels: map[string]string{preferredHolderLabel: "b"},
	}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	a, b := newPreferredLock(kc, lease.Name, "a"), newPreferredLock(kc, lease.Name, "b")
	for _, lock := range []*preferredLock{a, b} {
		if _, _, err := lock.Get(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Update(ctx, resourcelock.LeaderElectionRecord{HolderIdentity: "a"}); err == nil {
		t.Fatal("A acquired a vacant lease assigned to B")
	}
	lease.Labels[preferredHolderLabel] = "a"
	if _, err := leases.Update(ctx, lease, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := b.Update(ctx, resourcelock.LeaderElectionRecord{HolderIdentity: "b"}); err == nil {
		t.Fatal("B acquired the lease after it was reassigned to A")
	}
}

type prototypeElector func(context.Context)

func (e prototypeElector) Run(ctx context.Context) { e(ctx) }

func preferredHolderElector(kc kubernetes.Interface, cc leaderelection.ComponentConfig, bkt reconciler.Bucket,
	la reconciler.LeaderAware, enq func(reconciler.Bucket, types.NamespacedName),
) (leaderelection.Elector, error) {
	drainer, ok := la.(reconciler.BucketDrainer)
	if !ok {
		return nil, errors.New("handoff requires a bucket drainer")
	}
	lock := newPreferredLock(kc, bkt.Name(), cc.Identity)
	return prototypeElector(func(ctx context.Context) {
		for ctx.Err() == nil {
			term, release := context.WithCancel(ctx)
			var callback sync.Once
			runLeader := func(leader context.Context) {
				if leader.Err() != nil {
					return
				}
				if err := la.Promote(bkt, enq); err != nil {
					la.Demote(bkt)
					release()
					return
				}
				err := wait.PollUntilContextCancel(leader, cc.RetryPeriod, true, func(ctx context.Context) (bool, error) {
					l, err := kc.CoordinationV1().Leases(system.Namespace()).Get(ctx, bkt.Name(), metav1.GetOptions{})
					if err == nil {
						preferred := l.Labels[preferredHolderLabel]
						return preferred != "" && preferred != cc.Identity, nil
					}
					return false, nil // Retry while the election term is live.
				})
				la.Demote(bkt)
				if err == nil && drainer.Drain(leader, bkt) == nil {
					release()
				}
			}
			election.RunOrDie(term, election.LeaderElectionConfig{
				Lock: lock, LeaseDuration: cc.LeaseDuration, RenewDeadline: cc.RenewDeadline,
				RetryPeriod: cc.RetryPeriod, ReleaseOnCancel: true,
				Callbacks: election.LeaderCallbacks{
					OnStartedLeading: func(leader context.Context) {
						callback.Do(func() { runLeader(leader) })
					},
					OnStoppedLeading: func() {
						// Join the canceled term's callback, or prevent a late callback
						// from promoting after the term has ended.
						callback.Do(func() {})
					},
				},
			})
			release()
		}
	}), nil
}
