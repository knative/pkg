/*
Copyright 2020 The Knative Authors.

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
package testing

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	. "knative.dev/pkg/testing"
	"knative.dev/pkg/tracker"
)

func TestFakeTracker(t *testing.T) {
	t1 := &Resource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "foo",
		},
	}
	t1Name := types.NamespacedName{
		Namespace: t1.Namespace,
		Name:      t1.Name,
	}
	t2 := &Resource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "bar.baz.this-is-fine",
		},
	}
	t2Name := types.NamespacedName{
		Namespace: t2.Namespace,
		Name:      t2.Name,
	}

	obj1 := &Resource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "fakeapi",
			Kind:       "Fake",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "bar",
		},
	}
	ref1 := tracker.Reference{
		APIVersion: obj1.APIVersion,
		Kind:       obj1.Kind,
		Namespace:  obj1.Namespace,
		Name:       obj1.Name,
	}
	obj2 := &Resource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "fakeapi2",
			Kind:       "Fake2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo2",
			Name:      "bar2",
		},
	}
	ref2 := tracker.Reference{
		APIVersion: obj2.APIVersion,
		Kind:       obj2.Kind,
		Namespace:  obj2.Namespace,
		Name:       obj2.Name,
	}

	trk := &FakeTracker{}

	// Adding t1 to ref1 and then removing it results in ref1 stopping tracking.
	trk.TrackReference(ref1, t1)
	if !isTracking(trk, ref1) {
		t.Fatal("Tracker is not tracking", ref1)
	}
	if !hasObserver(trk, obj1, t1Name) {
		t.Fatalf("Object %v is not being observed by %v", obj1, t1Name)
	}

	trk.OnDeletedObserver(t1)
	if isTracking(trk, ref1) {
		t.Fatal("Tracker is still tracking", ref1)
	}
	if hasObserver(trk, obj1, t1Name) {
		t.Fatalf("Object %v is still being observed by %v", obj1, t1Name)
	}

	// Adding t1, t2 to ref1 and t2 to ref2, then removing t2 results in ref2 stopping
	// tracking.
	trk.TrackReference(ref1, t1)
	trk.TrackReference(ref1, t2)
	trk.TrackReference(ref2, t2)
	if !isTracking(trk, ref1) {
		t.Fatal("Tracker is not tracking", ref1)
	}
	if !isTracking(trk, ref2) {
		t.Fatal("Tracker is not tracking", ref2)
	}
	if !hasObserver(trk, obj1, t1Name) {
		t.Fatalf("Object %v is not being observed by %v", obj1, t1Name)
	}
	if !hasObserver(trk, obj1, t2Name) {
		t.Fatalf("Object %v is not being observed by %v", obj1, t2Name)
	}
	if hasObserver(trk, obj2, t1Name) {
		t.Fatalf("Object %v wrongly being observed by %v", obj2, t1Name)
	}
	if !hasObserver(trk, obj2, t2Name) {
		t.Fatalf("Object %v is not being observed by %v", obj2, t2Name)
	}

	trk.OnDeletedObserver(t2)
	if !isTracking(trk, ref1) {
		t.Fatal("Tracker is not tracking", ref1)
	}
	if !hasObserver(trk, obj1, t1Name) {
		t.Fatalf("Object %v is not being observed by %v", obj1, t1Name)
	}
	if isTracking(trk, ref2) {
		t.Fatal("Tracker is still tracking", ref2)
	}
	if hasObserver(trk, obj1, t2Name) {
		t.Fatalf("Object %v is still being observed by %v", obj1, t2Name)
	}
	if hasObserver(trk, obj2, t2Name) {
		t.Fatalf("Object %v is still being observed by %v", obj2, t2Name)
	}

	trk.OnDeletedObserver(t1)
	if isTracking(trk, ref1) {
		t.Fatal("Tracker is still tracking", ref1)
	}
	if hasObserver(trk, obj1, t1Name) {
		t.Fatalf("Object %v is still being observed by %v", obj1, t1Name)
	}
}

func isTracking(tracker *FakeTracker, ref1 tracker.Reference) bool {
	for _, tracking := range tracker.References() {
		if tracking == ref1 {
			return true
		}
	}
	return false
}

func hasObserver(tracker *FakeTracker, obj *Resource, obs types.NamespacedName) bool {
	observers := tracker.GetObservers(obj)
	for _, observer := range observers {
		if observer == obs {
			return true
		}
	}
	return false
}
