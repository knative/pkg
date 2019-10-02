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
	"errors"
	"fmt"
	"strconv"

	"knative.dev/pkg/testutils/gke"

	container "google.golang.org/api/container/v1beta1"
)

// GKESDKClient is a fake client for unit tests.
type GKESDKClient struct {
	// map of parent: clusters slice
	clusters map[string][]*container.Cluster
	// map of operationID: operation
	ops map[string]*container.Operation

	// An incremental number for new ops
	opNumber int
	// A lookup table for determining ops statuses
	OpStatus map[string]string
}

// NewGKESDKClient returns a new fake gkeSDKClient that can be used in unit tests.
func NewGKESDKClient() gke.SDKOperations {
	return &GKESDKClient{
		clusters: make(map[string][]*container.Cluster),
		ops:      make(map[string]*container.Operation),
		OpStatus: make(map[string]string),
	}
}

// automatically registers new ops, and mark it "DONE" by default. Update
// fgsc.opStatus by fgsc.opStatus[string(fgsc.opNumber+1)]="PENDING" to make the
// next operation pending
func (fgsc *GKESDKClient) newOp() *container.Operation {
	opName := strconv.Itoa(fgsc.opNumber)
	op := &container.Operation{
		Name:   opName,
		Status: "DONE",
	}
	if status, ok := fgsc.OpStatus[opName]; ok {
		op.Status = status
	}
	fgsc.opNumber++
	fgsc.ops[opName] = op
	return op
}

// CreateCluster creates a new cluster, fail if cluster already exists.
func (fgsc *GKESDKClient) CreateCluster(project, location string, rb *container.CreateClusterRequest) (*container.Operation, error) {
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

// DeleteCluster deletes the cluster with the given settings.
func (fgsc *GKESDKClient) DeleteCluster(project, location, clusterName string) (*container.Operation, error) {
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

// GetCluster gets the cluster with the given settings.
func (fgsc *GKESDKClient) GetCluster(project, location, cluster string) (*container.Cluster, error) {
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

// GetOperation gets the operation with the given settings.
func (fgsc *GKESDKClient) GetOperation(project, location, opName string) (*container.Operation, error) {
	if op, ok := fgsc.ops[opName]; ok {
		return op, nil
	}
	return nil, fmt.Errorf("op not found")
}

// SetAutoscaling sets autoscale for the given nodepool.
func (fgsc *GKESDKClient) SetAutoscaling(project, location, clusterName, nodepoolName string,
	rb *container.SetNodePoolAutoscalingRequest) (*container.Operation, error) {

	cluster, err := fgsc.GetCluster(project, location, clusterName)
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
