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

package testing

import (
	"time"
)

// FakeStatsReporter is a fake implementation of StatsReporter
type FakeStatsReporter struct{}

// ReportQueueDepth does nothing and returns success.
func (r *FakeStatsReporter) ReportQueueDepth(v int64) error {
	return nil
}

// ReportReconcile does nothing and returns success.
func (r *FakeStatsReporter) ReportReconcile(duration time.Duration, key, success string) error {
	return nil
}
