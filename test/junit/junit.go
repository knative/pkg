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

// junit.go defines types and functions specific to manipulating junit test result XML files

package junit

import (
	"encoding/xml"
	"fmt"
	"math"
)

// TestStatusEnum is a enum for test result status
type TestStatusEnum string

const (
	// Failed means junit test failed
	Failed TestStatusEnum = "failed"
	// Skipped means junit test skipped
	Skipped TestStatusEnum = "skipped"
	// Passed means junit test passed
	Passed TestStatusEnum = "passed"

	// acceptedFloatError is the maximum error between float type in the struct to be considered as equal.
	acceptedFloatError = 0.00001
)

// TestSuites holds a <testSuites/> list of TestSuite results
type TestSuites struct {
	XMLName xml.Name    `xml:"testsuites"`
	Suites  []TestSuite `xml:"testsuite"`
}

// TestSuite holds <testSuite/> results
type TestSuite struct {
	XMLName    xml.Name       `xml:"testsuite"`
	Name       string         `xml:"name,attr"`
	Time       float64        `xml:"time,attr"` // Seconds
	Failures   int            `xml:"failures,attr"`
	Tests      int            `xml:"tests,attr"`
	TestCases  []TestCase     `xml:"testcase"`
	Properties TestProperties `xml:"properties"`
}

// TestCase holds <testcase/> results
type TestCase struct {
	Name       string         `xml:"name,attr"`
	Time       float64        `xml:"time,attr"` // Seconds
	ClassName  string         `xml:"classname,attr"`
	Failure    *string        `xml:"failure,omitempty"`
	Output     *string        `xml:"system-out,omitempty"`
	Error      *string        `xml:"system-err,omitempty"`
	Skipped    *string        `xml:"skipped,omitempty"`
	Properties TestProperties `xml:"properties"`
}

// TestProperties is an array of test properties
type TestProperties struct {
	Properties []TestProperty `xml:"property"`
}

// TestProperty defines a property of the test
type TestProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// GetTestStatus returns the test status as a string
func (tc *TestCase) GetTestStatus() TestStatusEnum {
	testStatus := Passed
	switch {
	case tc.Failure != nil:
		testStatus = Failed
	case tc.Skipped != nil:
		testStatus = Skipped
	}
	return testStatus
}

// AddProperty adds property to testcase
func (tc *TestCase) AddProperty(name, val string) {
	property := TestProperty{Name: name, Value: val}
	tc.Properties.Properties = append(tc.Properties.Properties, property)
}

// Equal returns true if two TestCase are equal.
func (tc TestCase) Equal(tc2 TestCase) bool {
	return tc.Name == tc2.Name &&
		math.Abs(tc.Time-tc2.Time) < acceptedFloatError &&
		tc.ClassName == tc2.ClassName &&
		(tc.Failure == nil && tc2.Failure == nil || tc.Failure != nil && tc2.Failure != nil && *tc.Failure == *tc2.Failure) &&
		(tc.Output == nil && tc2.Output == nil || tc.Output != nil && tc2.Output != nil && *tc.Output == *tc2.Output) &&
		(tc.Error == nil && tc2.Error == nil || tc.Error != nil && tc2.Error != nil && *tc.Error == *tc2.Error) &&
		(tc.Skipped == nil && tc2.Skipped == nil || tc.Skipped != nil && tc2.Skipped != nil && *tc.Skipped == *tc2.Skipped) &&
		tc.Properties.Equal(tc2.Properties)
}

// Equal returns true if Two TestProperties are equal.
func (tps TestProperties) Equal(tps2 TestProperties) bool {
	if tps.Properties == nil && tps2.Properties == nil {
		return true
	}

	if tps.Properties == nil || tps2.Properties == nil || len(tps.Properties) != len(tps2.Properties) {
		return false
	}

	for i := range tps.Properties {
		tp := tps.Properties[i]
		tp2 := tps2.Properties[i]
		if !tp.Equal(tp2) {
			return false
		}
	}
	return true
}

// Equal returns true if two TestProperty are equal.
func (tp TestProperty) Equal(tp2 TestProperty) bool {
	return tp.Name == tp2.Name &&
		tp.Value == tp2.Value
}

// AddTestCase adds a testcase to the testsuite
func (ts *TestSuite) AddTestCase(tc TestCase) {
	ts.TestCases = append(ts.TestCases, tc)
}

// GetTestSuite gets TestSuite struct by name
func (testSuites *TestSuites) GetTestSuite(suiteName string) (*TestSuite, error) {
	for _, testSuite := range testSuites.Suites {
		if testSuite.Name == suiteName {
			return &testSuite, nil
		}
	}
	return nil, fmt.Errorf("Test suite '%s' not found", suiteName)
}

// AddTestSuite adds TestSuite to TestSuites
func (testSuites *TestSuites) AddTestSuite(testSuite *TestSuite) error {
	if _, err := testSuites.GetTestSuite(testSuite.Name); err == nil {
		return fmt.Errorf("Test suite '%s' already exists", testSuite.Name)
	}
	testSuites.Suites = append(testSuites.Suites, *testSuite)
	return nil
}

// ToBytes converts TestSuites struct to bytes array
func (testSuites *TestSuites) ToBytes(prefix, indent string) ([]byte, error) {
	return xml.MarshalIndent(testSuites, prefix, indent)
}

// UnMarshal converts bytes array to TestSuites struct,
// it works with both TestSuites and TestSuite structs, if
// input is a TestSuite struct it will still return a TestSuites
// struct, which is an empty wrapper TestSuites containing only
// the input Suite
func UnMarshal(buf []byte) (*TestSuites, error) {
	var testSuites TestSuites
	if err := xml.Unmarshal(buf, &testSuites); err == nil {
		return &testSuites, nil
	}

	// The input might be a TestSuite if reach here, try parsing with TestSuite
	testSuites.Suites = append([]TestSuite(nil), TestSuite{})
	if err := xml.Unmarshal(buf, &testSuites.Suites[0]); err != nil {
		return nil, err
	}
	return &testSuites, nil
}
