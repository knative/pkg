/*
Copyright 2021 The Knative Authors

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

package test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func TestCleanupOnInterrupt(t *testing.T) {
	if os.Getenv("CLEANUP") == "1" {
		OnInterrupt(func() { fmt.Println("cleanup 1") })
		OnInterrupt(func() { fmt.Println("cleanup 2") })
		OnInterrupt(func() { fmt.Println("cleanup 3") })

		// This signals to the parent test that it should proceed
		os.Remove(os.Getenv("READY_FILE"))

		time.Sleep(5 * time.Second)
		return
	}

	readyFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("failed to setup tests")
	}
	readyFile.Close()

	cmd := exec.Command(os.Args[0], "-test.run=TestCleanupOnInterrupt", "-test.v=true")
	cmd.Env = append(os.Environ(), "CLEANUP=1", "READY_FILE="+readyFile.Name())

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Start(); err != nil {
		t.Fatal("Running test failed", err)
	}

	p, err := os.FindProcess(cmd.Process.Pid)
	if err != nil {
		t.Fatal("Failed to find process", err)
	}

	// poll until the ready file is gone - indicating the subtest has been set up
	// with the cleanup functions
	err = wait.PollUntilContextTimeout(context.Background(), 100*time.Millisecond, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		_, err := os.Stat(readyFile.Name())
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	})

	if err != nil {
		t.Fatal("Test subprocess never became ready", err)
	}

	if err := p.Signal(os.Interrupt); err != nil {
		t.Fatal("Failed to interrupt", err)
	}

	err = cmd.Wait()
	var exitErr *exec.ExitError
	if ok := errors.As(err, &exitErr); err != nil && !ok {
		t.Fatal("Running test had abnormal exit", err)
	}

	testOutput := output.String()

	idx1 := strings.Index(testOutput, "cleanup 1")
	idx2 := strings.Index(testOutput, "cleanup 2")
	idx3 := strings.Index(testOutput, "cleanup 3")

	// Order is first in first out (3, 2, 1)
	if idx3 > idx2 || idx2 > idx1 || idx1 == -1 {
		t.Errorf("Cleanup functions were not invoked in the proper order")
	}
}
