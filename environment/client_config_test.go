/*
Copyright 2023 The Knative Authors

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

package environment

import (
	"flag"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInitFlag(t *testing.T) {
	t.Setenv("KUBE_API_BURST", "50")
	t.Setenv("KUBE_API_QPS", "60.1")
	t.Setenv("KUBECONFIG", "myconfig")

	c := new(ClientConfig)
	c.InitFlags(flag.CommandLine)

	// Override kube-api-burst via command line option.
	flag.CommandLine.Set("kube-api-burst", strconv.Itoa(100))

	// Call parse() here as InitFlags does not call it.
	flag.Parse()

	expect := &ClientConfig{
		Burst:      100,
		QPS:        60.1,
		Kubeconfig: "myconfig",
	}

	if !cmp.Equal(c, expect) {
		t.Errorf("ClientConfig mismatch: diff(-want,+got):\n%s", cmp.Diff(expect, c))
	}
}
