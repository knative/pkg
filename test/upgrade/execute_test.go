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
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"knative.dev/pkg/test/upgrade"
)

const (
	upgradeTestRunning = "🏃 Running upgrade test suite..."
	upgradeTestSuccess = "🥳🎉 Success! Upgrade suite completed without errors."
	upgradeTestFailure = "💣🤬💔️ Upgrade suite have failed!"

	shortWait = time.Millisecond
	longWait  = 5 * time.Millisecond
)

func TestExpectedTextsForEmptySuite(t *testing.T) {
	fp := notFailing
	suite := emptySuiteExample()
	txt := expectedTexts(suite, fp)
	expected := []string{
		"1) 💿 No base installation registered. Skipping.",
		"2) ✅️️ No pre upgrade tests registered. Skipping.",
		"3) 🔄 No continual tests registered. Skipping.",
		"4) 📀 No upgrade operations registered. Skipping.",
		"5) ✅️️ No post upgrade tests registered. Skipping.",
		"6) 💿 No downgrade operations registered. Skipping.",
		"7) ✅️️ No post downgrade tests registered. Skipping.",
	}
	assertArraysEqual(t, txt.elms, expected)
}

func TestExpectedTextsForCompleteSuite(t *testing.T) {
	fp := notFailing
	suite := completeSuiteExample(fp)
	txt := expectedTexts(suite, fp)
	expected := []string{
		"1) 💿 Installing base installations. 2 are registered.",
		`1.1) Installing base install of "Serving latest stable release".`,
		`1.2) Installing base install of "Eventing latest stable release".`,
		"2) ✅️️ Testing functionality before upgrade is performed. 2 tests are registered.",
		`2.1) Testing with "Serving pre upgrade test".`,
		`2.2) Testing with "Eventing pre upgrade test".`,
		"3) 🔄 Starting continual tests to run in background. 2 tests are registered.",
		`3.1) Starting continual tests of "Serving continual test".`,
		`3.2) Starting continual tests of "Eventing continual test".`,
		"4) 📀 Upgrading with 2 registered operations.",
		`4.1) Upgrading with "Serving HEAD".`,
		`4.2) Upgrading with "Eventing HEAD".`,
		"5) ✅️️ Testing functionality after upgrade is performed. 2 tests are registered.",
		`5.1) Testing with "Serving post upgrade test".`,
		`5.2) Testing with "Eventing post upgrade test".`,
		"6) 💿 Downgrading with 2 registered operations.",
		`6.1) Downgrading with "Serving latest stable release".`,
		`6.2) Downgrading with "Eventing latest stable release".`,
		"7) ✅️️ Testing functionality after downgrade is performed. 2 tests are registered.",
		`7.1) Testing with "Serving post downgrade test".`,
		`7.2) Testing with "Eventing post downgrade test".`,
		"8) ✋ Verifying 2 running continual tests.",
		`8.1) Verifying "Serving continual test".`,
		`8.2) Verifying "Eventing continual test".`,
	}
	assertArraysEqual(t, txt.elms, expected)
}

func TestExpectedTextsForFailingCompleteSuite(t *testing.T) {
	fp := failurePoint{
		step:    2,
		element: 1,
	}
	suite := completeSuiteExample(fp)
	txt := expectedTexts(suite, fp)
	expected := []string{
		"1) 💿 Installing base installations. 2 are registered.",
		`1.1) Installing base install of "Serving latest stable release".`,
		`1.2) Installing base install of "Eventing latest stable release".`,
		"2) ✅️️ Testing functionality before upgrade is performed. 2 tests are registered.",
		`2.1) Testing with "Serving pre upgrade test".`,
	}
	assertArraysEqual(t, txt.elms, expected)
}

func TestSuiteExecuteEmpty(t *testing.T) {
	c, buf := newConfig(t)
	fp := notFailing
	suite := emptySuiteExample()
	suite.Execute(c)
	output := buf.String()
	if c.T.Failed() {
		return
	}

	txt := expectedTexts(suite, fp)
	txt.append(upgradeTestRunning, upgradeTestSuccess)

	assertTextContains(t, output, txt)
}

func TestSuiteExecuteWithComplete(t *testing.T) {
	c, buf := newConfig(t)
	fp := notFailing
	suite := completeSuiteExample(fp)
	suite.Execute(c)
	output := buf.String()
	if c.T.Failed() {
		return
	}
	txt := expectedTexts(suite, fp)
	txt.append(upgradeTestRunning, upgradeTestSuccess)
	txt.append(
		"Installing Serving stable 0.17.1",
		"Installing Eventing stable 0.17.2",
		"Running Serving continual test",
		"Running Eventing continual test",
		"Installing Serving HEAD at e3c4563",
		"Installing Eventing HEAD at 12f67cc",
		"Installing Serving stable 0.17.1",
		"Installing Eventing stable 0.17.2",
		"Serving probe test have received a stop message",
		"Eventing probe test have received a stop message",
	)

	assertTextContains(t, output, txt)
}

