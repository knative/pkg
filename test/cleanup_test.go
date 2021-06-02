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
		CleanupOnInterrupt(func() { fmt.Println("cleanup 1") })
		CleanupOnInterrupt(func() { fmt.Println("cleanup 2") })
		CleanupOnInterrupt(func() { fmt.Println("cleanup 3") })
		fmt.Println("ready")
		time.Sleep(5 * time.Second)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestCleanupOnInterrupt", "-test.v=true")
	cmd.Env = append(os.Environ(), "CLEANUP=1")

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

	err = wait.PollImmediate(100*time.Millisecond, 2*time.Second, func() (bool, error) {
		return bytes.Contains(output.Bytes(), []byte("ready")), nil
	})
	if err != nil {
		t.Fatal("Test subprocess never became ready", err)
	}
	if err := p.Signal(os.Interrupt); err != nil {
		t.Fatal("Failed to interrupt", err)
	}

	err = cmd.Wait()
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		t.Fatal("Running test had abnormal exit", err)
	}

	testOutput := string(output.Bytes())

	idx1 := strings.Index(testOutput, "cleanup 1")
	idx2 := strings.Index(testOutput, "cleanup 2")
	idx3 := strings.Index(testOutput, "cleanup 3")

	// Order is first in first out (3, 2, 1)
	if idx3 > idx2 || idx2 > idx1 || idx1 == -1 {
		t.Errorf("Cleanup functions were not invoked in the proper order")
	}
}
