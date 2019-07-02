// Copyright 2018, OpenCensus Authors
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

package gcp // import "contrib.go.opencensus.io/resource/gcp"

import (
	"context"
	"log"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"contrib.go.opencensus.io/resource/resourcekeys"
	"go.opencensus.io/resource"
)

func DetectGCEInstance(context.Context) (*resource.Resource, error) {
	if !metadata.OnGCE() {
		return nil, nil
	}
	res := &resource.Resource{
		Type:   resourcekeys.GCPTypeGCEInstance,
		Labels: map[string]string{},
	}
	instanceID, err := metadata.InstanceID()
	logError(err)
	if instanceID != "" {
		res.Labels[resourcekeys.GCPKeyGCEInstanceID] = instanceID
	}

	projectID, err := metadata.ProjectID()
	logError(err)
	if projectID != "" {
		res.Labels[resourcekeys.GCPKeyGCEProjectID] = projectID
	}

	zone, err := metadata.Zone()
	logError(err)
	if zone != "" {
		res.Labels[resourcekeys.GCPKeyGCEZone] = zone
	}

	return res, nil
}

// logError logs error only if the error is present and it is not 'not defined'
func logError(err error) {
	if err != nil {
		if !strings.Contains(err.Error(), "not defined") {
			log.Printf("Error retrieving gcp metadata: %v", err)
		}
	}
}
