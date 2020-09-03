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
)

func TestExpectedTextsForEmptySuite(t *testing.T) {
	fp := notFailing
	suite := sampler{}.empty()
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
	suite := sampler{}.complete(fp)
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
	suite := sampler{}.complete(fp)
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
	suite := sampler{}.empty()
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
	suite := sampler{}.complete(fp)
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
	t.Skip("not yet implemented")
	c, buf := newConfig(t)
	fp := failurePoint{
		step:    5,
		element: 1,
	}
	suite := sampler{}.complete(fp)
	suite.Execute(c)
	output := buf.String()
	if !c.T.Failed() {
		t.Fatal("didn't failed, but should")
	}
	txt := expectedTexts(suite, fp)
	txt.append(upgradeTestRunning, upgradeTestFailure)

	assertTextContains(t, output, txt)
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

func waitForStopSignal(bc upgrade.BackgroundContext, name string, retcode int) {
	for {
		select {
		case sig := <-bc.Stop:
			bc.Log.Infof(
				"%s probe test have received a stop message: %s",
				name, sig.String())
			sig.Finished <- retcode
			return
		default:
			bc.Log.Debugf("Probing %s functionality...", name)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func expectedTexts(s upgrade.Suite, fp failurePoint) texts {
	steps := []step{{
		messages: messageFormatters.baseInstall,
		ops:      asNamed(s.Installations.Base),
	}, {
		messages: messageFormatters.preUpgrade,
		ops:      asNamed(s.Tests.PreUpgrade),
	}, {
		messages: messageFormatters.startContinual,
		ops:      asNamedFromBg(s.Tests.ContinualTests),
	}, {
		messages: messageFormatters.upgrade,
		ops:      asNamed(s.Installations.UpgradeWith),
	}, {
		messages: messageFormatters.postUpgrade,
		ops:      asNamed(s.Tests.PostUpgrade),
	}, {
		messages: messageFormatters.downgrade,
		ops:      asNamed(s.Installations.DowngradeWith),
	}, {
		messages: messageFormatters.postDowngrade,
		ops:      asNamed(s.Tests.PostDowngrade),
	}, {
		messages: messageFormatters.verifyContinual,
		ops:      asNamedFromBg(s.Tests.ContinualTests),
	}}
	tt := texts{elms: nil}
	for i, st := range steps {
		stepIdx := i + 1
		if len(st.ops) == 0 {
			tt.append(st.skipped(stepIdx))
		} else {
			tt.append(st.starting(stepIdx, len(st.ops)))
			for j, named := range st.ops {
				elemIdx := j + 1
				tt.append(st.element(stepIdx, elemIdx, named.Name()))
				if fp.step == stepIdx && fp.element == elemIdx {
					return tt
				}
			}
		}
	}
	return tt
}

func asNamed(ops []upgrade.Operation) []upgrade.Named {
	names := make([]upgrade.Named, len(ops))
	for idx, op := range ops {
		names[idx] = op
	}
	return names
}

func asNamedFromBg(ops []upgrade.BackgroundOperation) []upgrade.Named {
	names := make([]upgrade.Named, len(ops))
	for idx, op := range ops {
		names[idx] = op
	}
	return names
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

type suiteSampler interface {
	complete(fp failurePoint) upgrade.Suite
	empty()
}

type sampler struct{}

func (s sampler) complete(fp failurePoint) upgrade.Suite {
	return upgrade.Suite{
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
}

func (s sampler) empty() upgrade.Suite {
	return upgrade.Suite{
		Tests:         upgrade.Tests{},
		Installations: upgrade.Installations{},
	}
}

type step struct {
	messages
	ops []upgrade.Named
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
				time.Sleep(20 * time.Millisecond)
			}),
			head: upgrade.NewOperation("Serving HEAD", func(c upgrade.Context) {
				c.Log.Info("Installing Serving HEAD at e3c4563")
				time.Sleep(20 * time.Millisecond)
			}),
		},
		tests: tests{
			preUpgrade: upgrade.NewOperation("Serving pre upgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Serving pre upgrade test")
				time.Sleep(5 * time.Millisecond)
			}),
			postUpgrade: upgrade.NewOperation("Serving post upgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Serving post upgrade test")
				time.Sleep(5 * time.Millisecond)
			}),
			postDowngrade: upgrade.NewOperation("Serving post downgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Serving post downgrade test")
				time.Sleep(5 * time.Millisecond)
			}),
			continual: upgrade.NewBackgroundOperation("Serving continual test",
				func(c upgrade.Context) {
					c.Log.Info("Setup of Serving continual test")
					time.Sleep(5 * time.Millisecond)
				},
				func(bc upgrade.BackgroundContext) {
					bc.Log.Info("Running Serving continual test")
					waitForStopSignal(bc, "Serving", 12)
				}),
		},
	}
	eventing = component{
		installs: installs{
			stable: upgrade.NewOperation("Eventing latest stable release", func(c upgrade.Context) {
				c.Log.Info("Installing Eventing stable 0.17.2")
				time.Sleep(20 * time.Millisecond)
			}),
			head: upgrade.NewOperation("Eventing HEAD", func(c upgrade.Context) {
				c.Log.Info("Installing Eventing HEAD at 12f67cc")
				time.Sleep(20 * time.Millisecond)
			}),
		},
		tests: tests{
			preUpgrade: upgrade.NewOperation("Eventing pre upgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Eventing pre upgrade test")
				time.Sleep(5 * time.Millisecond)
			}),
			postUpgrade: upgrade.NewOperation("Eventing post upgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Eventing post upgrade test")
				time.Sleep(5 * time.Millisecond)
			}),
			postDowngrade: upgrade.NewOperation("Eventing post downgrade test", func(c upgrade.Context) {
				c.Log.Info("Running Eventing post downgrade test")
				time.Sleep(5 * time.Millisecond)
			}),
			continual: upgrade.NewBackgroundOperation("Eventing continual test",
				func(c upgrade.Context) {
					c.Log.Info("Setup of Eventing continual test")
					time.Sleep(5 * time.Millisecond)
				},
				func(bc upgrade.BackgroundContext) {
					bc.Log.Info("Running Eventing continual test")
					waitForStopSignal(bc, "Eventing", 13)
				}),
		},
	}
)
