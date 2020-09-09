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
	"go.uber.org/zap"
)

func (s *Suite) Execute(c Configuration) {
	l := c.logger()
	se := suiteExecution{
		suite:         s,
		configuration: c,
		failed:        false,
		logger:        l,
		stoppables:    make([]stoppable, 0),
	}
	l.Info("🏃 Running upgrade test suite...")

	se.execute()

	if !se.failed {
		l.Info("🥳🎉 Success! Upgrade suite completed without errors.")
	} else {
		l.Error("💣🤬💔️ Upgrade suite have failed!")
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

func (s *StopEvent) Name() string {
	return s.name
}
