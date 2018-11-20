/*
Copyright 2018 The Knative Authors
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

package metricsKey

const (
	// ResourceTypeKnativeRevision is the Stackdriver resource type for Knative revision
	ResourceTypeKnativeRevision = "knative_revision"

	// LabelProject is the label for project number (gaia id)
	LabelProject = "project"

	// LabelLocation is the label for location where the service is deployed
	LabelLocation = "location"

	// LabelClusterName is the label for immutable name of the cluster
	LabelClusterName = "cluster_name"

	// LabelNamespaceName is the label for immutable name of the namespace that the service is deployed
	LabelNamespaceName = "namespace_name"

	// LabelServiceName is the label for the deployed service name
	LabelServiceName = "service_name"

	// LabelConfigurationName is the label for the configuration which created the monitored revision
	LabelConfigurationName = "configuration_name"

	// LabelRevisionName is the label for the monitored revision
	LabelRevisionName = "revision_name"
)
