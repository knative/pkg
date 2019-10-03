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

package fake

import (
	"fmt"
	"testing"

	"knative.dev/pkg/testutils/gke"
)

func TestWait(t *testing.T) {
	client := NewGKESDKClient()
	request, _ := gke.NewCreateClusterRequest(&gke.Request{
		Project:     "test-project",
		ClusterName: "test-cluster-name",
		MinNodes:    1,
		MaxNodes:    1,
		NodeType:    "n1-standard-4",
	})
	err := client.CreateCluster("test-project", "test-region", request)
	fmt.Printf("error is %v", err)
}
