/*
Copyright 2019 The Knative Authors

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

package metrics

import (
	"testing"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"golang.org/x/net/context"
	"k8s.io/client-go/util/workqueue"

	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics/metricstest"
)

func newInt64(name string) *stats.Int64Measure {
	return stats.Int64(name, "bar", "wtfs/s")
}

func newFloat64(name string) *stats.Float64Measure {
	return stats.Float64(name, "bar", "wtfs/s")
}

type fixedRateLimiter struct {
	delay time.Duration
}

func (f *fixedRateLimiter) When(item interface{}) time.Duration {
	return f.delay
}

func (f *fixedRateLimiter) Forget(item interface{}) {}

func (f *fixedRateLimiter) NumRequeues(item interface{}) int {
	return 0
}

func TestWorkqueueMetrics(t *testing.T) {
	wp := &WorkqueueProvider{
		Adds:                           newInt64("adds"),
		Depth:                          newInt64("depth"),
		Latency:                        newFloat64("latency"),
		Retries:                        newInt64("retries"),
		WorkDuration:                   newFloat64("work_duration"),
		UnfinishedWorkSeconds:          newFloat64("unfinished_work_seconds"),
		LongestRunningProcessorSeconds: newFloat64("longest_running_processor_seconds"),
	}
	workqueue.SetProvider(wp)

	// Reset the metrics configuration to avoid leaked state from other tests.
	InitForTesting()

	_, ref, err := newPrometheusExporter(getCurMetricsConfig(), logging.FromContext(context.Background()))
	if err != nil {
		t.Fatalf("newPrometheusExporter() = %v", err)
	}
	setFactory(ref)

	views := wp.DefaultViews()
	if got, want := len(views), 7; got != want {
		t.Errorf("len(DefaultViews()) = %d, want %d", got, want)
	}
	if err := view.Register(views...); err != nil {
		t.Error("view.Register() =", err)
	}
	defer view.Unregister(views...)

	queueName := t.Name()
	limiter := &fixedRateLimiter{delay: 200 * time.Millisecond}
	wq := workqueue.NewNamedRateLimitingQueue(limiter, queueName)

	metricstest.CheckStatsNotReported(t, "adds", "depth", "latency", "retries", "work_duration",
		"unfinished_work_seconds", "longest_running_processor_seconds")

	wq.Add("foo")

	metricstest.AssertMetricExists(t, "adds", "depth")
	metricstest.AssertNoMetric(t, "latency", "retries", "work_duration",
		"unfinished_work_seconds", "longest_running_processor_seconds")
	wantAdd := metricstest.IntMetric("adds", 1, map[string]string{"name": queueName})
	wantDepth := metricstest.IntMetric("depth", 1, map[string]string{"name": queueName})
	metricstest.AssertMetric(t, wantAdd, wantDepth)

	wq.Add("bar")

	metricstest.AssertNoMetric(t, "latency", "retries", "work_duration",
		"unfinished_work_seconds", "longest_running_processor_seconds")
	*wantAdd.Values[0].Int64++
	*wantDepth.Values[0].Int64++
	metricstest.AssertMetric(t, wantAdd, wantDepth)

	if got, shutdown := wq.Get(); shutdown {
		t.Errorf("Get() = %v, true; want false", got)
	} else if want := "foo"; got != want {
		t.Errorf("Get() = %s, false; want %s", got, want)
	} else {
		wq.Forget(got)
		wq.Done(got)
	}

	metricstest.AssertMetricExists(t, "latency", "work_duration")
	metricstest.AssertNoMetric(t, "retries",
		"unfinished_work_seconds", "longest_running_processor_seconds")
	metricstest.AssertMetric(t, wantAdd)

	if got, shutdown := wq.Get(); shutdown {
		t.Errorf("Get() = %v, true; want false", got)
	} else if want := "bar"; got != want {
		t.Errorf("Get() = %s, false; want %s", got, want)
	} else {
		wq.AddRateLimited(got)
		wq.Done(got)
	}

	// It should show up as a retry now.
	metricstest.AssertMetricExists(t, "retries")
	metricstest.AssertNoMetric(t, "unfinished_work_seconds", "longest_running_processor_seconds")
	wantRetries := metricstest.IntMetric("retries", 1, map[string]string{"name": queueName})
	metricstest.AssertMetric(t, wantRetries, wantAdd) // It is not added right away.

	// It doesn't show up as an "add" until the rate limit has elapsed.
	time.Sleep(2 * limiter.delay)
	*wantAdd.Values[0].Int64++
	metricstest.AssertMetric(t, wantAdd)

	wq.ShutDown()

	if got, shutdown := wq.Get(); shutdown {
		t.Errorf("Get() = %v, true; want false", got)
	} else if want := "bar"; got != want {
		t.Errorf("Get() = %s, true; want %s", got, want)
	}

	if got, shutdown := wq.Get(); !shutdown {
		t.Errorf("Get() = %v, false; want true", got)
	}
}
