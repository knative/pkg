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

package slack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

const koDataPathEnvName = "KO_DATA_PATH"

// configFile saves all information we need to send slack message to channel(s)
// when performance regression happens in the automation tests.
const configFile = "slack-config.yaml"

// config contains all repo configs for performance regression alerting.
type config struct {
	repoConfigs []repoConfig `yaml:"repoConfigs"`
}

// repoConfig is initial configuration for a given repo, defines which channel(s) to alert
type repoConfig struct {
	repo     string    `yaml:"repo"` // repository to report issues
	channels []channel `yaml:"slackChannels,omitempty"`
}

// channel contains Slack channel's info
type channel struct {
	name     string `yaml:"name"`
	identity string `yaml:"identity"`
}

// loadConfig parses config from configFile
func loadConfig() ([]repoConfig, error) {
	koDataPath := os.Getenv(koDataPathEnvName)
	if koDataPath == "" {
		return nil, fmt.Errorf("%q does not exist or is empty", koDataPathEnvName)
	}
	fullFilename := filepath.Join(koDataPath, configFile)
	contents, err := ioutil.ReadFile(fullFilename)
	if err != nil {
		return nil, err
	}
	config := &config{}
	if err = yaml.Unmarshal(contents, &config); err != nil {
		return nil, err
	}
	return config.repoConfigs, nil
}
