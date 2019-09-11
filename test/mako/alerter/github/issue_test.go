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

package github

import (
	"fmt"
	"os"
	"testing"

	"knative.dev/pkg/test/ghutil"
	"knative.dev/pkg/test/ghutil/fakeghutil"
)

var gih IssueHandler

func TestMain(m *testing.M) {
	gih = IssueHandler{
		client: fakeghutil.NewFakeGithubClient(),
		config: config{org: "test_org", repo: "test_repo", dryrun: false},
	}
	os.Exit(m.Run())
}

func TestNewIssueWillBeAdded(t *testing.T) {
	testName := "test add new issue"
	testDesc := "test add new issue desc"
	if err := gih.CreateIssueForTest(testName, testDesc); err != nil {
		t.Fatalf("expected to create a new issue %v, but failed", testName)
	}
	issueTitle := fmt.Sprintf(issueTitleTemplate, testName)
	issueFound := gih.findIssue(gih.config.org, gih.config.repo, issueTitle, false)
	if issueFound == nil {
		t.Fatalf("expected to find the new created issue %v, but failed to", testName)
	}
}

func TestClosedIssueWillBeReopened(t *testing.T) {
	org := gih.config.org
	repo := gih.config.repo
	testName := "test reopening close issue"
	testDesc := "test reopening close issue desc"
	issueTitle := fmt.Sprintf(issueTitleTemplate, testName)
	issue, _ := gih.client.CreateIssue(org, repo, issueTitle, testDesc)
	gih.client.CloseIssue(org, repo, *issue.Number)

	if err := gih.CreateIssueForTest(testName, testDesc); err != nil {
		t.Fatalf("expected to update the existed issue %v, but failed", testName)
	}
	updatedIssue := gih.findIssue(org, repo, issueTitle, gih.config.dryrun)
	if updatedIssue == nil || *updatedIssue.State != string(ghutil.IssueOpenState) {
		t.Fatalf("expected to reopen the closed issue %v, but failed", testName)
	}
}

func TestIssueCanBeClosed(t *testing.T) {
	org := gih.config.org
	repo := gih.config.repo
	testName := "test closing existed issue"
	testDesc := "test closing existed issue desc"
	issueTitle := fmt.Sprintf(issueTitleTemplate, testName)
	if err := gih.CreateIssueForTest(testName, testDesc); err != nil {
		t.Fatalf("expected to create a new issue %v, but failed", testName)
	}

	if err := gih.CloseIssueForTest(testName); err != nil {
		t.Fatalf("tried to close the existed issue %v, but got an error %v", testName, err)
	}
	updatedIssue := gih.findIssue(org, repo, issueTitle, gih.config.dryrun)
	if updatedIssue == nil || *updatedIssue.State != string(ghutil.IssueCloseState) {
		t.Fatalf("expected to close the issue %v, but failed", testName)
	}
}
