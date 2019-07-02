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

package metricstest

import (
	"testing"

	"go.opencensus.io/stats/view"
)

// CheckStatsReported checks that there is a view registered with the given name for each string in names,
// and that each view has at least one record.
func CheckStatsReported(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		d, err := view.RetrieveData(name)
		if err != nil {
			t.Errorf("Reporter.Report() error = %v, wantErr %v", err, false)
		}
		if len(d) < 1 {
			t.Errorf("No data reported when data was expected. len(d)=%v", len(d))
		}
	}
}

// CheckStatsNotReported checks that there are no records for any views that a name matching a string in names.
// Names that do not match registered views are considered not reported.
func CheckStatsNotReported(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		if d, err := view.RetrieveData(name); err != nil {
			if len(d) > 0 {
				t.Errorf("Unexpected data reported when no data was expected. Reporter len(d) = %d", len(d))
			}
		}
	}
}

// CheckCountData checks the view with a name matching string name to verify that the CountData stats
// reported are tagged with the tags in wantTags and that wantValue matches reported count.
func CheckCountData(t *testing.T, name string, wantTags map[string]string, wantValue int) {
	t.Helper()
	if row := checkExactlyOneRow(t, name); row != nil {
		for _, got := range row.Tags {
			n := got.Key.Name()
			if want, ok := wantTags[n]; !ok {
				t.Errorf("Reporter got an extra tag %v: %v", n, got.Value)
			} else if got.Value != want {
				t.Errorf("Reporter expected a different tag value for key: %s, got: %s, want: %s", n, got.Value, want)
			}
		}

		if s, ok := row.Data.(*view.CountData); !ok {
			t.Error("Reporter expected a SumData type")
		} else if s.Value != int64(wantValue) {
			t.Errorf("For %s value = %v, want: %d", name, s.Value, wantValue)
		}
	}
}

// CheckDistributionData checks the view with a name matching string name to verify that the DistributionData stats reported
// are tagged with the tags in wantTags and that expectedCount number of records were reported.
// It also checks that expectedMin and expectedMax match the minimum and maximum reported values, respectively.
func CheckDistributionData(t *testing.T, name string, wantTags map[string]string, expectedCount int, expectedMin float64, expectedMax float64) {
	t.Helper()
	if row := checkExactlyOneRow(t, name); row != nil {
		for _, got := range row.Tags {
			n := got.Key.Name()
			if want, ok := wantTags[n]; !ok {
				t.Errorf("Reporter got an extra tag %v: %v", n, got.Value)
			} else if got.Value != want {
				t.Errorf("Reporter expected a different tag value for key: %s, got: %s, want: %s", n, got.Value, want)
			}
		}

		if s, ok := row.Data.(*view.DistributionData); !ok {
			t.Error("Reporter expected a DistributionData type")
		} else {
			if s.Count != int64(expectedCount) {
				t.Errorf("For metric %s: reporter count = %d, want = %d", name, s.Count, expectedCount)
			}
			if s.Min != expectedMin {
				t.Errorf("For metric %s: reporter count = %f, want = %f", name, s.Min, expectedMin)
			}
			if s.Max != expectedMax {
				t.Errorf("For metric %s: reporter count = %f, want = %f", name, s.Max, expectedMax)
			}
		}
	}
}

// CheckLastValueData checks the view with a name matching string name to verify that the LastValueData stats
// reported are tagged with the tags in wantTags and that wantValue matches reported last value.
func CheckLastValueData(t *testing.T, name string, wantValue float64) {
	t.Helper()
	if row := checkExactlyOneRow(t, name); row != nil {
		if s, ok := row.Data.(*view.LastValueData); !ok {
			t.Error("Reporter.Report() expected a LastValueData type")
		} else if s.Value != wantValue {
			t.Errorf("Reporter.Report() expected %v got %v. metric: %v", s.Value, wantValue, name)
		}
	}
}

// CheckSumData checks the view with a name matching string name to verify that the SumData stats
// reported are tagged with the tags in wantTags and that wantValue matches the reported sum.
func CheckSumData(t *testing.T, name string, wantTags map[string]string, wantValue int) {
	t.Helper()
	if row := checkExactlyOneRow(t, name); row != nil {
		for _, got := range row.Tags {
			n := got.Key.Name()
			if want, ok := wantTags[n]; !ok {
				t.Errorf("Reporter got an extra tag %v: %v", n, got.Value)
			} else if got.Value != want {
				t.Errorf("Reporter expected a different tag value for key: %s, got: %s, want: %s", n, got.Value, want)
			}
		}

		if s, ok := row.Data.(*view.SumData); !ok {
			t.Error("Reporter expected a SumData type")
		} else if s.Value != float64(wantValue) {
			t.Errorf("For %s value = %v, want: %d", name, s.Value, wantValue)
		}
	}
}

func checkExactlyOneRow(t *testing.T, name string) *view.Row {
	t.Helper()
	d, err := view.RetrieveData(name)
	if err != nil {
		t.Errorf("Reporter.Report() error = %v, wantErr %v", err, false)
		return nil
	}
	if len(d) != 1 {
		t.Errorf("Reporter.Report() len(d)=%v, want 1", len(d))
	}
	return d[0]
}
