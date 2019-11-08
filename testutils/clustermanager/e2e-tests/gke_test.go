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
	"knative.dev/pkg/test/gke"
	gkeFake "knative.dev/pkg/test/gke/fake"
	boskosFake "knative.dev/pkg/testutils/clustermanager/e2e-tests/boskos/fake"
	"knative.dev/pkg/testutils/clustermanager/e2e-tests/common"

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
	fakeBuildID := "1234"
	datas := []struct {
		r                          GKERequest
		isProw                     bool
		regionEnv, backupRegionEnv string
		expClusterOperations       *GKECluster
	}{
		{
			// Defaults, not running in Prow
			GKERequest{},
			false,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "us-central1",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: []string{"us-west1", "us-east1"},
				},
			},
		}, {
			// Defaults, running in Prow
			GKERequest{},
			true,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls-1234",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "us-central1",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: []string{"us-west1", "us-east1"},
				},
			},
		}, {
			// Project provided, not running in Prow
			GKERequest{
				Request: gke.Request{
					Project: fakeProj,
				},
			},
			false,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls",
						Project:     "b",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "us-central1",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: []string{"us-west1", "us-east1"},
				},
				Project:      fakeProj,
				NeedsCleanup: true,
			},
		}, {
			// Project provided, running in Prow
			GKERequest{
				Request: gke.Request{
					Project: fakeProj,
				},
			},
			true,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls-1234",
						Project:     "b",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "us-central1",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: []string{"us-west1", "us-east1"},
				},
				Project:      fakeProj,
				NeedsCleanup: true,
			},
		}, {
			// Cluster name provided, not running in Prow
			GKERequest{
				Request: gke.Request{
					ClusterName: "predefined-cluster-name",
				},
			},
			false,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "predefined-cluster-name",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "us-central1",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: []string{"us-west1", "us-east1"},
				},
			},
		}, {
			// Cluster name provided, running in Prow
			GKERequest{
				Request: gke.Request{
					ClusterName: "predefined-cluster-name",
				},
			},
			false,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "predefined-cluster-name",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "us-central1",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: []string{"us-west1", "us-east1"},
				},
			},
		}, {
			// Override other parts
			GKERequest{
				Request: gke.Request{
					MinNodes: minNodesOverride,
					MaxNodes: maxNodesOverride,
					NodeType: nodeTypeOverride,
					Region:   regionOverride,
					Zone:     zoneOverride,
				},
			},
			false,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls",
						MinNodes:    2,
						MaxNodes:    4,
						NodeType:    "foonode",
						Region:      "fooregion",
						Zone:        "foozone",
						Addons:      nil,
					},
					BackupRegions: []string{},
				},
			},
		}, {
			// Override other parts but not zone
			GKERequest{
				Request: gke.Request{
					MinNodes: minNodesOverride,
					MaxNodes: maxNodesOverride,
					NodeType: nodeTypeOverride,
					Region:   regionOverride,
				},
			},
			false,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls",
						MinNodes:    2,
						MaxNodes:    4,
						NodeType:    "foonode",
						Region:      "fooregion",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: nil,
				},
			},
		}, {
			// Set env Region
			GKERequest{},
			false,
			"customregion", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "customregion",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: []string{"us-west1", "us-east1"},
				},
			},
		}, {
			// Set env backupzone
			GKERequest{},
			false,
			"", "backupregion1 backupregion2",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "us-central1",
						Zone:        "",
						Addons:      nil,
					},
					BackupRegions: []string{"backupregion1", "backupregion2"},
				},
			},
		}, {
			// Set addons
			GKERequest{
				Request: gke.Request{
					Addons: []string{fakeAddons},
				},
			},
			false,
			"", "",
			&GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "kpkg-e2e-cls",
						MinNodes:    1,
						MaxNodes:    3,
						NodeType:    "n1-standard-4",
						Region:      "us-central1",
						Zone:        "",
						Addons:      []string{fakeAddons},
					},
					BackupRegions: []string{"us-west1", "us-east1"},
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
			out, err = oldExecFunc(name, args...)
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
			case "BUILD_NUMBER":
				return fakeBuildID
			case "PROW_JOB_ID": // needed to mock IsProw()
				if data.isProw {
					return "fake_job_id"
				}
				return ""
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

