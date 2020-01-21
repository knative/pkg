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

package ensure

import (
	"github.com/pkg/errors"
	"testing"
)

func TestNoError_GivenNil(t *testing.T) {
	var err error = nil

	NoError(err)
}

func TestNoError_GivenError(t *testing.T) {
	err := errors.New("expected")

	defer func() {
		r := recover()
		expectToRecoverFromPanic(t, r)

		err = r.(error)
		equalError(t, err, "unexpected error: expected")
	}()

	NoError(err)
}

func TestError_GivenNil(t *testing.T) {
	var err error = nil

	defer func() {
		r := recover()
		expectToRecoverFromPanic(t, r)

		err = r.(error)
		equalError(t, err, "expecting error, but none given")
	}()

	Error(err)
}

func TestError_GivenError(t *testing.T) {
	err := errors.New("expected")

	Error(err)
}

func TestErrorWithMessage(t *testing.T) {
	err1 := errors.New("expected")
	err2 := errors.New("expect")
	re := "^expect(?:ed)?$"

	ErrorWithMessage(err1, re)
	ErrorWithMessage(err2, re)
}

func TestErrorWithMessage_DifferentMessage(t *testing.T) {
	err := errors.New("dogs")
	re := "^cats$"

	defer func() {
		r := recover()
		expectToRecoverFromPanic(t, r)

		err = r.(error)
		equalError(t, err, "given error doesn't match given regexp (^cats$): dogs")
	}()

	ErrorWithMessage(err, re)
}

func equalError(t *testing.T, err error, expectedMessage string) {
	actual := err.Error()
	if actual != expectedMessage {
		t.Errorf("expecting error message to be: %s, but was: %s", expectedMessage, actual)
	}
}

func expectToRecoverFromPanic(t *testing.T, r interface{}) {
	if r == nil {
		t.Fatal(errors.New("expected to recover from panic, but didn't"))
	}
}
