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
	"bytes"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"knative.dev/pkg/test/upgrade"
)

const (
	failureTestingMessage = "This error is expected to be seen. Upgrade suite should fail."
)

func newConfig(t *testing.T) (upgrade.Configuration, fmt.Stringer) {
	buf := threadSafeBuffer{}
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = ""
	cfg.EncoderConfig.CallerKey = ""
	syncedBuf := zapcore.AddSync(&buf)
	logConfig := upgrade.LogConfig{
		Config: cfg,
		Options: []zap.Option{
			zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return zapcore.NewCore(
					zapcore.NewConsoleEncoder(cfg.EncoderConfig),
					zapcore.NewMultiWriteSyncer(syncedBuf), cfg.Level)
			}),
			zap.ErrorOutput(syncedBuf),
		},
	}
	logger, err := cfg.Build(logConfig.Options...)
	if err != nil {
		t.Fatal("Unable to create logger:", err)
	}
	c := upgrade.Configuration{
		T:         t,
		Log:       logger,
		LogConfig: logConfig,
	}
	return c, &buf
}

// threadSafeBuffer avoids race conditions on bytes.Buffer.
// See: https://stackoverflow.com/a/36226525/844449
type threadSafeBuffer struct {
	bytes.Buffer
	sync.Mutex
}

func (b *threadSafeBuffer) Read(p []byte) (n int, err error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.Read(p)
}

func (b *threadSafeBuffer) Write(p []byte) (n int, err error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.Write(p)
}

func createSteps(s upgrade.Suite) []*step {
	continualTestsGeneralized := generalizeOpsFromBg(s.Tests.Continual)
	return []*step{{
		messages: messageFormatters.baseInstall,
		ops:      generalizeOps(s.Installations.Base),
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Installations.Base = ops.asOperations()
		},
	}, {
		messages: messageFormatters.preUpgrade,
		ops:      generalizeOps(s.Tests.PreUpgrade),
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Tests.PreUpgrade = ops.asOperations()
		},
	}, {
		messages: messageFormatters.startContinual,
		ops:      continualTestsGeneralized,
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Tests.Continual = ops.asBackgroundOperation()
		},
	}, {
		messages: messageFormatters.upgrade,
		ops:      generalizeOps(s.Installations.UpgradeWith),
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Installations.UpgradeWith = ops.asOperations()
		},
	}, {
		messages: messageFormatters.postUpgrade,
		ops:      generalizeOps(s.Tests.PostUpgrade),
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Tests.PostUpgrade = ops.asOperations()
		},
	}, {
		messages: messageFormatters.downgrade,
		ops:      generalizeOps(s.Installations.DowngradeWith),
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Installations.DowngradeWith = ops.asOperations()
		},
	}, {
		messages: messageFormatters.postDowngrade,
		ops:      generalizeOps(s.Tests.PostDowngrade),
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Tests.PostDowngrade = ops.asOperations()
		},
	}}
}

func expectedTexts(s upgrade.Suite, fp failurePoint) texts {
	steps := createSteps(s)
	tt := texts{elms: nil}
	for i, st := range steps {
		stepIdx := i + 1
		if st.ops.length() == 0 {
			tt.append(st.skipped(stepIdx))
		} else {
			tt.append(st.starting(stepIdx, st.ops.length()))
			for j, op := range st.ops.ops {
				elemIdx := j + 1
				tt.append(st.element(stepIdx, elemIdx, op.Name()))
				if fp.step == stepIdx && fp.element == elemIdx {
					return tt
				}
			}
		}
	}
	return tt
}

func generalizeOps(ops []upgrade.Operation) operations {
	gen := make([]*operation, len(ops))
	for idx, op := range ops {
		gen[idx] = &operation{op: op}
	}
	return operations{ops: gen}
}

func generalizeOpsFromBg(ops []upgrade.BackgroundOperation) operations {
	gen := make([]*operation, len(ops))
	for idx, op := range ops {
		gen[idx] = &operation{bg: op}
	}
	return operations{ops: gen}
}

func createMessages(mf formats) messages {
	return messages{
		skipped: func(args ...interface{}) string {
			empty := ""
			if mf.skipped == empty {
				return empty
			}
			return fmt.Sprintf(mf.skipped, args...)
		},
		starting: func(args ...interface{}) string {
			return fmt.Sprintf(mf.starting, args...)
		},
		element: func(args ...interface{}) string {
			return fmt.Sprintf(mf.element, args...)
		},
	}
}

