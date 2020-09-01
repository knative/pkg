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

const skippingOperationTemplate = `Skipping "%s" as previous operation have failed`

func (s *Suite) Execute(c Configuration) {
	l := c.logger()
	se := suiteExecution{
		suite:         s,
		configuration: c,
		failed:        false,
		logger:        l,
		stopSignals:   make([]StopSignal, 0),
	}
	l.Info("🏃 Running upgrade suite...")

	for i, operation := range []func(num int){
		se.installingBase,
		se.preUpgradeTests,
		se.startContinualTests,
		se.upgradeWith,
		se.postUpgradeTests,
		se.downgradeWith,
		se.postDowngradeTests,
		se.verifyContinualTests,
	} {
		operation(i + 1)
		if se.failed {
			return
		}
	}

	if !se.failed {
		l.Info("🥳🎉 Success! Upgrade suite completed without errors.")
	}
}

// NewOperation creates a new upgrade operation or test
func NewOperation(name string, handler func(c Context)) Operation {
	return &operationHolder{name: name, handler: handler}
}

// NewBackgroundOperation creates a new upgrade operation or test that can be
// notified to stop operating
func NewBackgroundOperation(
	name string,
	setup func(c Context),
	handler func(bc BackgroundContext),
) BackgroundOperation {
	return &backgroundOperationHolder{
		name:    name,
		setup:   setup,
		handler: handler,
	}
}

func (c Configuration) logger() *zap.SugaredLogger {
	return c.Log.Sugar()
}

type suiteExecution struct {
	suite         *Suite
	configuration Configuration
	failed        bool
	logger        *zap.SugaredLogger
	stopSignals   []StopSignal
}

func (se *suiteExecution) processOperationGroup(op operationGroup) {
	l := se.logger
	se.configuration.T.Run(op.groupName, func(t *testing.T) {
		if len(op.operations) > 0 {
			l.Infof(op.groupTemplate, op.num, len(op.operations))
			for i, operation := range op.operations {
				l.Infof(op.elementTemplate, op.num, i+1, operation.Name())
				if se.failed {
					l.Debugf(skippingOperationTemplate, operation.Name())
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

func (s *StopSignal) String() string {
	return s.name
}
