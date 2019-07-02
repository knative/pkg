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

package jsonreport

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"sort"
	"time"

	"github.com/knative/test-infra/shared/common"
	"github.com/knative/test-infra/shared/prow"
)

const (
	filename = "flaky-tests.json"
	jobName  = "ci-knative-flakes-reporter" // flaky-test-reporter's Prow job name
	maxAge   = 4                            // maximum age in days that JSON data is valid
)

// Report contains concise information about current flaky tests
type Report struct {
	Repo  string   `json:"-"`
	Flaky []string `json:"flaky"`
}

// Initialize wraps prow's init, which must be called before any other prow functions are used.
func Initialize(serviceAccount string) error {
	return prow.Initialize(serviceAccount)
}

func (r *Report) writeToArtifactsDir() error {
	artifactsDir := prow.GetLocalArtifactsDir()
	if err := common.CreateDir(path.Join(artifactsDir, r.Repo)); nil != err {
		return err
	}
	outFilePath := path.Join(artifactsDir, r.Repo, filename)
	contents, err := json.Marshal(r)
	if nil != err {
		return err
	}
	return ioutil.WriteFile(outFilePath, contents, 0644)
}

func getJSONForBuild(repo string, buildID int) ([]byte, error) {
	relPath := path.Join(prow.ArtifactsDir, repo, filename)
	job := prow.NewJob(jobName, prow.PeriodicJob, "", 0)
	if buildID != -1 {
		return job.NewBuild(buildID).ReadFile(relPath)
	}
	return getLatestValidJSON(job, relPath)
}

// inexpensively sort and find the most recent JSON file. Assumes build IDs are in sequential order.
func getLatestValidJSON(job *prow.Job, relPath string) ([]byte, error) {
	maxElapsedTime, _ := time.ParseDuration(fmt.Sprintf("%dh", maxAge*24))
	buildIDs := job.GetBuildIDs()
	sort.Sort(sort.Reverse(sort.IntSlice(buildIDs)))
	for _, buildID := range buildIDs {
		build := job.NewBuild(buildID)
		// check for JSON
		if contents, err := build.ReadFile(relPath); err != nil {
			continue
		} else {
			// JSON exists, check timestamp
			startTimeInt, err := build.GetStartTime()
			if err != nil {
				continue
			}
			startTime := time.Unix(startTimeInt, 0)
			if time.Since(startTime) < maxElapsedTime {
				return contents, nil
			} else {
				return nil, fmt.Errorf("latest JSON log is outdated: %.2f days old", time.Since(startTime).Hours()/24)
			}
		}
	}
	return nil, fmt.Errorf("no JSON logs found in recent builds")
}

// GetReportForRepo gets a Report struct from a specific build for a given repo
// use buildID = -1 for most recent successful report
func GetReportForRepo(repo string, buildID int) (*Report, error) {
	report := &Report{
		Repo: repo,
	}
	contents, err := getJSONForBuild(repo, buildID)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(contents, report); err != nil {
		return nil, err
	}
	return report, nil
}

// CreateReportForRepo generates a Report struct and optionally writes it to disk
func CreateReportForRepo(repo string, flaky []string, writeFile bool) (*Report, error) {
	report := &Report{
		Repo:  repo,
		Flaky: flaky,
	}
	if writeFile {
		return report, report.writeToArtifactsDir()
	}
	return report, nil
}
