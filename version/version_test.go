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

package version

import (
	"errors"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/version"
)

type testVersioner struct {
	major string
	minor string
	err   error
}

func (t *testVersioner) ServerVersion() (*version.Info, error) {
	return &version.Info{Major: t.major, Minor: t.minor}, t.err
}

func TestVersionCheck(t *testing.T) {
	minimumVersion := fmt.Sprintf("v%d.%d", minimumMajor, minimumMinor)

	tests := []struct {
		name          string
		actualVersion *testVersioner
		wantError     bool
	}{{
		name:          "greater version (minor)",
		actualVersion: &testVersioner{major: "1", minor: "12"},
	}, {
		name:          "same version",
		actualVersion: &testVersioner{major: "1", minor: "11"},
	}, {
		name:          "smaller version",
		actualVersion: &testVersioner{major: "1", minor: "10"},
		wantError:     true,
	}, {
		name:          "error while fetching",
		actualVersion: &testVersioner{err: errors.New("random error")},
		wantError:     true,
	}}

	for _, test := range tests {
		err := CheckMinimumVersion(test.actualVersion)
		if err == nil && test.wantError {
			t.Errorf("Expected an error for minimum: %q, actual: %v", minimumVersion, test.actualVersion)
		}

		if err != nil && !test.wantError {
			t.Errorf("Expected no error but got %v for minimum: %q, actual: %v", err, minimumVersion, test.actualVersion)
		}
	}
}
