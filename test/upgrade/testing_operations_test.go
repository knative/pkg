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

package upgrade_test

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"knative.dev/pkg/test/upgrade"
)

func newConfig(t *testing.T) (upgrade.Configuration, fmt.Stringer) {
	log, buf := newExampleZap()
	c := upgrade.Configuration{T: t, Log: log}
	return c, buf
}

func newExampleZap() (*zap.Logger, fmt.Stringer) {
	ec := zap.NewDevelopmentEncoderConfig()
	ec.TimeKey = ""
	encoder := zapcore.NewConsoleEncoder(ec)
	buf := &buffer{
		Buffer: bytes.Buffer{},
		Mutex:  sync.Mutex{},
		Syncer: zaptest.Syncer{},
	}
	ws := zapcore.NewMultiWriteSyncer(buf, os.Stdout)
	core := zapcore.NewCore(encoder, ws, zap.DebugLevel)
	return zap.New(core).WithOptions(), buf
}

func waitForStopSignal(bc upgrade.BackgroundContext, name string, handler func(sig upgrade.StopSignal) int) {
	for {
		select {
		case sig := <-bc.Stop:
			bc.Log.Infof(
				"%s probe test have received a stop message: %s",
				name, sig.String())
			sig.Finished <- handler(sig)
			return
		default:
			bc.Log.Debugf("Probing %s functionality...", name)
		}
		time.Sleep(shortWait)
	}
}

func createSteps(s upgrade.Suite) []*step {
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
		ops:      generalizeOpsFromBg(s.Tests.ContinualTests),
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Tests.ContinualTests = ops.asBackgroundOperation()
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
	}, {
		messages: messageFormatters.verifyContinual,
		ops:      generalizeOpsFromBg(s.Tests.ContinualTests),
		updateSuite: func(ops operations, s *upgrade.Suite) {
			s.Tests.ContinualTests = ops.asBackgroundOperation()
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

func (tt *texts) append(msgs ...string) *texts {
	for _, msg := range msgs {
		if msg == "" {
			continue
		}
		tt.elms = append(tt.elms, msg)
	}
	return tt
}

func completeSuiteExample(fp failurePoint) upgrade.Suite {
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
			ContinualTests: []upgrade.BackgroundOperation{
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
				op.fail()
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
