// Copyright 2020 Knative Authors
// Copyright 2018 Solly Ross
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The useful parts of this file have been preserved
// from their origin at https://github.com/go-logr/zapr/tree/8f2487342d52a33a1793e50e3ca04bc1767aa65c

package logging

import (
	"github.com/go-logr/logr"
)

// noopInfoLogger is a logr.InfoLogger that's always disabled, and does nothing.
type noopInfoLogger struct{}

func (l *noopInfoLogger) Enabled() bool                                             { return false }
func (l *noopInfoLogger) Info(_ string, _ ...interface{})                           {}
func (l *noopInfoLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
func (l *noopInfoLogger) V(level int) logr.InfoLogger                               { return &noopInfoLogger{} }
func (l *noopInfoLogger) WithValues(keysAndValues ...interface{}) logr.InfoLogger {
	return &noopInfoLogger{}
}
func (l *noopInfoLogger) WithName(name string) logr.InfoLogger {
	return &noopInfoLogger{}
}

var disabledInfoLogger = &noopInfoLogger{}

// infoLogger is a logr.InfoLogger that uses Zap to log at a particular
// level.
type infoLogger struct {
	logrLevel int
	t         *TLogger
}

func (i *infoLogger) Enabled() bool { return true }
func (i *infoLogger) Info(msg string, keysAndVals ...interface{}) {
	i.indirectWrite(msg, keysAndVals...)
}
func (i *infoLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
func (i *infoLogger) V(level int) logr.InfoLogger                               { return &infoLogger{} }
func (i *infoLogger) WithValues(keysAndValues ...interface{}) logr.InfoLogger   { return &infoLogger{} }
func (i *infoLogger) WithName(name string) logr.InfoLogger                      { return &infoLogger{} }

// This function just exists to have consistent 2-level call depth for Zap proxying
func (i *infoLogger) indirectWrite(msg string, keysAndVals ...interface{}) {
	lvl := zapLevelFromLogrLevel(i.logrLevel)
	if checkedEntry := i.t.l.Check(lvl, msg); checkedEntry != nil {
		checkedEntry.Write(i.t.handleFields(keysAndVals)...)
	}
}
