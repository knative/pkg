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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

	"knative.dev/pkg/hash"
	"knative.dev/pkg/leaderelection"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
	"knative.dev/pkg/test/ha"
)

const (
	deploymentLabel      = "prototype.knative.dev/deployment"
	preferredHolderLabel = "prototype.knative.dev/preferred-holder"
)

// TestPreferredHolderHandoff runs the same long reconcile as the local balancer
// tests, with placement driven by a Deployment and ordinary Lease labels.
func TestPreferredHolderHandoff(t *testing.T) {
	testRebalanceInflight(t, preferredHolderHandoff)
}

func setupPreferredHolderPrototype(t *testing.T, ctx context.Context, kc *fakekube.Clientset, cc leaderelection.ComponentConfig) (func(string) context.Context, func()) {
	t.Helper()
	ha.EnableLeaseResourceVersions(kc)
	ns := system.Namespace()
	// Desired replicas are two throughout; only A is Ready at startup.
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "ctrl", Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](2),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "ctrl"}},
		},
	}
	if _, err := kc.AppsV1().Deployments(ns).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	names := sets.New[string]()
	for i := range cc.Buckets {
		name := cc.LeaseName(i)
		names.Insert(name)
		_, err := kc.CoordinationV1().Leases(ns).Create(ctx, &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: map[string]string{deploymentLabel: deployment.Name}},
		}, metav1.CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
	}
	buckets := hash.NewBucketSet(names).Buckets()
	replicaContext := func(id string) context.Context {
		cfg := cc
		cfg.Identity = id // This deployment uses the Pod UID as the elector identity.
		return leaderelection.WithLeaderElectorBuilder(ctx, func(_ context.Context, la reconciler.LeaderAware, _ string, enq func(reconciler.Bucket, types.NamespacedName)) (leaderelection.Elector, error) {
			var electors []leaderelection.Elector
			for _, bkt := range buckets {
				el, err := preferredHolderElector(kc, cfg, bkt, la, enq)
				if err != nil {
					return nil, err
				}
				electors = append(electors, el)
			}
			return prototypeElector(func(ctx context.Context) {
				var wg sync.WaitGroup
				for _, el := range electors {
					wg.Go(func() { el.Run(ctx) })
				}
				wg.Wait()
			}), nil
		})
	}
	// Drive the coordinator once at startup and once when B becomes Ready.
	// Electors learn each assignment exclusively by reading the Lease.
	return replicaContext, func() {
		t.Helper()
		if err := assignPreferredHolders(ctx, kc, cc.LeaseName(0)); err != nil {
			t.Fatal(err)
		}
	}
}
