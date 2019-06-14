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

package v1beta1

import (
	"testing"

	"github.com/knative/pkg/apis/duck/unversioned"
)

func TestTypesImplements(t *testing.T) {
	testCases := []struct {
		instance interface{}
		iface    unversioned.Implementable
	}{
		{instance: &AddressableType{}, iface: &Addressable{}},
		{instance: &KResource{}, iface: &Conditions{}},
	}
	for _, tc := range testCases {
		if err := unversioned.VerifyType(tc.instance, tc.iface); err != nil {
			t.Error(err)
		}
	}
}
