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

package configmap

import (
	"crypto/sha256"
	"fmt"
)

const (
	// ExampleKey signifies a given example configuration in a ConfigMap.
	ExampleKey = "_example"

	// ExampleHashLabel is the label that stores the computed hash.
	ExampleHashLabel = "knative.dev/example-hash"
)

// ExampleHash generates a hash for the example value to be compared against a respective label.
func ExampleHash(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))[:9]
}
