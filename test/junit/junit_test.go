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

// junit_test.go contains unit tests for junit package

package junit

import (
	"testing"
)

var emptySuites = `
<testsuites>
</testsuites>
`

var malSuitesString = `
<testsuites>
<testsuites>
`

var validSuiteString = `
<testsuite name="knative/test-infra">
	<properties>
		<property name="go.version" value="go1.6"/>
	</properties>
	<testcase name="TestBad" time="0.1">
		<failure>something bad</failure>
		<system-out>out: first line</system-out>
		<system-err>err: first line</system-err>
		<system-out>out: second line</system-out>
	</testcase>
	<testcase name="TestGood" time="0.1">
	</testcase>
	<testcase name="TestSkip" time="0.1">
		<skipped>do not test</skipped>
	</testcase>
</testsuite>
`

var validSuitesString = `
<testsuites>
	<testsuite name="knative/test-infra">
		<properties>
			<property name="go.version" value="go1.6"/>
		</properties>
		<testcase name="TestBad" time="0.1">
			<failure>something bad</failure>
			<system-out>out: first line</system-out>
			<system-err>err: first line</system-err>
			<system-out>out: second line</system-out>
		</testcase>
		<testcase name="TestGood" time="0.1">
		</testcase>
		<testcase name="TestSkip" time="0.1">
			<skipped>do not test</skipped>
		</testcase>
	</testsuite>
</testsuites>
`

func newTestCase(name string, status TestStatusEnum) *TestCase {
	testCase := TestCase{
		Name: name,
	}

	var tmp string // cast const to string
	switch {
	case status == Failed:
		tmp = string(Failed)
		testCase.Failure = &tmp
	case status == Skipped:
		tmp = string(Skipped)
		testCase.Skipped = &tmp
	}

	return &testCase
}

func TestUnmarshalEmptySuites(t *testing.T) {
	if _, err := UnMarshal([]byte(emptySuites)); err != nil {
		t.Errorf("Expected 'succeed', actual: 'failed parsing empty suites, '%s'", err)
	}
}

func TestUnmarshalMalFormed(t *testing.T) {
	if _, err := UnMarshal([]byte(malSuitesString)); err == nil {
		t.Errorf("Expected: failed, actual: succeeded parsing malformed xml, '%s'", err)
	}
}

func TestUnmarshalSuites(t *testing.T) {
	if _, err := UnMarshal([]byte(validSuitesString)); err != nil {
		t.Errorf("Expected: succeed, actual: failed parsing suites result, '%s'", err)
	}
}

func TestUnmarshalSuite(t *testing.T) {
	if _, err := UnMarshal([]byte(validSuiteString)); err != nil {
		t.Errorf("Expected: succeed, actual: failed parsing suite result, '%s'", err)
	}
}

func TestGetTestStatus(t *testing.T) {
	if status := newTestCase("TestGood", Passed).GetTestStatus(); Passed != status {
		t.Errorf("Expected '%s', actual '%s'", Passed, status)
	}
	if status := newTestCase("TestSkip", Skipped).GetTestStatus(); Skipped != status {
		t.Errorf("Expected '%s', actual '%s'", Skipped, status)
	}
	if status := newTestCase("TestBad", Failed).GetTestStatus(); Failed != status {
		t.Errorf("Expected '%s', actual '%s'", Failed, status)
	}
}

func TestAddTestSuite(t *testing.T) {
	testSuites := TestSuites{}
	testSuite0 := TestSuite{Name: "suite_0"}
	testSuite1 := TestSuite{Name: "suite_1"}

	if err := testSuites.AddTestSuite(&testSuite0); err != nil {
		t.Fatalf("Expected '', actual '%v'", err)
	}

	expectedErrString := "Test suite 'suite_0' already exists"
	if err := testSuites.AddTestSuite(&testSuite0); err == nil || err.Error() != expectedErrString {
		t.Fatalf("Expected: '%s', actual: '%v'", expectedErrString, err)
	}

	if err := testSuites.AddTestSuite(&testSuite1); err != nil {
		t.Fatalf("Expected '', actual '%v'", err)
	}

	if len(testSuites.Suites) != 2 {
		t.Fatalf("Expected 2, actual %d", len(testSuites.Suites))
	}
}

