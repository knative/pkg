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

	"go.uber.org/zap"
)

func (s *Suite) Execute(c Configuration) {
	l := c.logger()
	se := suiteExecution{
		suite:         s,
		configuration: c,
		failed:        false,
		logger:        l,
	}
	l.Info("🏃 Running upgrade suite...")

	for i, operation := range []func(num int){
		se.installingBase,
		se.preUpgradeTests,
	} {
		operation(i + 1)
		if se.failed {
			return
		}
	}

	if !se.failed {
		l.Info("🥳🎉 Success! Upgrade suite completed without errors.")
	}

	if len(s.Tests.PreUpgrade) > 0 {
		texts := []string{
			"3) 🔄 Staring continual tests to run in background. 2 tests are registered.",
			`3.1) Staring continual tests of "Serving continual test"`,
			"Running Serving continual test",
			`3.2) Staring continual tests of "Eventing continual test"`,
			"Running Eventing continual test",
			"4) 📀 Upgrading with 2 registered installations.",
			`4.1) Upgrading with "Serving HEAD"`,
			"Installing Serving HEAD at e3c4563",
			`4.1) Upgrading with "Eventing HEAD"`,
			"Installing Eventing HEAD at 12f67cc",
			"5) ✅️️ Testing functionality after upgrade is performed. 2 tests are registered.",
			`5.1) Testing with "Serving post upgrade test"`,
			`5.2) Testing with "Eventing post upgrade test"`,
			"6) 💿 Downgrading with 2 registered installations.",
			`6.1) Downgrading with "Serving latest stable release".`,
			"Installing Serving stable 0.17.1",
			`6.2) Downgrading with "Eventing latest stable release".`,
			"Installing Eventing stable 0.17.2",
			"7) ✅️️ Testing functionality after downgrade is performed. 2 tests are registered.",
			`7.1) Testing with "Serving post downgrade test"`,
			`7.2) Testing with "Eventing post downgrade test"`,
			"8) ✋ Stopping 2 running continual tests",
			`8.1) Stopping "Serving continual test"`,
			"Serving probe test have received a stop message",
			`8.2) Stopping "Eventing continual test"`,
			"Eventing probe test have received a stop message",
		}
		for _, text := range texts {
			c.Log.Info(text)
		}
	} else {
		texts := []string{
			"3) 🔄 No continual tests registered. Skipping.",
			"4) 📀 No upgrade installations registered. Skipping.",
			"5) ✅️️ No post upgrade tests registered. Skipping.",
			"6) 💿 No downgrade installations registered. Skipping.",
			"7) ✅️️ No post downgrade tests registered. Skipping.",
		}
		for _, text := range texts {
			c.Log.Info(text)
		}
	}
}

// NewOperation creates a new upgrade operation or test
func NewOperation(name string, handler func(c Context)) Operation {
	return &operationHolder{name: name, handler: handler}
}

// NewBackgroundOperation creates a new upgrade operation or test that can be
// notified to stop operating
func NewBackgroundOperation(name string, handler func(bc BackgroundContext)) BackgroundOperation {
	return &stoppableOperationHolder{name: name, handler: handler}
}

func (c Configuration) logger() *zap.SugaredLogger {
	return c.Log.Sugar()
}

type suiteExecution struct {
	suite         *Suite
	configuration Configuration
	failed        bool
	logger        *zap.SugaredLogger
}

func (se *suiteExecution) processOperationGroup(op operationGroup) {
	l := se.logger
	se.configuration.T.Run(op.groupName, func(t *testing.T) {
		if len(op.operations) > 0 {
			l.Infof(op.groupTemplate, op.num, len(op.operations))
			for i, operation := range op.operations {
				l.Infof(op.elementTemplate, op.num, i+1, operation.Name())
				if se.failed {
					l.Debugf(`Skipping "%s" as previous operation have failed`, operation.Name())
					return
				}
				handler := operation.Handler()
				t.Run(operation.Name(), func(t *testing.T) {
					handler(Context{T: t, Log: l})
				})
				se.failed = se.failed || t.Failed()
				if se.failed {
					return
				}
			}
		} else {
			l.Infof(op.skippingGroupTemplate, op.num)
		}
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

type stoppableOperationHolder struct {
	name    string
	handler func(bc BackgroundContext)
}

func (h *operationHolder) Name() string {
	return h.name
}

func (h *operationHolder) Handler() func(c Context) {
	return h.handler
}

func (s *stoppableOperationHolder) Name() string {
	return s.name
}

func (s *stoppableOperationHolder) Handler() func(bc BackgroundContext) {
	return s.handler
}
