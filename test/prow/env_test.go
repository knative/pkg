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

package prow

import (
	"testing"
)

func TestGetEnvConfig(t *testing.T) {
	t.Setenv("CI", "true")
	ec, err := GetEnvConfig()
	t.Log("EnvConfig is:", ec)
	if err != nil {
		t.Fatal("Error getting envconfig for Prow:", err)
	}
	if !ec.CI {
		t.Fatal("Expected CI to be true but is false")
	}

	t.Setenv("CI", "false")
	if _, err = GetEnvConfig(); err == nil {
		t.Fatal("Expected an error if called from a non-CI environment but got nil")
	}
}
