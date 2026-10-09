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

package controller

import (
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller/priorityqueue"
)

const slowLanePriority = -100

// twoLaneRateLimitingQueue is a small compatibility wrapper around the
// controller-runtime priority queue. Items enqueued through Add/EnqueueFast use
// the default priority, while items enqueued through AddSlow/EnqueueSlow use a
// low priority.
type twoLaneRateLimitingQueue struct {
	priorityqueue.PriorityQueue[any]
}

var _ workqueue.TypedRateLimitingInterface[any] = (*twoLaneRateLimitingQueue)(nil)

// Creates a new newTwoLaneWorkQueue
func newTwoLaneWorkQueue(name string, rl workqueue.TypedRateLimiter[any]) *twoLaneRateLimitingQueue {
	return &twoLaneRateLimitingQueue{
		PriorityQueue: priorityqueue.New[any](name, func(o *priorityqueue.Opts[any]) {
			o.RateLimiter = rl
			o.MetricProvider = globalMetricsProvider
		}),
	}
}

func (q *twoLaneRateLimitingQueue) EnqueueFast(item any) {
	q.Add(item)
}

func (q *twoLaneRateLimitingQueue) AddSlow(item any) {
	q.EnqueueSlow(item)
}

func (q *twoLaneRateLimitingQueue) EnqueueSlow(item any) {
	q.AddWithOpts(priorityqueue.AddOpts{Priority: ptrTo(slowLanePriority)}, item)
}

func ptrTo[T any](v T) *T {
	return &v
}
