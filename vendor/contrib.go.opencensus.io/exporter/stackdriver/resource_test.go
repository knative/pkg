// Copyright 2019, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stackdriver // import "contrib.go.opencensus.io/exporter/stackdriver"

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/resource"
	"go.opencensus.io/resource/resourcekeys"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
)

func TestDefaultMapResource(t *testing.T) {
	cases := []struct {
		input *resource.Resource
		want  *monitoredrespb.MonitoredResource
	}{
		// Verify that the mapping works and that we skip over the
		// first mapping that doesn't apply.
		{
			input: &resource.Resource{
				Type: resourcekeys.ContainerType,
				Labels: map[string]string{
					stackdriverProjectID:             "proj1",
					resourcekeys.K8SKeyClusterName:   "cluster1",
					resourcekeys.K8SKeyPodName:       "pod1",
					resourcekeys.K8SKeyNamespaceName: "namespace1",
					resourcekeys.ContainerKeyName:    "container-name1",
					resourcekeys.CloudKeyAccountID:   "proj1",
					resourcekeys.CloudKeyZone:        "zone1",
					resourcekeys.CloudKeyRegion:      "",
					"extra_key":                      "must be ignored",
				},
			},
			want: &monitoredrespb.MonitoredResource{
				Type: "k8s_container",
				Labels: map[string]string{
					"project_id":     "proj1",
					"location":       "zone1",
					"cluster_name":   "cluster1",
					"namespace_name": "namespace1",
					"pod_name":       "pod1",
					"container_name": "container-name1",
				},
			},
		},
		{
			input: &resource.Resource{
				Type: resourcekeys.CloudType,
				Labels: map[string]string{
					stackdriverProjectID:          "proj1",
					resourcekeys.CloudKeyProvider: resourcekeys.CloudProviderGCP,
					resourcekeys.HostKeyID:        "inst1",
					resourcekeys.CloudKeyZone:     "zone1",
					"extra_key":                   "must be ignored",
				},
			},
			want: &monitoredrespb.MonitoredResource{
				Type: "gce_instance",
				Labels: map[string]string{
					"project_id":  "proj1",
					"instance_id": "inst1",
					"zone":        "zone1",
				},
			},
		},
		{
			input: &resource.Resource{
				Type: resourcekeys.CloudType,
				Labels: map[string]string{
					stackdriverProjectID:           "proj1",
					resourcekeys.CloudKeyProvider:  resourcekeys.CloudProviderAWS,
					resourcekeys.HostKeyID:         "inst1",
					resourcekeys.CloudKeyRegion:    "region1",
					resourcekeys.CloudKeyAccountID: "account1",
					"extra_key":                    "must be ignored",
				},
			},
			want: &monitoredrespb.MonitoredResource{
				Type: "aws_ec2_instance",
				Labels: map[string]string{
					"project_id":  "proj1",
					"instance_id": "inst1",
					"region":      "aws:region1",
					"aws_account": "account1",
				},
			},
		},
		// Partial Match
		{
			input: &resource.Resource{
				Type: resourcekeys.CloudType,
				Labels: map[string]string{
					stackdriverProjectID:          "proj1",
					resourcekeys.CloudKeyProvider: resourcekeys.CloudProviderGCP,
					resourcekeys.HostKeyID:        "inst1",
				},
			},
			want: &monitoredrespb.MonitoredResource{
				Type: "gce_instance",
				Labels: map[string]string{
					"project_id":  "proj1",
					"instance_id": "inst1",
				},
			},
		},
		// Convert to Global.
		{
			input: &resource.Resource{
				Type: "",
				Labels: map[string]string{
					stackdriverProjectID:            "proj1",
					stackdriverLocation:             "zone1",
					stackdriverGenericTaskNamespace: "namespace1",
					stackdriverGenericTaskJob:       "job1",
					stackdriverGenericTaskID:        "task_id1",
					resourcekeys.HostKeyID:          "inst1",
				},
			},
			want: &monitoredrespb.MonitoredResource{
				Type: "global",
				Labels: map[string]string{
					"project_id": "proj1",
					"location":   "zone1",
					"namespace":  "namespace1",
					"job":        "job1",
					"task_id":    "task_id1",
				},
			},
		},
		// nil to Global.
		{
			input: nil,
			want: &monitoredrespb.MonitoredResource{
				Type:   "global",
				Labels: nil,
			},
		},
		// no label to Global.
		{
			input: &resource.Resource{
				Type: resourcekeys.K8SType,
			},
			want: &monitoredrespb.MonitoredResource{
				Type:   "global",
				Labels: nil,
			},
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			got := defaultMapResource(c.input)
			if diff := cmp.Diff(got, c.want); diff != "" {
				t.Errorf("Values differ -got +want: %s", diff)
			}
		})
	}
}
