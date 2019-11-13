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
	"testing"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestRunCommand(t *testing.T) {
	testCases := []struct {
		command string
		expectedOutput string
		shouldGetError bool
	}{
		{"", "", true},
		{"   ", "", true},
		{"echo hello, world", "hello, world\n", false},
		{"unknowncommand", "", true},
	}
	for _, c := range testCases {
		out, err := RunCommand(c.command)
		if c.expectedOutput != out {
			t.Fatalf("Expect %q but actual is %q", c.expectedOutput, out)
		}
		if err == nil && c.shouldGetError {
			t.Fatal("Expect to get an error but got nil")
		}
		if err != nil && !c.shouldGetError {
			t.Fatalf("Got an error %v but should get nil", err)
		}
	}
}

func TestRunCommands(t *testing.T) {
	testCases := []struct {
		commands []string
		expectedOutput string
		shouldGetError bool
	}{
		{
			[]string{"echo 123", "echo 234", "echo 345"},
			"123\n\n234\n\n345\n",
			false,
		},
		{
			[]string{"   ", "echo 123"},
			"",
			true,
		},
		{
			[]string{"echo 123", "", "echo 234"},
			"123\n\n",
			true,
		},
		{
			[]string{"unknowncommand"},
			"",
			true,
		},
	}
	for _, c := range testCases {
		out, err := RunCommands(c.commands...)
		if c.expectedOutput != out {
			t.Fatalf("Expect %q but actual is %q", c.expectedOutput, out)
		}
		if err == nil && c.shouldGetError {
			t.Fatal("Expect to get an error but got nil")
		}
		if err != nil && !c.shouldGetError {
			t.Fatalf("Got an error %v but should get nil", err)
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