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
	pods := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	selectors := metav1.ListOptions{LabelSelector: "app=test,tier=web", FieldSelector: "status.phase=Running"}
	first := clientgotesting.NewDeleteCollectionAction(pods, "ns", selectors)
	second := clientgotesting.NewDeleteCollectionAction(pods, "ns", metav1.ListOptions{LabelSelector: "other=true"})
	otherNamespace := clientgotesting.NewDeleteCollectionAction(pods, "other", selectors)
	otherResource := clientgotesting.NewDeleteCollectionAction(schema.GroupVersionResource{Group: "example.org", Version: "v1", Resource: "pods"}, "ns", selectors)
	otherFields := selectors
	otherFields.FieldSelector = "status.phase=Pending"
	reordered := selectors
	reordered.LabelSelector = "tier=web,app=test"
	limited := selectors
	limited.Limit = 10
	dryRun := clientgotesting.NewDeleteCollectionActionWithOptions(pods, "ns", metav1.DeleteOptions{DryRun: []string{metav1.DryRunAll}}, selectors)

	subresource := first
	subresource.Subresource = "status"
	literal := clientgotesting.DeleteCollectionActionImpl{ActionImpl: first.ActionImpl, ListOptions: reordered}
	selectorOnly := clientgotesting.DeleteCollectionActionImpl{ListRestrictions: first.ListRestrictions}
	root := clientgotesting.NewRootDeleteCollectionAction(pods, metav1.ListOptions{})
	noLabels := root
	noLabels.ListRestrictions.Labels = labels.Nothing()
	noFields := root
	noFields.ListRestrictions.Fields = fields.Nothing()

	cases := []struct {
		name                   string
		want, got              calls
		skipNamespace, matches bool
	}{{
		name:    "empty",
		matches: true,
	}, {
		name:    "match",
		want:    calls{first, second},
		got:     calls{first, second},
		matches: true,
	}, {
		name: "missing",
		want: calls{first},
	}, {
		name: "extra",
		got:  calls{first},
	}, {
		name: "duplicate",
		want: calls{first},
		got:  calls{first, first},
	}, {
		name: "order",
		want: calls{first, second},
		got:  calls{second, first},
	}, {
		name: "resource",
		want: calls{first},
		got:  calls{otherResource},
	}, {
		name: "namespace",
		want: calls{first},
		got:  calls{otherNamespace},
	}, {
		name:          "skip_namespace",
		want:          calls{first},
		got:           calls{otherNamespace},
		skipNamespace: true,
		matches:       true,
	}, {
		name: "labels",
		want: calls{first},
		got:  calls{clientgotesting.NewDeleteCollectionAction(pods, "ns", metav1.ListOptions{FieldSelector: selectors.FieldSelector})},
	}, {
		name: "fields",
		want: calls{first},
		got:  calls{clientgotesting.NewDeleteCollectionAction(pods, "ns", otherFields)},
	}, {
		name:    "equivalent_labels",
		want:    calls{first},
		got:     calls{clientgotesting.NewDeleteCollectionAction(pods, "ns", reordered)},
		matches: true,
	}, {
		name: "list_options",
		want: calls{first},
		got:  calls{clientgotesting.NewDeleteCollectionAction(pods, "ns", limited)},
	}, {
		name: "delete_options",
		want: calls{first},
		got:  calls{dryRun},
	}, {
		name: "subresource",
		want: calls{first},
		got:  calls{subresource},
	}, {
		name:    "list_options_literal",
		want:    calls{literal},
		got:     calls{first},
		matches: true,
	}, {
		name:    "selector_only",
		want:    calls{selectorOnly},
		got:     calls{first},
		matches: true,
	}, {
		name: "selector_only_wrong_namespace",
		want: calls{selectorOnly},
		got:  calls{otherNamespace},
	}, {
		name:          "selector_only_skip_namespace",
		want:          calls{selectorOnly},
		got:           calls{otherNamespace},
		skipNamespace: true,
		matches:       true,
	}, {
		name: "selector_only_wrong_labels",
		want: calls{selectorOnly},
		got:  calls{second},
	}, {
		name: "mixed_expectations_wrong_resource",
		want: calls{selectorOnly, first},
		got:  calls{first, otherResource},
	}, {
		name:    "cluster_scoped",
		want:    calls{root},
		got:     calls{root},
		matches: true,
	}, {
		name: "labels_match_nothing",
		want: calls{noLabels},
		got:  calls{root},
	}, {
		name: "fields_match_nothing",
		want: calls{noFields},
		got:  calls{root},
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := TableRow{Key: "ns/test", WantDeleteCollections: tc.want, SkipNamespaceValidation: tc.skipNamespace}
			rec := &collectionReconciler{calls: tc.got}
			recorders := ActionRecorderList{rec}
			events := EventList{Recorder: record.NewFakeRecorder(1)}
			if tc.matches {
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
