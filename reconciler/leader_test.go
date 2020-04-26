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
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"
)

func TestLeaderAwareFuncs(t *testing.T) {

	laf := LeaderAwareFuncs{}

	wantBkt := AllBuckets()

	wantKey := types.NamespacedName{
		Namespace: "foo",
		Name:      "bar",
	}
	called := false
	wantFunc := func(gotBkt Bucket, gotKey types.NamespacedName) {
		called = true
		if !cmp.Equal(gotKey, wantKey) {
			t.Errorf("key (-want, +got) = %s", cmp.Diff(wantKey, gotKey))
		}
		if !cmp.Equal(gotBkt, wantBkt) {
			t.Errorf("bucket (-want, +got) = %s", cmp.Diff(wantBkt, gotBkt))
		}
	}

	laf.PromoteFunc = func(bkt Bucket, gotFunc func(Bucket, types.NamespacedName)) {
		gotFunc(bkt, wantKey)
		if !called {
			t.Error("gotFunc didn't call wantFunc!")
		}

		// Check that we're not called while the lock is held,
		// and that we are already the Leader.
		if !laf.IsLeader(wantKey) {
			t.Error("IsLeader() = false, wanted true")
		}
	}
	laf.DemoteFunc = func(bkt Bucket) {
		// Check that we're not called while the lock is held,
		// and that we are no longer leader.
		if laf.IsLeader(wantKey) {
			t.Error("IsLeader() = true, wanted false")
		}
	}

	// We don't start as leader.
	if laf.IsLeader(wantKey) {
		t.Error("IsLeader() = true, wanted false")
	}

	// After Promote we are leader.
	laf.Promote(wantBkt, wantFunc)
	if !laf.IsLeader(wantKey) {
		t.Error("IsLeader() = false, wanted true")
	}

	// After Demote we are not leader.
	laf.Demote(wantBkt)
	if laf.IsLeader(wantKey) {
		t.Error("IsLeader() = true, wanted false")
	}
}
