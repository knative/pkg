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

package ha

import "testing"

func TestExtractDeployment(t *testing.T) {
	const want = "gke-cluster-michigan-pool-2"
	if got := extractDeployment("gke-cluster-michigan-pool-2-03f384a0-2zu1"); got != want {
		t.Errorf("Deployment = %q, want: %q", got, want)
	}
	if got := extractDeployment("a-b"); got != "" {
		t.Errorf("Deployment = %q, want empty string", got)
	}
}
