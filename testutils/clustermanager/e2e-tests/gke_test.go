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
	// Custom Comparer to ignore addresses
	opt := cmp.Options{
		cmp.Comparer(func(x, y *GKECluster) bool {
			if dif := cmp.Diff(x.Request, y.Request); dif != "" {
				return false
			}
			if x.Project != y.Project {
				return false
			}
			if x.NeedsCleanup != y.NeedsCleanup {
				return false
			}

			return true
		}),
	}

	tests := []struct {
		name string
		args GKERequest
		want *GKECluster
	}{
		{
			name: "No project provided in request",
			args: GKERequest{
				Request: gke.Request{ClusterName: "test-cluster"},
			},
			want: &GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						ClusterName: "test-cluster",
					},
				},
			},
		},
		{
			name: "Project provided in request",
			args: GKERequest{
				Request: gke.Request{Project: "test-project"},
			},
			want: &GKECluster{
				Request: &GKERequest{
					Request: gke.Request{
						Project: "test-project",
					},
				},
				Project:      "test-project",
				NeedsCleanup: true,
			},
		},
	}

	// mock GetOSEnv for testing
	oldDefaultCred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	tf, _ := ioutil.TempFile("", "foo")
	tf.WriteString(`{"type": "service_account"}`)
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tf.Name())
	defer func() {
		// restore
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", oldDefaultCred)
		os.Remove(tf.Name())
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := GKEClient{}
			errMsg := fmt.Sprintf("testing setup with:\n\t%+v", tt.args)
			if got := c.Setup(tt.args); !cmp.Equal(got, tt.want, opt) {
				gcgot := got.(*GKECluster)
				difReq := cmp.Diff(gcgot.Request, tt.want.Request, opt)
				t.Errorf("%s\nRequest got(+) wanted(-)\n%v", errMsg, difReq)
				t.Errorf("Project: got %q, want %q", gcgot.Project, tt.want.Project)
				t.Errorf("NeedsCleanup: got %v, want %v", gcgot.NeedsCleanup, tt.want.NeedsCleanup)
			}
		})
	}
}

