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
	"context"
	"sync/atomic"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"k8s.io/client-go/util/workqueue"
)

type counterMetric struct {
	mutators []tag.Mutator
	measure  *stats.Int64Measure
}

var (
	_ workqueue.CounterMetric = (*counterMetric)(nil)
)

// Inc implements CounterMetric
func (m counterMetric) Inc() {
	Record(context.Background(), m.measure.M(1), stats.WithTags(m.mutators...))
}

type gaugeMetric struct {
	mutators []tag.Mutator
	measure  *stats.Int64Measure
	total    int64
}

var (
	_ workqueue.GaugeMetric = (*gaugeMetric)(nil)
)

// Inc implements CounterMetric
func (m *gaugeMetric) Inc() {
	total := atomic.AddInt64(&m.total, 1)
	Record(context.Background(), m.measure.M(total), stats.WithTags(m.mutators...))
}

// Dec implements GaugeMetric
func (m *gaugeMetric) Dec() {
	total := atomic.AddInt64(&m.total, -1)
	Record(context.Background(), m.measure.M(total), stats.WithTags(m.mutators...))
}

type floatMetric struct {
	mutators []tag.Mutator
	measure  *stats.Float64Measure
}

var (
	_ workqueue.SummaryMetric = (*floatMetric)(nil)
)

// Observe implements SummaryMetric
func (m floatMetric) Observe(v float64) {
	Record(context.Background(), m.measure.M(v), stats.WithTags(m.mutators...))
}
