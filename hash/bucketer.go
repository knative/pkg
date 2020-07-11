/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains the utilities to make bucketing decisions.

package hash

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/reconciler"
)

var _ reconciler.Bucket = (*BucketSet)(nil)

// BucketSet answers to what bucket does key X belong in a
// consistent manner (consistent as in consistent hashing).
// In addition Bucket implements reconciler.Bucket interface, so it
// can be used both in leader election and in routing applications.
type BucketSet struct {
	// The name of this bucket. Required to use `Has`.
	name string
	// Stores the cached lookups. cache is internally thread safe.
	cache *lru.Cache

	// mu guards buckets.
	mu sync.RWMutex
	// All the bucket names. Needed for building hash universe.
	// `name` must be in this set.
	buckets sets.String
}

// Scientifically inferred preferred cache size.
const cacheSize = 4096

func newCache() *lru.Cache {
	c, _ := lru.New(cacheSize)
	return c
}

// NewBucketSet creates a new bucket set with the given name.
func NewBucketSet(name string, bucketList sets.String) *BucketSet {
	return &BucketSet{
		name:    name,
		cache:   newCache(),
		buckets: bucketList,
	}
}

// Name implements Bucket.
func (b *BucketSet) Name() string {
	return b.name
}

// Has returns true if this bucket owns the key and
// implements reconciler.Bucket interface.
func (b *BucketSet) Has(nn types.NamespacedName) bool {
	return b.Owner(nn.String()) == b.name
}

// Owner returns the owner of the key.
// Owner will cache the results for faster lookup.
func (b *BucketSet) Owner(key string) string {
	if v, ok := b.cache.Get(key); ok {
		return v.(string)
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	l := ChooseSubset(b.buckets, 1 /*single query wanted*/, key)
	ret := l.UnsortedList()[0]
	b.cache.Add(key, ret)
	return ret
}

// Update updates the universe of buckets.
func (b *BucketSet) Update(newB sets.String) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// In theory we can iterate over the map and
	// purge only the keys that moved to a new shard.
	// But this might be more expensive than re-build
	// the cache as reconciliations happen.
	b.cache.Purge()
	b.buckets = newB
}
