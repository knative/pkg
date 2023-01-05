/*
Copyright 2020 The Knative Authors

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

package upgrade_test

import (
	"fmt"
	"io"
	"os"
	"testing"

	"knative.dev/pkg/test/upgrade"
)

func TestSuiteExecuteWithFailures(t *testing.T) {
	suite := completeSuite()
	for i, st := range createSteps(suite) {
		for j := range st.ops.ops {
			fp := failurePoint{
				step:    i + 1,
				element: j + 1,
			}
			testSuiteExecuteWithFailingStep(suite, fp, t)
		}
	}
}

var allTestsFilter = func(_, _ string) (bool, error) { return true, nil }

func testSuiteExecuteWithFailingStep(suite upgrade.Suite, fp failurePoint, t *testing.T) {
	testName := fmt.Sprintf("FailAt-%d-%d", fp.step, fp.element)
	t.Run(testName, func(t *testing.T) {
		assert := assertions{tb: t}
		var (
			output string
			c      upgrade.Configuration
			buf    fmt.Stringer
		)
		suiteWithFailures := enrichSuiteWithFailures(suite, fp)
		txt := expectedTexts(suiteWithFailures, fp)
		txt.append(upgradeTestRunning, upgradeTestFailure)

		it := []testing.InternalTest{{
			Name: testName,
			F: func(t *testing.T) {
				c, buf = newConfig(t)
				suiteWithFailures.Execute(c)
			},
		}}
		var ok bool
		testOutput := captureStdOutput(func() {
			ok = testing.RunTests(allTestsFilter, it)
		})
		output = buf.String()

		if ok {
			t.Fatal("Didn't fail, but should")
		}

		assert.textContains(output, txt)
		assert.textContains(testOutput, texts{
			elms: []string{
				fmt.Sprintf("--- FAIL: FailAt-%d-%d", fp.step, fp.element),
			},
		})
	})
}

func captureStdOutput(call func()) string {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = rescueStdout
	}()

	call()

	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}
