/*
Copyright 2022 The Knative Authors

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

package exit

import (
	"fmt"
	"os"
)

var _real = func(code int) { os.Exit(code) }

// Exit normally terminates the process by calling os.Exit(code). If the package
// is stubbed, it instead records a call in the testing spy.
func Exit(code int) {
	_real(code)
}

// A StubbedExit is a testing fake for os.Exit.
type StubbedExit struct {
	Exited bool
	Code   int
	Panic  interface{}
	prev   func(code int)
}

// Stub substitutes a fake for the call to os.Exit(1).
func Stub() *StubbedExit {
	s := &StubbedExit{prev: _real}
	_real = s.exit
	return s
}

// WithStub runs the supplied function with Exit stubbed. It returns the stub
// used, so that users can test whether the process would have crashed.
func WithStub(fn func()) *StubbedExit {
	s := Stub()
	defer s.Unstub()
	panicCh := make(chan interface{})
	go handle(fn, panicCh)
	s.Panic = <-panicCh
	return s
}

// Unstub restores the previous exit function.
func (se *StubbedExit) Unstub() {
	_real = se.prev
}

func (se *StubbedExit) exit(code int) {
	se.Exited = true
	se.Code = code
	panic(fmt.Sprintf("exit with code: %d", code))
}

func handle(fn func(), panicCh chan interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// pass panic variable outside
			panicCh <- r
		}
	}()

	fn()
	close(panicCh)
}