func TestSuiteExecuteWithFailingStep(t *testing.T) {
	// for i := 1; i < 8; i++ {
	// 	for j := 1; j <= 2; j++ {
	// 		fp := failurePoint{
	// 			step:    i,
	// 			element: j,
	// 		}
	// 		testSuiteExecuteWithFailingStep(fp, t)
	// 	}
	// }
	fp := failurePoint{
		step:    1,
		element: 2,
	}
	testSuiteExecuteWithFailingStep(fp, t)
}

func testSuiteExecuteWithFailingStep(fp failurePoint, t *testing.T) {
	t.Run(fmt.Sprintf("FailAt-%d-%d", fp.step, fp.element), func(t *testing.T) {
		var output string
		suite := completeSuiteExample(fp)
		txt := expectedTexts(suite, fp)
		txt.append(upgradeTestRunning, upgradeTestFailure)
		log, buf := newExampleZap()

		tests := []testing.InternalTest{{
			Name: fmt.Sprintf("FailAt-%d-%d", fp.step, fp.element),
			F: func(t *testing.T) {
				c, _ := newConfig(t)
				c.Log = log
				suite.Execute(c)
			},
		}}
		var ok bool
		testOutput := captureStdOutput(func() {
			ok = testing.RunTests(allTestsFilter, tests)
		})
		output = buf.String()

		if ok {
			t.Fatal("didn't failed, but should")
		}

		assertTextContains(t, output, txt)
		assertTextContains(t, testOutput, texts{
			elms: []string{
				fmt.Sprintf("--- FAIL: FailAt-%d-%d", fp.step, fp.element),
			},
		})
	})
}

func captureStdOutput(call func()) string {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = rescueStdout
	}()

	call()

	_ = w.Close()
	out, _ := ioutil.ReadAll(r)
	return string(out)
}

func assertTextContains(t *testing.T, haystack string, needles texts) {
	for _, needle := range needles.elms {
		if !strings.Contains(haystack, needle) {
			t.Errorf(
				"expected \"%s\" is not in: `%s`",
				needle, haystack,
			)
		}
	}
}

func assertArraysEqual(t *testing.T, actual []string, expected []string) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("arrays differ:\n  actual: %#v\nexpected: %#v", actual, expected)
	}
}

var allTestsFilter = func(_, _ string) (bool, error) { return true, nil }

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

// To avoid race condition on zaptest.Buffer, see: https://stackoverflow.com/a/36226525/844449
type buffer struct {
	bytes.Buffer
	sync.Mutex
	zaptest.Syncer
}

