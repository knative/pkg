/*
Copyright 2026 The Knative Authors.

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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgotesting "k8s.io/client-go/testing"
)

func TestDeleteCollectionKey(t *testing.T) {
	gvr := schema.GroupVersionResource{Resource: "pods"}

	fromListOptions := clientgotesting.NewDeleteCollectionAction(gvr, "foo", metav1.ListOptions{LabelSelector: "app=foo"})
	literalWithSelector := clientgotesting.DeleteCollectionActionImpl{
		ActionImpl: clientgotesting.ActionImpl{
			Namespace: "foo",
			Resource:  gvr,
		},
		ListRestrictions: clientgotesting.ListRestrictions{
			Labels: labels.SelectorFromSet(labels.Set{"app": "foo"}),
		},
	}
	literalWithoutSelector := clientgotesting.DeleteCollectionActionImpl{
		ActionImpl: clientgotesting.ActionImpl{
			Namespace: "foo",
			Resource:  gvr,
		},
	}
	otherNamespace := clientgotesting.NewDeleteCollectionAction(gvr, "bar", metav1.ListOptions{LabelSelector: "app=foo"})

	// An action built via the constructor (as the fake clientset produces) and an
	// equivalent hand-built literal (as a test author would write in WantDeleteCollections)
	// must key identically, or a correct WantDeleteCollections would never match.
	if got, want := deleteCollectionKey(fromListOptions, false), deleteCollectionKey(literalWithSelector, false); got != want {
		t.Errorf("deleteCollectionKey() = %q, want %q", got, want)
	}

	// A DeleteCollection call with no selector at all (nil Labels/Fields) must not panic,
	// and must key the same as an explicit "everything" selector.
	everything := clientgotesting.NewDeleteCollectionAction(gvr, "foo", metav1.ListOptions{})
	if got, want := deleteCollectionKey(literalWithoutSelector, false), deleteCollectionKey(everything, false); got != want {
		t.Errorf("deleteCollectionKey() = %q, want %q", got, want)
	}

	// Differing namespaces must produce different keys unless namespace validation is skipped.
	if got := deleteCollectionKey(fromListOptions, false); got == deleteCollectionKey(otherNamespace, false) {
		t.Errorf("deleteCollectionKey() = %q, want distinct keys for different namespaces", got)
	}
	if got, want := deleteCollectionKey(fromListOptions, true), deleteCollectionKey(otherNamespace, true); got != want {
		t.Errorf("deleteCollectionKey() = %q, want %q when SkipNamespaceValidation is set", got, want)
	}
}
