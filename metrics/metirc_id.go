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

const (
	ActivatorComponentName  = "activator"
	AutoscalerComponentName = "autoscaler"
	QueueProxyComponentName = "queue"

	KnativeServingDomain = "knative.dev/serving"
)

// metricID is an ID of metric originally emitted from Knative component.
type metricID struct {
	domain    string
	component string
	name      string
}

func getActivatorMetricID(name string) metricID {
	return metricID{
		domain:    KnativeServingDomain,
		component: ActivatorComponentName,
		name:      name,
	}
}

func getAutoscalerMetricID(name string) metricID {
	return metricID{
		domain:    KnativeServingDomain,
		component: AutoscalerComponentName,
		name:      name,
	}
}

func getQueueProxyMetricID(name string) metricID {
	return metricID{
		domain:    KnativeServingDomain,
		component: QueueProxyComponentName,
		name:      name,
	}
}
