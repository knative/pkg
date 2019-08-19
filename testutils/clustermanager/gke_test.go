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
	"strings"
	"testing"
	"time"

	"google.golang.org/api/container/v1"

	"knative.dev/pkg/testutils/common"
)

var (
	fakeProj    = "b"
	fakeCluster = "d"
)

func setupFakeGKECluster() GKECluster {
	return GKECluster{
		operations: newFakeGKESDKClient(),
	}
}

type FakeGKESDKClient struct {
	// map of parent: clusters slice
	clusters     map[string][]*container.Cluster
	regionStatus map[string]string
}

func newFakeGKESDKClient() *FakeGKESDKClient {
	return &FakeGKESDKClient{
		clusters:     make(map[string][]*container.Cluster),
		regionStatus: make(map[string]string),
	}
}

// fake create cluster, fail if cluster already exists
func (fgsc *FakeGKESDKClient) create(project, location string, rb *container.CreateClusterRequest) (*container.Operation, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	name := rb.Cluster.Name
	if cls, ok := fgsc.clusters[parent]; ok {
		for _, cl := range cls {
			if cl.Name == name {
				return nil, fmt.Errorf("cluster already exist")
			}
		}
	} else {
		fgsc.clusters[parent] = make([]*container.Cluster, 0)
	}
	cluster := &container.Cluster{
		Name:     name,
		Location: location,
		Status:   "RUNNING",
	}
	if status, ok := fgsc.regionStatus[location]; ok {
		cluster.Status = status
	}

	fgsc.clusters[parent] = append(fgsc.clusters[parent], cluster)
	return nil, nil
}

func (fgsc *FakeGKESDKClient) get(project, location, cluster string) (*container.Cluster, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	if cls, ok := fgsc.clusters[parent]; ok {
		for _, cl := range cls {
			if cl.Name == cluster {
				return cl, nil
			}
		}
	}
	return nil, fmt.Errorf("cluster not found")
}

func TestGKECheckEnvironment(t *testing.T) {
	datas := []struct {
		kubectlOut   string
		kubectlErr   error
		gcloudOut    string
		gcloudErr    error
		clusterExist bool
		expProj      *string
		expCluster   *string
		expErr       error
	}{
		{
			// Base condition, kubectl shouldn't return empty string if there is no error
			"", nil, "", nil, false, nil, nil, nil,
		}, {
			// kubeconfig not set and gcloud not set
			"", fmt.Errorf("kubectl not set"), "", nil, false, nil, nil, nil,
		}, {
			// kubeconfig failed
			"failed", fmt.Errorf("kubectl other err"), "", nil, false, nil, nil, fmt.Errorf("failed running kubectl config current-context: 'failed'"),
		}, {
			// kubeconfig returned something other than "gke_PROJECT_REGION_CLUSTER"
			"gke_b_c", nil, "", nil, false, nil, nil, nil,
		}, {
			// kubeconfig returned something other than "gke_PROJECT_REGION_CLUSTER"
			"gke_b_c_d_e", nil, "", nil, false, nil, nil, nil,
		}, {
			// kubeconfig correctly set and cluster exist
			"gke_b_c_d", nil, "", nil, true, &fakeProj, &fakeCluster, nil,
		}, {
			// kubeconfig correctly set, but cluster doesn't exist
			"gke_b_c_d", nil, "", nil, false, &fakeProj, nil, fmt.Errorf("couldn't find cluster d in b in c, does it exist? cluster not found"),
		}, {
			// kubeconfig not set and gcloud failed
			"", fmt.Errorf("kubectl not set"), "", fmt.Errorf("gcloud failed"), false, nil, nil, fmt.Errorf("failed getting gcloud project: 'gcloud failed'"),
		}, {
			// kubeconfig not set and gcloud not set
			"", fmt.Errorf("kubectl not set"), "", nil, false, nil, nil, nil,
		}, {
			// kubeconfig not set and gcloud set
			"", fmt.Errorf("kubectl not set"), "b", nil, false, &fakeProj, nil, nil,
		},
	}

	oldFunc := common.StandardExec
	defer func() {
		// restore
		common.StandardExec = oldFunc
	}()

	for _, data := range datas {
		fgc := setupFakeGKECluster()
		if data.clusterExist {
			parts := strings.Split(data.kubectlOut, "_")
			fgc.operations.create(parts[1], parts[2], &container.CreateClusterRequest{
				Cluster: &container.Cluster{
					Name: parts[3],
				},
				ProjectId: parts[1],
			})
		}
		// mock for testing
		common.StandardExec = func(name string, args ...string) ([]byte, error) {
			var out []byte
			var err error
			switch name {
			case "gcloud":
				out = []byte(data.gcloudOut)
				err = data.gcloudErr
			case "kubectl":
				out = []byte(data.kubectlOut)
				err = data.kubectlErr
			}
			return out, err
		}

		err := fgc.checkEnvironment()
		var clusterGot *string
		if nil != fgc.Cluster {
			clusterGot = &fgc.Cluster.Name
		}

		if !reflect.DeepEqual(err, data.expErr) || !reflect.DeepEqual(fgc.Project, data.expProj) || !reflect.DeepEqual(clusterGot, data.expCluster) {
			t.Errorf("check environment with:\n\tkubectl output: '%s'\n\t\terror: '%v'\n\tgcloud output: '%s'\n\t\t"+
				"error: '%v'\nwant: project - '%v', cluster - '%v', err - '%v'\ngot: project - '%v', cluster - '%v', err - '%v'",
				data.kubectlOut, data.kubectlErr, data.gcloudOut, data.gcloudErr, data.expProj, data.expCluster, data.expErr, fgc.Project, fgc.Cluster, err)
		}
	}
}

