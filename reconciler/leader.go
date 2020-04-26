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

package reconciler

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

// Bucket is an opaque type used to scope leadership.
type Bucket interface {
	// String returns a string representing this bucket, which uniquely
	// identifies the bucket and is suitable for use as a resource lock name.
	String() string

	// Has determines whether this Bucket covers a particular key.
	Has(key types.NamespacedName) bool
}

// AllBuckets returns a Bucket that "Has()" all keys.
func AllBuckets() Bucket {
	return &bucket{}
}

// LeaderAware is implemented by Reconcilers that are aware of their leader status.
type LeaderAware interface {
	// IsLeader returns the leader status for the specified Bucket
	IsLeader(types.NamespacedName) bool

	// Promote is called when we become the leader of a given Bucket.  It must be
	// supplied with an enqueue function through which a Bucket resync may be triggered.
	Promote(b Bucket, enq func(Bucket, types.NamespacedName))

	// Demote is called when we stop being the leader for the specified Bucket.
	Demote(Bucket)
}

// LeaderAwareFuncs implements LeaderAware using the given functions for handling
// promotion and demotion.
type LeaderAwareFuncs struct {
	sync.RWMutex
	buckets map[string]Bucket

	PromoteFunc func(b Bucket, enq func(Bucket, types.NamespacedName))
	DemoteFunc  func(b Bucket)
}

var _ LeaderAware = (*LeaderAwareFuncs)(nil)

// IsLeader implements LeaderAware
func (laf *LeaderAwareFuncs) IsLeader(key types.NamespacedName) bool {
	laf.RLock()
	defer laf.RUnlock()

	for _, bkt := range laf.buckets {
		if bkt.Has(key) {
			return true
		}
	}
	return false
}

// Promote implements LeaderAware
func (laf *LeaderAwareFuncs) Promote(b Bucket, enq func(Bucket, types.NamespacedName)) {
	promote := func() func(Bucket, func(Bucket, types.NamespacedName)) {
		laf.Lock()
		defer laf.Unlock()
		if laf.buckets == nil {
			laf.buckets = make(map[string]Bucket, 1)
		}
		laf.buckets[b.String()] = b
		return laf.PromoteFunc
	}()

	if promote != nil {
		promote(b, enq)
	}
}

// Demote implements LeaderAware
func (laf *LeaderAwareFuncs) Demote(b Bucket) {
	demote := func() func(Bucket) {
		laf.Lock()
		defer laf.Unlock()
		delete(laf.buckets, b.String())
		return laf.DemoteFunc
	}()

	if demote != nil {
		demote(b)
	}
}

type bucket struct{}

var _ Bucket = (*bucket)(nil)

// String implements Bucket
func (b *bucket) String() string {
	return ""
}

// Has implements Bucket
func (b *bucket) Has(nn types.NamespacedName) bool {
	return true
}
