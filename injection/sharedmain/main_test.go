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

/*
copyright 2019 the knative authors

licensed under the apache license, version 2.0 (the "license");
you may not use this file except in compliance with the license.
you may obtain a copy of the license at

    http://www.apache.org/licenses/license-2.0

unless required by applicable law or agreed to in writing, software
distributed under the license is distributed on an "as is" basis,
without warranties or conditions of any kind, either express or implied.
see the license for the specific language governing permissions and
limitations under the license.
*/
package sharedmain

import "testing"

func TestComponentNameValidator(t *testing.T) {

	tests := []struct {
		name          string
		componentName string
		valid         bool
	}{
		{name: "invalid - empty name", componentName: "", valid: false},
		{name: "invalid - contains dashes", componentName: "inmemorychannel-controller", valid: false},
		{name: "valid", componentName: "foo_bar_is_correct", valid: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			ok := isComponentNameValid(test.componentName)
			if ok != test.valid {
				t.Errorf("got %t, want %t", ok, test.valid)
			}

		})
	}

}