func TestGKECheckEnvironment(t *testing.T) {
	datas := []struct {
		kubectlOut         string
		kubectlErr         error
		gcloudOut          string
		gcloudErr          error
		clusterExist       bool
		requestClusterName string
		requestProject     string
		expProj            string
		expCluster         *string
		expErr             error
	}{
		{
			// Base condition, kubectl shouldn't return empty string if there is no error
			"", nil, "", nil, false, "", "", "", nil, nil,
		}, {
			// kubeconfig not set and gcloud not set
			"", fmt.Errorf("kubectl not set"), "", nil, false, "", "", "", nil, nil,
		}, {
			// kubeconfig failed
			"failed", fmt.Errorf("kubectl other err"), "", nil, false, "", "", "", nil, fmt.Errorf("failed running kubectl config current-context: 'failed'"),
		}, {
			// kubeconfig returned something other than "gke_PROJECT_REGION_CLUSTER"
			"gke_b_c", nil, "", nil, false, "", "", "", nil, nil,
		}, {
			// kubeconfig returned something other than "gke_PROJECT_REGION_CLUSTER"
			"gke_b_c_d_e", nil, "", nil, false, "", "", "", nil, nil,
		}, {
			// kubeconfig correctly set and cluster exist
			"gke_b_c_d", nil, "", nil, true, "d", "b", fakeProj, &fakeCluster, nil,
		}, {
			// kubeconfig correctly set and cluster exist, project wasn't requested
			"gke_b_c_d", nil, "", nil, true, "d", "", fakeProj, &fakeCluster, nil,
		}, {
			// kubeconfig correctly set and cluster exist, project doesn't match
			"gke_b_c_d", nil, "", nil, true, "d", "doesntexist", "", nil, nil,
		}, {
			// kubeconfig correctly set and cluster exist, cluster wasn't requested
			"gke_b_c_d", nil, "", nil, true, "", "b", fakeProj, &fakeCluster, nil,
		}, {
			// kubeconfig correctly set and cluster exist, cluster doesn't match
			"gke_b_c_d", nil, "", nil, true, "doesntexist", "b", "", nil, nil,
		}, {
			// kubeconfig correctly set and cluster exist, none of project/cluster requested
			"gke_b_c_d", nil, "", nil, true, "", "", fakeProj, &fakeCluster, nil,
		}, {
			// kubeconfig correctly set, but cluster doesn't exist
			"gke_b_c_d", nil, "", nil, false, "d", "", "", nil, fmt.Errorf("couldn't find cluster d in b in c, does it exist? cluster not found"),
		}, {
			// kubeconfig not set and gcloud failed
			"", fmt.Errorf("kubectl not set"), "", fmt.Errorf("gcloud failed"), false, "", "", "", nil, fmt.Errorf("failed getting gcloud project: 'gcloud failed'"),
		}, {
			// kubeconfig not set and gcloud not set
			"", fmt.Errorf("kubectl not set"), "", nil, false, "", "", "", nil, nil,
		}, {
			// kubeconfig not set and gcloud set
			"", fmt.Errorf("kubectl not set"), "b", nil, false, "", "", fakeProj, nil, nil,
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
			fgc.operations.CreateClusterAsync(parts[1], parts[2], "", &container.CreateClusterRequest{
				Cluster: &container.Cluster{
					Name: parts[3],
				},
				ProjectId: parts[1],
			})
		}
		fgc.Request.ClusterName = data.requestClusterName
		fgc.Request.Project = data.requestProject
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

		errMsg := fmt.Sprintf("check environment with:\n\tkubectl output: %q\n\t\terror: '%v'\n\tgcloud output: %q\n\t\t"+
			"error: '%v'\n\t\tclustername requested: %q\n\t\tproject requested: %q",
			data.kubectlOut, data.kubectlErr, data.gcloudOut, data.gcloudErr, data.requestClusterName, data.requestProject)

		if !reflect.DeepEqual(err, data.expErr) || !reflect.DeepEqual(fgc.Project, data.expProj) || !reflect.DeepEqual(gotCluster, data.expCluster) {
			t.Errorf("%s\ngot: project - %q, cluster - '%v', err - '%v'\nwant: project - '%v', cluster - '%v', err - '%v'",
				errMsg, fgc.Project, fgc.Cluster, err, data.expProj, data.expCluster, data.expErr)
		}

		if !reflect.DeepEqual(data.expErr, err) {
			t.Errorf("%s\nerror got: '%v'\nerror want: '%v'", errMsg, data.expErr, err)
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
	fakeBoskosProj := "fake-boskos-proj-0"
	fakeBuildID := "1234"
	datas := []struct {
		isProw       bool
		project      string
		existCluster *container.Cluster
		addons       []string
		nextOpStatus []string
		boskosProjs  []string
		skipCreation bool
		expCluster   *container.Cluster
		expErr       error
		expPanic     bool
	}{
		{
			// cluster not exist, running in Prow and boskos not available
			true, fakeProj, nil, []string{}, []string{}, []string{}, false, nil, fmt.Errorf("failed acquiring boskos project: 'no GKE project available'"), false,
		}, {
			// cluster not exist, running in Prow and boskos available
			true, fakeProj, nil, []string{}, []string{}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:         predefinedClusterName,
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster not exist, project not set, running in Prow and boskos not available
			true, "", nil, []string{}, []string{}, []string{}, false, nil, fmt.Errorf("failed acquiring boskos project: 'no GKE project available'"), false,
		}, {
			// cluster not exist, project not set, running in Prow and boskos available
			true, "", nil, []string{}, []string{}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:         predefinedClusterName,
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// project not set, not in Prow and boskos not available
			false, "", nil, []string{}, []string{}, []string{}, false, nil, fmt.Errorf("GCP project must be set"), false,
		}, {
			// project not set, not in Prow and boskos available
			false, "", nil, []string{}, []string{}, []string{fakeBoskosProj}, false, nil, fmt.Errorf("GCP project must be set"), false,
		}, {
			// cluster exists, project set, running in Prow
			true, fakeProj, &container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, []string{}, []string{}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:         "customcluster",
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster exists, project set and not running in Prow
			false, fakeProj, &container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, []string{}, []string{}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:         "customcluster",
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster exist, not running in Prow and skip creation
			false, fakeProj, &container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, []string{}, []string{}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:         "customcluster",
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster exist, running in Prow and skip creation
			true, fakeProj, &container.Cluster{
				Name:     "customcluster",
				Location: "us-central1",
			}, []string{}, []string{}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:         "customcluster",
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster not exist, not running in Prow and skip creation
			false, fakeProj, nil, []string{}, []string{}, []string{fakeBoskosProj}, true, nil, fmt.Errorf("cannot acquire cluster if SkipCreation is set"), false,
		}, {
			// cluster not exist, running in Prow and skip creation
			true, fakeProj, nil, []string{}, []string{}, []string{fakeBoskosProj}, true, nil, fmt.Errorf("cannot acquire cluster if SkipCreation is set"), false,
		}, {
			// skipped cluster creation as SkipCreation is requested
			true, fakeProj, nil, []string{}, []string{}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:         predefinedClusterName,
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster creation succeeded with addon
			true, fakeProj, nil, []string{"istio"}, []string{}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:     predefinedClusterName,
				Location: "us-central1",
				Status:   "RUNNING",
				AddonsConfig: &container.AddonsConfig{
					IstioConfig: &container.IstioConfig{Disabled: false},
				},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster creation succeeded retry
			true, fakeProj, nil, []string{}, []string{"PENDING"}, []string{fakeBoskosProj}, false, &container.Cluster{
				Name:         predefinedClusterName,
				Location:     "us-west1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			}, nil, false,
		}, {
			// cluster creation failed all retry
			true, fakeProj, nil, []string{}, []string{"PENDING", "PENDING", "PENDING"}, []string{fakeBoskosProj}, false, nil, fmt.Errorf("got time out waiting"), false,
		}, {
			// cluster creation went bad state
			true, fakeProj, nil, []string{}, []string{"BAD", "BAD", "BAD"}, []string{fakeBoskosProj}, false, nil, fmt.Errorf("unexpected operation status: %q", "BAD"), false,
		}, {
			// bad addon, should get a panic
			true, fakeProj, nil, []string{"bad_addon"}, []string{}, []string{fakeBoskosProj}, false, nil, nil, true,
		},
	}

	oldEnvFunc := common.GetOSEnv
	oldExecFunc := common.StandardExec
	// mock timeout so it doesn't run forever
	oldCreationTimeout := gkeFake.CreationTimeout
	// wait function polls every 500ms, give it 1000 to avoid random timeout
	gkeFake.CreationTimeout = 1000 * time.Millisecond
	defer func() {
		// restore
		common.GetOSEnv = oldEnvFunc
		common.StandardExec = oldExecFunc
		gkeFake.CreationTimeout = oldCreationTimeout
	}()

	for _, data := range datas {
		defer func() {
			if r := recover(); r != nil && !data.expPanic {
				t.Errorf("got unexpected panic: '%v'", r)
			}
		}()
		// mock for testing
		common.StandardExec = func(name string, args ...string) ([]byte, error) {
			var out []byte
			var err error
			switch name {
			case "gcloud":
				out = []byte("")
				err = nil
				if data.project != "" {
					out = []byte(data.project)
					err = nil
				}
			case "kubectl":
				out = []byte("")
				err = fmt.Errorf("kubectl not set")
				if data.existCluster != nil {
					context := fmt.Sprintf("gke_%s_%s_%s", data.project, data.existCluster.Location, data.existCluster.Name)
					out = []byte(context)
					err = nil
				}
			default:
				out, err = oldExecFunc(name, args...)
			}
			return out, err
		}
		common.GetOSEnv = func(key string) string {
			switch key {
			case "BUILD_NUMBER":
				return fakeBuildID
			case "PROW_JOB_ID": // needed to mock IsProw()
				if data.isProw {
					return "fake_job_id"
				}
				return ""
			}
			return oldEnvFunc(key)
		}
		fgc := setupFakeGKECluster()
		// Set up fake boskos
		for _, bos := range data.boskosProjs {
			fgc.boskosOps.(*boskosFake.FakeBoskosClient).NewGKEProject(bos)
		}
		fgc.Request = &GKERequest{
			Request: gke.Request{
				ClusterName: predefinedClusterName,
				MinNodes:    DefaultGKEMinNodes,
				MaxNodes:    DefaultGKEMaxNodes,
				NodeType:    DefaultGKENodeType,
				Region:      DefaultGKERegion,
				Zone:        "",
				Addons:      data.addons,
			},
			BackupRegions: DefaultGKEBackupRegions,
		}
		opCount := 0
		if data.existCluster != nil {
			opCount++
			fgc.Request.ClusterName = data.existCluster.Name
			rb, _ := gke.NewCreateClusterRequest(&fgc.Request.Request)
			fgc.operations.CreateClusterAsync(data.project, data.existCluster.Location, "", rb)
			fgc.Cluster, _ = fgc.operations.GetCluster(data.project, data.existCluster.Location, "", data.existCluster.Name)
		}

		fgc.Project = data.project
		for i, status := range data.nextOpStatus {
			fgc.operations.(*gkeFake.GKESDKClient).OpStatus[strconv.Itoa(opCount+i)] = status
		}

		if data.skipCreation {
			fgc.Request.SkipCreation = true
		}
		// Set NeedsCleanup to false for easier testing, as it launches a
		// goroutine
		fgc.NeedsCleanup = false
		err := fgc.Acquire()
		errMsg := fmt.Sprintf("testing acquiring cluster, with:\n\tisProw: '%v'\n\tproject: '%v'\n\texisting cluster: '%+v'\n\tSkip creation: '%+v'\n\t"+
			"next operations outcomes: '%v'\n\taddons: '%v'\n\tboskos projects: '%v'",
			data.isProw, data.project, data.existCluster, data.skipCreation, data.nextOpStatus, data.addons, data.boskosProjs)
		if !reflect.DeepEqual(err, data.expErr) {
			t.Errorf("%s\nerror got: '%v'\nerror want: '%v'", errMsg, err, data.expErr)
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
				Name:         "customcluster",
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
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
				Name:         "customcluster",
				Location:     "us-central1",
				Status:       "RUNNING",
				AddonsConfig: &container.AddonsConfig{},
				NodePools: []*container.NodePool{
					{
						Name:             "default-pool",
						InitialNodeCount: DefaultGKEMinNodes,
						Config:           &container.NodeConfig{MachineType: "n1-standard-4"},
						Autoscaling:      &container.NodePoolAutoscaling{Enabled: true, MaxNodeCount: 3, MinNodeCount: 1},
					},
				},
				MasterAuth: &container.MasterAuth{
					Username: "admin",
				},
			},
			nil,
		},
	}

	// mock GetOSEnv for testing
	oldFunc := common.GetOSEnv
	// mock timeout to make it run faster
	oldCreationTimeout := gkeFake.CreationTimeout
	gkeFake.CreationTimeout = 100 * time.Millisecond
	defer func() {
		// restore
		common.GetOSEnv = oldFunc
		gkeFake.CreationTimeout = oldCreationTimeout
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
		fgc.Project = fakeProj
		fgc.NeedsCleanup = data.NeedsCleanup
		fgc.Request = &GKERequest{
			Request: gke.Request{
				MinNodes: DefaultGKEMinNodes,
				MaxNodes: DefaultGKEMaxNodes,
				NodeType: DefaultGKENodeType,
				Region:   DefaultGKERegion,
				Zone:     "",
			},
		}
		if data.cluster != nil {
			fgc.Request.ClusterName = data.cluster.Name
			rb, _ := gke.NewCreateClusterRequest(&fgc.Request.Request)
			fgc.operations.CreateClusterAsync(fakeProj, data.cluster.Location, "", rb)
			fgc.Cluster, _ = fgc.operations.GetCluster(fakeProj, data.cluster.Location, "", data.cluster.Name)
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
			gotCluster, _ = fgc.operations.GetCluster(fakeProj, data.cluster.Location, "", data.cluster.Name)
		}
		gotBoskos := fgc.boskosOps.(*boskosFake.FakeBoskosClient).GetResources()
		errMsg := fmt.Sprintf("testing deleting cluster, with:\n\tIs Prow: '%v'\n\tNeed cleanup: '%v'\n\t"+
			"Request cleanup: '%v'\n\texisting cluster: '%v'\n\tboskos state: '%v'",
			data.isProw, data.NeedsCleanup, data.requestCleanup, data.cluster, data.boskosState)
		if !reflect.DeepEqual(err, data.expErr) {
			t.Errorf("%s\nerror got: '%v'\nerror want: '%v'", errMsg, data.expErr, err)
		}
		if dif := cmp.Diff(data.expCluster, gotCluster); dif != "" {
			t.Errorf("%s\nCluster got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
		if dif := cmp.Diff(data.expBoskos, gotBoskos); dif != "" {
			t.Errorf("%s\nBoskos got(+) is different from wanted(-)\n%v", errMsg, dif)
		}
	}
}
