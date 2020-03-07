/*
Copyright 2020 The Knative Authors

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

// env.go provides a central point to read all environment variables defined by Prow.

package prow

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type EnvConfig struct {
	CI          bool   `required:"true"`
	Artifacts   string `required:"true"`
	JobName     string `required:"true" split_words:"true"`
	JobType     string `required:"true" split_words:"true"`
	JobSpec     string `required:"true" split_words:"true"`
	BuildID     string `required:"true" envconfig:"BUILD_ID"`
	ProwJobID   string `required:"true" envconfig:"PROW_JOB_ID"`
	RepoOwner   string `split_words:"true"`
	RepoName    string `split_words:"true"`
	PullBaseRef string `split_words:"true"`
	PullBaseSha string `split_words:"true"`
	PullRefs    string `split_words:"true"`
	PullNumber  uint   `split_words:"true"`
	PullPullSha string `split_words:"true"`
}

func GetEnvConfig() (*EnvConfig, error) {
	var ec EnvConfig
	if err := envconfig.Process("", &ec); err != nil {
		return nil, fmt.Errorf("failed getting environment variables for Prow: %w", err)
	}
	return &ec, nil
}
