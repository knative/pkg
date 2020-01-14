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

package logging

import (
	"testing"
)

type abc struct {
	A int
	b string
	C *de
	F func()
}

type de struct {
	D string
	e float64
}

const runFailingTests = false

var someStruct abc

func init() {
	someStruct = abc{
		A: 42,
		b: "some string",
		C: &de{
			D: "hello world",
			e: 72.3,
		},
		F: InitializeLogger,
	}
}

func TestTLogger(legacy *testing.T) {
	Verbosity = 5
	InitializeLogger()
	t, cancel := NewTLogger(legacy)
	defer cancel()

	var blank interface{}
	blank = &someStruct

	t.V(6).Info("Should not be printed")
	t.V(4).Info("Should be printed!")
	t.Run("A-Nice-Subtest", func(ts *TLogger) {
		ts.V(0).Info("This is pretty important; everyone needs to see it!",
			"some pointer", blank,
			"some number", 42.0)
		t.Run("A-Nested-Subtest", func(ts *TLogger) {
			ts.Parallel()
			ts.V(1).Info("I am visible!")
			ts.V(6).Info("I am invisible!")
		})
		t.Run("A-2nd-Nested-Subtest", func(ts *TLogger) {
			ts.Parallel()
			ts.V(1).Info("I am visible!")
			ts.V(6).Info("I am also invisible!")
		})
	})
	if runFailingTests {
		t.Run("Failing", func(ts *TLogger) {
			ts.Error("I am an error", "hello", "world")
		})
	}
	t.Run("Skipped", func(ts *TLogger) {
		ts.SkipNow()
	})
	t.ErrorIfErr(nil, "I won't fail because no error!")
	t.FatalIfErr(nil, "I won't fail because no error!")
}

func TestStructuredError(legacy *testing.T) {
	Verbosity = 5
	InitializeLogger()
	t, cancel := NewTLogger(legacy)
	defer cancel()
	err := Error("Hello World", "key", "value", "current function", TestStructuredError, "deep struct", someStruct, "z", 4, "y", 3, "x", 2, "w", 1)
	t.Log(err.Error())
	// TODO(coryrc): Big problem! zap is printing '\' 'n' instead of a newline, which makes the errors not readable on the screen
	// need a different zapcore or something
	if runFailingTests {
		t.Error(err)
		t.ErrorIfErr(err, "Demonstrate twice!")
	}
}