func TestTestCaseEqual(t *testing.T) {
	failMsg1 := "failure"
	failMsg2 := "failure"
	diffFailMsg := "not-the-same-failure"

	type args struct {
		tc1 *TestCase
		tc2 *TestCase
	}
	tests := []struct {
		name string
		args *args
		want bool
	}{
		{
			name: "default struct is equal",
			args: &args{tc1: &TestCase{}, tc2: &TestCase{}},
			want: true,
		},
		{
			name: "accepted float difference and different pointer address, same value result in same struct",
			args: &args{
				tc1: &TestCase{
					Name:      "dummy case",
					Time:      0,
					ClassName: "classname",
					Failure:   &failMsg1,
					Output:    &failMsg1,
					Error:     &failMsg1,
					Skipped:   &failMsg1,
					Properties: TestProperties{
						Properties: []TestProperty{{Name: "Random property", Value: "Random Value"}},
					},
				},
				tc2: &TestCase{
					Name:      "dummy case",
					Time:      0.000000000000001,
					ClassName: "classname",
					Failure:   &failMsg2,
					Output:    &failMsg2,
					Error:     &failMsg2,
					Skipped:   &failMsg2,
					Properties: TestProperties{
						Properties: []TestProperty{{Name: "Random property", Value: "Random Value"}},
					},
				},
			},
			want: true,
		},
		{
			name: "Different Name is not equal",
			args: &args{tc1: &TestCase{Name: "test1"}, tc2: &TestCase{Name: "test2"}},
			want: false,
		},
		{
			name: "Different time is not equal",
			args: &args{tc1: &TestCase{Time: 1}, tc2: &TestCase{Time: 200}},
			want: false,
		},
		{
			name: "Different ClaseName is not equal",
			args: &args{tc1: &TestCase{ClassName: "test1"}, tc2: &TestCase{ClassName: "test2"}},
			want: false,
		},
		{
			name: "Different Failure is not equal",
			args: &args{tc1: &TestCase{Failure: &failMsg1}, tc2: &TestCase{Failure: &diffFailMsg}},
			want: false,
		},
		{
			name: "Different Output is not equal",
			args: &args{tc1: &TestCase{Output: &failMsg1}, tc2: &TestCase{Output: &diffFailMsg}},
			want: false,
		},
		{
			name: "Different Error is not equal",
			args: &args{tc1: &TestCase{Error: &failMsg1}, tc2: &TestCase{Error: &diffFailMsg}},
			want: false,
		},
		{
			name: "Different Skipped is not equal",
			args: &args{tc1: &TestCase{Skipped: &failMsg1}, tc2: &TestCase{Skipped: &diffFailMsg}},
			want: false,
		},
		{
			name: "Different TestProperties is not equal",
			args: &args{
				tc1: &TestCase{Properties: TestProperties{
					Properties: []TestProperty{{Name: "Random property", Value: "Random Value"}},
				}},
				tc2: &TestCase{Properties: TestProperties{
					Properties: []TestProperty{{Name: "Another", Value: "Random Value"}},
				}},
			},
			want: false,
		},
		{
			name: "Different fields is not equal",
			args: &args{tc1: &TestCase{Name: "test"}, tc2: &TestCase{ClassName: "test"}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.tc1.Equal(*tt.args.tc2); got != tt.want {
				t.Errorf("TestCase.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestProperties_Equal(t *testing.T) {
	type args struct {
		tps1 TestProperties
		tps2 TestProperties
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Both nil properties is equal",
			args: args{
				tps1: TestProperties{},
				tps2: TestProperties{},
			},
			want: true,
		},
		{
			name: "Both properties with values is equal",
			args: args{
				tps1: TestProperties{
					Properties: []TestProperty{
						{Name: "Random property", Value: "Random Value"},
						{Name: "Another property", Value: "Another Random Value"},
					},
				},
				tps2: TestProperties{
					Properties: []TestProperty{
						{Name: "Random property", Value: "Random Value"},
						{Name: "Another property", Value: "Another Random Value"},
					},
				},
			},
			want: true,
		},
		{
			name: "Single nil returns false (LHS)",
			args: args{
				tps1: TestProperties{
					Properties: nil,
				},
				tps2: TestProperties{
					Properties: []TestProperty{
						{Name: "Random property", Value: "Random Value"},
						{Name: "Another property", Value: "Another Random Value"},
					},
				},
			},
			want: false,
		},
		{
			name: "Single nil returns false (RHS)",
			args: args{
				tps1: TestProperties{
					Properties: []TestProperty{
						{Name: "Random property", Value: "Random Value"},
						{Name: "Another property", Value: "Another Random Value"},
					},
				},
				tps2: TestProperties{
					Properties: nil,
				},
			},
			want: false,
		},
		{
			name: "Both properties with diff values is non-equal",
			args: args{
				tps1: TestProperties{
					Properties: []TestProperty{
						{Name: "Random property", Value: "Random Value"},
						{Name: "Another property", Value: "Another Random Value"},
					},
				},
				tps2: TestProperties{
					Properties: []TestProperty{
						{Name: "Random property", Value: "Random Value"},
						{Name: "Another property", Value: "Another Random UNIQUE Value"},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.tps1.Equal(tt.args.tps2); got != tt.want {
				t.Errorf("TestProperties.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}
