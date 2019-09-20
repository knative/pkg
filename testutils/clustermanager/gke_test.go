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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	container "google.golang.org/api/container/v1beta1"
	boskoscommon "k8s.io/test-infra/boskos/common"
	boskosFake "knative.dev/pkg/testutils/clustermanager/boskos/fake"
	"knative.dev/pkg/testutils/common"

	"github.com/google/go-cmp/cmp"
)

var (
	fakeProj    = "b"
	fakeCluster = "d"
)

func setupFakeGKECluster() GKECluster {
	return GKECluster{
		operations: newFakeGKESDKClient(),
		boskosOps:  &boskosFake.FakeBoskosClient{},
	}
}

type FakeGKESDKClient struct {
	// map of parent: clusters slice
	clusters map[string][]*container.Cluster
	// map of operationID: operation
	ops map[string]*container.Operation

	// An incremental number for new ops
	opNumber int
	// A lookup table for determining ops statuses
	opStatus map[string]string
}

func newFakeGKESDKClient() *FakeGKESDKClient {
	return &FakeGKESDKClient{
		clusters: make(map[string][]*container.Cluster),
		ops:      make(map[string]*container.Operation),
		opStatus: make(map[string]string),
	}
}

// automatically registers new ops, and mark it "DONE" by default. Update
// fgsc.opStatus by fgsc.opStatus[string(fgsc.opNumber+1)]="PENDING" to make the
// next operation pending
func (fgsc *FakeGKESDKClient) newOp() *container.Operation {
	opName := strconv.Itoa(fgsc.opNumber)
	op := &container.Operation{
		Name:   opName,
		Status: "DONE",
	}
	if status, ok := fgsc.opStatus[opName]; ok {
		op.Status = status
	}
	fgsc.opNumber++
	fgsc.ops[opName] = op
	return op
}

// fake create cluster, fail if cluster already exists
func (fgsc *FakeGKESDKClient) create(project, location string, rb *container.CreateClusterRequest) (*container.Operation, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	name := rb.Cluster.Name
	if cls, ok := fgsc.clusters[parent]; ok {
		for _, cl := range cls {
			if cl.Name == name {
				return nil, errors.New("cluster already exist")
			}
		}
	} else {
		fgsc.clusters[parent] = make([]*container.Cluster, 0)
	}
	cluster := &container.Cluster{
		Name:         name,
		Location:     location,
		Status:       "RUNNING",
		AddonsConfig: rb.Cluster.AddonsConfig,
		NodePools: []*container.NodePool{
			{
				Name: "default-pool",
			},
		},
	}
	if rb.Cluster.MasterAuth != nil {
		cluster.MasterAuth = &container.MasterAuth{
			Username: rb.Cluster.MasterAuth.Username,
		}
	}

	fgsc.clusters[parent] = append(fgsc.clusters[parent], cluster)
	return fgsc.newOp(), nil
}

