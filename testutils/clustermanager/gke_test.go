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
	"io/ioutil"
	"os"
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
	opName := string(fgsc.opNumber)
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

	fgsc.clusters[parent] = append(fgsc.clusters[parent], cluster)
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

func TestSetup(t *testing.T) {
	numNodesOverride := int64(2)
	nodeTypeOverride := "foonode"
	regionOverride := "fooregion"
	zoneOverride := "foozone"
	datas := []struct {
		numNodes                        *int64
		nodeType, region, zone, project *string
		regionEnv, backupRegionEnv      string
		expClusterOperations            *GKECluster
	}{
		{
			// Defaults
			nil, nil, nil, nil, nil, "", "",
			&GKECluster{
				Request: &GKERequest{
					NumNodes:      1,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
				},
			},
		}, {
			// Project provided
			nil, nil, nil, nil, &fakeProj, "", "",
			&GKECluster{
				Request: &GKERequest{
					NumNodes:      1,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
				},
				Project:     &fakeProj,
				NeedCleanup: true,
			},
		}, {
			// Override other parts
			&numNodesOverride, &nodeTypeOverride, &regionOverride, &zoneOverride, nil, "", "",
			&GKECluster{
				Request: &GKERequest{
					NumNodes:      2,
					NodeType:      "foonode",
					Region:        "fooregion",
					Zone:          "foozone",
					BackupRegions: []string{},
				},
			},
		}, {
			// Override other parts but not zone
			&numNodesOverride, &nodeTypeOverride, &regionOverride, nil, nil, "", "",
			&GKECluster{
				Request: &GKERequest{
					NumNodes:      2,
					NodeType:      "foonode",
					Region:        "fooregion",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
				},
			},
		}, {
			// Set env Region
			nil, nil, nil, nil, nil, "customregion", "",
			&GKECluster{
				Request: &GKERequest{
					NumNodes:      1,
					NodeType:      "n1-standard-4",
					Region:        "customregion",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
				},
			},
		}, {
			// Set env backupzone
			nil, nil, nil, nil, nil, "", "backupregion1 backupregion2",
			&GKECluster{
				Request: &GKERequest{
					NumNodes:      1,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"backupregion1", "backupregion2"},
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
		co := c.Setup(data.numNodes, data.nodeType, data.region, data.zone, data.project)
		errPrefix := fmt.Sprintf("testing setup with:\n\tnumNodes: %v\n\tnodeType: %v\n\tregion: %v\n\tone: %v\n\tproject: %v\n\tregionEnv: %v\n\tbackupRegionEnv: %v",
			data.numNodes, data.nodeType, data.region, data.zone, data.project, data.regionEnv, data.backupRegionEnv)
		gotCo := co.(*GKECluster)
		// mock for easier comparison
		gotCo.operations = nil
		if !reflect.DeepEqual(co, data.expClusterOperations) {
			t.Fatalf("%s\nwant GKECluster:\n'%v'\ngot GKECluster:\n'%v'", errPrefix, data.expClusterOperations, co)
		}
	}
}

func TestInitialize(t *testing.T) {
	customProj := "customproj"
	datas := []struct {
		project      *string
		clusterExist bool
		gcloudSet    bool
		expProj      *string
		expCluster   *container.Cluster
		expErr       error
	}{
		{
			// User defines project
			&fakeProj, false, false, &fakeProj, nil, nil,
		}, {
			// kubeconfig set
			nil, true, false, &fakeProj, &container.Cluster{
				Name:     "d",
				Location: "c",
				Status:   "RUNNING",
			}, nil,
		}, {
			// kubeconfig not set and gcloud not set
			nil, false, true, &customProj, nil, nil,
		}, {
			// kubeconfig not set and gcloud set
			nil, false, false, nil, nil, fmt.Errorf("gcp project must be set"),
		},
	}

	oldEnvFunc := common.GetOSEnv
	oldExecFunc := common.StandardExec
	defer func() {
		// restore
		common.GetOSEnv = oldEnvFunc
		common.StandardExec = oldExecFunc
	}()

	// Mock to make IsProw() always return false, otherwise it will actually
	// acquire a boskos project
	common.GetOSEnv = func(s string) string {
		switch s {
		case "PROW_JOB_ID":
			return ""
		}
		return oldEnvFunc(s)
	}

	for _, data := range datas {
		fgc := setupFakeGKECluster()
		if nil != data.project {
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

		err := fgc.Initialize()
		if !reflect.DeepEqual(err, data.expErr) || !reflect.DeepEqual(fgc.Project, data.expProj) || !reflect.DeepEqual(fgc.Cluster, data.expCluster) {
			t.Errorf("test initialize with:\n\tpreset project: '%v'\n\tkubeconfig set: '%v'\n\tgcloud set: '%v'\n"+
				"want:\n\tproject - '%v'\n\tcluster - '%v'\n\terr - '%v'\ngot:\n\tproject - '%v'\n\tcluster - '%v'\n\terr - '%v'",
				data.project, data.clusterExist, data.gcloudSet, data.expProj, data.expCluster, data.expErr, fgc.Project, fgc.Cluster, err)
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
		var clusterGot *string
		if nil != fgc.Cluster {
			clusterGot = &fgc.Cluster.Name
		}

		if !reflect.DeepEqual(err, data.expErr) || !reflect.DeepEqual(fgc.Project, data.expProj) || !reflect.DeepEqual(clusterGot, data.expCluster) {
			t.Errorf("check environment with:\n\tkubectl output: %q\n\t\terror: '%v'\n\tgcloud output: %q\n\t\t"+
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
		nextOpStatus       []string
		expClusterName     string
		expClusterLocation string
		expErr             error
	}{
		{
			// cluster already found
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, []string{}, "customcluster", "us-central1", nil,
		}, {
			// cluster creation succeeded
			nil, []string{}, fakeClusterName, "us-central1", nil,
		}, {
			// cluster creation succeeded retry
			nil, []string{"PENDING"}, fakeClusterName, "us-west1", nil,
		}, {
			// cluster creation failed all retry
			nil, []string{"PENDING", "PENDING", "PENDING"},
			"", "", fmt.Errorf("timed out waiting"),
		}, {
			// cluster creation went bad state
			nil, []string{"BAD", "BAD", "BAD"}, "", "", fmt.Errorf("unexpected operation status: %q", "BAD"),
		},
	}

	// mock GetOSEnv for testing
	oldFunc := common.GetOSEnv
	// mock timeout so it doesn't run forever
	oldTimeout := creationTimeout
	// wait function polls every 500ms, give it 1000 to avoid random timeout
	creationTimeout = 1000 * time.Millisecond
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
		for i, status := range data.nextOpStatus {
			fgc.operations.(*FakeGKESDKClient).opStatus[string(i)] = status
		}

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
			t.Errorf("testing acquiring cluster, with:\n\texisting cluster: '%v'\n\tnext operations outcomes: '%v'\nwant: cluster name - %q, location - %q, err - '%v'\ngot: cluster name - %q, location - %q, err - '%v'",
				data.existCluster, data.nextOpStatus, data.expClusterName, data.expClusterLocation, data.expErr, gotName, gotLocation, err)
		}
	}
}
