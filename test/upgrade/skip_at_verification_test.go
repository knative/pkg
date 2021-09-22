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
	log, buf := upgrade.NewInMemoryLoggerBuffer()
	skipMsg := "It is expected to be skipped"
	s := upgrade.Suite{
		Tests: upgrade.Tests{
			Continual: []upgrade.BackgroundOperation{
				upgrade.NewBackgroundVerification("ShouldBeSkipped",
					func(c upgrade.Context) {
						log.Info("Setup 1")
					},
					func(c upgrade.Context) {
						log.Warn(skipMsg)
						c.T.Skip(skipMsg)
					},
				),
				upgrade.NewBackgroundVerification("ShouldNotBeSkipped",
					func(c upgrade.Context) {
						log.Info("Setup 2")
					},
					func(c upgrade.Context) {
						log.Info("Verify 2")
					},
				),
			},
		},
	}
	s.Execute(upgrade.Configuration{
		T:   t,
		Log: log,
	})
	out := buf.String()
	assert := assertions{t: t}
	assert.textContains(out, texts{elms: []string{
		upgradeTestRunning,
		"INFO\tSetup 1",
		"INFO\tSetup 2",
		"WARN\t" + skipMsg,
		"INFO\tVerify 2",
		"DEBUG\tFinished \"ShouldNotBeSkipped\"",
		upgradeTestSuccess,
	}})
}