func (fgsc *FakeGKESDKClient) delete(project, clusterName, location string) (*container.Operation, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	found := -1
	if clusters, ok := fgsc.clusters[parent]; ok {
		for i, cluster := range clusters {
			if cluster.Name == clusterName {
				found = i
			}
		}
	}
	if found == -1 {
		return nil, fmt.Errorf("cluster %q not found for deletion", clusterName)
	}
	// Delete this cluster
	fgsc.clusters[parent] = append(fgsc.clusters[parent][:found], fgsc.clusters[parent][found+1:]...)
	return fgsc.newOp(), nil
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

func (fgsc *FakeGKESDKClient) getOperation(project, location, opName string) (*container.Operation, error) {
	if op, ok := fgsc.ops[opName]; ok {
		return op, nil
	}
	return nil, fmt.Errorf("op not found")
}

func (fgsc *FakeGKESDKClient) setAutoscaling(project, clusterName, location, nodepoolName string,
	rb *container.SetNodePoolAutoscalingRequest) (*container.Operation, error) {

	cluster, err := fgsc.get(project, location, clusterName)
	if err != nil {
		return nil, err
	}
	for _, np := range cluster.NodePools {
		if np.Name == nodepoolName {
			np.Autoscaling = rb.Autoscaling
		}
	}
	return fgsc.newOp(), nil
}

func TestSetup(t *testing.T) {
	minNodesOverride := int64(2)
	maxNodesOverride := int64(4)
	nodeTypeOverride := "foonode"
	regionOverride := "fooregion"
	zoneOverride := "foozone"
	fakeAddons := "fake-addon"
	datas := []struct {
		minNodes                        *int64
		maxNodes                        *int64
		nodeType, region, zone, project *string
		addons                          []string
		regionEnv, backupRegionEnv      string
		expClusterOperations            *GKECluster
	}{
		{
			// Defaults
			nil, nil, nil, nil, nil, nil, []string{}, "", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        []string{},
				},
			},
		}, {
			// Project provided
			nil, nil, nil, nil, nil, &fakeProj, []string{}, "", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        []string{},
				},
				Project:     &fakeProj,
				NeedCleanup: true,
			},
		}, {
			// Override other parts
			&minNodesOverride, &maxNodesOverride, &nodeTypeOverride, &regionOverride, &zoneOverride, nil, []string{}, "", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      2,
					MaxNodes:      4,
					NodeType:      "foonode",
					Region:        "fooregion",
					Zone:          "foozone",
					BackupRegions: []string{},
					Addons:        []string{},
				},
			},
		}, {
			// Override other parts but not zone
			&minNodesOverride, &maxNodesOverride, &nodeTypeOverride, &regionOverride, nil, nil, []string{}, "", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      2,
					MaxNodes:      4,
					NodeType:      "foonode",
					Region:        "fooregion",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        []string{},
				},
			},
		}, {
			// Set env Region
			nil, nil, nil, nil, nil, nil, []string{}, "customregion", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "customregion",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        []string{},
				},
			},
		}, {
			// Set env backupzone
			nil, nil, nil, nil, nil, nil, []string{}, "", "backupregion1 backupregion2",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"backupregion1", "backupregion2"},
					Addons:        []string{},
				},
			},
		}, {
			// Set addons
			nil, nil, nil, nil, nil, nil, []string{fakeAddons}, "", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        []string{fakeAddons},
				},
			},
		},
	}

	// mock GetOSEnv for testing
	oldEnvFunc := common.GetOSEnv
	oldExecFunc := common.StandardExec
	oldDefaultCred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	tf, _ := ioutil.TempFile("", "foo")
	tf.WriteString(`{"type": "service_account"}`)
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tf.Name())
	defer func() {
		// restore
		common.GetOSEnv = oldEnvFunc
		common.StandardExec = oldExecFunc
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", oldDefaultCred)
		os.Remove(tf.Name())
	}()
	// mock as kubectl not set and gcloud set as "b", so check environment
	// return project as "b"
	common.StandardExec = func(name string, args ...string) ([]byte, error) {
		var out []byte
		var err error
		switch name {
		case "gcloud":
			out = []byte("b")
			err = nil
		case "kubectl":
			out = []byte("")
			err = fmt.Errorf("kubectl not set")
		default:
			out, err = oldExecFunc(name)
		}
		return out, err
	}

	for _, data := range datas {
		common.GetOSEnv = func(s string) string {
			switch s {
			case "E2E_CLUSTER_REGION":
				return data.regionEnv
			case "E2E_CLUSTER_BACKUP_REGIONS":
				return data.backupRegionEnv
			}
			return oldEnvFunc(s)
		}
		c := GKEClient{}
		co := c.Setup(data.minNodes, data.maxNodes, data.nodeType, data.region, data.zone, data.project, data.addons)
		errMsg := fmt.Sprintf("testing setup with:\n\tminNodes: %v\n\tnodeType: %v\n\tregion: %v\n\tzone: %v\n\tproject: %v\n\taddons: %v\n\tregionEnv: %v\n\tbackupRegionEnv: %v",
			data.minNodes, data.nodeType, data.region, data.zone, data.project, data.addons, data.regionEnv, data.backupRegionEnv)
		gotCo := co.(*GKECluster)
		// mock for easier comparison
		gotCo.operations = nil
		gotCo.boskosOps = nil
		if !reflect.DeepEqual(co, data.expClusterOperations) {
			t.Fatalf("%s\nwant GKECluster:\n'%v'\ngot GKECluster:\n'%v'", errMsg, data.expClusterOperations, co)
		}
	}
}

