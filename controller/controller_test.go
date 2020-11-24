/*
Copyright 2017 The Knative Authors

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

package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/atomic"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"knative.dev/pkg/leaderelection"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"

	. "knative.dev/pkg/controller/testing"
	. "knative.dev/pkg/logging/testing"
	_ "knative.dev/pkg/system/testing"
	. "knative.dev/pkg/testing"
)

const (
	oldObj = "foo"
	newObj = "bar"
)

func TestPassNew(t *testing.T) {
	PassNew(func(got interface{}) {
		if newObj != got.(string) {
			t.Errorf("PassNew() = %v, wanted %v", got, newObj)
		}
	})(oldObj, newObj)
}

func TestHandleAll(t *testing.T) {
	ha := HandleAll(func(got interface{}) {
		if newObj != got.(string) {
			t.Errorf("HandleAll() = %v, wanted %v", got, newObj)
		}
	})

	ha.OnAdd(newObj)
	ha.OnUpdate(oldObj, newObj)
	ha.OnDelete(newObj)
}

var gvk = schema.GroupVersionKind{
	Group:   "pkg.knative.dev",
	Version: "v1meta1",
	Kind:    "Parent",
}

func TestFilterWithNameAndNamespace(t *testing.T) {
	filter := FilterWithNameAndNamespace("test-namespace", "test-name")

	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{{
		name:  "not a metav1.Object",
		input: "foo",
	}, {
		name:  "nil",
		input: nil,
	}, {
		name: "name matches, namespace does not",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "wrong-namespace",
			},
		},
	}, {
		name: "namespace matches, name does not",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wrong-name",
				Namespace: "test-namespace",
			},
		},
	}, {
		name: "neither matches",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wrong-name",
				Namespace: "wrong-namespace",
			},
		},
	}, {
		name: "matches",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		},
		want: true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := filter(test.input)
			if test.want != got {
				t.Errorf("FilterWithNameAndNamespace() = %v, wanted %v", got, test.want)
			}
		})
	}
}

func TestFilterWithName(t *testing.T) {
	filter := FilterWithName("test-name")

	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{{
		name:  "not a metav1.Object",
		input: "foo",
	}, {
		name:  "nil",
		input: nil,
	}, {
		name: "name matches, namespace does not",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "wrong-namespace",
			},
		},
		want: true, // Unlike FilterWithNameAndNamespace this passes
	}, {
		name: "namespace matches, name does not",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wrong-name",
				Namespace: "test-namespace",
			},
		},
	}, {
		name: "neither matches",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wrong-name",
				Namespace: "wrong-namespace",
			},
		},
	}, {
		name: "matches",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		},
		want: true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := filter(test.input)
			if test.want != got {
				t.Errorf("FilterWithNameAndNamespace() = %v, wanted %v", got, test.want)
			}
		})
	}
}

func TestFilterGroupKind(t *testing.T) {
	filter := FilterGroupKind(gvk.GroupKind())

	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{{
		name:  "not a metav1.Object",
		input: "foo",
	}, {
		name:  "nil",
		input: nil,
	}, {
		name: "no owner reference",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
		},
	}, {
		name: "wrong owner reference, not controller",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "another.knative.dev/v1beta3",
					Kind:       "Parent",
					Controller: ptr.Bool(false),
				}},
			},
		},
	}, {
		name: "right owner reference, not controller",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Controller: ptr.Bool(false),
				}},
			},
		},
	}, {
		name: "wrong owner reference, but controller",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "another.knative.dev/v1beta3",
					Kind:       "Parent",
					Controller: ptr.Bool(true),
				}},
			},
		},
		want: false,
	}, {
		name: "right owner reference, is controller",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Controller: ptr.Bool(true),
				}},
			},
		},
		want: true,
	}, {
		name: "right owner reference, is controller, different version",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: schema.GroupVersion{Group: gvk.Group, Version: "other"}.String(),
					Kind:       gvk.Kind,
					Controller: ptr.Bool(true),
				}},
			},
		},
		want: true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := filter(test.input)
			if test.want != got {
				t.Errorf("Filter() = %v, wanted %v", got, test.want)
			}
		})
	}
}

func TestFilterGroupVersionKind(t *testing.T) {
	filter := FilterGroupVersionKind(gvk)

	tests := []struct {
		name  string
		input interface{}
		want  bool
	}{{
		name:  "not a metav1.Object",
		input: "foo",
	}, {
		name:  "nil",
		input: nil,
	}, {
		name: "no owner reference",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
		},
	}, {
		name: "wrong owner reference, not controller",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "another.knative.dev/v1beta3",
					Kind:       "Parent",
					Controller: ptr.Bool(false),
				}},
			},
		},
	}, {
		name: "right owner reference, not controller",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Controller: ptr.Bool(false),
				}},
			},
		},
	}, {
		name: "wrong owner reference, but controller",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "another.knative.dev/v1beta3",
					Kind:       "Parent",
					Controller: ptr.Bool(true),
				}},
			},
		},
	}, {
		name: "right owner reference, is controller",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Controller: ptr.Bool(true),
				}},
			},
		},
		want: true,
	}, {
		name: "right owner reference, is controller, wrong version",
		input: &Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: schema.GroupVersion{Group: gvk.Group, Version: "other"}.String(),
					Kind:       gvk.Kind,
					Controller: ptr.Bool(true),
				}},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := filter(test.input)
			if test.want != got {
				t.Errorf("Filter() = %v, wanted %v", got, test.want)
			}
		})
	}
}

type nopReconciler struct{}

func (nr *nopReconciler) Reconcile(context.Context, string) error {
	return nil
}

type testRateLimiter struct {
	t     *testing.T
	delay time.Duration
}

func (t testRateLimiter) When(interface{}) time.Duration { return t.delay }
func (t testRateLimiter) Forget(interface{})             {}
func (t testRateLimiter) NumRequeues(interface{}) int    { return 0 }

var _ workqueue.RateLimiter = (*testRateLimiter)(nil)

func TestEnqueue(t *testing.T) {
	tests := []struct {
		name      string
		work      func(*Impl)
		wantQueue []types.NamespacedName
	}{{
		name: "do nothing",
		work: func(*Impl) {},
	}, {
		name: "enqueue key",
		work: func(impl *Impl) {
			impl.EnqueueKey(types.NamespacedName{Namespace: "foo", Name: "bar"})
		},
		wantQueue: []types.NamespacedName{{Namespace: "foo", Name: "bar"}},
	}, {
		name: "enqueue duplicate key",
		work: func(impl *Impl) {
			impl.EnqueueKey(types.NamespacedName{Namespace: "foo", Name: "bar"})
			impl.EnqueueKey(types.NamespacedName{Namespace: "foo", Name: "bar"})
		},
		// The queue deduplicates.
		wantQueue: []types.NamespacedName{{Namespace: "foo", Name: "bar"}},
	}, {
		name: "enqueue different keys",
		work: func(impl *Impl) {
			impl.EnqueueKey(types.NamespacedName{Namespace: "foo", Name: "bar"})
			impl.EnqueueKey(types.NamespacedName{Namespace: "foo", Name: "baz"})
		},
		wantQueue: []types.NamespacedName{{Namespace: "foo", Name: "bar"}, {Namespace: "foo", Name: "baz"}},
	}, {
		name: "enqueue resource",
		work: func(impl *Impl) {
			impl.Enqueue(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "bar", Name: "foo"}},
	}, {
		name: "enqueue resource slow",
		work: func(impl *Impl) {
			impl.EnqueueSlow(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "bar", Name: "foo"}},
	}, {
		name: "enqueue sentinel resource",
		work: func(impl *Impl) {
			e := impl.EnqueueSentinel(types.NamespacedName{Namespace: "foo", Name: "bar"})
			e(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "baz",
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "foo", Name: "bar"}},
	}, {
		name: "enqueue duplicate sentinel resource",
		work: func(impl *Impl) {
			e := impl.EnqueueSentinel(types.NamespacedName{Namespace: "foo", Name: "bar"})
			e(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "baz-1",
				},
			})
			e(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "foo",
					Name:      "baz-2",
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "foo", Name: "bar"}},
	}, {
		name: "enqueue bad resource",
		work: func(impl *Impl) {
			impl.Enqueue("baz/blah")
		},
	}, {
		name: "enqueue controller of bad resource",
		work: func(impl *Impl) {
			impl.EnqueueControllerOf("baz/blah")
		},
	}, {
		name: "enqueue controller of resource without owner",
		work: func(impl *Impl) {
			impl.EnqueueControllerOf(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			})
		},
	}, {
		name: "enqueue controller of resource with owner",
		work: func(impl *Impl) {
			impl.EnqueueControllerOf(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: gvk.GroupVersion().String(),
						Kind:       gvk.Kind,
						Name:       "baz",
						Controller: ptr.Bool(true),
					}},
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "bar", Name: "baz"}},
	}, {
		name: "enqueue controller of deleted resource with owner",
		work: func(impl *Impl) {
			impl.EnqueueControllerOf(cache.DeletedFinalStateUnknown{
				Key: "foo/bar",
				Obj: &Resource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						OwnerReferences: []metav1.OwnerReference{{
							APIVersion: gvk.GroupVersion().String(),
							Kind:       gvk.Kind,
							Name:       "baz",
							Controller: ptr.Bool(true),
						}},
					},
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "bar", Name: "baz"}},
	}, {
		name: "enqueue controller of deleted bad resource",
		work: func(impl *Impl) {
			impl.EnqueueControllerOf(cache.DeletedFinalStateUnknown{
				Key: "foo/bar",
				Obj: "bad-resource",
			})
		},
	}, {
		name: "enqueue label of namespaced resource bad resource",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfNamespaceScopedResource("test-ns", "test-name")("baz/blah")
		},
	}, {
		name: "enqueue label of namespaced resource without label",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfNamespaceScopedResource("ns-key", "name-key")(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Labels: map[string]string{
						"ns-key": "bar",
					},
				},
			})
		},
	}, {
		name: "enqueue label of namespaced resource without namespace label",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfNamespaceScopedResource("ns-key", "name-key")(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Labels: map[string]string{
						"name-key": "baz",
					},
				},
			})
		},
	}, {
		name: "enqueue label of namespaced resource with labels",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfNamespaceScopedResource("ns-key", "name-key")(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Labels: map[string]string{
						"ns-key":   "qux",
						"name-key": "baz",
					},
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "qux", Name: "baz"}},
	}, {
		name: "enqueue label of namespaced resource with empty namespace label",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfNamespaceScopedResource("", "name-key")(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Labels: map[string]string{
						"name-key": "baz",
					},
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "bar", Name: "baz"}},
	}, {
		name: "enqueue label of deleted namespaced resource with label",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfNamespaceScopedResource("ns-key", "name-key")(cache.DeletedFinalStateUnknown{
				Key: "foo/bar",
				Obj: &Resource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Labels: map[string]string{
							"ns-key":   "qux",
							"name-key": "baz",
						},
					},
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "qux", Name: "baz"}},
	}, {
		name: "enqueue label of deleted bad namespaced resource",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfNamespaceScopedResource("ns-key", "name-key")(cache.DeletedFinalStateUnknown{
				Key: "foo/bar",
				Obj: "bad-resource",
			})
		},
	}, {
		name: "enqueue label of cluster scoped resource bad resource",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfClusterScopedResource("name-key")("baz")
		},
	}, {
		name: "enqueue label of cluster scoped resource without label",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfClusterScopedResource("name-key")(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Labels:    map[string]string{},
				},
			})
		},
	}, {
		name: "enqueue label of cluster scoped resource with label",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfClusterScopedResource("name-key")(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
					Labels: map[string]string{
						"name-key": "baz",
					},
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "", Name: "baz"}},
	}, {
		name: "enqueue label of deleted cluster scoped resource with label",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfClusterScopedResource("name-key")(cache.DeletedFinalStateUnknown{
				Key: "foo/bar",
				Obj: &Resource{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
						Labels: map[string]string{
							"name-key": "baz",
						},
					},
				},
			})
		},
		wantQueue: []types.NamespacedName{{Namespace: "", Name: "baz"}},
	}, {
		name: "enqueue namespace of object",
		work: func(impl *Impl) {
			impl.EnqueueNamespaceOf(&Resource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				}})
		},
		wantQueue: []types.NamespacedName{{Name: "bar"}},
	}, {
		name: "enqueue label of deleted bad cluster scoped resource",
		work: func(impl *Impl) {
			impl.EnqueueLabelOfClusterScopedResource("name-key")(cache.DeletedFinalStateUnknown{
				Key: "bar",
				Obj: "bad-resource",
			})
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var rl workqueue.RateLimiter = testRateLimiter{t, 100 * time.Millisecond}
			impl := NewImplFull(&nopReconciler{}, ControllerOptions{WorkQueueName: "Testing", Logger: TestLogger(t), RateLimiter: rl})
			test.work(impl)

			impl.WorkQueue().ShutDown()
			gotQueue := drainWorkQueue(impl.WorkQueue())

			if diff := cmp.Diff(test.wantQueue, gotQueue); diff != "" {
				t.Error("unexpected queue (-want +got):", diff)
			}
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			impl := NewImplWithStats(&nopReconciler{}, TestLogger(t), "Testing", &FakeStatsReporter{})
			test.work(impl)

			impl.WorkQueue().ShutDown()
			gotQueue := drainWorkQueue(impl.WorkQueue())

			if diff := cmp.Diff(test.wantQueue, gotQueue); diff != "" {
				t.Error("unexpected queue (-want +got):", diff)
			}
		})
	}
}

const (
	// longDelay is longer than we expect the test to run.
	longDelay = time.Minute
	// shortDelay is short enough for the test to execute quickly, but long
	// enough to reasonably delay the enqueuing of an item.
	shortDelay = 50 * time.Millisecond

	// time we allow the queue length checker to keep polling the
	// workqueue.
	queueCheckTimeout = shortDelay + 500*time.Millisecond
)

func pollQ(q workqueue.RateLimitingInterface, sig chan int) func() (bool, error) {
	return func() (bool, error) {
		if ql := q.Len(); ql > 0 {
			sig <- ql
			return true, nil
		}
		return false, nil
	}
}

func TestEnqueueAfter(t *testing.T) {
	impl := NewImplWithStats(&nopReconciler{}, TestLogger(t), "Testing", &FakeStatsReporter{})
	t.Cleanup(func() {
		impl.WorkQueue().ShutDown()
	})

	// Enqueue two items with a long delay.
	impl.EnqueueAfter(&Resource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "for",
			Namespace: "waiting",
		},
	}, longDelay)
	impl.EnqueueAfter(&Resource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "waterfall",
			Namespace: "the",
		},
	}, longDelay)

	// Enqueue one item with a short delay.
	enqueueTime := time.Now()
	impl.EnqueueAfter(&Resource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fall",
			Namespace: "to",
		},
	}, shortDelay)

	// Keep checking the queue length until 'to/fall' gets enqueued, send to channel to indicate success.
	queuePopulated := make(chan int)
	ctx, cancel := context.WithTimeout(context.Background(), queueCheckTimeout)

	t.Cleanup(func() {
		close(queuePopulated)
		cancel()
	})

	go wait.PollImmediateUntil(5*time.Millisecond,
		pollQ(impl.WorkQueue(), queuePopulated), ctx.Done())

	select {
	case qlen := <-queuePopulated:
		if enqueueDelay := time.Since(enqueueTime); enqueueDelay < shortDelay {
			t.Errorf("Item enqueued within %v, expected at least a %v delay", enqueueDelay, shortDelay)
		}
		if got, want := qlen, 1; got != want {
			t.Errorf("|Queue| = %d, want: %d", got, want)
		}

	case <-ctx.Done():
		t.Fatal("Timed out waiting for item to be put onto the workqueue")
	}

	impl.WorkQueue().ShutDown()

	got, want := drainWorkQueue(impl.WorkQueue()), []types.NamespacedName{{Namespace: "to", Name: "fall"}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Unexpected workqueue state (-:expect, +:got):\n%s", diff)
	}
}

func TestEnqueueKeyAfter(t *testing.T) {
	impl := NewImplWithStats(&nopReconciler{}, TestLogger(t), "Testing", &FakeStatsReporter{})
	t.Cleanup(func() {
		impl.WorkQueue().ShutDown()
	})

	// Enqueue two items with a long delay.
	impl.EnqueueKeyAfter(types.NamespacedName{Namespace: "waiting", Name: "for"}, longDelay)
	impl.EnqueueKeyAfter(types.NamespacedName{Namespace: "the", Name: "waterfall"}, longDelay)

	// Enqueue one item with a short delay.
	enqueueTime := time.Now()
	impl.EnqueueKeyAfter(types.NamespacedName{Namespace: "to", Name: "fall"}, shortDelay)

	// Keep checking the queue length until 'to/fall' gets enqueued, send to channel to indicate success.
	queuePopulated := make(chan int)

	ctx, cancel := context.WithTimeout(context.Background(), queueCheckTimeout)

	t.Cleanup(func() {
		close(queuePopulated)
		cancel()
	})

	go wait.PollImmediateUntil(5*time.Millisecond,
		pollQ(impl.WorkQueue(), queuePopulated), ctx.Done())

	select {
	case qlen := <-queuePopulated:
		if enqueueDelay := time.Since(enqueueTime); enqueueDelay < shortDelay {
			t.Errorf("Item enqueued within %v, expected at least a %v delay", enqueueDelay, shortDelay)
		}
		if got, want := qlen, 1; got != want {
			t.Errorf("|Queue| = %d, want: %d", got, want)
		}

	case <-ctx.Done():
		t.Fatal("Timed out waiting for item to be put onto the workqueue")
	}

	impl.WorkQueue().ShutDown()

	got, want := drainWorkQueue(impl.WorkQueue()), []types.NamespacedName{{Namespace: "to", Name: "fall"}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Unexpected workqueue state (-:expect, +:got):\n%s", diff)
	}
}

type CountingReconciler struct {
	count atomic.Int32
}

func (cr *CountingReconciler) Reconcile(context.Context, string) error {
	cr.count.Inc()
	return nil
}

func TestStartAndShutdown(t *testing.T) {
	r := &CountingReconciler{}
	impl := NewImplWithStats(r, TestLogger(t), "Testing", &FakeStatsReporter{})

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		StartAll(ctx, impl)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	select {
	case <-time.After(10 * time.Millisecond):
		// We don't expect completion before the context is cancelled.
	case <-doneCh:
		t.Error("StartAll finished early.")
	}
	cancel()

	select {
	case <-time.After(time.Second):
		t.Error("Timed out waiting for controller to finish.")
	case <-doneCh:
		// We expect the work to complete.
	}

	if got, want := r.count.Load(), int32(0); got != want {
		t.Errorf("count = %v, wanted %v", got, want)
	}
}

type countingLeaderAwareReconciler struct {
	reconciler.LeaderAwareFuncs

	count atomic.Int32
}

var _ reconciler.LeaderAware = (*countingLeaderAwareReconciler)(nil)

func (cr *countingLeaderAwareReconciler) Reconcile(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	if cr.IsLeaderFor(types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}) {
		cr.count.Inc()
	}
	return nil
}

func TestStartAndShutdownWithLeaderAwareNoElection(t *testing.T) {
	promoted := make(chan struct{})
	r := &countingLeaderAwareReconciler{
		LeaderAwareFuncs: reconciler.LeaderAwareFuncs{
			PromoteFunc: func(bkt reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
				close(promoted)
				return nil
			},
		},
	}
	impl := NewImplWithStats(r, TestLogger(t), "Testing", &FakeStatsReporter{})

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		StartAll(ctx, impl)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	select {
	case <-promoted:
		// We expect to be promoted immediately, since there is no
		// ElectorBuilder attached to the context.
	case <-doneCh:
		t.Fatal("StartAll finished early.")
	case <-time.After(10 * time.Second):
		t.Error("Timed out waiting for StartAll.")
	}

	cancel()

	select {
	case <-time.After(time.Second):
		t.Error("Timed out waiting for controller to finish.")
	case <-doneCh:
		// We expect the work to complete.
	}

	if got, want := r.count.Load(), int32(0); got != want {
		t.Errorf("reconcile count = %v, wanted %v", got, want)
	}
}

func TestStartAndShutdownWithLeaderAwareWithLostElection(t *testing.T) {
	promoted := make(chan struct{})
	r := &countingLeaderAwareReconciler{
		LeaderAwareFuncs: reconciler.LeaderAwareFuncs{
			PromoteFunc: func(bkt reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
				close(promoted)
				return nil
			},
		},
	}
	cc := leaderelection.ComponentConfig{
		Component:     "component",
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
	}
	kc := fakekube.NewSimpleClientset(
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      "component.testing.00-of-01",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       ptr.String("not-us"),
				LeaseDurationSeconds: ptr.Int32(3000),
				AcquireTime:          &metav1.MicroTime{Time: time.Now()},
				RenewTime:            &metav1.MicroTime{Time: time.Now().Add(3000 * time.Second)},
			},
		},
	)

	impl := NewImplWithStats(r, TestLogger(t), "Testing", &FakeStatsReporter{})

	ctx, cancel := context.WithCancel(context.Background())
	ctx = leaderelection.WithStandardLeaderElectorBuilder(ctx, kc, cc)
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		StartAll(ctx, impl)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	select {
	case <-promoted:
		t.Fatal("Unexpected promotion.")
	case <-time.After(3 * time.Second):
		// Wait for 3 seconds for good measure.
	case <-doneCh:
		t.Error("StartAll finished early.")
	}

	cancel()

	select {
	case <-time.After(time.Second):
		t.Error("Timed out waiting for controller to finish.")
	case <-doneCh:
		// We expect the work to complete.
	}

	if got, want := r.count.Load(), int32(0); got != want {
		t.Errorf("reconcile count = %v, wanted %v", got, want)
	}
}

func TestStartAndShutdownWithWork(t *testing.T) {
	r := &CountingReconciler{}
	reporter := &FakeStatsReporter{}
	impl := NewImplWithStats(r, TestLogger(t), "Testing", reporter)

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		StartAll(ctx, impl)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	impl.EnqueueKey(types.NamespacedName{Namespace: "foo", Name: "bar"})

	select {
	case <-time.After(10 * time.Millisecond):
		// We don't expect completion before the context is cancelled.
	case <-doneCh:
		t.Error("StartAll finished early.")
	}
	cancel()

	select {
	case <-time.After(time.Second):
		t.Error("Timed out waiting for controller to finish.")
	case <-doneCh:
		// We expect the work to complete.
	}

	if got, want := r.count.Load(), int32(1); got != want {
		t.Errorf("reconcile count = %v, wanted %v", got, want)
	}
	if got, want := impl.WorkQueue().NumRequeues(types.NamespacedName{Namespace: "foo", Name: "bar"}), 0; got != want {
		t.Errorf("requeues = %v, wanted %v", got, want)
	}

	checkStats(t, reporter, 1, 0, 1, trueString)
}

type fakeError struct{}

var _ error = (*fakeError)(nil)

func (*fakeError) Error() string {
	return "I always error"
}

func TestPermanentError(t *testing.T) {
	err := new(fakeError)
	permErr := NewPermanentError(err)
	if !IsPermanentError(permErr) {
		t.Errorf("Expected type %T to be a permanentError", permErr)
	}
	if IsPermanentError(err) {
		t.Errorf("Expected type %T to not be a permanentError", err)
	}

	wrapPermErr := fmt.Errorf("wrapped: %w", permErr)
	if !IsPermanentError(wrapPermErr) {
		t.Error("Expected wrapped permanentError to be equivalent to a permanentError")
	}

	unwrapErr := new(fakeError)
	if !errors.As(permErr, &unwrapErr) {
		t.Errorf("Could not unwrap %T from permanentError", unwrapErr)
	}
}

type errorReconciler struct{}

func (er *errorReconciler) Reconcile(context.Context, string) error {
	return new(fakeError)
}

func TestStartAndShutdownWithErroringWork(t *testing.T) {
	const testTimeout = 500 * time.Millisecond

	item := types.NamespacedName{Namespace: "", Name: "bar"}

	impl := NewImplWithStats(&errorReconciler{}, TestLogger(t), "Testing", &FakeStatsReporter{})
	impl.EnqueueKey(item)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		// StartAll blocks until all the worker threads finish, which shouldn't
		// be until we cancel the context.
		StartAll(ctx, impl)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	// Keep checking the number of requeues, send to channel to indicate success.
	itemRequeued := make(chan struct{})
	defer close(itemRequeued)

	var successCheck wait.ConditionFunc = func() (bool, error) {
		// Check that the work was requeued in RateLimiter, as NumRequeues
		// can't fully reflect the real state of queue length.
		// Here we need to wait for NumRequeues to be more than 1, to ensure
		// the key get re-queued and reprocessed as expect.
		if impl.WorkQueue().NumRequeues(item) > 1 {
			itemRequeued <- struct{}{}
			return true, nil
		}
		return false, nil
	}
	go wait.PollImmediateUntil(5*time.Millisecond, successCheck, ctx.Done())

	select {
	case <-itemRequeued:
		// shut down reconciler
		cancel()

	case <-doneCh:
		t.Fatal("StartAll finished early")

	case <-ctx.Done():
		t.Fatal("Timed out waiting for item to be requeued")
	}
}

type permanentErrorReconciler struct{}

func (er *permanentErrorReconciler) Reconcile(context.Context, string) error {
	return NewPermanentError(new(fakeError))
}

func TestStartAndShutdownWithPermanentErroringWork(t *testing.T) {
	r := &permanentErrorReconciler{}
	reporter := &FakeStatsReporter{}
	impl := NewImplWithStats(r, TestLogger(t), "Testing", reporter)

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		StartAll(ctx, impl)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	impl.EnqueueKey(types.NamespacedName{Namespace: "foo", Name: "bar"})

	select {
	case <-time.After(20 * time.Millisecond):
		// We don't expect completion before the context is cancelled.
	case <-doneCh:
		t.Error("StartAll finished early.")
	}
	cancel()

	select {
	case <-time.After(time.Second):
		t.Error("Timed out waiting for controller to finish.")
	case <-doneCh:
		// We expect the work to complete.
	}

	// Check that the work was not requeued in RateLimiter.
	if got, want := impl.WorkQueue().NumRequeues(types.NamespacedName{Namespace: "foo", Name: "bar"}), 0; got != want {
		t.Errorf("Requeue count = %v, wanted %v", got, want)
	}

	checkStats(t, reporter, 1, 0, 1, falseString)
}

func drainWorkQueue(wq workqueue.RateLimitingInterface) (hasQueue []types.NamespacedName) {
	for {
		key, shutdown := wq.Get()
		if key == nil && shutdown {
			break
		}
		hasQueue = append(hasQueue, key.(types.NamespacedName))
	}
	return
}

type fakeInformer struct {
	cache.SharedInformer
}

type fakeStore struct {
	cache.Store
}

func (*fakeInformer) GetStore() cache.Store {
	return &fakeStore{}
}

var (
	fakeKeys = []string{"foo/bar", "bar/foo", "fizz/buzz"}
	fakeObjs = []interface{}{
		&Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bar",
				Namespace: "foo",
			},
		},
		&Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
		},
		&Resource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "buzz",
				Namespace: "fizz",
			},
		},
	}
)

func (*fakeStore) ListKeys() []string {
	return fakeKeys
}

func (*fakeStore) List() []interface{} {
	return fakeObjs
}

func TestImplGlobalResync(t *testing.T) {
	r := &CountingReconciler{}
	impl := NewImplWithStats(r, TestLogger(t), "Testing", &FakeStatsReporter{})

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		StartAll(ctx, impl)
	}()
	t.Cleanup(func() {
		cancel()
		<-doneCh
	})

	impl.GlobalResync(&fakeInformer{})

	// The global resync delays enqueuing things by a second with a jitter that
	// goes up to len(fakeObjs) times a second: time.Duration(1+len(fakeObjs)) * time.Second.
	// In this test, the fast lane is empty, so we can assume immediate enqueuing.
	select {
	case <-time.After(50 * time.Millisecond):
		// We don't expect completion before the context is cancelled.
	case <-doneCh:
		t.Error("StartAll finished early.")
	}
	cancel()

	select {
	case <-time.After(time.Second):
		t.Error("Timed out waiting for controller to finish.")
	case <-doneCh:
		// We expect the work to complete.
	}

	if got, want := r.count.Load(), int32(3); want != got {
		t.Errorf("GlobalResync: want = %v, got = %v", want, got)
	}
}

func checkStats(t *testing.T, r *FakeStatsReporter, reportCount, lastQueueDepth, reconcileCount int, lastReconcileSuccess string) {
	qd := r.GetQueueDepths()
	if got, want := len(qd), reportCount; got != want {
		t.Errorf("Queue depth reports = %v, wanted %v", got, want)
	}
	if got, want := qd[len(qd)-1], int64(lastQueueDepth); got != want {
		t.Errorf("Queue depth report = %v, wanted %v", got, want)
	}
	rd := r.GetReconcileData()
	if got, want := len(rd), reconcileCount; got != want {
		t.Errorf("Reconcile reports = %v, wanted %v", got, want)
	}
	if got, want := rd[len(rd)-1].Success, lastReconcileSuccess; got != want {
		t.Errorf("Reconcile success = %v, wanted %v", got, want)
	}
}

type fixedInformer struct {
	m    sync.Mutex
	sunk bool
	done bool
}

var _ Informer = (*fixedInformer)(nil)

func (fi *fixedInformer) Run(stopCh <-chan struct{}) {
	<-stopCh

	fi.m.Lock()
	defer fi.m.Unlock()
	fi.done = true
}

func (fi *fixedInformer) HasSynced() bool {
	fi.m.Lock()
	defer fi.m.Unlock()
	return fi.sunk
}

func (fi *fixedInformer) ToggleSynced(b bool) {
	fi.m.Lock()
	defer fi.m.Unlock()
	fi.sunk = b
}

func (fi *fixedInformer) Done() bool {
	fi.m.Lock()
	defer fi.m.Unlock()
	return fi.done
}

func TestStartInformersSuccess(t *testing.T) {
	errCh := make(chan error)
	defer close(errCh)

	fi := &fixedInformer{sunk: true}

	stopCh := make(chan struct{})
	defer close(stopCh)
	go func() {
		errCh <- StartInformers(stopCh, fi)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Error("Unexpected error:", err)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for informers to sync.")
	}
}

func TestStartInformersEventualSuccess(t *testing.T) {
	errCh := make(chan error)
	defer close(errCh)

	fi := &fixedInformer{sunk: false}

	stopCh := make(chan struct{})
	defer close(stopCh)
	go func() {
		errCh <- StartInformers(stopCh, fi)
	}()

	select {
	case <-time.After(50 * time.Millisecond):
		// Wait a brief period to ensure nothing is sent.
	case err := <-errCh:
		t.Fatal("Unexpected send on errCh:", err)
	}

	// Let the Sync complete.
	fi.ToggleSynced(true)

	select {
	case err := <-errCh:
		if err != nil {
			t.Error("Unexpected error:", err)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for informers to sync.")
	}
}

func TestStartInformersFailure(t *testing.T) {
	errCh := make(chan error)
	defer close(errCh)

	fi := &fixedInformer{sunk: false}

	stopCh := make(chan struct{})
	go func() {
		errCh <- StartInformers(stopCh, fi)
	}()

	select {
	case <-time.After(50 * time.Millisecond):
		// Wait a brief period to ensure nothing is sent.
	case err := <-errCh:
		t.Fatal("Unexpected send on errCh:", err)
	}

	// Now close the stopCh and we should see an error sent.
	close(stopCh)

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("Unexpected success syncing informers after stopCh closed.")
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for informers to sync.")
	}
}

func TestRunInformersSuccess(t *testing.T) {
	errCh := make(chan error)
	defer close(errCh)

	fi := &fixedInformer{sunk: true}

	stopCh := make(chan struct{})
	go func() {
		_, err := RunInformers(stopCh, fi)
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for informers to sync.")
	}

	close(stopCh)
}

func TestRunInformersEventualSuccess(t *testing.T) {
	errCh := make(chan error)
	defer close(errCh)

	fi := &fixedInformer{sunk: false}

	stopCh := make(chan struct{})
	go func() {
		_, err := RunInformers(stopCh, fi)
		errCh <- err
	}()

	select {
	case <-time.After(50 * time.Millisecond):
		// Wait a brief period to ensure nothing is sent.
	case err := <-errCh:
		t.Fatal("Unexpected send on errCh:", err)
	}

	// Let the Sync complete.
	fi.ToggleSynced(true)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for informers to sync.")
	}

	close(stopCh)
}

func TestRunInformersFailure(t *testing.T) {
	errCh := make(chan error)
	defer close(errCh)

	fi := &fixedInformer{sunk: false}

	stopCh := make(chan struct{})
	go func() {
		_, err := RunInformers(stopCh, fi)
		errCh <- err
	}()

	select {
	case <-time.After(50 * time.Millisecond):
		// Wait a brief period to ensure nothing is sent.
	case err := <-errCh:
		t.Fatal("Unexpected send on errCh:", err)
	}

	// Now close the stopCh and we should see an error sent.
	close(stopCh)

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("Unexpected success syncing informers after stopCh closed.")
		}
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for informers to sync.")
	}
}

func TestRunInformersFinished(t *testing.T) {
	fi := &fixedInformer{sunk: true}
	defer func() {
		if !fi.Done() {
			t.Fatalf("Test didn't wait for informers to finish")
		}
	}()

	ctx, cancel := context.WithCancel(TestContextWithLogger(t))
	t.Cleanup(cancel)

	waitInformers, err := RunInformers(ctx.Done(), fi)
	if err != nil {
		t.Fatal("Failed to start informers:", err)
	}

	cancel()

	ch := make(chan struct{})
	go func() {
		waitInformers()
		ch <- struct{}{}
	}()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for informers to finish.")
	}
}

func TestGetResyncPeriod(t *testing.T) {
	ctx := context.Background()

	if got := GetResyncPeriod(ctx); got != DefaultResyncPeriod {
		t.Errorf("GetResyncPeriod() = %v, wanted %v", got, nil)
	}

	bob := 30 * time.Second
	ctx = WithResyncPeriod(ctx, bob)

	if want, got := bob, GetResyncPeriod(ctx); got != want {
		t.Errorf("GetResyncPeriod() = %v, wanted %v", got, want)
	}

	tribob := 90 * time.Second
	if want, got := tribob, GetTrackerLease(ctx); got != want {
		t.Errorf("GetTrackerLease() = %v, wanted %v", got, want)
	}
}

func TestGetEventRecorder(t *testing.T) {
	ctx := context.Background()

	if got := GetEventRecorder(ctx); got != nil {
		t.Errorf("GetEventRecorder() = %v, wanted nil", got)
	}

	ctx = WithEventRecorder(ctx, record.NewFakeRecorder(1000))

	if got := GetEventRecorder(ctx); got == nil {
		t.Error("GetEventRecorder() = nil, wanted non-nil")
	}
}

func TestFilteredGlobalResync(t *testing.T) {
	tests := []struct {
		name       string
		filterFunc filterFunc
		wantQueue  []types.NamespacedName
	}{{
		name:       "do nothing",
		filterFunc: func(interface{}) bool { return false },
	}, {
		name:       "always true",
		filterFunc: alwaysTrue,
		wantQueue:  []types.NamespacedName{{Namespace: "foo", Name: "bar"}, {Namespace: "bar", Name: "foo"}, {Namespace: "fizz", Name: "buzz"}},
	}, {
		name: "filter namespace foo",
		filterFunc: func(obj interface{}) bool {
			if mo, ok := obj.(metav1.Object); ok {
				if mo.GetNamespace() == "foo" {
					return true
				} else {
					return false
				}
			}
			return false
		},
		wantQueue: []types.NamespacedName{{Namespace: "foo", Name: "bar"}},
	}, {
		name: "filter object named foo",
		filterFunc: func(obj interface{}) bool {
			if mo, ok := obj.(metav1.Object); ok {
				if mo.GetName() == "foo" {
					return true
				} else {
					return false
				}
			}
			return false
		},
		wantQueue: []types.NamespacedName{{Namespace: "bar", Name: "foo"}},
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			impl := NewImplFull(&nopReconciler{}, ControllerOptions{WorkQueueName: "FilteredTesting", Logger: TestLogger(t), GlobalResyncFilterFunc: test.filterFunc})
			impl.GlobalResync(&fakeInformer{})

			impl.WorkQueue().ShutDown()
			gotQueue := drainWorkQueue(impl.WorkQueue())

			if diff := cmp.Diff(test.wantQueue, gotQueue); diff != "" {
				t.Error("unexpected queue (-want +got):", diff)
			}
		})
	}
}
