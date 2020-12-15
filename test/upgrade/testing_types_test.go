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

import "knative.dev/pkg/test/upgrade"

type failurePoint struct {
	step    int
	element int
}

type texts struct {
	elms []string
}

type messageFormatter func(args ...interface{}) string

type step struct {
	messages
	ops         operations
	updateSuite func(ops operations, s *upgrade.Suite)
}

type operations struct {
	ops []*operation
}

type operation struct {
	op upgrade.Operation
	bg upgrade.BackgroundOperation
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
