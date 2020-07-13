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

package hash

import (
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	thisBucket  = "monsoon"
	knownKey    = "hagel"
	otherBucket = "chubasco"
	unknownKey  = "snow" // snow maps to "chubasco", originally.
)

var buckets = sets.NewString(thisBucket, otherBucket, "aguacero", "chaparrón")

func TestBucketOwner(t *testing.T) {
	b := NewBucketSet(buckets)
	if got := b.Owner(knownKey); got != thisBucket {
		t.Errorf("Owner = %q, want: %q", got, thisBucket)
	}
	if l := b.cache.Len(); l != 1 {
		t.Errorf("|Cache| = %d, want: 1", l)
	}
	if n, ok := b.cache.Get(knownKey); !ok || n.(string) != thisBucket {
		t.Errorf("Cache[%s] = %q, want: %q", knownKey, n, thisBucket)
	}
	// Verify nothing is added to the cache.
	if got := b.Owner(knownKey); got != thisBucket {
		t.Errorf("Owner = %q, want: %q", got, thisBucket)
	}
	if l := b.cache.Len(); l != 1 {
		t.Errorf("|Cache| = %d, want: 1", l)
	}

	if got := b.Owner(unknownKey); got != otherBucket {
		t.Errorf("Owner = %q, want: %q", got, otherBucket)
	}
	if l := b.cache.Len(); l != 2 {
		t.Errorf("|Cache| = %d, want: 2", l)
	}
	if n, ok := b.cache.Get(unknownKey); !ok || n.(string) != otherBucket {
		t.Errorf("Cache[%s] = %q, want: %q", unknownKey, n, otherBucket)
	}
}

func TestBucketUpdate(t *testing.T) {
	b := NewBucketSet(buckets)
	b.Owner(knownKey)

	// Need a clone.
	newNames := buckets.Difference(sets.NewString(otherBucket))
	b.Update(newNames)
	if b.cache.Len() != 0 {
		t.Error("cache was not emptied")
	}

	// Verify the mapping is stable.
	if got := b.Owner(knownKey); got != thisBucket {
		t.Errorf("Owner = %q, want: %q", got, thisBucket)
	}
	if l := b.cache.Len(); l != 1 {
		t.Errorf("|Cache| = %d, want: 1", l)
	}
	if n, ok := b.cache.Get(knownKey); !ok || n.(string) != thisBucket {
		t.Errorf("Cache[%s] = %q, want: %q", knownKey, n, thisBucket)
	}
	// unknownKey should've migrated.
	if got := b.Owner(unknownKey); got == otherBucket {
		t.Errorf("Owner = %q, don't want: %q", got, otherBucket)
	}
}

func TestBucketHas(t *testing.T) {
	bs := NewBucketSet(buckets)
	b := NewBucket(thisBucket, bs)
	thisNN := types.NamespacedName{Namespace: "snow", Name: "hail"}
	if !b.Has(thisNN) {
		t.Errorf("Has(%v) = false", thisNN)
	}
	b = NewBucket(otherBucket, bs)
	if b.Has(thisNN) {
		t.Errorf("Other bucket Has(%v) = true", thisNN)
	}
}

func TestBucketName(t *testing.T) {
	bs := NewBucketSet(buckets)
	b := NewBucket(thisBucket, bs)
	if got, want := b.Name(), thisBucket; got != want {
		t.Errorf("Name = %q, want: %q", got, want)
	}
}
