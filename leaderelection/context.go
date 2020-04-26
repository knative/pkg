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
	"errors"
	"fmt"
	"hash/fnv"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
)

// WithLeaderElectorBuilder infuses a context with the ability to build
// LeaderElectors with the provided component configuration acquiring resource
// locks via the provided kubernetes client.
func WithLeaderElectorBuilder(ctx context.Context, kc kubernetes.Interface, cc ComponentConfig) context.Context {
	return context.WithValue(ctx, builderKey{}, &builder{
		kc:  kc,
		lec: cc,
	})
}

// HasLeaderElection returns whether there is leader election configuration
// associated with the context
func HasLeaderElection(ctx context.Context) bool {
	val := ctx.Value(builderKey{})
	return val != nil
}

// BuildElector builds a leaderelection.LeaderElector for the named LeaderAware
// reconciler using a builder added to the context via WithLeaderElectorBuilder.
func BuildElector(ctx context.Context, la reconciler.LeaderAware, name string, enq func(reconciler.Bucket, types.NamespacedName)) ([]*leaderelection.LeaderElector, error) {
	val := ctx.Value(builderKey{})
	if val == nil {
		return nil, errors.New("Builder not found")
	}
	return val.(*builder).BuildElector(ctx, la, name, enq)
}

type builderKey struct{}

type builder struct {
	kc  kubernetes.Interface
	lec ComponentConfig
}

func (b *builder) BuildElector(ctx context.Context, la reconciler.LeaderAware, name string, enq func(reconciler.Bucket, types.NamespacedName)) ([]*leaderelection.LeaderElector, error) {
	logger := logging.FromContext(ctx)

	id, err := UniqueID()
	if err != nil {
		return nil, err
	}

	// TODO(mattmoor): Extract this from b.lec for this name?
	const count uint32 = 1

	buckets := make([]*leaderelection.LeaderElector, 0, count)
	for i := uint32(0); i < count; i++ {
		bkt := &bucket{
			component: b.lec.Component,
			name:      name,
			index:     i,
			total:     count,
		}

		rl, err := resourcelock.New(b.lec.ResourceLock,
			system.Namespace(), // use namespace we are running in
			bkt.String(),
			b.kc.CoreV1(),
			b.kc.CoordinationV1(),
			resourcelock.ResourceLockConfig{
				Identity: id,
			})
		if err != nil {
			return nil, err
		}
		logger.Infof("%s will run in leader-elected mode with id %q", bkt.String(), rl.Identity())

		le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: b.lec.LeaseDuration,
			RenewDeadline: b.lec.RenewDeadline,
			RetryPeriod:   b.lec.RetryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(context.Context) {
					logger.Infof("%q has started leading %q", rl.Identity(), bkt.String())
					la.Promote(bkt, enq)
				},
				OnStoppedLeading: func() {
					logger.Infof("%q has stopped leading %q", rl.Identity(), bkt.String())
					la.Demote(bkt)
				},
			},
			ReleaseOnCancel: true,

			// TODO: use health check watchdog, knative/pkg#1048
			Name: b.lec.Component,
		})
		if err != nil {
			return nil, err
		}
		// TODO: use health check watchdog, knative/pkg#1048
		// if lec.WatchDog != nil {
		// 	lec.WatchDog.SetLeaderElection(le)
		// }
		buckets = append(buckets, le)
	}
	return buckets, nil
}

type bucket struct {
	component string
	name      string

	// We are bucket {index} of {total}
	index uint32
	total uint32
}

var _ reconciler.Bucket = (*bucket)(nil)

// String implements reconciler.Bucket
func (b *bucket) String() string {
	// The resource name is the lowercase:
	//   {component}.{workqueue}.{index}-of-{total}
	return strings.ToLower(fmt.Sprintf("%s.%s.%02d-of-%02d", b.component, b.name, b.index, b.total))
}

// Has implements reconciler.Bucket
func (b *bucket) Has(nn types.NamespacedName) bool {
	h := fnv.New32a()
	h.Write([]byte(nn.Namespace + "." + nn.Name))
	ii := h.Sum32() % b.total
	return b.index == ii
}
