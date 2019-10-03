package main

import (
	"log"

	"knative.dev/pkg/testutils/clustermanager"
	"knative.dev/pkg/testutils/gke"
)

func main() {
	var (
		clusterName       = "test1-chizhg"
		minNodes    int64 = 1
		maxNodes    int64 = 1
		nodeType          = "n1-standard-8"
		region            = "us-central1"
		zone              = ""
		project           = "test-project-chizhg"
		addons            = []string{"istio"}
	)
	gkeClient := clustermanager.GKEClient{}
	clusterOps := gkeClient.Setup(clustermanager.GKERequest{
		Request: gke.Request{
			MinNodes:    minNodes,
			MaxNodes:    maxNodes,
			NodeType:    nodeType,
			Region:      region,
			Zone:        zone,
			Project:     project,
			Addons:      addons,
			ClusterName: clusterName,
		}})
	// Cast to GKEOperation
	gkeOps := clusterOps.(*clustermanager.GKECluster)
	if err := gkeOps.Acquire(); err != nil {
		log.Fatalf("failed acquire cluster: '%v'", err)
	}
	log.Printf("GKE project is: %s", *gkeOps.Project)
	log.Printf("GKE cluster is: %v", gkeOps.Cluster)
}
