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
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestSuiteExecuteEmpty(t *testing.T) {
	c, buf := newConfig(t)
	suite := Suite{
		Tests:         Tests{},
		Installations: Installations{},
	}
	suite.Execute(c)
	output := buf.String()
	if c.T.Failed() {
		return
	}
	texts := []string{
		"🏃 Running upgrade suite...",
		"1) 💿 No base installation registered. Skipping.",
		"2) ✅️️ No pre upgrade tests registered. Skipping.",
		"3) 🔄 No continual tests registered. Skipping.",
		"4) 📀 No upgrade installations registered. Skipping.",
		"5) ✅️️ No post upgrade tests registered. Skipping.",
		"6) 💿 No downgrade installations registered. Skipping.",
		"7) ✅️️ No post downgrade tests registered. Skipping.",
		"🥳🎉 Success! Upgrade suite completed without errors.",
	}
	for _, text := range texts {
		assert.Contains(t, output, text)
	}
}

func TestSuiteExecuteWithTestsAndInstallations(t *testing.T) {
	c, buf := newConfig(t)
	suite := Suite{
		Tests: Tests{
			PreUpgrade: []Operation{
				servingPreUpgradeTest, eventingPreUpgradeTest,
			},
			PostUpgrade: []Operation{
				servingPostUpgradeTest, eventingPostUpgradeTest,
			},
			PostDowngrade: []Operation{
				servingPostDowngradeTest, eventingPostDowngradeTest,
			},
			ContinualTests: []BackgroundOperation{
				servingContinualTest, eventingContinualTest,
			},
		},
		Installations: Installations{
			Base: []Operation{
				servingStableInstall, eventingStableInstall,
			},
			UpgradeWith: []Operation{
				servingHeadInstall, eventingHeadInstall,
			},
			DowngradeWith: []Operation{
				servingStableInstall, eventingStableInstall,
			},
		},
	}
	suite.Execute(c)
	output := buf.String()
	if c.T.Failed() {
		return
	}
	texts := []string{
		"🏃 Running upgrade suite...",
		"1) 💿 Installing base installations. 2 are registered.",
		`1.1) Installing base install of "Serving latest stable release".`,
		"Installing Serving stable 0.17.1",
		`1.2) Installing base install of "Eventing latest stable release".`,
		"Installing Eventing stable 0.17.2",
		"2) ✅️️ Testing functionality before upgrade is performed. 2 tests are registered.",
		`2.1) Testing with "Serving pre upgrade test"`,
		`2.2) Testing with "Eventing pre upgrade test"`,
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
		"🥳🎉 Success! Upgrade suite completed without errors.",
	}
	for _, text := range texts {
		assert.Contains(t, output, text)
	}
}

func newConfig(t *testing.T) (Configuration, fmt.Stringer) {
	log, buf := newExampleZap()
	c := Configuration{T: t, Log: log}
	return c, buf
}

func newExampleZap() (*zap.Logger, *zaptest.Buffer) {
	ec := zap.NewDevelopmentEncoderConfig()
	ec.TimeKey = ""
	encoder := zapcore.NewConsoleEncoder(ec)
	buf := &zaptest.Buffer{
		Buffer: bytes.Buffer{},
		Syncer: zaptest.Syncer{},
	}
	ws := zapcore.NewMultiWriteSyncer(buf, os.Stdout)
	core := zapcore.NewCore(encoder, ws, zap.DebugLevel)
	return zap.New(core).WithOptions(), buf
}

var (
	servingStableInstall = NewOperation("Serving latest stable release", func(c Context) {
		c.Log.Info("Installing Serving stable 0.17.1")
		time.Sleep(20 * time.Millisecond)
	})
	servingHeadInstall = NewOperation("Serving HEAD", func(c Context) {
		c.Log.Info("Installing Serving HEAD at e3c4563")
		time.Sleep(20 * time.Millisecond)
	})
	eventingStableInstall = NewOperation("Eventing latest stable release", func(c Context) {
		c.Log.Info("Installing Eventing stable 0.17.2")
		time.Sleep(20 * time.Millisecond)
	})
	eventingHeadInstall = NewOperation("Eventing HEAD", func(c Context) {
		c.Log.Info("Installing Eventing HEAD at 12f67cc")
		time.Sleep(20 * time.Millisecond)
	})
	servingPreUpgradeTest = NewOperation("Serving pre upgrade test", func(c Context) {
		c.Log.Info("Running Serving pre upgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	eventingPreUpgradeTest = NewOperation("Eventing pre upgrade test", func(c Context) {
		c.Log.Info("Running Eventing pre upgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	servingPostUpgradeTest = NewOperation("Serving post upgrade test", func(c Context) {
		c.Log.Info("Running Serving post upgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	eventingPostUpgradeTest = NewOperation("Eventing post upgrade test", func(c Context) {
		c.Log.Info("Running Eventing post upgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	servingPostDowngradeTest = NewOperation("Serving post downgrade test", func(c Context) {
		c.Log.Info("Running Serving post downgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	eventingPostDowngradeTest = NewOperation("Eventing post downgrade test", func(c Context) {
		c.Log.Info("Running Eventing post downgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	servingContinualTest = NewBackgroundOperation("Serving continual test", func(bc BackgroundContext) {
		bc.Log.Info("Running Serving continual test")
		for {
			select {
			case msg := <-bc.Stop:
				bc.Log.Info("Serving probe test have received a stop message", msg)
				return
			default:
				bc.Log.Debug("Probing serving functionality...")
			}
			time.Sleep(5 * time.Millisecond)
		}
	})
	eventingContinualTest = NewBackgroundOperation("Eventing continual test", func(bc BackgroundContext) {
		bc.Log.Info("Running Eventing continual test")
		for {
			select {
			case msg := <-bc.Stop:
				bc.Log.Info("Eventing probe test have received a stop message", msg)
				return
			default:
				bc.Log.Debug("Probing eventing functionality...")
			}
			time.Sleep(5 * time.Millisecond)
		}
	})
)
