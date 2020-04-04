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

package clustermanager

import (
	"fmt"
	"log"
	"strings"

	"knative.dev/pkg/test/cmd"
)

const (
	defaultVersion = "default"
	latestVersion  = "latest"
)

func resolveGKEVersion(raw, location string) (string, error) {
	switch raw {
	case defaultVersion:
		defaultCmd := "gcloud container get-server-config --format='value(defaultClusterVersion)' --zone=" + location
		version, err := cmd.RunCommand(defaultCmd)
		if err != nil && version != "" {
			return "", fmt.Errorf("failed getting the default version: %w", err)
		}
		log.Printf("Using default version, %s", version)
		return version, nil
	case latestVersion:
		validCmd := "gcloud container get-server-config --format='value(validMasterVersions)' --zone=" + location
		versionsStr, err := cmd.RunCommand(validCmd)
		if err != nil && versionsStr != "" {
			return "", fmt.Errorf("failed getting the list of valid versions: %w", err)
		}
		versions := strings.Split(versionsStr, ";")
		log.Printf("Using the latest version, %s", versions[0])
		return versions[0], nil
	default:
		log.Printf("Using the custom version, %s", raw)
		return raw, nil
	}
}
