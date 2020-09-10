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
	"time"

	"go.uber.org/zap"
)

func (s *Suite) Execute(c Configuration) {
	l := c.logger()
	se := suiteExecution{
		suite:         enrichSuite(s),
		configuration: c,
		failed:        false,
		logger:        l,
	}
	l.Info("🏃 Running upgrade test suite...")

	se.execute()

	if !se.failed {
		l.Info("🥳🎉 Success! Upgrade suite completed without errors.")
	} else {
		l.Error("💣🤬💔️ Upgrade suite have failed!")
	}
}

// NewOperation creates a new upgrade operation or test.
func NewOperation(name string, handler func(c Context)) Operation {
	return &simpleOperation{name: name, handler: handler}
}

// NewBackgroundVerification is convenience function to easily setup a
// background operation that will setup environment and then verify environment
// status after receiving a StopEvent.
func NewBackgroundVerification(name string, setup func(c Context), verify func(c Context)) BackgroundOperation {
	return NewBackgroundOperation(name, setup, func(bc BackgroundContext) {
		WaitForStopEvent(bc, WaitOnStopEventConfiguration{
			Name: name,
			OnStop: func(event StopEvent) {
				verify(Context{
					T:   event.T,
					Log: bc.Log,
				})
			},
			OnWait:   DefaultOnWait,
			WaitTime: DefaultWaitTime,
		})
	})
}

// NewBackgroundOperation creates a new background operation or test that can be
// notified to stop its operation.
func NewBackgroundOperation(name string, setup func(c Context),
	handler func(bc BackgroundContext)) BackgroundOperation {
	return &simpleBackgroundOperation{
		name:    name,
		setup:   setup,
		handler: handler,
	}
}

// WaitForStopEvent will wait until upgrade suite sends a stop event to it.
// After that happen a handler is invoked to verify environment state and report
// failures.
func WaitForStopEvent(bc BackgroundContext, w WaitOnStopEventConfiguration) {
	log := bc.Log
	for {
		select {
		case stopEvent := <-bc.Stop:
			log.Infof("%s have received a stop event: %s", w.Name, stopEvent.Name())
			w.OnStop(stopEvent)
			close(stopEvent.Finished)
			return
		default:
			w.OnWait(bc, w)
		}
		time.Sleep(w.WaitTime)
	}
}

func (c Configuration) logger() *zap.SugaredLogger {
	return c.Log.Sugar()
}

func (s *StopEvent) Name() string {
	return s.name
}

func enrichSuite(s *Suite) *enrichedSuite {
	es := &enrichedSuite{
		installations: s.Installations,
		tests: enrichedTests{
			preUpgrade:     s.Tests.PreUpgrade,
			postUpgrade:    s.Tests.PostUpgrade,
			postDowngrade:  s.Tests.PostDowngrade,
			continualTests: make([]stoppableOperation, len(s.Tests.ContinualTests)),
		},
	}
	for i, test := range s.Tests.ContinualTests {
		es.tests.continualTests[i] = stoppableOperation{
			BackgroundOperation: test,
			stop:                make(chan StopEvent),
		}
	}
	return es
}

func (h *simpleOperation) Name() string {
	return h.name
}

func (h *simpleOperation) Handler() func(c Context) {
	return h.handler
}

func (s *simpleBackgroundOperation) Name() string {
	return s.name
}

func (s *simpleBackgroundOperation) Setup() func(c Context) {
	return s.setup
}

func (s *simpleBackgroundOperation) Handler() func(bc BackgroundContext) {
	return s.handler
}
