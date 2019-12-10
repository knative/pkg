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

	"github.com/davecgh/go-spew/spew"
)

type StructuredError struct {
	msg           string
	keysAndValues []interface{}
}

// Implement `error` interface
func (e StructuredError) Error() string {
	// TODO(coryrc): accept zap.Field entries?
	return fmt.Sprint(e.msg, spew.Sprintf("%+#v", e.keysAndValues))
}

// Create a StructuredError. Gives a little better logging when given to a TLogger.
// TODO(coryrc): problem; if we don't convert them right away and they get mutated
//   maybe save string representation right away just in case?
func Error(msg string, keysAndValues ...interface{}) *StructuredError {
	return &StructuredError{msg, keysAndValues}
}

func (e *StructuredError) WithValues(keysAndValues ...interface{}) *StructuredError {
	newKAV := make([]interface{}, 0, len(keysAndValues)+len(e.keysAndValues))
	newKAV = append(newKAV, e.keysAndValues...)
	newKAV = append(newKAV, keysAndValues...)
	return &StructuredError{e.msg, newKAV}
}
