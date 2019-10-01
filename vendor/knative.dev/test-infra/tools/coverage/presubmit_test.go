/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	"knative.dev/test-infra/tools/coverage/artifacts/artsTest"
	"knative.dev/test-infra/tools/coverage/gcs"
	"knative.dev/test-infra/tools/coverage/gcs/gcsFakes"
	"knative.dev/test-infra/tools/coverage/githubUtil/githubFakes"
	"knative.dev/test-infra/tools/coverage/githubUtil/githubPr"
	"knative.dev/test-infra/tools/coverage/test"
)

const (
	testPresubmitBuild = 787
)

func repoDataForTest() *githubPr.GithubPr {
	ctx := context.Background()
	log.Printf("creating fake repo data \n")

	return &githubPr.GithubPr{
		RepoOwner:     "fakeRepoOwner",
		RepoName:      "fakeRepoName",
		Pr:            7,
		RobotUserName: "fakeCovbot",
		GithubClient:  githubFakes.FakeGithubClient(),
		Ctx:           ctx,
	}
}

func gcsArtifactsForTest() *gcs.GcsArtifacts {
	return &gcs.GcsArtifacts{
		Ctx:       context.Background(),
		Bucket:    "fakeBucket",
		Client:    gcsFakes.NewFakeStorageClient(),
		Artifacts: artsTest.LocalArtsForTest("gcsArts-").Artifacts,
	}
}

func preSubmitForTest() (data *gcs.PreSubmit) {
	repoData := repoDataForTest()
	build := gcs.GcsBuild{
		Client: gcsFakes.NewFakeStorageClient(),
		Bucket: gcsFakes.FakeGcsBucketName,
		Job:    gcsFakes.FakePreSubmitProwJobName,
		Build:  testPresubmitBuild,
	}
	pbuild := gcs.PresubmitBuild{
		GcsBuild:      build,
		Artifacts:     *gcsArtifactsForTest(),
		PostSubmitJob: gcsFakes.FakePostSubmitProwJobName,
	}
	data = &gcs.PreSubmit{
		GithubPr:       *repoData,
		PresubmitBuild: pbuild,
	}
	log.Println("finished preSubmitForTest()")
	return
}

func TestRunPresubmit(t *testing.T) {
	log.Println("Starting TestRunPresubmit")
	arts := artsTest.LocalArtsForTest("TestRunPresubmit")
	arts.ProduceProfileFile("./" + test.CovTargetRelPath)
	p := preSubmitForTest()
	RunPresubmit(p, arts)
	if !test.FileOrDirExists(arts.LineCovFilePath()) {
		t.Fatalf("No line cov file found in %s\n", arts.LineCovFilePath())
	}
}

// tests the construction of gcs url from PreSubmit
func TestK8sGcsAddress(t *testing.T) {
	data := preSubmitForTest()
	data.Build = 1286
	got := data.UrlGcsLineCovLinkWithMarker(3)

	want := fmt.Sprintf("https://storage.cloud.google.com/%s/pr-logs/pull/"+
		"%s_%s/%s/%s/%s/artifacts/line-cov.html#file3",
		gcsFakes.FakeGcsBucketName, data.RepoOwner, data.RepoName, data.PrStr(), gcsFakes.FakePreSubmitProwJobName, "1286")
	if got != want {
		t.Fatal(test.StrFailure("", want, got))
	}
	fmt.Printf("line cov link=%s", got)
}
