/*
Copyright 2021 The Knative Authors

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
	"testing"

	"knative.dev/pkg/test/upgrade"
)

func TestSkipAtBackgroundVerification(t *testing.T) {
	config, buf := newConfig(t)
	skipMsg := "It is expected to be skipped"
	expectedTexts := []string{
		upgradeTestRunning,
		"DEBUG\tFinished \"ShouldNotBeSkipped\"",
		upgradeTestSuccess,
		"INFO\tSetup 1",
		"INFO\tSetup 2",
		"INFO\tVerify 2",
		"WARN\t" + skipMsg,
	}
	s := upgrade.Suite{
		Tests: upgrade.Tests{
			Continual: []upgrade.BackgroundOperation{
				upgrade.NewBackgroundVerification("ShouldBeSkipped",
					// Setup
					func(c upgrade.Context) {
						c.Log.Info("Setup 1")
					},
					// Verify
					func(c upgrade.Context) {
						c.Log.Warn(skipMsg)
						c.T.Skip(skipMsg)
						c.Log.Info("Verify 1")
					},
				),
				upgrade.NewBackgroundVerification("ShouldNotBeSkipped",
					func(c upgrade.Context) {
						c.Log.Info("Setup 2")
					},
					func(c upgrade.Context) {
						c.Log.Info("Verify 2")
					},
				),
			},
		},
	}
	s.Execute(config)

	assert := assertions{tb: t}

	out := buf.String()
	assert.textNotContains(out, texts{elms: []string{
		"INFO\tVerify 1",
	}})
	assert.textContains(out, texts{elms: expectedTexts})
}

func TestFailAtBackgroundVerification(t *testing.T) {
	const failingVerification = "FailAtVerification"
	expectedTexts := []string{
		upgradeTestRunning,
		"INFO\tSetup 1",
		"INFO\tVerify 1",
	}
	s := upgrade.Suite{
		Tests: upgrade.Tests{
			Continual: []upgrade.BackgroundOperation{
				upgrade.NewBackgroundVerification(failingVerification,
					// Setup
					func(c upgrade.Context) {
						c.Log.Info("Setup 1")
					},
					// Verify
					func(c upgrade.Context) {
						c.Log.Info("Verify 1")
						c.T.Fatal(failureTestingMessage)
						c.Log.Info("Verify 2")
					},
				),
			},
		},
	}
	var (
		buf fmt.Stringer
		c   upgrade.Configuration
		ok  bool
	)
	it := []testing.InternalTest{{
		Name: t.Name(),
		F: func(t *testing.T) {
			c, buf = newConfig(t)
			s.Execute(c)
		},
	}}
	testOutput := captureStdOutput(func() {
		ok = testing.RunTests(allTestsFilter, it)
	})
	if ok {
		t.Fatal("Didn't fail, but should")
	}
	out := buf.String()
	assert := assertions{tb: t}
	assert.textNotContains(out, texts{elms: []string{
		"INFO\tVerify 2",
	}})
	assert.textContains(out, texts{elms: expectedTexts})
	assert.textContains(testOutput, texts{
		elms: []string{
			fmt.Sprintf("--- FAIL: %s/Parallel/%s", t.Name(), failingVerification),
		},
	})
}
