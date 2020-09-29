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

package gke

import "testing"

func TestServiceEndpoint(t *testing.T) {
	datas := []struct {
		env           string
		want          string
		errorExpected bool
	}{
		{"", "", true},
		{testEnv, testEndpoint, false},
		{stagingEnv, stagingEndpoint, false},
		{staging2Env, staging2Endpoint, false},
		{prodEnv, prodEndpoint, false},
		{"invalid_url", "", true},
		{"https://custom.container.googleapis.com/", "https://custom.container.googleapis.com/", false},
	}
	for _, data := range datas {
		got, err := ServiceEndpoint(data.env)
		if got != data.want {
			t.Errorf("Service endpoint for %q = %q, want: %q",
				data.env, got, data.want)
		}
		if err != nil && !data.errorExpected {
			t.Error("Error is not expected by got", err)
		}
		if err == nil && data.errorExpected {
			t.Error("Expected one error but got nil")
		}
	}
}