func TestAcquire(t *testing.T) {
	fakeClusterName := "kpkg-e2e-cls-1234"
	fakeBuildID := "1234"
	datas := []struct {
		existCluster       *container.Cluster
		regionStates       map[string]string
		expClusterName     string
		expClusterLocation string
		expErr             error
	}{
		{
			// cluster already found
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, map[string]string{}, "customcluster", "us-central1", nil,
		}, {
			// cluster creation succeeded
			nil, map[string]string{}, fakeClusterName, "us-central1", nil,
		}, {
			// cluster creation succeeded retry
			nil, map[string]string{"us-central1": "PROVISIONING"}, fakeClusterName, "us-west1", nil,
		}, {
			// cluster creation failed all retry
			nil, map[string]string{"us-central1": "PROVISIONING", "us-west1": "PROVISIONING", "us-east1": "PROVISIONING"},
			"", "", fmt.Errorf("timed out waiting for cluster creation"),
		}, {
			// cluster creation went bad state
			nil, map[string]string{"us-central1": "BAD", "us-west1": "BAD", "us-east1": "BAD"}, "", "", fmt.Errorf("cluster in bad state: 'BAD'"),
		},
	}

	// mock GetOSEnv for testing
	oldFunc := common.GetOSEnv
	// mock timeout so it doesn't run forever
	oldTimeout := creationTimeout
	creationTimeout = 100 * time.Millisecond
	defer func() {
		// restore
		common.GetOSEnv = oldFunc
		creationTimeout = oldTimeout
	}()

	for _, data := range datas {
		common.GetOSEnv = func(key string) string {
			switch key {
			case "BUILD_NUMBER":
				return fakeBuildID
			case "PROW_JOB_ID": // needed to mock IsProw()
				return "jobid"
			}
			return oldFunc(key)
		}
		fgc := setupFakeGKECluster()
		if nil != data.existCluster {
			fgc.Cluster = data.existCluster
		}
		fgc.Project = &fakeProj
		fgc.operations.(*FakeGKESDKClient).regionStatus = data.regionStates

		fgc.Request = &GKERequest{
			NumNodes:      DefaultGKENumNodes,
			NodeType:      DefaultGKENodeType,
			Region:        DefaultGKERegion,
			Zone:          "",
			BackupRegions: DefaultGKEBackupRegions,
		}
		err := fgc.Acquire()
		var gotName, gotLocation string
		if nil != fgc.Cluster {
			gotName = fgc.Cluster.Name
			gotLocation = fgc.Cluster.Location
		}
		if !reflect.DeepEqual(err, data.expErr) || data.expClusterName != gotName || data.expClusterLocation != gotLocation {
			t.Errorf("testing acquiring cluster, with:\n\texisting cluster: '%v'\n\tbad regions: '%v'\nwant: cluster name - '%s', location - '%s', err - '%v'\ngot: cluster name - '%s', location - '%s', err - '%v'",
				data.existCluster, data.regionStates, data.expClusterName, data.expClusterLocation, data.expErr, gotName, gotLocation, err)
		}
	}
}
