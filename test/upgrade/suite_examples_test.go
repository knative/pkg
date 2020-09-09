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
	"time"

	"knative.dev/pkg/test/upgrade"
)

const (
	shortWait = 50 * time.Microsecond
	longWait  = 750 * time.Microsecond
)

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
					waitForStopSignal(bc, "Serving", func(sig upgrade.StopEvent) interface{} {
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
					waitForStopSignal(bc, "Eventing", func(sig upgrade.StopEvent) interface{} {
						return 0
					})
				}),
		},
	}
)