func TestInitialize(t *testing.T) {
	customProj := "customproj"
	fakeBoskosProj := "fake-boskos-proj-0"
	datas := []struct {
		project      *string
		clusterExist bool
		gcloudSet    bool
		isProw       bool
		boskosProjs  []string
		expProj      *string
		expCluster   *container.Cluster
		expErr       error
	}{
		{
			// User defines project
			&fakeProj, false, false, false, []string{}, &fakeProj, nil, nil,
		}, {
			// User defines project, and running in Prow
			&fakeProj, false, false, true, []string{}, &fakeProj, nil, nil,
		}, {
			// kubeconfig set
			nil, true, false, false, []string{}, &fakeProj, &container.Cluster{
				Name:     "d",
				Location: "c",
				Status:   "RUNNING",
				NodePools: []*container.NodePool{
					{
						Name: "default-pool",
					},
				},
			}, nil,
		}, {
			// kubeconfig not set and gcloud set
			nil, false, true, false, []string{}, &customProj, nil, nil,
		}, {
			// kubeconfig not set and gcloud set, running in Prow and boskos not available
			nil, false, false, true, []string{}, nil, nil, fmt.Errorf("failed acquire boskos project: 'no GKE project available'"),
		}, {
			// kubeconfig not set and gcloud set, running in Prow and boskos available
			nil, false, false, true, []string{fakeBoskosProj}, &fakeBoskosProj, nil, nil,
		}, {
			// kubeconfig not set and gcloud set, not in Prow and boskos not available
			nil, false, false, false, []string{}, nil, nil, fmt.Errorf("gcp project must be set"),
		}, {
			// kubeconfig not set and gcloud set, not in Prow and boskos available
			nil, false, false, false, []string{fakeBoskosProj}, nil, nil, fmt.Errorf("gcp project must be set"),
		},
	}

	oldEnvFunc := common.GetOSEnv
	oldExecFunc := common.StandardExec
	defer func() {
		// restore
		common.GetOSEnv = oldEnvFunc
		common.StandardExec = oldExecFunc
	}()

	for _, data := range datas {
		fgc := setupFakeGKECluster()
		if data.project != nil {
			fgc.Project = data.project
		}
		if data.clusterExist {
			parts := strings.Split("gke_b_c_d", "_")
			fgc.operations.create(parts[1], parts[2], &container.CreateClusterRequest{
				Cluster: &container.Cluster{
					Name: parts[3],
				},
				ProjectId: parts[1],
			})
		}
		// Set up fake boskos
		for _, bos := range data.boskosProjs {
			fgc.boskosOps.(*boskosFake.FakeBoskosClient).NewGKEProject(bos)
		}
		// mock for testing
		common.StandardExec = func(name string, args ...string) ([]byte, error) {
			var out []byte
			var err error
			switch name {
			case "gcloud":
				out = []byte("")
				err = nil
				if data.gcloudSet {
					out = []byte(customProj)
					err = nil
				}
			case "kubectl":
				out = []byte("")
				err = fmt.Errorf("kubectl not set")
				if data.clusterExist {
					out = []byte("gke_b_c_d")
					err = nil
				}
			default:
				out, err = oldExecFunc(name, args...)
			}
			return out, err
		}
		// Mock IsProw()
		common.GetOSEnv = func(s string) string {
			var res string
			switch s {
			case "PROW_JOB_ID":
				if data.isProw {
					res = "fake_job_id"
				}
			default:
				res = oldEnvFunc(s)
			}
			return res
		}

		err := fgc.Initialize()
		errMsg := fmt.Sprintf("test initialize with:\n\tuser defined project: '%v'\n\tkubeconfig set: '%v'\n\tgcloud set: '%v'\n\trunning in prow: '%v'\n\tboskos set: '%v'",
			data.project, data.clusterExist, data.gcloudSet, data.isProw, data.boskosProjs)
		if !reflect.DeepEqual(data.expErr, err) {
			t.Errorf("%s\nerror want: '%v'\nerror got: '%v'", errMsg, err, data.expErr)
		}
		if dif := cmp.Diff(data.expCluster, fgc.Cluster); dif != "" {
			t.Errorf("%s\nCluster got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
		if dif := cmp.Diff(data.expProj, fgc.Project); dif != "" {
			t.Errorf("%s\nProject got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
	}
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
		var gotCluster *string
		if fgc.Cluster != nil {
			gotCluster = &fgc.Cluster.Name
		}

		if !reflect.DeepEqual(err, data.expErr) || !reflect.DeepEqual(fgc.Project, data.expProj) || !reflect.DeepEqual(gotCluster, data.expCluster) {
			t.Errorf("check environment with:\n\tkubectl output: %q\n\t\terror: '%v'\n\tgcloud output: %q\n\t\t"+
				"error: '%v'\nwant: project - '%v', cluster - '%v', err - '%v'\ngot: project - '%v', cluster - '%v', err - '%v'",
				data.kubectlOut, data.kubectlErr, data.gcloudOut, data.gcloudErr, data.expProj, data.expCluster, data.expErr, fgc.Project, fgc.Cluster, err)
		}

		errMsg := fmt.Sprintf("check environment with:\n\tkubectl output: %q\n\t\terror: '%v'\n\tgcloud output: %q\n\t\terror: '%v'",
			data.kubectlOut, data.kubectlErr, data.gcloudOut, data.gcloudErr)
		if !reflect.DeepEqual(data.expErr, err) {
			t.Errorf("%s\nerror want: '%v'\nerror got: '%v'", errMsg, err, data.expErr)
		}
		if dif := cmp.Diff(data.expCluster, gotCluster); dif != "" {
			t.Errorf("%s\nCluster got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
		if dif := cmp.Diff(data.expProj, fgc.Project); dif != "" {
			t.Errorf("%s\nProject got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
	}
}

func TestAcquire(t *testing.T) {
	fakeClusterName := "kpkg-e2e-cls-1234"
	fakeBuildID := "1234"
	datas := []struct {
		existCluster  *container.Cluster
		kubeconfigSet bool
		addons        []string
		nextOpStatus  []string
		expCluster    *container.Cluster
		expErr        error
		expPanic      bool
	}{
		{
			// cluster already found
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, true, []string{}, []string{}, &container.Cluster{
				Name:         "customcluster",
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name: "default-pool",
					},
				},
			}, nil, false,
		}, {
			// cluster exists but not set in kubeconfig, cluster will be deleted
			// then created
			&container.Cluster{
				Name:     fakeClusterName,
				Location: "us-central1",
			}, false, []string{}, []string{}, &container.Cluster{
				Name:         fakeClusterName,
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:        "default-pool",
						Autoscaling: &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster exists but not set in kubeconfig, cluster deletion
			// failed, will recreate in us-west1
			&container.Cluster{
				Name:     fakeClusterName,
				Location: "us-central1",
			}, false, []string{}, []string{"BAD"}, &container.Cluster{
				Name:         fakeClusterName,
				Location:     "us-west1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:        "default-pool",
						Autoscaling: &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster creation succeeded
			nil, false, []string{}, []string{}, &container.Cluster{
				Name:         fakeClusterName,
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:        "default-pool",
						Autoscaling: &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster creation succeeded with addon
			nil, false, []string{"istio"}, []string{}, &container.Cluster{
				Name:     fakeClusterName,
				Location: "us-central1",
				Status:   "RUNNING",
				AddonsConfig: &container.AddonsConfig{
					IstioConfig: &container.IstioConfig{Disabled: false},
				},
				NodePools: []*container.NodePool{
					{
						Name:        "default-pool",
						Autoscaling: &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster creation succeeded retry
			nil, false, []string{}, []string{"PENDING"}, &container.Cluster{
				Name:         fakeClusterName,
				Location:     "us-west1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:        "default-pool",
						Autoscaling: &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster creation failed set addon, but succeeded retry
			nil, false, []string{}, []string{"DONE", "PENDING"}, &container.Cluster{
				Name:         fakeClusterName,
				Location:     "us-west1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:        "default-pool",
						Autoscaling: &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster creation failed all retry
			nil, false, []string{}, []string{"PENDING", "PENDING", "PENDING"}, nil, fmt.Errorf("timed out waiting"), false,
		}, {
			// cluster creation went bad state
			nil, false, []string{}, []string{"BAD", "BAD", "BAD"}, nil, fmt.Errorf("unexpected operation status: %q", "BAD"), false,
		}, {
			// bad addon, should get a panic
			nil, false, []string{"bad_addon"}, []string{}, nil, nil, true,
		},
	}

	// mock GetOSEnv for testing
	oldFunc := common.GetOSEnv
	// mock timeout so it doesn't run forever
	oldCreationTimeout := creationTimeout
	oldAutoscalingTimeout := autoscalingTimeout
	// wait function polls every 500ms, give it 1000 to avoid random timeout
	creationTimeout = 1000 * time.Millisecond
	autoscalingTimeout = 1000 * time.Millisecond
	defer func() {
		// restore
		common.GetOSEnv = oldFunc
		creationTimeout = oldCreationTimeout
		autoscalingTimeout = oldAutoscalingTimeout
	}()

	for _, data := range datas {
		defer func() {
			if r := recover(); r != nil && !data.expPanic {
				t.Errorf("got unexpected panic: '%v'", r)
			}
		}()
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
		opCount := 0
		if data.existCluster != nil {
			opCount++
			ac := &container.AddonsConfig{}
			for _, addon := range data.addons {
				if addon == "istio" {
					ac.IstioConfig = &container.IstioConfig{Disabled: false}
				}
			}
			fgc.operations.create(fakeProj, data.existCluster.Location, &container.CreateClusterRequest{
				Cluster: &container.Cluster{
					Name:         data.existCluster.Name,
					AddonsConfig: ac,
				},
				ProjectId: fakeProj,
			})
			if data.kubeconfigSet {
				fgc.Cluster, _ = fgc.operations.get(fakeProj, data.existCluster.Location, data.existCluster.Name)
			}
		}
		fgc.Project = &fakeProj
		for i, status := range data.nextOpStatus {
			fgc.operations.(*FakeGKESDKClient).opStatus[strconv.Itoa(opCount+i)] = status
		}

		fgc.Request = &GKERequest{
			MinNodes:      DefaultGKEMinNodes,
			MaxNodes:      DefaultGKEMaxNodes,
			NodeType:      DefaultGKENodeType,
			Region:        DefaultGKERegion,
			Zone:          "",
			BackupRegions: DefaultGKEBackupRegions,
			Addons:        data.addons,
		}
		// Set NeedCleanup to false for easier testing, as it launches a
		// goroutine
		fgc.NeedCleanup = false
		err := fgc.Acquire()
		errMsg := fmt.Sprintf("testing acquiring cluster, with:\n\texisting cluster: '%+v'\n\tnext operations outcomes: '%v'\n\tkubeconfig set: '%v'\n\taddons: '%v'",
			data.existCluster, data.nextOpStatus, data.kubeconfigSet, data.addons)
		if !reflect.DeepEqual(err, data.expErr) {
			t.Errorf("%s\nerror want: '%v'\nerror got: '%v'", errMsg, err, data.expErr)
		}
		if dif := cmp.Diff(data.expCluster, fgc.Cluster); dif != "" {
			t.Errorf("%s\nCluster got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
	}
}

func TestDelete(t *testing.T) {
	datas := []struct {
		isProw      bool
		needCleanup bool
		boskosState []*boskoscommon.Resource
		cluster     *container.Cluster
		expBoskos   []*boskoscommon.Resource
		expCluster  *container.Cluster
		expErr      error
	}{
		{
			// Not in prow, NeedCleanup is false
			false,
			false,
			[]*boskoscommon.Resource{},
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			},
			nil,
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
				Status:   "RUNNING",
				NodePools: []*container.NodePool{
					{
						Name: "default-pool",
					},
				},
			},
			nil,
		}, {
			// Not in prow, NeedCleanup is true
			false,
			true,
			[]*boskoscommon.Resource{},
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			},
			nil,
			nil,
			nil,
		}, {
			// Not in prow, NeedCleanup is true, but cluster doesn't exist
			false,
			true,
			[]*boskoscommon.Resource{},
			nil,
			nil,
			nil,
			fmt.Errorf("cluster doesn't exist"),
		}, {
			// In prow, only need to release boskos
			true,
			true,
			[]*boskoscommon.Resource{{
				Name: fakeProj,
			}},
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			},
			[]*boskoscommon.Resource{{
				Type:  "gke-project",
				Name:  fakeProj,
				State: boskoscommon.Free,
			}},
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
				Status:   "RUNNING",
				NodePools: []*container.NodePool{
					{
						Name: "default-pool",
					},
				},
			},
			nil,
		},
	}

	// mock GetOSEnv for testing
	oldFunc := common.GetOSEnv
	// mock timeout so it doesn't run forever
	oldCreationTimeout := creationTimeout
	creationTimeout = 100 * time.Millisecond
	defer func() {
		// restore
		common.GetOSEnv = oldFunc
		creationTimeout = oldCreationTimeout
	}()

	for _, data := range datas {
		common.GetOSEnv = func(key string) string {
			switch key {
			case "PROW_JOB_ID": // needed to mock IsProw()
				if data.isProw {
					return "fake_job_id"
				}
				return ""
			}
			return oldFunc(key)
		}
		fgc := setupFakeGKECluster()
		fgc.Project = &fakeProj
		fgc.NeedCleanup = data.needCleanup
		if data.cluster != nil {
			fgc.operations.create(fakeProj, data.cluster.Location, &container.CreateClusterRequest{
				Cluster: &container.Cluster{
					Name: data.cluster.Name,
				},
				ProjectId: fakeProj,
			})
			fgc.Cluster = data.cluster
		}
		// Set up fake boskos
		for _, bos := range data.boskosState {
			fgc.boskosOps.(*boskosFake.FakeBoskosClient).NewGKEProject(bos.Name)
			// Acquire with default user
			fgc.boskosOps.(*boskosFake.FakeBoskosClient).AcquireGKEProject(nil)
		}

		err := fgc.Delete()
		var gotCluster *container.Cluster
		if data.cluster != nil {
			gotCluster, _ = fgc.operations.get(fakeProj, data.cluster.Location, data.cluster.Name)
		}
		gotBoskos := fgc.boskosOps.(*boskosFake.FakeBoskosClient).GetResources()
		errMsg := fmt.Sprintf("testing deleting cluster, with:\n\tIs Prow: '%v'\n\texisting cluster: '%v'\n\tboskos state: '%v'",
			data.isProw, data.cluster, data.boskosState)
		if !reflect.DeepEqual(err, data.expErr) {
			t.Errorf("%s\nerror want: '%v'\nerror got: '%v'", errMsg, err, data.expErr)
		}
		if dif := cmp.Diff(data.expCluster, gotCluster); dif != "" {
			t.Errorf("%s\nCluster got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
		if dif := cmp.Diff(data.expBoskos, gotBoskos); dif != "" {
			t.Errorf("%s\nBoskos got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
	}
}
