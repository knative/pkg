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
	"testing"
)

const skippingOperationTemplate = `Skipping "%s" as previous operation have failed`

func (se *suiteExecution) installingBase(t *testing.T, num int) {
	se.processOperationGroup(t, operationGroup{
		num:                   num,
		operations:            se.suite.installations.Base,
		groupName:             "InstallingBase",
		elementTemplate:       `%d.%d) Installing base install of "%s".`,
		skippingGroupTemplate: "%d) 💿 No base installation registered. Skipping.",
		groupTemplate:         "%d) 💿 Installing base installations. %d are registered.",
	})
}

func (se *suiteExecution) preUpgradeTests(t *testing.T, num int) {
	se.processOperationGroup(t, operationGroup{
		num:                   num,
		operations:            se.suite.tests.preUpgrade,
		groupName:             "PreUpgradeTests",
		elementTemplate:       `%d.%d) Testing with "%s".`,
		skippingGroupTemplate: "%d) ✅️️ No pre upgrade tests registered. Skipping.",
		groupTemplate: "%d) ✅️️ Testing functionality before upgrade is performed." +
			" %d tests are registered.",
	})
}

func (se *suiteExecution) runContinualTests(t *testing.T, num int, stopCh <-chan struct{}) {
	l := se.logger
	operations := se.suite.tests.continual
	groupTemplate := "%d) 🔄 Starting continual tests. " +
		"%d tests are registered."
	elementTemplate := `%d.%d) Starting continual tests of "%s".`
	numOps := len(operations)
	if numOps > 0 {
		l.Infof(groupTemplate, num, numOps)
		for i := range operations {
			operation := operations[i]
			l.Debugf(elementTemplate, num, i+1, operation.Name())
			t.Run(operation.Name(), func(t *testing.T) {
				setup := operation.Setup()
				setup(Context{T: t, Log: l})

				//go func() {
				//	//TODO: This could possibly wait directly for done instead of operation.stop?
				//	handler(BackgroundContext{
				//		Log:  l,
				//		Stop: operation.stop,
				//	})
				//}()

				// Will be run in parallel with "UpgradeDowngrade" test
				t.Parallel()

				// Waiting for done signal to be sent after Upgrades/Downgrades are finished.
				//<-done

				handler := operation.Handler()
				// Blocking operation.
				handler(BackgroundContext{
					T:    t,
					Log:  l,
					Stop: stopCh,
				})

				//finished := make(chan struct{})
				//operation.stop <- StopEvent{
				//	T:        t,
				//	//Finished: finished,
				//	logger:   l, // is this necessary? maybe only for test purposes
				//}
				//<-finished
				l.Debugf(`Finished "%s"`, operation.Name())
			})
		}
	} else {
		l.Infof("%d) 🔄 No continual tests registered. Skipping.", num)
	}
	//
	//se.configuration.T.Run("ContinualTests", func(t *testing.T) {
	//	l, err := se.configuration.logger(t)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	if numOps > 0 {
	//		l.Infof(groupTemplate, num, numOps)
	//		for i := range operations {
	//			operation := operations[i]
	//			l.Infof(elementTemplate, num, i+1, operation.Name())
	//			if se.failed {
	//				l.Debugf(skippingOperationTemplate, operation.Name())
	//				return
	//			}
	//			operation.Setup()(Context{T: t}) // Note: Log is not passed here, we'll use t.Logf directly
	//
	//			logger, buffer := newInMemoryLoggerBuffer(se.configuration)
	//			t.Run("Setup"+operation.Name(), func(t *testing.T) {
	//				l, err = se.configuration.logger(t)
	//				if err != nil {
	//					t.Fatal(err)
	//				}
	//				setup(Context{T: t, Log: logger.Sugar()})
	//			})
	//
	//			handler := operation.Handler()
	//			go func() {
	//				handler(BackgroundContext{
	//					Log:       logger.Sugar(),
	//					Stop:      operation.stop,
	//					logBuffer: buffer,
	//				})
	//			}()
	//			se.failed = se.failed || t.Failed()
	//			if se.failed {
	//				// need to dump logs here, because verify will not be executed.
	//				l.Error(wrapLog(buffer.Dump()))
	//				return
	//			}
	//		}
	//
	//	} else {
	//		l.Infof("%d) 🔄 No continual tests registered. Skipping.", num)
	//	}
	//})
}

//func (se *suiteExecution) verifyContinualTests(num int) {
//	testsCount := len(se.suite.tests.continual)
//	if testsCount > 0 {
//		se.configuration.T.Run("VerifyContinualTests", func(t *testing.T) {
//			l, err := se.configuration.logger(t)
//			if err != nil {
//				t.Fatal(err)
//			}
//			l.Infof("%d) ✋ Verifying %d running continual tests.", num, testsCount)
//			for i, operation := range se.suite.tests.continual {
//				t.Run(operation.Name(), func(t *testing.T) {
//					l, err = se.configuration.logger(t)
//					if err != nil {
//						t.Fatal(err)
//					}
//					l.Infof(`%d.%d) Verifying "%s".`, num, i+1, operation.Name())
//					finished := make(chan struct{})
//					operation.stop <- StopEvent{
//						T:        t,
//						Finished: finished,
//						logger:   l,
//						name:     "Stop of " + operation.Name(),
//					}
//					<-finished
//					se.failed = se.failed || t.Failed()
//					l.Debugf(`Finished "%s"`, operation.Name())
//				})
//			}
//		})
//	}
//}

func (se *suiteExecution) upgradeWith(t *testing.T, num int) {
	se.processOperationGroup(t, operationGroup{
		num:                   num,
		operations:            se.suite.installations.UpgradeWith,
		groupName:             "UpgradeWith",
		elementTemplate:       `%d.%d) Upgrading with "%s".`,
		skippingGroupTemplate: "%d) 📀 No upgrade operations registered. Skipping.",
		groupTemplate:         "%d) 📀 Upgrading with %d registered operations.",
	})
}

func (se *suiteExecution) postUpgradeTests(t *testing.T, num int) {
	se.processOperationGroup(t, operationGroup{
		num:                   num,
		operations:            se.suite.tests.postUpgrade,
		groupName:             "PostUpgradeTests",
		elementTemplate:       `%d.%d) Testing with "%s".`,
		skippingGroupTemplate: "%d) ✅️️ No post upgrade tests registered. Skipping.",
		groupTemplate: "%d) ✅️️ Testing functionality after upgrade is performed." +
			" %d tests are registered.",
	})
}

func (se *suiteExecution) downgradeWith(t *testing.T, num int) {
	se.processOperationGroup(t, operationGroup{
		num:                   num,
		operations:            se.suite.installations.DowngradeWith,
		groupName:             "DowngradeWith",
		elementTemplate:       `%d.%d) Downgrading with "%s".`,
		skippingGroupTemplate: "%d) 💿 No downgrade operations registered. Skipping.",
		groupTemplate:         "%d) 💿 Downgrading with %d registered operations.",
	})
}

func (se *suiteExecution) postDowngradeTests(t *testing.T, num int) {
	se.processOperationGroup(t, operationGroup{
		num:                   num,
		operations:            se.suite.tests.postDowngrade,
		groupName:             "PostDowngradeTests",
		elementTemplate:       `%d.%d) Testing with "%s".`,
		skippingGroupTemplate: "%d) ✅️️ No post downgrade tests registered. Skipping.",
		groupTemplate: "%d) ✅️️ Testing functionality after downgrade is performed." +
			" %d tests are registered.",
	})
}
