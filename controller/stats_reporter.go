/*
Copyright 2018 The Knative Authors

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
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	workQueueDepthStat   = stats.Float64("work_queue_depth", "Depth of the work queue", stats.UnitNone)
	reconcileCountStat   = stats.Int64("reconcile_count", "Number of reconcile operations", stats.UnitNone)
	reconcileLatencyStat = stats.Int64("reconcile_latency", "Latency of reconcile operations", stats.UnitMilliseconds)

	reconcilerTagKey tag.Key
	keyTagKey        tag.Key
	successTagKey    tag.Key
)

func init() {
	var err error
	// Create the tag keys that will be used to add tags to our measurements.
	// Tag keys must conform to the restrictions described in
	// go.opencensus.io/tag/validate.go. Currently those restrictions are:
	// - length between 1 and 255 inclusive
	// - characters are printable US-ASCII
	reconcilerTagKey, err = tag.NewKey("reconciler")
	if err != nil {
		panic(err)
	}
	keyTagKey, err = tag.NewKey("key")
	if err != nil {
		panic(err)
	}
	successTagKey, err = tag.NewKey("success")
	if err != nil {
		panic(err)
	}

	// Create views to see our measurements. This can return an error if
	// a previously-registered view has the same name with a different value.
	// View name defaults to the measure name if unspecified.
	err = view.Register(
		&view.View{
			Description: "Depth of the work queue",
			Measure:     workQueueDepthStat,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{reconcilerTagKey},
		},
		&view.View{
			Description: "Number of reconcile operations",
			Measure:     reconcileCountStat,
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{reconcilerTagKey, keyTagKey, successTagKey},
		},
		&view.View{
			Description: "Latency of reconcile operations",
			Measure:     reconcileLatencyStat,
			Aggregation: view.Distribution(0, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000, 60000),
			TagKeys:     []tag.Key{reconcilerTagKey, keyTagKey, successTagKey},
		},
	)
	if err != nil {
		panic(err)
	}
}

// StatsReporter defines the interface for sending metrics
type StatsReporter interface {
	ReportQueueDepth(v float64) error
	ReportReconcile(duration time.Duration, key, success string) error
}

// Reporter holds cached metric objects to report metrics
type reporter struct {
	reconciler string
	globalCtx  context.Context
}

// NewStatsReporter creates a reporter that collects and reports metrics
func NewStatsReporter(reconciler string) (StatsReporter, error) {
	// Reconciler tag is static. Create a context containing that and cache it.
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(reconcilerTagKey, reconciler))
	if err != nil {
		return nil, err
	}

	return &reporter{reconciler: reconciler, globalCtx: ctx}, nil
}

// ReportQueueDepth reports the queue depth metric
func (r *reporter) ReportQueueDepth(v float64) error {
	if r.globalCtx == nil {
		return errors.New("reporter is not initialized correctly")
	}
	stats.Record(r.globalCtx, workQueueDepthStat.M(v))
	return nil
}

// ReportReconcile reports the count and latency metrics for a reconcile operation.
func (r *reporter) ReportReconcile(duration time.Duration, key, success string) error {
	ctx, err := tag.New(
		context.Background(),
		tag.Insert(reconcilerTagKey, r.reconciler),
		tag.Insert(keyTagKey, key),
		tag.Insert(successTagKey, success))
	if err != nil {
		return err
	}

	stats.Record(ctx, reconcileCountStat.M(1))
	stats.Record(ctx, reconcileLatencyStat.M(int64(duration)))
	return nil
}
