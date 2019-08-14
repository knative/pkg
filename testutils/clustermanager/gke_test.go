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
	"fmt"
	"reflect"
	"testing"
)

var (
	fakeProj    = "b"
	fakeCluster = "d"
)

func TestGKECheckEnvironment(t *testing.T) {
	datas := []struct {
		kubectlOut string
		kubectlErr error
		gcloudOut  string
		gcloudErr  error
		expProj    *string
		expCluster *string
		expErr     error
	}{
		{
			// Base condition, kubectl shouldn't return empty string if there is no error
			"", nil, "", nil, nil, nil, fmt.Errorf("kubectl current-context is malformed: ''"),
		}, {
			// kubeconfig not set and gcloud not set
			"", fmt.Errorf("kubectl not set"), "", nil, nil, nil, nil,
		}, {
			// kubeconfig failed
			"failed", fmt.Errorf("kubectl other err"), "", nil, nil, nil, fmt.Errorf("failed running kubectl config current-context: 'failed'"),
		}, {
			// kubeconfig returned something other than "gke_PROJECT_REGION_CLUSTER"
			"a_b_c", nil, "", nil, nil, nil, fmt.Errorf("kubectl current-context is malformed: 'a_b_c'"),
		}, {
			// kubeconfig returned something other than "gke_PROJECT_REGION_CLUSTER"
			"a_b_c_d_e", nil, "", nil, nil, nil, fmt.Errorf("kubectl current-context is malformed: 'a_b_c_d_e'"),
		}, {
			// kubeconfig correctly set
			"a_b_c_d", nil, "", nil, &fakeProj, &fakeCluster, nil,
		}, {
			// kubeconfig not set and gcloud failed
			"", fmt.Errorf("kubectl not set"), "", fmt.Errorf("gcloud failed"), nil, nil, fmt.Errorf("failed getting gcloud project: 'gcloud failed'"),
		}, {
			// kubeconfig not set and gcloud not set
			"", fmt.Errorf("kubectl not set"), "", nil, nil, nil, nil,
		}, {
			// kubeconfig not set and gcloud set
			"", fmt.Errorf("kubectl not set"), "b", nil, &fakeProj, nil, nil,
		},
	}

	for _, data := range datas {
		gc := GKECluster{
			Exec: func(name string, args ...string) ([]byte, error) {
				var out []byte
				var err error
				if "gcloud" == name {
					out = []byte(data.gcloudOut)
					err = data.gcloudErr
				} else if "kubectl" == name {
					out = []byte(data.kubectlOut)
					err = data.kubectlErr
				}
				return out, err
			},
		}
		if err := gc.checkEnvironment(); !reflect.DeepEqual(err, data.expErr) || !reflect.DeepEqual(gc.Project, data.expProj) || !reflect.DeepEqual(gc.Cluster, data.expCluster) {
			t.Errorf("check environment with:\n\tkubectl output: '%s'\n\t\terror: '%v'\n\tgcloud output: '%s'\n\t\t"+
				"error: '%v'\nwant: project - '%v', cluster - '%v', err - '%v'\ngot: project - '%v', cluster - '%v', err - '%v'",
				data.kubectlOut, data.kubectlErr, data.gcloudOut, data.gcloudErr, data.expProj, data.expCluster, data.expErr, gc.Project, gc.Cluster, err)
		}
	}
}