func TestApplyRequestDefaults(t *testing.T) {
	tests := []struct {
		name    string
		args    GKERequest
		envVars map[string]string
		want    *GKERequest
	}{
		{
			name: "apply default for all fields",
			args: GKERequest{},
			want: &GKERequest{
				Request: gke.Request{
					ClusterName: "kpkg-e2e-cls",
					MinNodes:    1,
					MaxNodes:    3,
					NodeType:    "n1-standard-4",
					Region:      "us-central1",
				},
				BackupRegions: []string{"us-west1", "us-east1"},
				SkipCreation:  false,
				NeedsCleanup:  false,
			},
		},
		{
			name: "override nodes with min > max",
			args: GKERequest{
				Request: gke.Request{
					MinNodes: 5,
					MaxNodes: 3,
				},
			},
			want: &GKERequest{
				Request: gke.Request{
					ClusterName: "kpkg-e2e-cls",
					MinNodes:    5,
					MaxNodes:    5,
					NodeType:    "n1-standard-4",
					Region:      "us-central1",
				},
				BackupRegions: []string{"us-west1", "us-east1"},
				SkipCreation:  false,
				NeedsCleanup:  false,
			},
		},
		{
			name: "zone overrides has no backup regions",
			args: GKERequest{
				Request: gke.Request{
					Zone: "us-central1-a",
				},
			},
			want: &GKERequest{
				Request: gke.Request{
					ClusterName: "kpkg-e2e-cls",
					MinNodes:    1,
					MaxNodes:    3,
					NodeType:    "n1-standard-4",
					Region:      "us-central1",
					Zone:        "us-central1-a",
				},
				BackupRegions: []string{},
				SkipCreation:  false,
				NeedsCleanup:  false,
			},
		},
		{
			name: "all fields provided, no defaults applied",
			args: GKERequest{
				Request: gke.Request{
					ClusterName: "custom-cluster",
					MinNodes:    10,
					MaxNodes:    20,
					NodeType:    "n1-standard-8",
					Region:      "us-west1",
				},
				BackupRegions: []string{"us-central1"},
				SkipCreation:  true,
				NeedsCleanup:  true,
			},
			want: &GKERequest{
				Request: gke.Request{
					ClusterName: "custom-cluster",
					MinNodes:    10,
					MaxNodes:    20,
					NodeType:    "n1-standard-8",
					Region:      "us-west1",
				},
				BackupRegions: []string{"us-central1"},
				SkipCreation:  true,
				NeedsCleanup:  true,
			},
		},
		{
			name: "EnvVar:E2E_CLUSTER_REGION sets region",
			args: GKERequest{},
			envVars: map[string]string{
				regionEnv: "us-west1",
			},
			want: &GKERequest{
				Request: gke.Request{
					ClusterName: "kpkg-e2e-cls",
					MinNodes:    1,
					MaxNodes:    3,
					NodeType:    "n1-standard-4",
					Region:      "us-west1",
				},
				BackupRegions: []string{"us-west1", "us-east1"},
				SkipCreation:  false,
				NeedsCleanup:  false,
			},
		},
		{
			name: "EnvVar:E2E_CLUSTER_BACKUP_REGIONS sets backup regions",
			args: GKERequest{},
			envVars: map[string]string{
				backupRegionEnv: "europe-west1 asia-east1",
			},
			want: &GKERequest{
				Request: gke.Request{
					ClusterName: "kpkg-e2e-cls",
					MinNodes:    1,
					MaxNodes:    3,
					NodeType:    "n1-standard-4",
					Region:      "us-central1",
				},
				BackupRegions: []string{"europe-west1", "asia-east1"},
				SkipCreation:  false,
				NeedsCleanup:  false,
			},
		},
	}

	// mock GetOSEnv for testing
	oldFunc := common.GetOSEnv
	defer func() {
		// restore
		common.GetOSEnv = oldFunc
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			common.GetOSEnv = func(s string) string {
				return tt.envVars[s]
			}

			got := ApplyRequestDefaults(tt.args)
			if dif := cmp.Diff(tt.want, got); dif != "" {
				t.Errorf("ApplyRequestDefaults() returned diff (-want +got):\n%s", dif)
			}
		})
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
			true, fakeProj, nil, []string{}, []string{"PENDING", "PENDING", "PENDING"}, []string{fakeBoskosProj}, false, nil, fmt.Errorf("timed out waiting"), false,
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
	type testdata struct {
		isProw         bool
		NeedsCleanup   bool
		requestCleanup bool
		boskosState    []*boskoscommon.Resource
		cluster        *container.Cluster
	}
	type wantResult struct {
		Boskos  []*boskoscommon.Resource
		Cluster *container.Cluster
		Err     error
	}
	tests := []struct {
		name string
		td   testdata
		want wantResult
	}{
		{
			name: "Not in prow, NeedsCleanup is false",
			td: testdata{
				isProw:         false,
				NeedsCleanup:   false,
				requestCleanup: false,
				boskosState:    []*boskoscommon.Resource{},
				cluster: &container.Cluster{
					Name:     "customcluster",
					Location: "us-central1",
				},
			},
			want: wantResult{
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
			},
		},
		{
			name: "Not in prow, NeedsCleanup is true",
			td: testdata{
				isProw:         false,
				NeedsCleanup:   true,
				requestCleanup: false,
				boskosState:    []*boskoscommon.Resource{},
				cluster: &container.Cluster{
					Name:     "customcluster",
					Location: "us-central1",
				},
			},
			want: wantResult{
				nil,
				nil,
				nil,
			},
		},
		{
			name: "Not in prow, NeedsCleanup is false, requestCleanup is true",
			td: testdata{
				isProw:         false,
				NeedsCleanup:   false,
				requestCleanup: true,
				boskosState:    []*boskoscommon.Resource{},
				cluster: &container.Cluster{
					Name:     "customcluster",
					Location: "us-central1",
				},
			},
			want: wantResult{
				nil,
				nil,
				nil,
			},
		},
		{
			name: "Not in prow, NeedsCleanup is true, but cluster doesn't exist",
			td: testdata{
				isProw:         false,
				NeedsCleanup:   true,
				requestCleanup: false,
				boskosState:    []*boskoscommon.Resource{},
				cluster:        nil,
			},
			want: wantResult{
				nil,
				nil,
				fmt.Errorf("cluster doesn't exist"),
			},
		},
		{
			name: "In prow, only need to release boskos",
			td: testdata{
				isProw:         true,
				NeedsCleanup:   true,
				requestCleanup: false,
				boskosState: []*boskoscommon.Resource{{
					Name: fakeProj,
				}},
				cluster: &container.Cluster{
					Name:     "customcluster",
					Location: "us-central1",
				},
			},
			want: wantResult{
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
		},
	}

	oldEnvFunc := common.GetOSEnv
	oldExecFunc := common.StandardExec
	defer func() {
		// restore
		common.GetOSEnv = oldEnvFunc
		common.StandardExec = oldExecFunc
	}()

	// Mocked StandardExec so it does not actually run kubectl, gcloud commands.
	// Override so checkEnvironment returns nil all the time and each test use
	// the provided testdata.
	common.StandardExec = func(name string, args ...string) ([]byte, error) {
		var out []byte
		var err error
		switch name {
		case "gcloud":
			out = []byte("")
			err = nil
		case "kubectl":
			out = []byte("")
			err = fmt.Errorf("kubectl not set")
		default:
			out, err = oldExecFunc(name, args...)
		}
		return out, err
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.td
			common.GetOSEnv = func(key string) string {
				switch key {
				case "PROW_JOB_ID": // needed to mock IsProw()
					if data.isProw {
						return "fake_job_id"
					}
					return ""
				}
				return oldEnvFunc(key)
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
			if !reflect.DeepEqual(err, tt.want.Err) {
				t.Errorf("%s\nerror got: '%v'\nerror want: '%v'", errMsg, err, tt.want.Err)
			}
			if dif := cmp.Diff(tt.want.Cluster, gotCluster); dif != "" {
				t.Errorf("%s\nCluster got(+) is different from wanted(-)\n%v", errMsg, dif)
			}
			if dif := cmp.Diff(tt.want.Boskos, gotBoskos); dif != "" {
				t.Errorf("%s\nBoskos got(+) is different from wanted(-)\n%v", errMsg, dif)
			}
		})
	}
}
