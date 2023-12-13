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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	thisBucket  = "monsoon"
	knownKey    = "hagel"
	otherBucket = "chubasco"
	unknownKey  = "snow" // snow maps to "chubasco", originally.
)

var buckets = sets.New[string](thisBucket, otherBucket, "aguacero", "chaparrón")

func TestBucketSetOwner(t *testing.T) {
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

func TestBucketSetList(t *testing.T) {
	bs := NewBucketSet(buckets)

	got := bs.BucketList()
	if want := sets.List(buckets); !reflect.DeepEqual(got, want) {
		t.Errorf("Name = %q, want: %q, diff(-want,+got):\n%s", got, want, cmp.Diff(want, got))
	}
}

func TestBucketSetUpdate(t *testing.T) {
	b := NewBucketSet(buckets)
	b.Owner(knownKey)

	// Need a clone.
	newNames := buckets.Difference(sets.New[string](otherBucket))
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

func TestBucketSetBuckets(t *testing.T) {
	bs := NewBucketSet(buckets)
	bkts := bs.Buckets()

	// Sorted
	want := []string{"aguacero", "chaparrón", "chubasco", "monsoon"}
	got := make([]string, len(bkts))
	for i, b := range bkts {
		got[i] = b.Name()
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got %q, want %q, diff(-want,+got):\n%s", got, want, cmp.Diff(want, got))
	}
}

func TestBucketSetHasBucket(t *testing.T) {
	bs := NewBucketSet(sets.New[string](thisBucket, "aguacero", "chaparrón"))
	if !bs.HasBucket(thisBucket) {
		t.Errorf("HasBucket(%v) = false", thisBucket)
	}
	if bs.HasBucket(otherBucket) {
		t.Errorf("HasBucket(%v) = true", otherBucket)
	}
}

func TestBucketHas(t *testing.T) {
	bs := NewBucketSet(buckets)
	b := Bucket{
		name:    thisBucket,
		buckets: bs,
	}
	thisNN := types.NamespacedName{Namespace: "snow", Name: "hail"}
	if !b.Has(thisNN) {
		t.Errorf("Has(%v) = false", thisNN)
	}
	b = Bucket{
		name:    otherBucket,
		buckets: bs,
	}
	if b.Has(thisNN) {
		t.Errorf("Other bucket Has(%v) = true", thisNN)
	}
}

func TestBucketName(t *testing.T) {
	bs := NewBucketSet(buckets)
	b := Bucket{
		name:    thisBucket,
		buckets: bs,
	}
	if got, want := b.Name(), thisBucket; got != want {
		t.Errorf("Name = %q, want: %q", got, want)
	}
}
