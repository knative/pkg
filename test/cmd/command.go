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
	"os/exec"
	"strings"
)

// Run will run the command and return the standard output and error if there is one.
func Run(cmdLine string) (string, error) {
	cmdSplit := strings.Fields(cmdLine)
	cmd := cmdSplit[0]
	args := cmdSplit[1:]
	cmdOut, err := exec.Command(cmd, args...).Output()
	if err != nil {
		errorCode := getErrorCode(err)
		commandLineErr := CommandLineError{Command: cmdLine, ErrorOutput: err.Error(), ErrorCode: errorCode}
		return string(cmdOut), commandLineErr
	}

	return string(cmdOut), nil
}

// RunBatchSequentially will run the command sequentially.
// If there is an error when running a command, it will return directly and the following commands will not be run.
func RunBatchSequentially(cmdLines ...string) (string, error) {
	var sb strings.Builder
	for _, cmdLine := range cmdLines {
		output, err := Run(cmdLine)
		sb.WriteString(output)
		sb.WriteRune('\n')
		if err != nil {
			return strings.TrimRight(sb.String(), "\n"), err
		}
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// RunBatchParallelly will run the command in parallel.
// It will always finish running all commands, and return all standard output and errors together.
func RunBatchParallelly(cmdLines ...string) (string, error) {

	return "", nil
}

// getErrorCode returns the exit code of the command run
func getErrorCode(err error) int {
	errorCode := -1
	if exitError, ok := err.(*exec.ExitError); ok {
		errorCode = exitError.ExitCode()
	}
	return errorCode
}
