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

package gke

import (
	"fmt"

	container "google.golang.org/api/container/v1beta1"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
)

// SDKOperations wraps GKE SDK related functions
type SDKOperations interface {
	CreateCluster(string, string, *container.CreateClusterRequest) (*container.Operation, error)
	DeleteCluster(string, string, string) (*container.Operation, error)
	GetCluster(string, string, string) (*container.Cluster, error)
	GetOperation(string, string, string) (*container.Operation, error)
	SetAutoscaling(string, string, string, string, *container.SetNodePoolAutoscalingRequest) (*container.Operation, error)
}

// sdkClient Implement SDKOperations
type sdkClient struct {
	*container.Service
}

// NewSDKClient returns an SDKClient that implements SDKOperations
func NewSDKClient() (SDKOperations, error) {
	ctx := context.Background()
	c, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google client: '%v'", err)
	}

	containerService, err := container.New(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create container service: '%v'", err)
	}
	return &sdkClient{containerService}, nil
}

// CreateCluster creates a new GKE cluster, and wait until it finishes or timeout or there is an error.
func (gsc *sdkClient) CreateCluster(project, location string, rb *container.CreateClusterRequest) (*container.Operation, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	op, err := gsc.Projects.Locations.Clusters.Create(parent, rb).Context(context.Background()).Do()
	if err != nil {
		return op, err
	}
	return op, gsc.wait(project, location, op.Name, creationTimeout)
}

// DeleteCluster deletes GKE cluster, and wait until it finishes or timeout or there is an error.
func (gsc *sdkClient) DeleteCluster(project, location, clusterName string) (*container.Operation, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, clusterName)
	op, err := gsc.Projects.Locations.Clusters.Delete(parent).Context(context.Background()).Do()
	if err != nil {
		return op, err
	}
	return op, gsc.wait(project, location, op.Name, deletionTimeout)
}

// GetCluster gets the GKE cluster with the given cluster name.
func (gsc *sdkClient) GetCluster(project, location, clusterName string) (*container.Cluster, error) {
	clusterFullPath := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, clusterName)
	return gsc.Projects.Locations.Clusters.Get(clusterFullPath).Context(context.Background()).Do()
}

// GetOperation gets the operation ref with the given operation name.
func (gsc *sdkClient) GetOperation(project, location, opName string) (*container.Operation, error) {
	name := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, opName)
	return gsc.Service.Projects.Locations.Operations.Get(name).Do()
}

// SetAutoscaling sets up autoscaling for a nodepool, and wait until it finishes or timeout or there is an error.
// This function is not covered by either `Clusters.Update` or `NodePools.Update`, so can not really
// make it as generic as the others.
func (gsc *sdkClient) SetAutoscaling(project, location, clusterName, nodepoolName string,
	rb *container.SetNodePoolAutoscalingRequest) (*container.Operation, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", project, location, clusterName, nodepoolName)
	op, err := gsc.Service.Projects.Locations.Clusters.NodePools.SetAutoscaling(parent, rb).Do()
	if err != nil {
		return op, err
	}
	return op, gsc.wait(project, location, op.Name, autoscalingTimeout)
}
