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

/*
Package main prototypes uploading resource (go test coverage profile) to GCS
if enable debug, then the reading from GCS feature would be run as well
*/

package gcs

import (
	"path"
	"strconv"

	"github.com/knative/test-infra/tools/coverage/artifacts"
	"github.com/knative/test-infra/tools/coverage/githubUtil/githubPr"
)

const (
	ArtifactsDirNameOnGcs = "artifacts"
	gcsUrlHost            = "storage.cloud.google.com/"
)

type PresubmitBuild struct {
	GcsBuild
	Artifacts     GcsArtifacts
	PostSubmitJob string
}

type PreSubmit struct {
	githubPr.GithubPr
	PresubmitBuild
}

func (p *PreSubmit) relDirOfJob() (result string) {
	return path.Join("pr-logs", "pull", p.RepoOwner+"_"+p.RepoName,
		p.PrStr(),
		p.Job)
}

func (p *PreSubmit) relDirOfBuild() (result string) {
	return path.Join(p.relDirOfJob(), p.BuildStr())
}

func (p *PreSubmit) relDirOfArtifacts() (result string) {
	return path.Join(p.relDirOfBuild(), ArtifactsDirNameOnGcs)
}

func (p *PreSubmit) urlArtifactsDir() (result string) {
	return path.Join(gcsUrlHost, p.Bucket, p.relDirOfArtifacts())
}

func (p *PreSubmit) MakeGcsArtifacts(localArts artifacts.LocalArtifacts) *GcsArtifacts {
	localArts.SetDirectory(p.relDirOfArtifacts())
	res := NewGcsArtifacts(p.Ctx, p.StorageClient, p.Bucket, localArts.Artifacts)
	return res
}

func (p *PreSubmit) urlLineCov() (result string) {
	return path.Join(p.urlArtifactsDir(), artifacts.LineCovFileName)
}

func (p *PreSubmit) UrlGcsLineCovLinkWithMarker(section int) (result string) {
	return "https://" + p.urlLineCov() + "#file" + strconv.Itoa(section)
}
