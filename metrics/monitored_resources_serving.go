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

package metrics

import (
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/resource"
	"knative.dev/pkg/metrics/metricskey"
)

// TODO should be moved to serving. See https://github.com/knative/pkg/issues/608

type KnativeRevision map[string]string

func (kr *KnativeRevision) MonitoredResource() (resType string, labels map[string]string) {
	return metricskey.ResourceTypeKnativeRevision, *kr
}

func GetKnativeRevisionMonitoredResource(
	des *metricdata.Descriptor, tags map[string]string, gm *gcpMetadata, r *resource.Resource) (map[string]string, monitoredresource.Interface) {
	kr := KnativeRevision{
		// The first three resource labels are from metadata.
		metricskey.LabelProject:     gm.project,
		metricskey.LabelLocation:    gm.location,
		metricskey.LabelClusterName: gm.cluster,
		// The rest resource labels are from metrics labels.
		metricskey.LabelNamespaceName:     metricskey.ValueUnknown,
		metricskey.LabelServiceName:       metricskey.ValueUnknown,
		metricskey.LabelConfigurationName: metricskey.ValueUnknown,
		metricskey.LabelRevisionName:      metricskey.ValueUnknown,
	}

	metricLabels := make(map[string]string, len(tags))
	for k, v := range tags {
		if _, ok := kr[k]; ok {
			kr[k] = v
		} else {
			metricLabels[k] = v
		}
	}

	if r != nil {
		for k, v := range r.Labels {
			if _, ok := kr[k]; ok {
				kr[k] = v
			}
		}
	}

	return metricLabels, &kr
}
