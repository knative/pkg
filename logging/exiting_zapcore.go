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

package logging

import (
	"errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"knative.dev/pkg/system/posix/exit"
	"knative.dev/pkg/system/posix/retcode"
)

// ExitingZapcore is a zap.Option which will exit the process with deterministic
// POSIX retcode for every Fatal+ invocation.
var ExitingZapcore = zap.WrapCore(func(core zapcore.Core) zapcore.Core {
	return retcodeCore{base: core}
})

type retcodeCore struct {
	base   zapcore.Core
	fields []zapcore.Field
}

func (r retcodeCore) Enabled(level zapcore.Level) bool {
	return r.base.Enabled(level)
}

func (r retcodeCore) With(fields []zapcore.Field) zapcore.Core {
	return retcodeCore{
		base:   r.base.With(fields),
		fields: fields,
	}
}

func (r retcodeCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if entry.Level >= zapcore.DPanicLevel {
		return ce.AddCore(entry, r)
	}
	return r.base.Check(entry, ce)
}

func (r retcodeCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if err := r.base.Write(entry, fields); err != nil {
		return err
	}
	code := r.calculateRetcode(entry, fields)
	_ = r.Sync()
	exit.Exit(code)
	return nil
}

func (r retcodeCore) Sync() error {
	return r.base.Sync()
}

func (r retcodeCore) calculateRetcode(entry zapcore.Entry, fields []zapcore.Field) int {
	err := errors.New(entry.Message)
	for _, field := range append(r.fields, fields...) {
		if field.Type == zapcore.ErrorType {
			err = field.Interface.(error)
			break
		}
	}
	return retcode.Calc(err)
}
