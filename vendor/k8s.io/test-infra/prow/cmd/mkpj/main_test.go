/*
Copyright 2018 The Kubernetes Authors.

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
	"testing"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

func TestOptions_Validate(t *testing.T) {
	var testCases = []struct {
		name        string
		input       options
		expectedErr bool
	}{
		{
			name: "all ok",
			input: options{
				jobName:    "job",
				configPath: "somewhere",
			},
			expectedErr: false,
		},
		{
			name: "missing config",
			input: options{
				jobName: "job",
			},
			expectedErr: true,
		},
		{
			name: "missing job",
			input: options{
				configPath: "somewhere",
			},
			expectedErr: true,
		},
	}

	for _, testCase := range testCases {
		err := testCase.input.Validate()
		if testCase.expectedErr && err == nil {
			t.Errorf("%s: expected an error but got none", testCase.name)
		}
		if !testCase.expectedErr && err != nil {
			t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
		}
	}
}

func TestDefaultPR(t *testing.T) {
	author := "Bernardo Soares"
	sha := "Esther Greenwood"
	fakeGitHubClient := &fakegithub.FakeClient{}
	fakeGitHubClient.PullRequests = map[int]*github.PullRequest{2: {
		User: github.User{Login: author},
		Head: github.PullRequestBranch{SHA: sha},
	}}
	o := &options{pullNumber: 2, githubClient: fakeGitHubClient}
	pjs := &prowapi.ProwJobSpec{Refs: &prowapi.Refs{Pulls: []prowapi.Pull{{Number: 2}}}}
	if err := o.defaultPR(pjs); err != nil {
		t.Fatalf("Expected no err when defaulting PJ, but got %v", err)
	}
	if pjs.Refs.Pulls[0].Author != author {
		t.Errorf("Expected author to get defaulted to %s but got %s", author, pjs.Refs.Pulls[0].Author)
	}
	if pjs.Refs.Pulls[0].SHA != sha {
		t.Errorf("Expectged sha to get defaulted to %s but got %s", sha, pjs.Refs.Pulls[0].SHA)
	}
}

func TestDefaultBaseRef(t *testing.T) {
	testCases := []struct {
		name            string
		baseRef         string
		expectedBaseSha string
		pullNumber      int
		prBaseSha       string
	}{
		{
			name:            "Default for Presubmit",
			expectedBaseSha: "Theodore Decker",
			pullNumber:      2,
			prBaseSha:       "Theodore Decker",
		},
		{
			name:            "Default for Postsubmit",
			baseRef:         "master",
			expectedBaseSha: fakegithub.TestRef,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			fakeGitHubClient := &fakegithub.FakeClient{}
			fakeGitHubClient.PullRequests = map[int]*github.PullRequest{2: {Base: github.PullRequestBranch{
				SHA: test.prBaseSha,
			}}}
			o := &options{pullNumber: test.pullNumber, githubClient: fakeGitHubClient}
			pjs := &prowapi.ProwJobSpec{Refs: &prowapi.Refs{BaseRef: test.baseRef}}
			if err := o.defaultBaseRef(pjs); err != nil {
				t.Fatalf("Error when calling defaultBaseRef: %v", err)
			}
			if pjs.Refs.BaseSHA != test.expectedBaseSha {
				t.Errorf("Expected BaseSHA to be %s after defaulting but was %s",
					test.expectedBaseSha, pjs.Refs.BaseSHA)
			}
		})
	}

}
