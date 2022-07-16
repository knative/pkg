/*
Copyright 2022 The Knative Authors

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

package changeset

import (
	"errors"
	"regexp"
	"runtime/debug"
	"strconv"
	"sync"
)

var (
	shaRegexp = regexp.MustCompile(`^[a-f0-9]{40,64}$`)
	rev       string
	err       error
	once      sync.Once

	// for testing
	readBuildInfo = debug.ReadBuildInfo
)

// Get returns the 'vcs.revision' property from the embedded build information
// This function will return an error if value is not a valid Git SHA
//
// The result will have a '-dirty' suffix if the workspace was not clean
func Get() (string, error) {
	once.Do(func() {
		rev, err = get()
	})

	return rev, err
}

func get() (string, error) {
	info, ok := readBuildInfo()
	if !ok {
		return "", errors.New("unable to read build info")
	}

	var revision string
	var modified bool

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified, _ = strconv.ParseBool(s.Value)
		}
	}

	if revision == "" {
		return "unknown", nil
	}

	if shaRegexp.MatchString(revision) {
		revision = revision[:7]
	}

	if modified {
		revision += "-dirty"
	}

	return revision, nil
}
