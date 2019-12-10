/*
Copyright 2019 The Knative Authors

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
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

//  1. Structured versions of t.Error() and t.Fatal()
//  2. A replacement t.Run() for subtests, which calls a subfunction func(t *TLogger) instead
//  3. Implement test.T and test.TLegacy for compat reasons

type TLogger struct {
	l     *zap.Logger
	level int
	t     *testing.T
}

func (o *TLogger) V(level int) logr.InfoLogger {
	if level <= o.level ||
		(level <= logrZapDebugLevel && o.l.Core().Enabled(zapLevelFromLogrLevel(level))) {
		return &infoLogger{
			logrLevel: o.level,
			t:         o,
		}
	}
	return disabledInfoLogger
}

func (o *TLogger) WithValues(keysAndValues ...interface{}) *TLogger {
	return o.cloneWithNewLogger(o.l.With(o.handleFields(keysAndValues)...))
}

func (o *TLogger) WithName(name string) *TLogger {
	return o.cloneWithNewLogger(o.l.Named(name))
}

// Custom additions:

func (o *TLogger) ErrorIfErr(err error, msg string, keysAndValues ...interface{}) {
	if err != nil {
		o.error(err, msg, keysAndValues)
		o.t.Fail()
	}
}

func (o *TLogger) FatalIfErr(err error, msg string, keysAndValues ...interface{}) {
	if err != nil {
		o.error(err, msg, keysAndValues)
		o.t.FailNow()
	}
}

// Proper usage is Error(msg string, key-value alternating arguments)
// Generic definition for compatibility with test.T interface
func (o *TLogger) Error(keysAndValues ...interface{}) {
	// Using o.error to have consistent call depth for Error, FatalIfErr, Info, etc
	o.error(o.errorWithRuntimeCheck(keysAndValues...))
	o.t.Fail()
}

// Proper usage is Fatal(msg string, key-value alternating arguments)
// Generic definition for compatibility with test.TLegacy interface
func (o *TLogger) Fatal(keysAndValues ...interface{}) {
	o.error(o.errorWithRuntimeCheck(keysAndValues...))
	o.t.FailNow()
}

func (o *TLogger) errorWithRuntimeCheck(keysAndValues ...interface{}) (error, string, []interface{}) {
	if len(keysAndValues) == 0 {
		return nil, "", nil
	} else {
		s, isString := keysAndValues[0].(string)
		if isString {
			// Desired case (probably)
			return nil, s, keysAndValues[1:]
		} else {
			// Treat as untrustworthy data
			if o.V(8).Enabled() {
				o.l.Sugar().Debugw("DEPRECATED Error/Fatal usage", zap.Stack("callstack"))
			}
			_, isError := keysAndValues[0].(error)
			fields := make([]interface{}, 2*len(keysAndValues))
			for i, d := range keysAndValues {
				if i == 0 && isError {
					fields[0] = "error"
					fields[1] = d
				} else {
					fields[i*2] = fmt.Sprintf("arg %d", i)
					fields[i*2+1] = d
				}
			}
			return nil, "unstructured error", fields
		}
	}
}

func (o *TLogger) Run(name string, f func(t *TLogger)) {
	tfunc := func(ts *testing.T) {
		f(newTLogger(ts, o.level))
	}
	o.t.Run(name, tfunc)
}

// Interface test.T

// Just like testing.T.Name()
func (o *TLogger) Name() string {
	return o.t.Name()
}

// T.Helper() cannot work as an indirect call, so just do nothing
func (o *TLogger) Helper() {
}

// Just like testing.T.SkipNow()
func (o *TLogger) SkipNow() {
	o.t.SkipNow()
}

// Deprecated: only existing for test.T compatibility
// Will panic if given data incompatible with Info() function
func (o *TLogger) Log(args ...interface{}) {
	o.V(2).Info(args[0].(string), args[1:]...)
}

// Just like testing.T.Parallel()
func (o *TLogger) Parallel() {
	o.t.Parallel()
}

// Interface test.TLegacy
// Fatal() is an intended function

// Deprecated. Just like testing.T.Logf()
func (o *TLogger) Logf(fmtS string, args ...interface{}) {
	o.V(2).Info(fmt.Sprintf(fmtS, args...))
}

func (o *TLogger) error(err error, msg string, keysAndValues []interface{}) {
	structuredError, ok := err.(StructuredError)
	if ok {
		newLen := len(keysAndValues) + len(structuredError.keysAndValues)
		newKAV := make([]interface{}, 0, newLen+2)
		newKAV = append(newKAV, keysAndValues...)
		newKAV = append(newKAV, structuredError.keysAndValues...)
		// This first case used if just the error is given to .Error() or .Fatal()
		if msg == "" {
			msg = structuredError.msg
		} else {
			newKAV = append(newKAV, "error", structuredError.msg)
			err = nil
		}
		keysAndValues = newKAV
	} else {
		newKAV := make([]interface{}, 0, len(keysAndValues)+1)
		newKAV = append(newKAV, keysAndValues...)
		newKAV = append(newKAV, zap.Error(err))
		keysAndValues = newKAV
	}
	if checkedEntry := o.l.Check(zap.ErrorLevel, msg); checkedEntry != nil {
		checkedEntry.Write(o.handleFields(keysAndValues)...)
	}
}

// Creation and Teardown

// Create a TLogger object using the global Zap logger and the current testing.T
func NewTLogger(t *testing.T) *TLogger {
	return newTLogger(t, Verbosity)
}

func newTLogger(t *testing.T, verbosity int) *TLogger {
	testOptions := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(2), //TODO(coryrc): some AddCallerSkip probably
		zap.Development(),
	}
	core := zaptest.NewLogger(t).Core()
	if zapCore != nil {
		core = zapcore.NewTee(
			zapCore,
			core,
			// TODO(coryrc): Open new file (maybe creating JUnit!?) with test output?
		)
	}
	log := zap.New(core).Named(t.Name()).WithOptions(testOptions...)
	tlogger := TLogger{
		l:     log,
		level: verbosity,
		t:     t,
	}
	return &tlogger
}

func (o *TLogger) cloneWithNewLogger(l *zap.Logger) *TLogger {
	t := TLogger{
		l:     l,
		level: o.level,
		t:     o.t,
	}
	return &t
}

// Please `defer t.CleanUp()` after invoking NewTLogger()
func (o *TLogger) CleanUp() {
	// Ensure nothing can log to t after test is complete
	// TODO(coryrc): except .WithName(), etc create a new logger
	//   can we somehow overwrite the core?
	//   or change the core's LevelEnabler so it can't fire!
	o.l = logger
	o.t = nil
}
