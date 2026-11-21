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

package testing

import (
	"context"
	"fmt"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgotesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

	"knative.dev/pkg/controller"
)

type collectionReconciler struct {
	clientgotesting.Fake
	calls []clientgotesting.DeleteCollectionActionImpl
}

func (r *collectionReconciler) Reconcile(context.Context, string) error {
	for _, call := range r.calls {
		if _, err := r.Invokes(call, nil); err != nil {
			return err
		}
	}
	return nil
}

type rowErrors struct {
	*testing.T
	messages []string
}

func (r *rowErrors) Errorf(format string, args ...any) {
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

func TestTableRowDeleteCollections(t *testing.T) {
	type calls = []clientgotesting.DeleteCollectionActionImpl
	first := collectionDeleteAction()
	second := collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
		a.ListOptions = metav1.ListOptions{LabelSelector: "other=true"}
	})
	otherNamespace := collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
		a.Namespace = "other"
	})
	otherResource := collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
		a.Resource.Group = "example.org"
	})
	selectorOnly := clientgotesting.DeleteCollectionActionImpl{ListRestrictions: first.ListRestrictions}
	root := clientgotesting.NewRootDeleteCollectionAction(first.Resource, metav1.ListOptions{})

	cases := []struct {
		name          string
		want, got     calls
		skipNamespace bool
		wantMismatch  bool
	}{{
		name: "no expected or recorded collection deletes",
	}, {
		name: "collection deletes match expectations in order",
		want: calls{first, second},
		got:  calls{first, second},
	}, {
		name:         "missing expected collection delete is rejected",
		want:         calls{first},
		wantMismatch: true,
	}, {
		name:         "unexpected collection delete is rejected",
		got:          calls{first},
		wantMismatch: true,
	}, {
		name:         "duplicate collection delete is rejected",
		want:         calls{first},
		got:          calls{first, first},
		wantMismatch: true,
	}, {
		name:         "collection deletes in the wrong order are rejected",
		want:         calls{first, second},
		got:          calls{second, first},
		wantMismatch: true,
	}, {
		name:         "collection delete with the wrong API group is rejected",
		want:         calls{first},
		got:          calls{otherResource},
		wantMismatch: true,
	}, {
		name:         "collection delete in the wrong namespace is rejected",
		want:         calls{first},
		got:          calls{otherNamespace},
		wantMismatch: true,
	}, {
		name:          "namespace mismatch is allowed when namespace validation is skipped",
		want:          calls{first},
		got:           calls{otherNamespace},
		skipNamespace: true,
	}, {
		name: "collection delete without the expected label selector is rejected",
		want: calls{first},
		got: calls{collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
			a.ListOptions.LabelSelector = ""
		})},
		wantMismatch: true,
	}, {
		name: "collection delete with a different field selector is rejected",
		want: calls{first},
		got: calls{collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
			a.ListOptions.FieldSelector = "status.phase=Pending"
		})},
		wantMismatch: true,
	}, {
		name: "equivalent label selectors with reordered requirements match",
		want: calls{first},
		got: calls{collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
			a.ListOptions.LabelSelector = "tier=web,app=test"
		})},
	}, {
		name: "collection delete with an unexpected list limit is rejected",
		want: calls{first},
		got: calls{collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
			a.ListOptions.Limit = 10
		})},
		wantMismatch: true,
	}, {
		name: "collection delete with unexpected dry run options is rejected",
		want: calls{first},
		got: calls{collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
			a.DeleteOptions.DryRun = []string{metav1.DryRunAll}
		})},
		wantMismatch: true,
	}, {
		name: "collection delete with an unexpected subresource is rejected",
		want: calls{first},
		got: calls{collectionDeleteAction(func(a *clientgotesting.DeleteCollectionActionImpl) {
			a.Subresource = "status"
		})},
		wantMismatch: true,
	}, {
		name: "selectors supplied only through list options match recorded restrictions",
		want: calls{{
			ActionImpl: first.ActionImpl,
			ListOptions: metav1.ListOptions{
				LabelSelector: "tier=web,app=test",
				FieldSelector: "status.phase=Running",
			},
		}},
		got: calls{first},
	}, {
		name: "selector-only expectation matches a collection delete in the row namespace",
		want: calls{selectorOnly},
		got:  calls{first},
	}, {
		name:         "selector-only expectation rejects a collection delete outside the row namespace",
		want:         calls{selectorOnly},
		got:          calls{otherNamespace},
		wantMismatch: true,
	}, {
		name:          "selector-only expectation allows another namespace when namespace validation is skipped",
		want:          calls{selectorOnly},
		got:           calls{otherNamespace},
		skipNamespace: true,
	}, {
		name:         "selector-only expectation rejects a different label selector",
		want:         calls{selectorOnly},
		got:          calls{second},
		wantMismatch: true,
	}, {
		name:         "selector-only expectation does not disable resource validation for other expectations",
		want:         calls{selectorOnly, first},
		got:          calls{first, otherResource},
		wantMismatch: true,
	}, {
		name: "cluster-scoped collection delete matches an explicit root action expectation",
		want: calls{root},
		got:  calls{root},
	}, {
		name: "label selector matching nothing does not match a selector matching everything",
		want: calls{{
			ActionImpl: root.ActionImpl,
			ListRestrictions: clientgotesting.ListRestrictions{
				Labels: labels.Nothing(),
				Fields: root.ListRestrictions.Fields,
			},
		}},
		got:          calls{root},
		wantMismatch: true,
	}, {
		name: "field selector matching nothing does not match a selector matching everything",
		want: calls{{
			ActionImpl: root.ActionImpl,
			ListRestrictions: clientgotesting.ListRestrictions{
				Labels: root.ListRestrictions.Labels,
				Fields: fields.Nothing(),
			},
		}},
		got:          calls{root},
		wantMismatch: true,
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := TableRow{
				Key:                     "ns/test",
				WantDeleteCollections:   tc.want,
				SkipNamespaceValidation: tc.skipNamespace,
			}
			rec := &collectionReconciler{calls: tc.got}
			recorders := ActionRecorderList{rec}
			events := EventList{Recorder: record.NewFakeRecorder(1)}
			if !tc.wantMismatch {
				row.Test(t, func(*testing.T, *TableRow) (controller.Reconciler, ActionRecorderList, EventList) {
					return rec, recorders, events
				})
				return
			}

			// Capture the row's expected diagnostic without failing the enclosing test.
			errs := &rowErrors{T: t}
			row.run(errs, rec, recorders, events)
			if len(errs.messages) != 1 || !strings.HasPrefix(errs.messages[0], "Unexpected DeleteCollection calls") {
				t.Errorf("expected a DeleteCollection mismatch, got %q", errs.messages)
			}
		})
	}
}

// collectionDeleteAction constructs a namespaced pod collection delete with
// label and field selectors. Apply overrides before constructing the action so
// client-go keeps ListOptions and ListRestrictions consistent.
func collectionDeleteAction(overrides ...func(*clientgotesting.DeleteCollectionActionImpl)) clientgotesting.DeleteCollectionActionImpl {
	a := clientgotesting.DeleteCollectionActionImpl{
		ActionImpl: clientgotesting.ActionImpl{
			Resource:  schema.GroupVersionResource{Version: "v1", Resource: "pods"},
			Namespace: "ns",
		},
		ListOptions: metav1.ListOptions{
			LabelSelector: "app=test,tier=web",
			FieldSelector: "status.phase=Running",
		},
	}
	for _, override := range overrides {
		override(&a)
	}
	call := clientgotesting.NewDeleteCollectionActionWithOptions(a.Resource, a.Namespace, a.DeleteOptions, a.ListOptions)
	call.Subresource = a.Subresource
	return call
}
