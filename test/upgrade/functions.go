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

package upgrade

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"

	"go.uber.org/zap"
)

// Execute the Suite of upgrade tests with a Configuration given.
func (s *Suite) Execute(c Configuration) {
	l, err := c.logger()
	if err != nil {
		l.Error("Failed to build logger")
		return
	}
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
		WaitForStopEvent(bc, WaitForStopEventConfiguration{
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
func WaitForStopEvent(bc BackgroundContext, w WaitForStopEventConfiguration) {
	for {
		select {
		case stopEvent := <-bc.Stop:
			handleStopEvent(stopEvent, bc, w)
			return
		default:
			w.OnWait(bc, w)
		}
		time.Sleep(w.WaitTime)
	}
}

func (c Configuration) logger() (*zap.SugaredLogger, error) {
	if c.LogConfig != nil {
		var (
			log *zap.Logger
			err error
		)
		if c.LogConfig.Build != nil {
			// Build the logger using the provided config and build function.
			log, err = c.LogConfig.Build(c.LogConfig.Config)
		} else {
			log, err = c.LogConfig.Config.Build()
		}
		if err != nil {
			return nil, err
		}
		return log.Sugar(), nil
	}
	return c.Log.Sugar(), nil
}

// Name returns a friendly human readable text.
func (s *StopEvent) Name() string {
	return s.name
}

func handleStopEvent(
	se StopEvent,
	bc BackgroundContext,
	wc WaitForStopEventConfiguration,
) {
	bc.Log.Infof("%s have received a stop event: %s", wc.Name, se.Name())
	defer close(se.Finished)
	wc.OnStop(se)
	se.T.Log(wrapLogs(bc.logBuffer))
}

func enrichSuite(s *Suite) *enrichedSuite {
	es := &enrichedSuite{
		installations: s.Installations,
		tests: enrichedTests{
			preUpgrade:    s.Tests.PreUpgrade,
			postUpgrade:   s.Tests.PostUpgrade,
			postDowngrade: s.Tests.PostDowngrade,
			continual:     make([]stoppableOperation, len(s.Tests.Continual)),
		},
	}
	for i, test := range s.Tests.Continual {
		es.tests.continual[i] = stoppableOperation{
			BackgroundOperation: test,
			stop:                make(chan StopEvent),
		}
	}
	return es
}

// Name is a human readable operation title, and it will be used in t.Run.
func (h *simpleOperation) Name() string {
	return h.name
}

// Handler is a function that will be called to perform an operation.
func (h *simpleOperation) Handler() func(c Context) {
	return h.handler
}

// Name is a human readable operation title, and it will be used in t.Run.
func (s *simpleBackgroundOperation) Name() string {
	return s.name
}

// Setup method may be used to set up environment before upgrade/downgrade is
// performed.
func (s *simpleBackgroundOperation) Setup() func(c Context) {
	return s.setup
}

// Handler will be executed in background while upgrade/downgrade is being
// executed. It can be used to constantly validate environment during that
// time and/or wait for StopEvent being sent. After StopEvent is received
// user should validate environment, clean up resources, and report found
// issues to testing.T forwarded in StepEvent.
func (s *simpleBackgroundOperation) Handler() func(bc BackgroundContext) {
	return s.handler
}

func (b *ThreadSafeBuffer) Read(p []byte) (n int, err error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.Read(p)
}

func (b *ThreadSafeBuffer) Write(p []byte) (n int, err error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.Write(p)
}

func (b *ThreadSafeBuffer) String() string {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.String()
}

// newInMemoryLoggerBuffer creates a logger that writes logs into a byte buffer.
// This byte buffer is returned and can be used to process the logs at later stage.
func newInMemoryLoggerBuffer(logConfig *LogConfig) (*zap.Logger, *ThreadSafeBuffer) {
	buf := &ThreadSafeBuffer{}
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(logConfig.Config.EncoderConfig),
		zapcore.AddSync(buf),
		logConfig.Config.Level)

	opts := []zap.Option{}
	if logConfig.Config.Development {
		opts = append(opts, zap.Development())
	}
	if !logConfig.Config.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}
	stackLevel := zap.ErrorLevel
	if logConfig.Config.Development {
		stackLevel = zap.WarnLevel
	}
	if !logConfig.Config.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(stackLevel))
	}
	return zap.New(core, opts...), buf
}

func wrapLogs(stringer fmt.Stringer) string {
	return fmt.Sprintf("\n=== Background Log Start ===\n%s=== Background Log End ===", stringer)
}
