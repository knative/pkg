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

package helpers

import (
	"regexp"
	"strings"
	"testing"
)

var matcher = regexp.MustCompile("abcd-[a-z]{8}")

func TestAppendRandomString(t *testing.T) {
	const s = "abcd"
	w := AppendRandomString(s)
	o := AppendRandomString(s)
	if !matcher.MatchString(w) || !matcher.MatchString(o) || o == w {
		t.Fatalf("Generated string(s) are incorrect: %q, %q", w, o)
	}
}

func TestMakeK8sNamePrefix(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"abcd123", "abcd123"},
		{"AbCdef", "ab-cdef"},
		{"ABCD", "a-b-c-d"},
		{"aBc*ef&d", "a-bc-ef-d"},
		{"*aBc*ef&d", "a-bc-ef-d"},
		{"AutoTLS", "auto-tls"},
		{"GRPCLoadBalancing", "grpc-load-balancing"},
		{"HTTPSTermination", "https-termination"},
		{"HTTPIsNotHTTP2", "http-is-not-http2"},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := MakeK8sNamePrefix(tc.input)
			if tc.expected != actual {
				t.Errorf("MakeK8sNamePrefix = %q, want: %q", actual, tc.expected)
			}
		})
	}
}

func TestGetBaseFuncName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"test/e2e.TestMain", "TestMain"},
		{"e2e.TestMain", "TestMain"},
		{"test/TestMain", "TestMain"},
		{"TestMain", "TestMain"},
	}
	for _, v := range testCases {
		actual := GetBaseFuncName(v.input)
		if v.expected != actual {
			t.Fatalf("Expect %q but actual is %q", v.expected, actual)
		}
	}
}

func TestObjectNameForTest(t *testing.T) {
	testCases := []struct {
		input          testNamed
		expectedPrefix string
	}{
		{testNamed{name: "TestFooBar"}, "foo-bar-"},
		{testNamed{name: "Foo-bar"}, "foo-bar-"},
		{testNamed{name: "with_underscore"}, "with-underscore-"},
		{testNamed{name: "WithHTTP"}, "with-http-"},
		{testNamed{name: "ANameExceedingTheLimitLength-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, "a-name-exceeding-the-limit-length-aaaaaaa-"},
	}
	for _, v := range testCases {
		actual := ObjectNameForTest(&v.input)
		if !strings.HasPrefix(actual, v.expectedPrefix) {
			t.Fatalf("Expect prefix %q but actual is %q", v.expectedPrefix, actual)
		}
	}
}

type testNamed struct {
	name string
}

func (n *testNamed) Name() string {
	return n.name
}
