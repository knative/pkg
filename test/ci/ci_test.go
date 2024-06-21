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

package ci

import (
	"testing"
)

func TestIsCI(t *testing.T) {
	t.Setenv("CI", "true")
	if ic := IsCI(); !ic {
		t.Fatal("Expected: true, actual: false")
	}
}

func TestGetArtifacts(t *testing.T) {
	// Test we can read from the env var
	t.Setenv("ARTIFACTS", "test")
	v := GetLocalArtifactsDir()
	if v != "test" {
		t.Fatalf("Actual artifacts dir: '%s' and Expected: 'test'", v)
	}

	// Test we can use the default
	t.Setenv("ARTIFACTS", "")
	v = GetLocalArtifactsDir()
	if v != "artifacts" {
		t.Fatalf("Actual artifacts dir: '%s' and Expected: 'artifacts'", v)
	}
}
