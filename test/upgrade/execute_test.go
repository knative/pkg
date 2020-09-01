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

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"knative.dev/pkg/test/upgrade"
)

func TestSuiteExecuteEmpty(t *testing.T) {
	c, buf := newConfig(t)
	suite := upgrade.Suite{
		Tests:         upgrade.Tests{},
		Installations: upgrade.Installations{},
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
	suite := upgrade.Suite{
		Tests: upgrade.Tests{
			PreUpgrade: []upgrade.Operation{
				servingPreUpgradeTest, eventingPreUpgradeTest,
			},
			PostUpgrade: []upgrade.Operation{
				servingPostUpgradeTest, eventingPostUpgradeTest,
			},
			PostDowngrade: []upgrade.Operation{
				servingPostDowngradeTest, eventingPostDowngradeTest,
			},
			ContinualTests: []upgrade.BackgroundOperation{
				servingContinualTest, eventingContinualTest,
			},
		},
		Installations: upgrade.Installations{
			Base: []upgrade.Operation{
				servingStableInstall, eventingStableInstall,
			},
			UpgradeWith: []upgrade.Operation{
				servingHeadInstall, eventingHeadInstall,
			},
			DowngradeWith: []upgrade.Operation{
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
		`4.2) Upgrading with "Eventing HEAD"`,
		"Installing Eventing HEAD at 12f67cc",
		"5) ✅️️ Testing functionality after upgrade is performed. 2 tests are registered.",
		`5.1) Testing with "Serving post upgrade test"`,
		`5.2) Testing with "Eventing post upgrade test"`,
		"6) 💿 Downgrading with 2 registered installations.",
		`6.1) Downgrading with "Serving latest stable release"`,
		"Installing Serving stable 0.17.1",
		`6.2) Downgrading with "Eventing latest stable release"`,
		"Installing Eventing stable 0.17.2",
		"7) ✅️️ Testing functionality after downgrade is performed. 2 tests are registered.",
		`7.1) Testing with "Serving post downgrade test"`,
		`7.2) Testing with "Eventing post downgrade test"`,
		"8) ✋ Verifying 2 running continual tests",
		`8.1) Verifying "Serving continual test"`,
		"Serving probe test have received a stop message",
		`8.2) Verifying "Eventing continual test"`,
		"Eventing probe test have received a stop message",
		"🥳🎉 Success! Upgrade suite completed without errors.",
	}
	for _, text := range texts {
		assert.Contains(t, output, text)
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

var (
	servingStableInstall = upgrade.NewOperation("Serving latest stable release", func(c upgrade.Context) {
		c.Log.Info("Installing Serving stable 0.17.1")
		time.Sleep(20 * time.Millisecond)
	})
	servingHeadInstall = upgrade.NewOperation("Serving HEAD", func(c upgrade.Context) {
		c.Log.Info("Installing Serving HEAD at e3c4563")
		time.Sleep(20 * time.Millisecond)
	})
	eventingStableInstall = upgrade.NewOperation("Eventing latest stable release", func(c upgrade.Context) {
		c.Log.Info("Installing Eventing stable 0.17.2")
		time.Sleep(20 * time.Millisecond)
	})
	eventingHeadInstall = upgrade.NewOperation("Eventing HEAD", func(c upgrade.Context) {
		c.Log.Info("Installing Eventing HEAD at 12f67cc")
		time.Sleep(20 * time.Millisecond)
	})
	servingPreUpgradeTest = upgrade.NewOperation("Serving pre upgrade test", func(c upgrade.Context) {
		c.Log.Info("Running Serving pre upgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	eventingPreUpgradeTest = upgrade.NewOperation("Eventing pre upgrade test", func(c upgrade.Context) {
		c.Log.Info("Running Eventing pre upgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	servingPostUpgradeTest = upgrade.NewOperation("Serving post upgrade test", func(c upgrade.Context) {
		c.Log.Info("Running Serving post upgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	eventingPostUpgradeTest = upgrade.NewOperation("Eventing post upgrade test", func(c upgrade.Context) {
		c.Log.Info("Running Eventing post upgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	servingPostDowngradeTest = upgrade.NewOperation("Serving post downgrade test", func(c upgrade.Context) {
		c.Log.Info("Running Serving post downgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	eventingPostDowngradeTest = upgrade.NewOperation("Eventing post downgrade test", func(c upgrade.Context) {
		c.Log.Info("Running Eventing post downgrade test")
		time.Sleep(5 * time.Millisecond)
	})
	servingContinualTest = upgrade.NewBackgroundOperation("Serving continual test",
		func(c upgrade.Context) {
			c.Log.Info("Setup of Serving continual test")
		},
		func(bc upgrade.BackgroundContext) {
			bc.Log.Info("Running Serving continual test")
			for {
				select {
				case sig := <-bc.Stop:
					bc.Log.Infof("Serving probe test have received a stop message: %s", sig.String())
					sig.Finished <- 12
					return
				default:
					bc.Log.Debug("Probing Serving functionality...")
				}
				time.Sleep(5 * time.Millisecond)
			}
		})
	eventingContinualTest = upgrade.NewBackgroundOperation("Eventing continual test",
		func(c upgrade.Context) {
			c.Log.Info("Setup of Eventing continual test")
		},
		func(bc upgrade.BackgroundContext) {
			bc.Log.Info("Running Eventing continual test")
			for {
				select {
				case sig := <-bc.Stop:
					bc.Log.Infof("Eventing probe test have received a stop message: %s", sig.String())
					sig.Finished <- 13
					return
				default:
					bc.Log.Debug("Probing Eventing functionality...")
				}
				time.Sleep(5 * time.Millisecond)
			}
		})
)
