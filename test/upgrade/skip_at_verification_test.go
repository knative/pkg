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
	"testing"

	"knative.dev/pkg/test/upgrade"
)

func TestSkipAtBackgroundVerification(t *testing.T) {
	config, buf := newConfig(t)
	bgLog, bgBuf := newBackgroundTestLogger(t)
	skipMsg := "It is expected to be skipped"
	s := upgrade.Suite{
		Tests: upgrade.Tests{
			Continual: []upgrade.BackgroundOperation{
				upgrade.NewBackgroundVerification("ShouldBeSkipped",
					func(c upgrade.Context) {
						bgLog.Info("Setup 1")
					},
					func(c upgrade.Context) {
						bgLog.Warn(skipMsg)
						c.T.Skip(skipMsg)
					},
				),
				upgrade.NewBackgroundVerification("ShouldNotBeSkipped",
					func(c upgrade.Context) {
						bgLog.Info("Setup 2")
					},
					func(c upgrade.Context) {
						bgLog.Info("Verify 2")
					},
				),
			},
		},
	}
	s.Execute(config)

	assert := assertions{t: t}

	out := buf.String()
	assert.textContains(out, texts{elms: []string{
		upgradeTestRunning,
		"DEBUG\tFinished \"ShouldNotBeSkipped\"",
		upgradeTestSuccess,
	}})

	bgOut := bgBuf.String()
	assert.textContains(bgOut, texts{elms: []string{
		"INFO\tSetup 1",
		"INFO\tSetup 2",
		"WARN\t" + skipMsg,
		"INFO\tVerify 2",
	}})
}
