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

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestConfig(t *testing.T) {
	content, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", slackConfigFile))
	if err != nil {
		t.Fatalf("failed to read the test config file: %v", err)
	}
	channels, err := parseConfig(content)
	if err != nil {
		t.Fatalf("failed to parse the test config file: %v", err)
	}

	if len(channels) != 1 || channels[0].Name != "test_channel_name" || channels[0].Identity != "test_channel_identity" {
		t.Fatalf("the channels parsed from the test config file is not correct: %v", channels)
	}
}