func (b *buffer) Read(p []byte) (n int, err error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.Read(p)
}
func (b *buffer) Write(p []byte) (n int, err error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.Write(p)
}
func (b *buffer) String() string {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.String()
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

type failurePoint struct {
	step    int
	element int
}

type texts struct {
	elms []string
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

type messageFormatter func(args ...interface{}) string

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
	return inlaySuite(suite, fp)
}

func emptySuiteExample() upgrade.Suite {
	return upgrade.Suite{
		Tests:         upgrade.Tests{},
		Installations: upgrade.Installations{},
	}
}

func inlaySuite(suite upgrade.Suite, fp failurePoint) upgrade.Suite {
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

type step struct {
	messages
	ops         operations
	updateSuite func(ops operations, s *upgrade.Suite)
}

type operations struct {
	ops []*operation
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

type operation struct {
	op upgrade.Operation
	bg upgrade.BackgroundOperation
}

func (o operation) Name() string {
	if o.op != nil {
		return o.op.Name()
	} else {
		return o.bg.Name()
	}
}

func (o *operation) fail() {
	failureTestingMessage := "Testing a failure"
	if o.op != nil {
		prev := o.op
		o.op = upgrade.NewOperation(prev.Name(), func(c upgrade.Context) {
			handler := prev.Handler()
			handler(c)
			c.T.Error(failureTestingMessage)
		})
	} else {
		prev := o.bg
		o.bg = upgrade.NewBackgroundOperation(prev.Name(), func(c upgrade.Context) {
			handler := prev.Setup()
			handler(c)
			c.T.Error(failureTestingMessage)
		}, func(bc upgrade.BackgroundContext) {
			waitForStopSignal(bc, prev.Name(), func(sig upgrade.StopSignal) int {
				sig.T.Error(failureTestingMessage)
				return 17
			})
		})
	}
}

type formats struct {
	skipped  string
	starting string
	element  string
}

type messages struct {
	starting messageFormatter
	element  messageFormatter
	skipped  messageFormatter
}

type messageFormatterRepository struct {
	baseInstall     messages
	preUpgrade      messages
	startContinual  messages
	upgrade         messages
	postUpgrade     messages
	downgrade       messages
	postDowngrade   messages
	verifyContinual messages
}

type component struct {
	installs
	tests
}

type installs struct {
	stable upgrade.Operation
	head   upgrade.Operation
}

type tests struct {
	preUpgrade    upgrade.Operation
	postUpgrade   upgrade.Operation
	continual     upgrade.BackgroundOperation
	postDowngrade upgrade.Operation
}

var (
	notFailing        = failurePoint{step: -1, element: -1}
	messageFormatters = messageFormatterRepository{
		baseInstall: createMessages(formats{
			starting: "%d) 💿 Installing base installations. %d are registered.",
			element:  `%d.%d) Installing base install of "%s".`,
			skipped:  "%d) 💿 No base installation registered. Skipping.",
		}),
		preUpgrade: createMessages(formats{
			starting: "%d) ✅️️ Testing functionality before upgrade is performed. %d tests are registered.",
			element:  `%d.%d) Testing with "%s".`,
			skipped:  "%d) ✅️️ No pre upgrade tests registered. Skipping.",
		}),
		startContinual: createMessages(formats{
			starting: "%d) 🔄 Starting continual tests to run in background. %d tests are registered.",
			element:  `%d.%d) Starting continual tests of "%s".`,
			skipped:  "%d) 🔄 No continual tests registered. Skipping.",
		}),
		upgrade: createMessages(formats{
			starting: "%d) 📀 Upgrading with %d registered operations.",
			element:  `%d.%d) Upgrading with "%s".`,
			skipped:  "%d) 📀 No upgrade operations registered. Skipping.",
		}),
		postUpgrade: createMessages(formats{
			starting: "%d) ✅️️ Testing functionality after upgrade is performed. %d tests are registered.",
			element:  `%d.%d) Testing with "%s".`,
			skipped:  "%d) ✅️️ No post upgrade tests registered. Skipping.",
		}),
		downgrade: createMessages(formats{
			starting: "%d) 💿 Downgrading with %d registered operations.",
			element:  `%d.%d) Downgrading with "%s".`,
			skipped:  "%d) 💿 No downgrade operations registered. Skipping.",
		}),
		postDowngrade: createMessages(formats{
			starting: "%d) ✅️️ Testing functionality after downgrade is performed. %d tests are registered.",
			element:  `%d.%d) Testing with "%s".`,
			skipped:  "%d) ✅️️ No post downgrade tests registered. Skipping.",
		}),
		verifyContinual: createMessages(formats{
			starting: "%d) ✋ Verifying %d running continual tests.",
			element:  `%d.%d) Verifying "%s".`,
			skipped:  "",
		}),
	}
	serving = component{
		installs: installs{
			stable: upgrade.NewOperation("Serving latest stable release", func(c upgrade.Context) {
				c.Log.Info("Installing Serving stable 0.17.1")
				time.Sleep(longWait)
			}),
			head: upgrade.NewOperation("Serving HEAD", func(c upgrade.Context) {
				c.Log.Info("Installing Serving HEAD at e3c4563")
				time.Sleep(longWait)
			}),
		},
		tests: tests{
			preUpgrade: upgrade.NewOperation("Serving pre upgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Serving pre upgrade test")
				time.Sleep(shortWait)
			}),
			postUpgrade: upgrade.NewOperation("Serving post upgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Serving post upgrade test")
				time.Sleep(shortWait)
			}),
			postDowngrade: upgrade.NewOperation("Serving post downgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Serving post downgrade test")
				time.Sleep(shortWait)
			}),
			continual: upgrade.NewBackgroundOperation("Serving continual test",
				func(c upgrade.Context) {
					c.Log.Info("Setup of Serving continual test")
					time.Sleep(shortWait)
				},
				func(bc upgrade.BackgroundContext) {
					bc.Log.Info("Running Serving continual test")
					waitForStopSignal(bc, "Serving", func(sig upgrade.StopSignal) int {
						return 0
					})
				}),
		},
	}
	eventing = component{
		installs: installs{
			stable: upgrade.NewOperation("Eventing latest stable release", func(c upgrade.Context) {
				c.Log.Info("Installing Eventing stable 0.17.2")
				time.Sleep(longWait)
			}),
			head: upgrade.NewOperation("Eventing HEAD", func(c upgrade.Context) {
				c.Log.Info("Installing Eventing HEAD at 12f67cc")
				time.Sleep(longWait)
			}),
		},
		tests: tests{
			preUpgrade: upgrade.NewOperation("Eventing pre upgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Eventing pre upgrade test")
				time.Sleep(shortWait)
			}),
			postUpgrade: upgrade.NewOperation("Eventing post upgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Eventing post upgrade test")
				time.Sleep(shortWait)
			}),
			postDowngrade: upgrade.NewOperation("Eventing post downgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Eventing post downgrade test")
				time.Sleep(shortWait)
			}),
			continual: upgrade.NewBackgroundOperation("Eventing continual test",
				func(c upgrade.Context) {
					c.Log.Info("Setup of Eventing continual test")
					time.Sleep(shortWait)
				},
				func(bc upgrade.BackgroundContext) {
					bc.Log.Info("Running Eventing continual test")
					waitForStopSignal(bc, "Eventing", func(sig upgrade.StopSignal) int {
						return 0
					})
				}),
		},
	}
)
