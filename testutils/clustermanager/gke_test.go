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
	"strconv"
	"strings"
	"testing"
	"time"

	container "google.golang.org/api/container/v1beta1"
	boskoscommon "k8s.io/test-infra/boskos/common"
	boskosFake "knative.dev/pkg/testutils/clustermanager/boskos/fake"
	"knative.dev/pkg/testutils/common"
	gkeFake "knative.dev/pkg/testutils/gke/fake"

	"github.com/google/go-cmp/cmp"
)

var (
	fakeProj    = "b"
	fakeCluster = "d"
)

func setupFakeGKECluster() GKECluster {
	return GKECluster{
		Request:    &GKERequest{},
		operations: gkeFake.NewGKESDKClient(),
		boskosOps:  &boskosFake.FakeBoskosClient{},
	}
}

func TestSetup(t *testing.T) {
	minNodesOverride := int64(2)
	maxNodesOverride := int64(4)
	nodeTypeOverride := "foonode"
	regionOverride := "fooregion"
	zoneOverride := "foozone"
	fakeAddons := "fake-addon"
	datas := []struct {
		r                          GKERequest
		regionEnv, backupRegionEnv string
		expClusterOperations       *GKECluster
	}{
		{
			// Defaults
			GKERequest{},
			"", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        nil,
				},
			},
		}, {
			// Project provided
			GKERequest{
				Project: fakeProj,
			},
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Project:       "b",
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        nil,
				},
				Project:      &fakeProj,
				NeedsCleanup: true,
			},
		}, {
			// Cluster name provided
			GKERequest{
				ClusterName: "predefined-cluster-name",
			},
			"", "",
			&GKECluster{
				Request: &GKERequest{
					ClusterName:   "predefined-cluster-name",
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        nil,
				},
			},
		}, {
			// Override other parts
			GKERequest{
				MinNodes: minNodesOverride,
				MaxNodes: maxNodesOverride,
				NodeType: nodeTypeOverride,
				Region:   regionOverride,
				Zone:     zoneOverride,
			},
			"", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      2,
					MaxNodes:      4,
					NodeType:      "foonode",
					Region:        "fooregion",
					Zone:          "foozone",
					BackupRegions: []string{},
					Addons:        nil,
				},
			},
		}, {
			// Override other parts but not zone
			GKERequest{
				MinNodes: minNodesOverride,
				MaxNodes: maxNodesOverride,
				NodeType: nodeTypeOverride,
				Region:   regionOverride,
			},
			"", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      2,
					MaxNodes:      4,
					NodeType:      "foonode",
					Region:        "fooregion",
					Zone:          "",
					BackupRegions: nil,
					Addons:        nil,
				},
			},
		}, {
			// Set env Region
			GKERequest{},
			"customregion", "",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "customregion",
					Zone:          "",
					BackupRegions: []string{"us-west1", "us-east1"},
					Addons:        nil,
				},
			},
		}, {
			// Set env backupzone
			GKERequest{},
			"", "backupregion1 backupregion2",
			&GKECluster{
				Request: &GKERequest{
					MinNodes:      1,
					MaxNodes:      3,
					NodeType:      "n1-standard-4",
					Region:        "us-central1",
					Zone:          "",
					BackupRegions: []string{"backupregion1", "backupregion2"},
					Addons:        nil,
				},
			},
		}, {
			// Set addons
			GKERequest{
				Addons: []string{fakeAddons},
			},
			"", "",
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
		co := c.Setup(data.r)
		errMsg := fmt.Sprintf("testing setup with:\n\t%+v\n\tregionEnv: %v\n\tbackupRegionEnv: %v",
			data.r, data.regionEnv, data.backupRegionEnv)
		gotCo := co.(*GKECluster)
		// mock for easier comparison
		gotCo.operations = nil
		gotCo.boskosOps = nil
		if dif := cmp.Diff(gotCo.Request, data.expClusterOperations.Request); dif != "" {
			t.Errorf("%s\nRequest got(+) is different from wanted(-)\n%v", errMsg, dif)
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
			fgc.operations.CreateCluster(parts[1], parts[2], &container.CreateClusterRequest{
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

		err := fgc.initialize()
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
			fgc.operations.CreateCluster(parts[1], parts[2], &container.CreateClusterRequest{
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
	predefinedClusterName := "predefined-cluster-name"
	fakeClusterName := "kpkg-e2e-cls-1234"
	fakeBuildID := "1234"
	datas := []struct {
		existCluster          *container.Cluster
		predefinedClusterName string
		kubeconfigSet         bool
		addons                []string
		nextOpStatus          []string
		skipCreation          bool
		expCluster            *container.Cluster
		expErr                error
		expPanic              bool
	}{
		{
			// cluster already found
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, "", true, []string{}, []string{}, false, &container.Cluster{
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
			// cluster already found and clustername predefined
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, predefinedClusterName, true, []string{}, []string{}, false, &container.Cluster{
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
			}, "", false, []string{}, []string{}, false, &container.Cluster{
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
			}, "", false, []string{}, []string{"BAD"}, false, &container.Cluster{
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
			// cluster exists but not set in kubeconfig, clusterName defined
			&container.Cluster{
				Name:     fakeClusterName,
				Location: "us-central1",
			}, predefinedClusterName, false, []string{}, []string{}, false, &container.Cluster{
				Name:         predefinedClusterName,
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
			// cluster not exist, but clustername defined
			nil, predefinedClusterName, false, []string{}, []string{}, false, &container.Cluster{
				Name:         predefinedClusterName,
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
			// cluster creation succeeded
			nil, "", false, []string{}, []string{}, true, nil, nil, false,
		}, {
			// skipped cluster creation as SkipCreation is requested
			nil, "", false, []string{}, []string{}, false, &container.Cluster{
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
			nil, "", false, []string{"istio"}, []string{}, false, &container.Cluster{
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
			nil, "", false, []string{}, []string{"PENDING"}, false, &container.Cluster{
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
			nil, "", false, []string{}, []string{"DONE", "PENDING"}, false, &container.Cluster{
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
			nil, "", false, []string{}, []string{"PENDING", "PENDING", "PENDING"}, false, nil, fmt.Errorf("timed out waiting"), false,
		}, {
			// cluster creation went bad state
			nil, "", false, []string{}, []string{"BAD", "BAD", "BAD"}, false, nil, fmt.Errorf("unexpected operation status: %q", "BAD"), false,
		}, {
			// bad addon, should get a panic
			nil, "", false, []string{"bad_addon"}, []string{}, false, nil, nil, true,
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
			fgc.operations.CreateCluster(fakeProj, data.existCluster.Location, &container.CreateClusterRequest{
				Cluster: &container.Cluster{
					Name:         data.existCluster.Name,
					AddonsConfig: ac,
				},
				ProjectId: fakeProj,
			})
			if data.kubeconfigSet {
				fgc.Cluster, _ = fgc.operations.GetCluster(fakeProj, data.existCluster.Location, data.existCluster.Name)
			}
		}
		fgc.Project = &fakeProj
		for i, status := range data.nextOpStatus {
			fgc.operations.(*gkeFake.GKESDKClient).OpStatus[strconv.Itoa(opCount+i)] = status
		}

		fgc.Request = &GKERequest{
			ClusterName:   data.predefinedClusterName,
			MinNodes:      DefaultGKEMinNodes,
			MaxNodes:      DefaultGKEMaxNodes,
			NodeType:      DefaultGKENodeType,
			Region:        DefaultGKERegion,
			Zone:          "",
			BackupRegions: DefaultGKEBackupRegions,
			Addons:        data.addons,
		}
		if data.skipCreation {
			fgc.Request.SkipCreation = true
		}
		// Set NeedsCleanup to false for easier testing, as it launches a
		// goroutine
		fgc.NeedsCleanup = false
		err := fgc.Acquire()
		errMsg := fmt.Sprintf("testing acquiring cluster, with:\n\texisting cluster: '%+v'\n\tSkip creation: '%+v'\n\t"+
			"next operations outcomes: '%v'\n\tkubeconfig set: '%v'\n\taddons: '%v'",
			data.existCluster, data.skipCreation, data.nextOpStatus, data.kubeconfigSet, data.addons)
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
		isProw         bool
		NeedsCleanup   bool
		requestCleanup bool
		boskosState    []*boskoscommon.Resource
		cluster        *container.Cluster
		expBoskos      []*boskoscommon.Resource
		expCluster     *container.Cluster
		expErr         error
	}{
		{
			// Not in prow, NeedsCleanup is false
			false,
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
			// Not in prow, NeedsCleanup is true
			false,
			true,
			false,
			[]*boskoscommon.Resource{},
			&container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			},
			nil,
			nil,
			nil,
		}, {
			// Not in prow, NeedsCleanup is false, requestCleanup is true
			false,
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
			// Not in prow, NeedsCleanup is true, but cluster doesn't exist
			false,
			true,
			false,
			[]*boskoscommon.Resource{},
			nil,
			nil,
			nil,
			fmt.Errorf("cluster doesn't exist"),
		}, {
			// In prow, only need to release boskos
			true,
			true,
			false,
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
		fgc.NeedsCleanup = data.NeedsCleanup
		if data.cluster != nil {
			fgc.operations.CreateCluster(fakeProj, data.cluster.Location, &container.CreateClusterRequest{
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
		if data.requestCleanup {
			fgc.Request = &GKERequest{
				NeedsCleanup: true,
			}
		}

		err := fgc.Delete()
		var gotCluster *container.Cluster
		if data.cluster != nil {
			gotCluster, _ = fgc.operations.GetCluster(fakeProj, data.cluster.Location, data.cluster.Name)
		}
		gotBoskos := fgc.boskosOps.(*boskosFake.FakeBoskosClient).GetResources()
		errMsg := fmt.Sprintf("testing deleting cluster, with:\n\tIs Prow: '%v'\n\tNeed cleanup: '%v'\n\t"+
			"Request cleanup: '%v'\n\texisting cluster: '%v'\n\tboskos state: '%v'",
			data.isProw, data.NeedsCleanup, data.requestCleanup, data.cluster, data.boskosState)
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
