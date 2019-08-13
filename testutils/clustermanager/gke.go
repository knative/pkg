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

	"knative.dev/pkg/testutils/clustermanager/boskos"
	"knative.dev/pkg/testutils/common"
)

// GKEClient implements Client
type GKEClient struct {
}

// GKECluster implements ClusterOperations
type GKECluster struct {
	// Project might be GKE specific, so put it here
	Project *string
	// NeedCleanup tells whether the cluster needs to be deleted afterwards
	// This probably should be part of task wrapper's logic
	NeedCleanup bool
	// TODO: evaluate returning "google.golang.org/api/container/v1.Cluster" when implementing the creation logic
	Cluster *string
}

// Setup sets up a GKECluster client.
// numNodes: default to 3 if not provided
// nodeType: default to n1-standard-4 if not provided
// region: default to regional cluster if not provided, and use default backup regions
// zone: default is none, must be provided together with region
func (gs *GKEClient) Setup(numNodes *int, nodeType *string, region *string, zone *string, project *string) (ClusterOperations, error) {
	var err error
	gc := &GKECluster{}
	if nil != project { // use provided project and create cluster
		gc.Project = project
		gc.NeedCleanup = true
	} else if err = gc.checkEnvironment(); nil != err {
		return nil, fmt.Errorf("failed checking existing cluster: '%v'", err)
	}
	if nil != gc.Cluster {
		return gc, nil
	}
	if common.IsProw() {
		if *gc.Project, err = boskos.AcquireGKEProject(); nil != err {
			return nil, fmt.Errorf("failed acquire boskos project: '%v'", err)
		}
	}
	return gc, nil
}

// Provider returns gke
func (gc *GKECluster) Provider() string {
	return "gke"
}

// Acquire gets existing cluster or create a new one
func (gc *GKECluster) Acquire() error {
	// Check if using existing cluster
	if nil != gc.Cluster {
		return nil
	}
	// TODO: Perform GKE specific cluster creation logics
	return nil
}

// Delete deletes a GKE cluster
func (gc *GKECluster) Delete() error {
	if !gc.NeedCleanup {
		return nil
	}
	// TODO: Perform GKE specific cluster deletion logics
	return nil
}

// checks for existing cluster by looking at kubeconfig,
// and sets up gc.Project and gc.Cluster properly, otherwise fail it.
// if project can be derived from gcloud, sets it up as well
func (gc *GKECluster) checkEnvironment() error {
	// TODO: implement this
	return nil
}
