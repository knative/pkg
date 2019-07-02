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

package auto // import "contrib.go.opencensus.io/resource/auto"

import (
	"context"

	"contrib.go.opencensus.io/resource/aws"
	"contrib.go.opencensus.io/resource/gcp"
	"go.opencensus.io/resource"
)

// Detect sequentially runs resource detection from environment varibales, AWS, and GCP.
func Detect(ctx context.Context) (*resource.Resource, error) {
	return resource.MultiDetector(
		resource.FromEnv,
		gcp.DetectGCEInstance,
		aws.DetectEC2Instance,
	)(ctx)
}
