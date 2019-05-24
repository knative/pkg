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
	"testing"
)

var matcher = regexp.MustCompile("abcd-[a-z]{8}")

func TestAppendRandomString(tst *testing.T) {
	const s = "abcd"
	t := AppendRandomString(s)
	o := AppendRandomString(s)
	if !matcher.MatchString(t) || !matcher.MatchString(o) || o == t {
		tst.Fatal()
	}
}

func TestMakeK8sNamePrefix(tst *testing.T) {
	testCases := map[string]string{
		"abcd123":  "abcd123",
		"AbCdef":   "ab-cdef",
		"ABCD":     "a-b-c-d",
		"aBc*ef&d": "a-bc-ef-d",
	}
	for k := range testCases {
		expected := testCases[k]
		actual := MakeK8sNamePrefix(k)
		if expected != actual {
			tst.Fatalf("Expect %q but actual is %q", expected, actual)
		}
	}
}

func TestGetBaseFuncName(tst *testing.T) {
	testCases := map[string]string{
		"test/e2e.TestMain": "TestMain",
		"e2e.TestMain":      "TestMain",
		"test/TestMain":     "TestMain",
		"TestMain":          "TestMain",
	}
	for k := range testCases {
		expected := testCases[k]
		actual := GetBaseFuncName(k)
		if expected != actual {
			tst.Fatalf("Expect %q but actual is %q", expected, actual)
		}
	}
}
