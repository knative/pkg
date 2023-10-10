/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shell_test

import (
	"bytes"
	"strconv"
	"testing"

	"knative.dev/pkg/test/shell"
)

func TestNewPrefixer(t *testing.T) {
	assert := assertions{t: t}
	var lineno int64 = 0
	tests := []struct {
		name   string
		prefix func() string
		want   string
	}{{
		"static",
		func() string {
			return "[prefix] "
		},
		`[prefix] test string 1
[prefix] test string 2
`,
	}, {
		"empty",
		func() string {
			return ""
		},
		`test string 1
test string 2
`,
	}, {
		"dynamic",
		func() string {
			lineno++
			return strconv.FormatInt(lineno, 10) + ") "
		},
		`1) test string 1
2) test string 2
`,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &bytes.Buffer{}
			wr := shell.NewPrefixer(writer, tt.prefix)
			_, err := wr.Write([]byte("test string 1\ntest string 2\n"))
			assert.NoError(err)
			got := writer.String()
			assert.Equal(tt.want, got)
		})
	}
}
