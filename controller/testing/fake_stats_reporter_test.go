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

package testing

import (
	"testing"
	"time"

	"github.com/knative/pkg/controller"
)

var _ controller.StatsReporter = (*FakeStatsReporter)(nil)

func TestReportQueueDepth(t *testing.T) {
	r := &FakeStatsReporter{}
	r.ReportQueueDepth(10)
	if got, want := len(r.QueueDepths), 1; want != got {
		t.Errorf("queue depth len: want: %v, got: %v", want, got)
	}
	if got, want := r.QueueDepths[0], int64(10); want != got {
		t.Errorf("queue depth value: want: %v, got: %v", want, got)
	}
}

func TestReportReconcile(t *testing.T) {
	r := &FakeStatsReporter{}
	r.ReportReconcile(time.Duration(123), "testkey", "False")
	if got, want := len(r.ReconcileData), 1; want != got {
		t.Errorf("reconcile data len: want: %v, got: %v", want, got)
	}
	if got, want := r.ReconcileData[0].Duration, time.Duration(123); want != got {
		t.Errorf("reconcile data duration: want: %v, got: %v", want, got)
	}
	if got, want := r.ReconcileData[0].Key, "testkey"; want != got {
		t.Errorf("reconcile data key: want: %v, got: %v", want, got)
	}
	if got, want := r.ReconcileData[0].Success, "False"; want != got {
		t.Errorf("reconcile data success: want: %v, got: %v", want, got)
	}
}
