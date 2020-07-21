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
	"strings"
	"testing"
)

func TestLoggerWriter(t *testing.T) {
	var strs []string
	fn := FormatLogger(func(template string, args ...interface{}) {
		strs = append(strs, template)
	})

	loggerWriter := NewLoggerWriter("hello: ", fn)
	_, err := loggerWriter.Write([]byte("aaaa"))
	if err != nil {
		t.Fatalf("Error while writing: %v", err)
	}
	_, err = loggerWriter.Write([]byte("bbb\nbbb\nbbb"))
	if err != nil {
		t.Fatalf("Error while writing: %v", err)
	}
	_, err = loggerWriter.Write([]byte("\nccc\n"))
	if err != nil {
		t.Fatalf("Error while writing: %v", err)
	}
	_, err = loggerWriter.Write([]byte("ddd\n"))
	if err != nil {
		t.Fatalf("Error while writing: %v", err)
	}

	actual := strings.Join(strs, "-")
	expected := "hello: aaaabbb-hello: bbb-hello: bbb-hello: ccc-hello: ddd"
	if actual != expected {
		t.Fatalf("expected: %s, actual: %s", expected, actual)
	}

}
