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
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"knative.dev/pkg/test/helpers"
)

// Run will run the command and return the standard output, plus error if there is one.
func Run(cmdLine string) (string, error) {
	cmdSplit := strings.Fields(cmdLine)
	if len(cmdSplit) == 0 {
		return "", fmt.Errorf("the command line %q cannot be empty", cmdLine)
	}

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
// If there is an error when running a command, it will return directly with all standard output so far and the error.
func RunBatchSequentially(cmdLines ...string) (string, error) {
	var outputs []string
	for _, cmdLine := range cmdLines {
		output, err := Run(cmdLine)
		outputs = append(outputs, output)
		if err != nil {
			return combineOutputs(outputs), fmt.Errorf("error happened when running %q: %v", cmdLine, err)
		}
	}
	return combineOutputs(outputs), nil
}

// RunBatchParallelly will run the command in parallel.
// It will always finish running all commands, and return all standard output and errors together.
func RunBatchParallelly(cmdLines ...string) (string, error) {
	errCh := make(chan error)
	outputCh := make(chan string)
	wg := sync.WaitGroup{}
	for i := range cmdLines {
		cmdLine := cmdLines[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			output, err := Run(cmdLine)
			outputCh <- output
			if err != nil {
				errCh <- err
			}
		}()
	}

	go func() {
		wg.Wait()
		close(outputCh)
		close(errCh)
	}()

	outputs := make([]string, 0)
	errs := make([]error, 0)
	for output := range outputCh {
		outputs = append(outputs, output)
	}
	for err := range errCh {
		errs = append(errs, err)
	}

	return combineOutputs(outputs), helpers.CombineErrors(errs)
}

// combineOutputs will combine the slice of output strings to a single string
func combineOutputs(outputs []string) string {
	var sb strings.Builder
	for _, output := range outputs {
		sb.WriteString("===================================")
		sb.WriteString(output)
		sb.WriteRune('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

// getErrorCode returns the exit code of the command run
func getErrorCode(err error) int {
	errorCode := -1
	if exitError, ok := err.(*exec.ExitError); ok {
		errorCode = exitError.ExitCode()
	}
	return errorCode
}
