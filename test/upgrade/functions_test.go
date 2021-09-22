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

import "testing"

const (
	upgradeTestRunning = "🏃 Running upgrade test suite..."
	upgradeTestSuccess = "🥳🎉 Success! Upgrade suite completed without errors."
	upgradeTestFailure = "💣🤬💔️ Upgrade suite have failed!"
)

func TestExpectedTextsForEmptySuite(t *testing.T) {
	assert := assertions{t: t}
	fp := notFailing
	suite := emptySuiteExample()
	txt := expectedTexts(suite, fp)
	expected := []string{
		"1) 💿 No base installation registered. Skipping.",
		"2) ✅️️ No pre upgrade tests registered. Skipping.",
		"3) 🔄 No continual tests registered. Skipping.",
		"4) 📀 No upgrade operations registered. Skipping.",
		"5) ✅️️ No post upgrade tests registered. Skipping.",
		"6) 💿 No downgrade operations registered. Skipping.",
		"7) ✅️️ No post downgrade tests registered. Skipping.",
	}
	assert.arraysEqual(txt.elms, expected)
}

func TestExpectedTextsForCompleteSuite(t *testing.T) {
	assert := assertions{t: t}
	fp := notFailing
	suite := completeSuiteExample(fp)
	txt := expectedTexts(suite, fp)
	expected := []string{
		"1) 💿 Installing base installations. 2 are registered.",
		`1.1) Installing base install of "Serving latest stable release".`,
		`1.2) Installing base install of "Eventing latest stable release".`,
		"2) ✅️️ Testing functionality before upgrade is performed. 2 tests are registered.",
		`2.1) Testing with "Serving pre upgrade test".`,
		`2.2) Testing with "Eventing pre upgrade test".`,
		"3) 🔄 Starting continual tests. 2 tests are registered.",
		`3.1) Starting continual tests of "Serving continual test".`,
		`3.2) Starting continual tests of "Eventing continual test".`,
		"4) 📀 Upgrading with 2 registered operations.",
		`4.1) Upgrading with "Serving HEAD".`,
		`4.2) Upgrading with "Eventing HEAD".`,
		"5) ✅️️ Testing functionality after upgrade is performed. 2 tests are registered.",
		`5.1) Testing with "Serving post upgrade test".`,
		`5.2) Testing with "Eventing post upgrade test".`,
		"6) 💿 Downgrading with 2 registered operations.",
		`6.1) Downgrading with "Serving latest stable release".`,
		`6.2) Downgrading with "Eventing latest stable release".`,
		"7) ✅️️ Testing functionality after downgrade is performed. 2 tests are registered.",
		`7.1) Testing with "Serving post downgrade test".`,
		`7.2) Testing with "Eventing post downgrade test".`,
		"8) ✋ Verifying 2 running continual tests.",
		`8.1) Verifying "Serving continual test".`,
		`8.2) Verifying "Eventing continual test".`,
	}
	assert.arraysEqual(txt.elms, expected)
}

func TestExpectedTextsForFailingCompleteSuite(t *testing.T) {
	assert := assertions{t: t}
	fp := failurePoint{
		step:    2,
		element: 1,
	}
	suite := completeSuiteExample(fp)
	txt := expectedTexts(suite, fp)
	expected := []string{
		"1) 💿 Installing base installations. 2 are registered.",
		`1.1) Installing base install of "Serving latest stable release".`,
		`1.2) Installing base install of "Eventing latest stable release".`,
		"2) ✅️️ Testing functionality before upgrade is performed. 2 tests are registered.",
		`2.1) Testing with "FailingOfServing pre upgrade test".`,
	}
	assert.arraysEqual(txt.elms, expected)
}

func TestSuiteExecuteEmpty(t *testing.T) {
	assert := assertions{t: t}
	c, buf := newConfig(t)
	fp := notFailing
	suite := emptySuiteExample()
	suite.Execute(c)
	output := buf.String()
	if c.T.Failed() {
		return
	}

	txt := expectedTexts(suite, fp)
	txt.append(upgradeTestRunning, upgradeTestSuccess)

	assert.textContains(output, txt)
}

func TestSuiteExecuteWithComplete(t *testing.T) {
	assert := assertions{t: t}
	c, buf := newConfig(t)
	fp := notFailing
	suite := completeSuiteExample(fp)
	suite.Execute(c)
	output := buf.String()
	if c.T.Failed() {
		return
	}
	expected := expectedTexts(suite, fp)
	expected.append(upgradeTestRunning, upgradeTestSuccess)
	expected.append(
		"Installing Serving stable 0.17.1",
		"Installing Eventing stable 0.17.2",
		"Installing Serving HEAD at e3c4563",
		"Installing Eventing HEAD at 12f67cc",
		"Installing Serving stable 0.17.1",
		"Installing Eventing stable 0.17.2",
	)
	assert.textContains(output, expected)

	unexpected := texts{elms: nil}
	unexpected.append(
		"Running Serving continual test",
		"Stopping and verify of Eventing continual test",
		"Serving have received a stop event",
		"Eventing continual test have received a stop event",
		"Serving - probing functionality...",
		"Eventing continual test - probing functionality...",
	)
	assert.textDoesNotContain(output, unexpected)
}
