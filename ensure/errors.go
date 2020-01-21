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
	"fmt"
	"github.com/pkg/errors"
	"regexp"
)

// NoError will panic if given an error, as it was unexpected
func NoError(err error) {
	if err != nil {
		panic(errors.WithMessage(err, "unexpected error"))
	}
}

// Error will panic if given no error, as it expected one
func Error(err error) {
	if err == nil {
		panic(errors.New("expecting error, but none given"))
	}
}

// ErrorWithMessage will panic if given no error, or error message don't match provided regexp
func ErrorWithMessage(err error, messageRegexp string) {
	Error(err)
	validErrorMessage := regexp.MustCompile(messageRegexp)
	if !validErrorMessage.MatchString(err.Error()) {
		panic(errors.WithMessage(
			err,
			fmt.Sprintf("given error doesn't match given regexp (%s)", messageRegexp),
		))
	}
}
