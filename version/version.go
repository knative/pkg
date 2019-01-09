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
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/version"
)

// ServerVersioner is an interface to mock the `ServerVersion`
// method of the Kubernetes client's Discovery interface.
// In an application `kubeClient.Discovery()` can be used to
// suffice this interface.
type ServerVersioner interface {
	ServerVersion() (*version.Info, error)
}

// Knative currently needs Kubernetes 1.11 at a minimum
const minimumMajor = 1
const minimumMinor = 11

// CheckMinimumVersion checks if the currently installed version of
// Kubernetes is compatible with the minimum version required.
// Returns an error if its not.
//
// A Kubernetes discovery client can be passed in as the versioner
// like `CheckMinimumVersion(kubeClient.Discovery())`.
func CheckMinimumVersion(versioner ServerVersioner) error {
	v, err := versioner.ServerVersion()
	if err != nil {
		return err
	}

	currentMajor, err := strconv.Atoi(v.Major)
	if err != nil {
		return err
	}

	currentMinor, err := strconv.Atoi(v.Minor)
	if err != nil {
		return err
	}

	if minimumMajor > currentMajor || (minimumMajor == currentMajor && minimumMinor > currentMinor) {
		minimumVersion := fmt.Sprintf("v%d.%d", minimumMajor, minimumMinor)
		currentVersion := v.String()
		return fmt.Errorf("kubernetes version %q is not compatible, need at least %q", currentVersion, minimumVersion)
	}
	return nil
}
