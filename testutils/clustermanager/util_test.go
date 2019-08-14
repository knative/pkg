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

package clustermanager

import (
	"testing"

	"knative.dev/pkg/testutils/common"
)

func TestGetResourceName(t *testing.T) {
	buildNumStr := "12345678901234567890fakebuildnum"
	expOut := "kpkg-e2e-cls"
	if common.IsProw() {
		expOut = "kpkg-e2e-cls-12345678901234567890"
	}

	// mock GetOSEnv for testing
	oldFunc := common.GetOSEnv
	defer func() {
		// restore GetOSEnv
		common.GetOSEnv = oldFunc
	}()
	common.GetOSEnv = func(key string) string {
		return buildNumStr
	}

	out, err := getResourceName(ClusterResource)
	if nil != err {
		t.Fatalf("getting resource name for cluster, wanted: 'no error', got: '%v'", err)
	}
	if out != expOut {
		t.Fatalf("getting resource name for cluster, wanted: '%s', got: '%s'", expOut, out)
	}
}
