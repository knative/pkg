/*
 * Copyright 2020 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package upgrade

import (
	"testing"
)

const skippingOperationTemplate = `Skipping "%s" as previous operation have failed`

func (se *suiteExecution) installingBase(num int) {
	se.processOperationGroup(operationGroup{
		num:                   num,
		operations:            se.suite.Installations.Base,
		groupName:             "InstallingBase",
		elementTemplate:       `%d.%d) Installing base install of "%s".`,
		skippingGroupTemplate: "%d) 💿 No base installation registered. Skipping.",
		groupTemplate:         "%d) 💿 Installing base installations. %d are registered.",
	})
}

func (se *suiteExecution) preUpgradeTests(num int) {
	se.processOperationGroup(operationGroup{
		num:                   num,
		operations:            se.suite.Tests.PreUpgrade,
		groupName:             "PreUpgradeTests",
		elementTemplate:       `%d.%d) Testing with "%s".`,
		skippingGroupTemplate: "%d) ✅️️ No pre upgrade tests registered. Skipping.",
		groupTemplate: "%d) ✅️️ Testing functionality before upgrade is performed." +
			" %d tests are registered.",
	})
}

func (se *suiteExecution) startContinualTests(num int) {
	l := se.logger
	operations := se.suite.Tests.ContinualTests
	groupTemplate := "%d) 🔄 Starting continual tests to run in background. " +
		"%d tests are registered."
	elementTemplate := `%d.%d) Starting continual tests of "%s".`
	numOps := len(operations)
	se.configuration.T.Run("ContinualTests", func(t *testing.T) {
		if numOps > 0 {
			l.Infof(groupTemplate, num, numOps)
			for i, operation := range operations {
				l.Infof(elementTemplate, num, i+1, operation.Name())
				if se.failed {
					l.Debugf(skippingOperationTemplate, operation.Name())
					return
				}
				setup := operation.Setup()
				t.Run("Setup"+operation.Name(), func(t *testing.T) {
					setup(Context{T: t, Log: l})
				})
				if se.failed {
					l.Debugf(skippingOperationTemplate, operation.Name())
					return
				}
				stop := make(chan StopSignal)
				se.stopSignals = append(se.stopSignals, StopSignal{
					T:       nil,
					name:    operation.Name(),
					channel: stop,
				})
				handler := operation.Handler()
				go func() {
					bc := BackgroundContext{Log: l, Stop: stop}
					handler(bc)
				}()

				se.failed = se.failed || t.Failed()
				if se.failed {
					return
				}
			}

		} else {
			l.Infof("%d) 🔄 No continual tests registered. Skipping.", num)
		}
	})
}

func (se *suiteExecution) verifyContinualTests(num int) {
	l := se.logger
	testsCount := len(se.suite.Tests.ContinualTests)
	if testsCount > 0 {
		se.configuration.T.Run("VerifyContinualTests", func(t *testing.T) {
			l.Infof("%d) ✋ Verifying %d running continual tests.", num, testsCount)
			for i, signal := range se.stopSignals {
				t.Run(signal.name, func(t *testing.T) {
					l.Infof(`%d.%d) Verifying "%s".`, num, i+1, signal.name)
					signal.T = t
					signal.Finished = make(chan int)
					signal.channel <- signal
					retcode := <-signal.Finished
					se.failed = se.failed || t.Failed()
					l.Debugf(`Finished "%s" with: %d`, signal.name, retcode)
				})
			}
		})
	}
}

func (se *suiteExecution) upgradeWith(num int) {
	se.processOperationGroup(operationGroup{
		num:                   num,
		operations:            se.suite.Installations.UpgradeWith,
		groupName:             "UpgradeWith",
		elementTemplate:       `%d.%d) Upgrading with "%s".`,
		skippingGroupTemplate: "%d) 📀 No upgrade operations registered. Skipping.",
		groupTemplate:         "%d) 📀 Upgrading with %d registered operations.",
	})
}

func (se *suiteExecution) postUpgradeTests(num int) {
	se.processOperationGroup(operationGroup{
		num:                   num,
		operations:            se.suite.Tests.PostUpgrade,
		groupName:             "PostUpgradeTests",
		elementTemplate:       `%d.%d) Testing with "%s".`,
		skippingGroupTemplate: "%d) ✅️️ No post upgrade tests registered. Skipping.",
		groupTemplate: "%d) ✅️️ Testing functionality after upgrade is performed." +
			" %d tests are registered.",
	})
}

func (se *suiteExecution) downgradeWith(num int) {
	se.processOperationGroup(operationGroup{
		num:                   num,
		operations:            se.suite.Installations.DowngradeWith,
		groupName:             "DowngradeWith",
		elementTemplate:       `%d.%d) Downgrading with "%s".`,
		skippingGroupTemplate: "%d) 💿 No downgrade operations registered. Skipping.",
		groupTemplate:         "%d) 💿 Downgrading with %d registered operations.",
	})
}

func (se *suiteExecution) postDowngradeTests(num int) {
	se.processOperationGroup(operationGroup{
		num:                   num,
		operations:            se.suite.Tests.PostDowngrade,
		groupName:             "PostDowngradeTests",
		elementTemplate:       `%d.%d) Testing with "%s".`,
		skippingGroupTemplate: "%d) ✅️️ No post downgrade tests registered. Skipping.",
		groupTemplate: "%d) ✅️️ Testing functionality after downgrade is performed." +
			" %d tests are registered.",
	})
}

type operationGroup struct {
	num                   int
	operations            []Operation
	groupName             string
	groupTemplate         string
	elementTemplate       string
	skippingGroupTemplate string
}

type operationHolder struct {
	name    string
	handler func(c Context)
}

type backgroundOperationHolder struct {
	name    string
	setup   func(c Context)
	handler func(bc BackgroundContext)
}

func (h *operationHolder) Name() string {
	return h.name
}

func (h *operationHolder) Handler() func(c Context) {
	return h.handler
}

func (s *backgroundOperationHolder) Name() string {
	return s.name
}

func (s *backgroundOperationHolder) Setup() func(c Context) {
	return s.setup
}

func (s *backgroundOperationHolder) Handler() func(bc BackgroundContext) {
	return s.handler
}
