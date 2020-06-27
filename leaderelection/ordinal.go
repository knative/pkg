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

package leaderelection

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// If run a process on Kubernetes, the value of this environment variable
// should be set to the pod name via the downward API.
const controllerOrdinalEnv = "CONTROLLER_ORDINAL"

// ParseControllerOrdinal returns the parts of the StatefulSet's pod name.
func ParseControllerOrdinal() (string, uint64, error) {
	v := os.Getenv(controllerOrdinalEnv)
	if i := strings.LastIndex(v, "-"); i != -1 {
		ui, err := strconv.ParseUint(v[i+1:], 10, 64)
		return v[:i], ui, err
	}

	return "", 0, fmt.Errorf("ordinal not found in %s=%s", controllerOrdinalEnv, v)
}

// ControllerOrdinal tries to get ordinal from the pod name of a StatefulSet,
// which is provided from the environment variable CONTROLLER_ORDINAL.
func ControllerOrdinal() (uint64, error) {
	_, ui, err := ParseControllerOrdinal()
	return ui, err
}
