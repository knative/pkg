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
	"testing"

	"go.uber.org/zap"
)

// Suite represents a upgrade tests suite that can be executed and will perform
// execution in predictable manner
type Suite struct {
	Tests         Tests
	Installations Installations
}

// Tests holds a list of operations for various part of upgrade suite
type Tests struct {
	PreUpgrade     []Operation
	PostUpgrade    []Operation
	PostDowngrade  []Operation
	ContinualTests []BackgroundOperation
}

// Installations holds a list of operations that will install Knative components
// in different versions
type Installations struct {
	Base          []Operation
	UpgradeWith   []Operation
	DowngradeWith []Operation
}

// Named interface represents a type that is named in user friendly way
type Named interface {
	Name() string
}

// Operation represents a upgrade test operation like test or installation that
// can be provided by specific component or reused in aggregating components
type Operation interface {
	Named
	Handler() func(c Context)
}

// BackgroundOperation represents a upgrade test operation that will be
// performed in background while other operations is running. To achieve that
// a passed BackgroundContext should be used to synchronize it's operations with
// Ready and Stop channels.
type BackgroundOperation interface {
	Named
	Setup() func(c Context)
	Handler() func(bc BackgroundContext)
}

// Context is an object that is passed to every operation
type Context struct {
	T   *testing.T
	Log *zap.SugaredLogger
}

// BackgroundContext is a upgrade test execution context that will be passed down to each
// handler of StoppableOperation. It contains a T reference and a stop channel
// which handler should listen to to know when to stop its operations.
type BackgroundContext struct {
	Log  *zap.SugaredLogger
	Stop chan StopSignal
}

// StopSignal represents a signal that is to be received by background operation
// to indicate that is should stop it's operations and validate results using
// passed T
type StopSignal struct {
	T        T
	Finished chan int
	name     string
	channel  chan StopSignal
}

// Configuration holds required and optional configuration to run upgrade tests
type Configuration struct {
	T   T
	Log *zap.Logger
}

// SuiteExecutor is to execute upgrade test suite
type SuiteExecutor interface {
	Execute(c Configuration)
}

// T is the interface of testing.T (copy of it).
type T interface {
	Cleanup(func())
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Helper()
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Name() string
	Skip(args ...interface{})
	SkipNow()
	Skipf(format string, args ...interface{})
	Skipped() bool
	Run(name string, f func(t *testing.T)) bool
}
