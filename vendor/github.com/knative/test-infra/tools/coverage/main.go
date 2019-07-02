/* Copyright 2018 The Knative Authors.

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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/knative/test-infra/tools/coverage/artifacts"
	"github.com/knative/test-infra/tools/coverage/gcs"
	"github.com/knative/test-infra/tools/coverage/githubUtil/githubPr"
	"github.com/knative/test-infra/tools/coverage/logUtil"
	"github.com/knative/test-infra/tools/coverage/testgrid"
)

const (
	keyCovProfileFileName      = "key-cov-prof.txt"
	defaultStdoutRedirect      = "stdout.txt"
	defaultCoverageTargetDir   = "."
	defaultGcsBucket           = "knative-prow"
	defaultPostSubmitJobName   = ""
	defaultCovThreshold        = 50
	defaultArtifactsDir        = "./artifacts/"
	defaultCoverageProfileName = "coverage_profile.txt"
)

func main() {
	fmt.Println("entering code coverage main")

	gcsBucketName := flag.String("postsubmit-gcs-bucket", defaultGcsBucket, "gcs bucket name")
	postSubmitJobName := flag.String("postsubmit-job-name", defaultPostSubmitJobName, "name of the prow job")
	artifactsDir := flag.String("artifacts", defaultArtifactsDir, "directory for artifacts")
	coverageTargetDir := flag.String("cov-target", defaultCoverageTargetDir, "target directory for test coverage")
	coverageProfileName := flag.String("profile-name", defaultCoverageProfileName, "file name for coverage profile")
	githubTokenPath := flag.String("github-token", "", "path to token to access github repo")
	covThresholdFlag := flag.Int("cov-threshold-percentage", defaultCovThreshold, "token to access github repo")
	postingBotUserName := flag.String("posting-robot", "knative-metrics-robot", "github user name for coverage robot")
	flag.Parse()

	log.Printf("container flag list: postsubmit-gcs-bucket=%s; postSubmitJobName=%s; "+
		"artifacts=%s; cov-target=%s; profile-name=%s; github-token=%s; "+
		"cov-threshold-percentage=%d; posting-robot=%s;",
		*gcsBucketName, *postSubmitJobName, *artifactsDir, *coverageTargetDir, *coverageProfileName,
		*githubTokenPath, *covThresholdFlag, *postingBotUserName)

	log.Println("Getting env values")
	pr := os.Getenv("PULL_NUMBER")
	pullSha := os.Getenv("PULL_PULL_SHA")
	baseSha := os.Getenv("PULL_BASE_SHA")
	repoOwner := os.Getenv("REPO_OWNER")
	repoName := os.Getenv("REPO_NAME")
	jobType := os.Getenv("JOB_TYPE")
	jobName := os.Getenv("JOB_NAME")

	fmt.Printf("Running coverage for PR %s with PR commit SHA %s and base SHA %s", pr, pullSha, baseSha)

	localArtifacts := artifacts.NewLocalArtifacts(
		*artifactsDir,
		*coverageProfileName,
		keyCovProfileFileName,
		defaultStdoutRedirect,
	)

	localArtifacts.ProduceProfileFile(*coverageTargetDir)

	switch jobType {
	case "presubmit":
		buildStr := os.Getenv("BUILD_NUMBER")
		build, err := strconv.Atoi(buildStr)
		if err != nil {
			logUtil.LogFatalf("BUILD_NUMBER(%s) cannot be converted to int, err=%v",
				buildStr, err)
		}

		prData := githubPr.New(*githubTokenPath, repoOwner, repoName, pr, *postingBotUserName)
		gcsData := &gcs.PresubmitBuild{GcsBuild: gcs.GcsBuild{
			StorageClient: gcs.NewStorageClient(prData.Ctx),
			Bucket:        *gcsBucketName,
			Job:           jobName,
			Build:         build,
			CovThreshold:  *covThresholdFlag,
		},
			PostSubmitJob: *postSubmitJobName,
		}
		presubmit := &gcs.PreSubmit{
			GithubPr:       *prData,
			PresubmitBuild: *gcsData,
		}

		presubmit.Artifacts = *presubmit.MakeGcsArtifacts(*localArtifacts)

		isCoverageLow := RunPresubmit(presubmit, localArtifacts)

		if isCoverageLow {
			logUtil.LogFatalf("Code coverage is below threshold (%d%%), "+
				"fail presubmit workflow intentionally", *covThresholdFlag)
		}
	case "periodic":
		log.Printf("job type is %v, producing testsuite xml...\n", jobType)
		testgrid.ProfileToTestsuiteXML(localArtifacts, *covThresholdFlag)
	}

	fmt.Println("end of code coverage main")
}
