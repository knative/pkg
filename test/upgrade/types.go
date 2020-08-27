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

// Operation represents a upgrade test operation like test or installation that
// can be provided by specific component or reused in aggregating components
type Operation interface {
	Name() string
	Handler() func(c Context)
}

// BackgroundOperation represents a upgrade test operation that will be
// performed in background while other operations is running. To achieve that
// a passed BackgroundContext should be used to synchronize it's operations with
// Ready and Stop channels.
type BackgroundOperation interface {
	Name() string
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
	Context
	Ready chan string
	Stop  chan string
}

// Configuration holds required and optional configuration to run upgrade tests
type Configuration struct {
	T   *testing.T
	Log *zap.Logger
}

// SuiteExecutor is to execute upgrade test suite
type SuiteExecutor interface {
	Execute(c Configuration)
}