func (tt *texts) append(messages ...string) {
	for _, msg := range messages {
		if msg == "" {
			continue
		}
		tt.elms = append(tt.elms, msg)
	}
}

func completeSuiteExample(fp failurePoint) upgrade.Suite {
	serving := servingComponent()
	eventing := eventingComponent()
	suite := upgrade.Suite{
		Tests: upgrade.Tests{
			PreUpgrade: []upgrade.Operation{
				serving.tests.preUpgrade, eventing.tests.preUpgrade,
			},
			PostUpgrade: []upgrade.Operation{
				serving.tests.postUpgrade, eventing.tests.postUpgrade,
			},
			PostDowngrade: []upgrade.Operation{
				serving.tests.postDowngrade, eventing.tests.postDowngrade,
			},
			Continual: []upgrade.BackgroundOperation{
				serving.tests.continual, eventing.tests.continual,
			},
		},
		Installations: upgrade.Installations{
			Base: []upgrade.Operation{
				serving.installs.stable, eventing.installs.stable,
			},
			UpgradeWith: []upgrade.Operation{
				serving.installs.head, eventing.installs.head,
			},
			DowngradeWith: []upgrade.Operation{
				serving.installs.stable, eventing.installs.stable,
			},
		},
	}
	return enrichSuiteWithFailures(suite, fp)
}

func emptySuiteExample() upgrade.Suite {
	return upgrade.Suite{
		Tests:         upgrade.Tests{},
		Installations: upgrade.Installations{},
	}
}

func enrichSuiteWithFailures(suite upgrade.Suite, fp failurePoint) upgrade.Suite {
	steps := createSteps(suite)
	for i, st := range steps {
		for j, op := range st.ops.ops {
			if fp.step == i+1 && fp.element == j+1 {
				op.fail(fp.step == 3)
			}
		}
	}
	return recreateSuite(steps)
}

func recreateSuite(steps []*step) upgrade.Suite {
	suite := &upgrade.Suite{
		Tests:         upgrade.Tests{},
		Installations: upgrade.Installations{},
	}
	for _, st := range steps {
		st.updateSuite(st.ops, suite)
	}
	return *suite
}

func (o operation) Name() string {
	if o.op != nil {
		return o.op.Name()
	}
	return o.bg.Name()
}

func (o *operation) fail(setupFail bool) {
	testName := fmt.Sprintf("FailingOf%s", o.Name())
	if o.op != nil {
		prev := o.op
		o.op = upgrade.NewOperation(testName, func(c upgrade.Context) {
			handler := prev.Handler()
			handler(c)
			c.T.Error(failureTestingMessage)
			c.Log.Error(failureTestingMessage)
		})
	} else {
		prev := o.bg
		o.bg = upgrade.NewBackgroundOperation(testName, func(c upgrade.Context) {
			setup := prev.Setup()
			setup(c)
			if setupFail {
				c.T.Error(failureTestingMessage)
				c.Log.Error(failureTestingMessage)
			}
		}, func(bc upgrade.BackgroundContext) {
			upgrade.WaitForStopEvent(bc, upgrade.WaitForStopEventConfiguration{
				Name: testName,
				OnStop: func() {
					if !setupFail {
						bc.T.Error(failureTestingMessage)
						bc.Log.Error(failureTestingMessage)
					}
				},
				OnWait: func(bc upgrade.BackgroundContext, self upgrade.WaitForStopEventConfiguration) {
					bc.Log.Debugf("%s - probing functionality...", self.Name)
				},
				WaitTime: shortWait,
			})
		})
	}
}

func (o operations) length() int {
	return len(o.ops)
}

func (o operations) asOperations() []upgrade.Operation {
	ops := make([]upgrade.Operation, o.length())
	for i, op := range o.ops {
		ops[i] = op.op
	}
	return ops
}

func (o operations) asBackgroundOperation() []upgrade.BackgroundOperation {
	ops := make([]upgrade.BackgroundOperation, o.length())
	for i, op := range o.ops {
		ops[i] = op.bg
	}
	return ops
}
