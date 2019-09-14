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

package config

import yaml "gopkg.in/yaml.v2"

// slackConfigFile saves all information we need to send slack message to channel(s)
// when performance regression happens in the automation tests.
const slackConfigFile = "config-slack.yaml"

// SlackConfig contains all repo configs for performance regression alerting.
type SlackConfig struct {
	Channels []Channel `yaml:"channels"`
}

// Channel contains Slack channel's info
type Channel struct {
	Name     string `yaml:"name"`
	Identity string `yaml:"identity"`
}

// LoadSlackConfig parses config from configFile and return
func LoadSlackConfig() ([]Channel, error) {
	content, err := readFileFromKoData(slackConfigFile)
	if err != nil {
		return nil, err
	}

	return parseConfig(content)
}

func parseConfig(configYaml []byte) ([]Channel, error) {
	conf := &SlackConfig{}
	if err := yaml.Unmarshal(configYaml, conf); err != nil {
		return nil, err
	}
	return conf.Channels, nil
}
