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

package logging_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"go.uber.org/zap"
	"gotest.tools/v3/assert/cmp"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/system/posix/exit"

	"gotest.tools/v3/assert"
)

func TestExitingZapcore(t *testing.T) {
	config, err := logging.NewConfigFromMap(map[string]string{})
	assert.NilError(t, err)
	logfile := path.Join(t.TempDir(), "test.log")
	config.LoggingConfig = fmt.Sprintf(`{"outputPaths": ["%s"]}`, logfile)
	log, _ := logging.NewLoggerFromConfig(config, "test", logging.ExitingZapcore)
	err = errors.New("bar")
	log = log.With(zap.Error(err))

	ex := exit.WithStub(func() {
		log.Fatal("foo")
	})

	assert.Check(t, ex.Exited)
	assert.Equal(t, ex.Code, 129)

	logBytes, err := ioutil.ReadFile(logfile)
	assert.NilError(t, err)
	logs := string(logBytes)
	assert.Check(t, cmp.Contains(logs, `"message":"foo"`))
	assert.Check(t, cmp.Contains(logs, `"error":"bar"`))
}
