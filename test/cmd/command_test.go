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

package cmd

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestRunCommand(t *testing.T) {
	testCases := []struct {
		command string
		expectedOutput string
		expectedErrorCode int
	}{
		{"", "", 1},
		{"   ", "", 1},
		{"echo hello, world", "hello, world\n", 0},
		{"unknowncommand", "", 1},
	}
	for _, c := range testCases {
		out, err := RunCommand(c.command)
		if c.expectedOutput != out {
			t.Fatalf("Expect %q but actual is %q", c.expectedOutput, out)
		}
		if err != nil {
			if ce, ok := err.(*CommandLineError); ok {
				if ce.ErrorCode != c.expectedErrorCode {
					t.Fatalf("Expect to get error code %d but got %d", c.expectedErrorCode, ce.ErrorCode)
				}
			} else {
				t.Fatalf("Expect to get a CommandLineError but got %s", reflect.TypeOf(err))
			}
		} else {
			if c.expectedErrorCode != 0 {
				t.Fatalf("Expect to get an error code %d but got no error", c.expectedErrorCode)
			}
		}
	}
}

func TestRunCommands(t *testing.T) {
	testCases := []struct {
		commands []string
		expectedOutput string
		expectedErrorCode  int
	}{
		{
			[]string{"echo 123", "echo 234", "echo 345"},
			"123\n\n234\n\n345\n",
			0,
		},
		{
			[]string{"   ", "echo 123"},
			"",
			1,
		},
		{
			[]string{"echo 123", "", "echo 234"},
			"123\n\n",
			1,
		},
		{
			[]string{"unknowncommand"},
			"",
			1,
		},
		{
			[]string{"unknowncommand", "echo 123"},
			"",
			1,
		},
	}
	for _, c := range testCases {
		out, err := RunCommands(c.commands...)
		if c.expectedOutput != out {
			t.Fatalf("Expect %q but actual is %q", c.expectedOutput, out)
		}
		if err != nil {
			if ce, ok := err.(*CommandLineError); ok {
				if ce.ErrorCode != c.expectedErrorCode {
					t.Fatalf("Expect to get error code %d but got %d", c.expectedErrorCode, ce.ErrorCode)
				}
			} else {
				t.Fatalf("Expect to get a CommandLineError but got %s", reflect.TypeOf(err))
			}
		} else {
			if c.expectedErrorCode != 0 {
				t.Fatalf("Expect to get an error code %d but got no error", c.expectedErrorCode)
			}
		}
	}
}

func TestRunCommandsInParallel(t *testing.T) {
	testCases := []struct {
		commands []string
		possibleOutput sets.String
		shouldGetError bool
	}{
		{
			[]string{"echo 123", "echo 234"},
			sets.NewString("123\n\n234\n", "234\n\n123\n"),
			false,
		},
		{
			[]string{"   ", "echo 123"},
			sets.NewString("\n123\n", "123\n\n"),
			true,
		},
		{
			[]string{"echo 123", ""},
			sets.NewString("\n123\n", "123\n\n"),
			true,
		},
		{
			[]string{"unknowncommand"},
			sets.NewString(""),
			true,
		},
	}
	for _, c := range testCases {
		out, err := RunCommandsInParallel(c.commands...)
		if !c.possibleOutput.Has(out) {
			t.Fatalf("Expect output in %v but actual is %q", c.possibleOutput, out)
		}
		if err == nil && c.shouldGetError {
			t.Fatal("Expect to get an error but got nil")
		}
		if err != nil && !c.shouldGetError {
			t.Fatalf("Got an error %v but should get nil", err)
		}
	}
}
