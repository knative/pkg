/*
Copyright 2017 The Kubernetes Authors.

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

package blunderbuss

import (
	"context"
	"errors"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/plugins"
	"k8s.io/test-infra/prow/repoowners"
)

type fakeGitHubClient struct {
	pr        *github.PullRequest
	changes   []github.PullRequestChange
	requested []string
}

func newFakeGitHubClient(pr *github.PullRequest, filesChanged []string) *fakeGitHubClient {
	changes := make([]github.PullRequestChange, 0, len(filesChanged))
	for _, name := range filesChanged {
		changes = append(changes, github.PullRequestChange{Filename: name})
	}
	return &fakeGitHubClient{pr: pr, changes: changes}
}

func (c *fakeGitHubClient) RequestReview(org, repo string, number int, logins []string) error {
	if org != "org" {
		return errors.New("org should be 'org'")
	}
	if repo != "repo" {
		return errors.New("repo should be 'repo'")
	}
	if number != 5 {
		return errors.New("number should be 5")
	}
	c.requested = append(c.requested, logins...)
	return nil
}

func (c *fakeGitHubClient) GetPullRequestChanges(org, repo string, num int) ([]github.PullRequestChange, error) {
	if org != "org" {
		return nil, errors.New("org should be 'org'")
	}
	if repo != "repo" {
		return nil, errors.New("repo should be 'repo'")
	}
	if num != 5 {
		return nil, errors.New("number should be 5")
	}
	return c.changes, nil
}

func (c *fakeGitHubClient) GetPullRequest(org, repo string, num int) (*github.PullRequest, error) {
	return c.pr, nil
}

func (f *fakeGitHubClient) Query(ctx context.Context, q interface{}, vars map[string]interface{}) error {
	sq, ok := q.(*githubAvailabilityQuery)
	if !ok {
		return errors.New("unexpected query type")
	}
	sq.User.Login = vars["user"].(githubql.String)
	if sq.User.Login == githubql.String("busy-user") {
		sq.User.Status.IndicatesLimitedAvailability = githubql.Boolean(true)
	}
	return nil
}

type fakeRepoownersClient struct {
	foc *fakeOwnersClient
}

func (froc fakeRepoownersClient) LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error) {
	return froc.foc, nil
}

type fakeOwnersClient struct {
	owners            map[string]string
	approvers         map[string]sets.String
	leafApprovers     map[string]sets.String
	reviewers         map[string]sets.String
	requiredReviewers map[string]sets.String
	leafReviewers     map[string]sets.String
	dirBlacklist      []*regexp.Regexp
}

func (foc *fakeOwnersClient) Approvers(path string) sets.String {
	return foc.approvers[path]
}

func (foc *fakeOwnersClient) LeafApprovers(path string) sets.String {
	return foc.leafApprovers[path]
}

func (foc *fakeOwnersClient) FindApproverOwnersForFile(path string) string {
	return foc.owners[path]
}

func (foc *fakeOwnersClient) Reviewers(path string) sets.String {
	return foc.reviewers[path]
}

func (foc *fakeOwnersClient) RequiredReviewers(path string) sets.String {
	return foc.requiredReviewers[path]
}

func (foc *fakeOwnersClient) LeafReviewers(path string) sets.String {
	return foc.leafReviewers[path]
}

func (foc *fakeOwnersClient) FindReviewersOwnersForFile(path string) string {
	return foc.owners[path]
}

func (foc *fakeOwnersClient) FindLabelsForFile(path string) sets.String {
	return sets.String{}
}

func (foc *fakeOwnersClient) IsNoParentOwners(path string) bool {
	return false
}

func (foc *fakeOwnersClient) ParseSimpleConfig(path string) (repoowners.SimpleConfig, error) {
	dir := filepath.Dir(path)
	for _, re := range foc.dirBlacklist {
		if re.MatchString(dir) {
			return repoowners.SimpleConfig{}, filepath.SkipDir
		}
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return repoowners.SimpleConfig{}, err
	}
	full := new(repoowners.SimpleConfig)
	err = yaml.Unmarshal(b, full)
	return *full, err
}

func (foc *fakeOwnersClient) ParseFullConfig(path string) (repoowners.FullConfig, error) {
	dir := filepath.Dir(path)
	for _, re := range foc.dirBlacklist {
		if re.MatchString(dir) {
			return repoowners.FullConfig{}, filepath.SkipDir
		}
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return repoowners.FullConfig{}, err
	}
	full := new(repoowners.FullConfig)
	err = yaml.Unmarshal(b, full)
	return *full, err
}

var (
	owners = map[string]string{
		"a.go":  "1",
		"b.go":  "2",
		"bb.go": "3",
		"c.go":  "4",

		"e.go":  "5",
		"ee.go": "5",
	}
	reviewers = map[string]sets.String{
		"a.go": sets.NewString("al"),
		"b.go": sets.NewString("al"),
		"c.go": sets.NewString("charles"),

		"e.go":  sets.NewString("erick", "evan"),
		"ee.go": sets.NewString("erick", "evan"),
		"f.go":  sets.NewString("author", "non-author"),
	}
	requiredReviewers = map[string]sets.String{
		"a.go": sets.NewString("ben"),

		"ee.go": sets.NewString("chris", "charles"),
	}
	leafReviewers = map[string]sets.String{
		"a.go":  sets.NewString("alice"),
		"b.go":  sets.NewString("bob"),
		"bb.go": sets.NewString("bob", "ben"),
		"c.go":  sets.NewString("cole", "carl", "chad"),

		"e.go":  sets.NewString("erick", "ellen"),
		"ee.go": sets.NewString("erick", "ellen"),
		"f.go":  sets.NewString("author"),
	}
	testcases = []struct {
		name                       string
		filesChanged               []string
		reviewerCount              int
		maxReviewerCount           int
		expectedRequested          []string
		alternateExpectedRequested []string
	}{
		{
			name:              "one file, 3 leaf reviewers, 1 parent, request 3",
			filesChanged:      []string{"c.go"},
			reviewerCount:     3,
			expectedRequested: []string{"cole", "carl", "chad"},
		},
		{
			name:              "one file, 3 leaf reviewers, 1 parent reviewer, request 4",
			filesChanged:      []string{"c.go"},
			reviewerCount:     4,
			expectedRequested: []string{"cole", "carl", "chad", "charles"},
		},
		{
			name:              "two files, 2 leaf reviewers, 1 common parent, request 2",
			filesChanged:      []string{"a.go", "b.go"},
			reviewerCount:     2,
			expectedRequested: []string{"alice", "ben", "bob"},
		},
		{
			name:              "two files, 2 leaf reviewers, 1 common parent, request 3",
			filesChanged:      []string{"a.go", "b.go"},
			reviewerCount:     3,
			expectedRequested: []string{"alice", "ben", "bob", "al"},
		},
		{
			name:              "one files, 1 leaf reviewers, request 1",
			filesChanged:      []string{"a.go"},
			reviewerCount:     1,
			maxReviewerCount:  1,
			expectedRequested: []string{"alice", "ben"},
		},
		{
			name:              "one file, 2 leaf reviewer, 2 parent reviewers (1 dup), request 3",
			filesChanged:      []string{"e.go"},
			reviewerCount:     3,
			expectedRequested: []string{"erick", "ellen", "evan"},
		},
		{
			name:                       "two files, 2 leaf reviewer, 2 parent reviewers (1 dup), request 1",
			filesChanged:               []string{"e.go"},
			reviewerCount:              1,
			expectedRequested:          []string{"erick"},
			alternateExpectedRequested: []string{"ellen"},
		},
		{
			name:              "two files, 1 common leaf reviewer, one additional leaf, one parent, request 1",
			filesChanged:      []string{"b.go", "bb.go"},
			reviewerCount:     1,
			expectedRequested: []string{"bob", "ben"},
		},
		{
			name:              "two files, 2 leaf reviewers, 1 common parent, request 1",
			filesChanged:      []string{"a.go", "b.go"},
			reviewerCount:     1,
			expectedRequested: []string{"alice", "ben", "bob"},
		},
		{
			name:                       "two files, 2 leaf reviewers, 1 common parent, request 1, limit 2",
			filesChanged:               []string{"a.go", "b.go"},
			reviewerCount:              1,
			maxReviewerCount:           1,
			expectedRequested:          []string{"alice", "ben"},
			alternateExpectedRequested: []string{"ben", "bob"},
		},
		{
			name:              "exclude author",
			filesChanged:      []string{"f.go"},
			reviewerCount:     1,
			expectedRequested: []string{"non-author"},
		},
	}
)

// TestHandleWithExcludeApprovers tests that the handle function requests
// reviews from the correct number of unique users when ExcludeApprovers is
// true.
func TestHandleWithExcludeApproversOnlyReviewers(t *testing.T) {
	froc := &fakeRepoownersClient{
		foc: &fakeOwnersClient{
			owners:            owners,
			reviewers:         reviewers,
			requiredReviewers: requiredReviewers,
			leafReviewers:     leafReviewers,
		},
	}

	for _, tc := range testcases {
		pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}}
		repo := github.Repo{Owner: github.User{Login: "org"}, Name: "repo"}
		fghc := newFakeGitHubClient(&pr, tc.filesChanged)

		if err := handle(
			fghc, froc, logrus.WithField("plugin", PluginName),
			&tc.reviewerCount, nil, tc.maxReviewerCount, true, false, &repo, &pr,
		); err != nil {
			t.Errorf("[%s] unexpected error from handle: %v", tc.name, err)
			continue
		}

		sort.Strings(fghc.requested)
		sort.Strings(tc.expectedRequested)
		sort.Strings(tc.alternateExpectedRequested)
		if !reflect.DeepEqual(fghc.requested, tc.expectedRequested) {
			if len(tc.alternateExpectedRequested) > 0 {
				if !reflect.DeepEqual(fghc.requested, tc.alternateExpectedRequested) {
					t.Errorf("[%s] expected the requested reviewers to be %q or %q, but got %q.", tc.name, tc.expectedRequested, tc.alternateExpectedRequested, fghc.requested)
				}
				continue
			}
			t.Errorf("[%s] expected the requested reviewers to be %q, but got %q.", tc.name, tc.expectedRequested, fghc.requested)
		}
	}
}

// TestHandleWithoutExcludeApprovers verifies that behavior is the same
// when ExcludeApprovers is false and only approvers exist in the OWNERS files.
// The owners fixture and test cases should always be the same as the ones in
// TestHandleWithExcludeApprovers.
func TestHandleWithoutExcludeApproversNoReviewers(t *testing.T) {
	froc := &fakeRepoownersClient{
		foc: &fakeOwnersClient{
			owners:            owners,
			approvers:         reviewers,
			leafApprovers:     leafReviewers,
			requiredReviewers: requiredReviewers,
		},
	}

	for _, tc := range testcases {
		pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}}
		repo := github.Repo{Owner: github.User{Login: "org"}, Name: "repo"}
		fghc := newFakeGitHubClient(&pr, tc.filesChanged)

		if err := handle(
			fghc, froc, logrus.WithField("plugin", PluginName),
			&tc.reviewerCount, nil, tc.maxReviewerCount, false, false, &repo, &pr,
		); err != nil {
			t.Errorf("[%s] unexpected error from handle: %v", tc.name, err)
			continue
		}

		sort.Strings(fghc.requested)
		sort.Strings(tc.expectedRequested)
		sort.Strings(tc.alternateExpectedRequested)
		if !reflect.DeepEqual(fghc.requested, tc.expectedRequested) {
			if len(tc.alternateExpectedRequested) > 0 {
				if !reflect.DeepEqual(fghc.requested, tc.alternateExpectedRequested) {
					t.Errorf("[%s] expected the requested reviewers to be %q or %q, but got %q.", tc.name, tc.expectedRequested, tc.alternateExpectedRequested, fghc.requested)
				}
				continue
			}
			t.Errorf("[%s] expected the requested reviewers to be %q, but got %q.", tc.name, tc.expectedRequested, fghc.requested)
		}
	}
}

func TestHandleWithoutExcludeApproversMixed(t *testing.T) {
	froc := &fakeRepoownersClient{
		foc: &fakeOwnersClient{
			owners: map[string]string{
				"a.go":  "1",
				"b.go":  "2",
				"bb.go": "3",
				"c.go":  "4",

				"e.go":  "5",
				"ee.go": "5",
			},
			approvers: map[string]sets.String{
				"a.go": sets.NewString("al"),
				"b.go": sets.NewString("jeff"),
				"c.go": sets.NewString("jeff"),

				"e.go":  sets.NewString(),
				"ee.go": sets.NewString("larry"),
			},
			leafApprovers: map[string]sets.String{
				"a.go": sets.NewString("alice"),
				"b.go": sets.NewString("brad"),
				"c.go": sets.NewString("evan"),

				"e.go":  sets.NewString("erick", "evan"),
				"ee.go": sets.NewString("erick", "evan"),
			},
			reviewers: map[string]sets.String{
				"a.go": sets.NewString("al"),
				"b.go": sets.NewString(),
				"c.go": sets.NewString("charles"),

				"e.go":  sets.NewString("erick", "evan"),
				"ee.go": sets.NewString("erick", "evan"),
			},
			leafReviewers: map[string]sets.String{
				"a.go":  sets.NewString("alice"),
				"b.go":  sets.NewString("bob"),
				"bb.go": sets.NewString("bob", "ben"),
				"c.go":  sets.NewString("cole", "carl", "chad"),

				"e.go":  sets.NewString("erick", "ellen"),
				"ee.go": sets.NewString("erick", "ellen"),
			},
		},
	}

	var testcases = []struct {
		name                       string
		filesChanged               []string
		reviewerCount              int
		maxReviewerCount           int
		expectedRequested          []string
		alternateExpectedRequested []string
	}{
		{
			name:              "1 file, 1 leaf reviewer, 1 leaf approver, 1 approver, request 3",
			filesChanged:      []string{"b.go"},
			reviewerCount:     3,
			expectedRequested: []string{"bob", "brad", "jeff"},
		},
		{
			name:              "1 file, 1 leaf reviewer, 1 leaf approver, 1 approver, request 1, limit 1",
			filesChanged:      []string{"b.go"},
			reviewerCount:     1,
			expectedRequested: []string{"bob"},
		},
		{
			name:              "2 file, 2 leaf reviewers, 1 parent reviewers, 1 leaf approver, 1 approver, request 5",
			filesChanged:      []string{"a.go", "b.go"},
			reviewerCount:     5,
			expectedRequested: []string{"alice", "bob", "al", "brad", "jeff"},
		},
		{
			name:              "1 file, 1 leaf reviewer+approver, 1 reviewer+approver, request 3",
			filesChanged:      []string{"a.go"},
			reviewerCount:     3,
			expectedRequested: []string{"alice", "al"},
		},
		{
			name:              "1 file, 2 leaf reviewers, request 2",
			filesChanged:      []string{"e.go"},
			reviewerCount:     2,
			expectedRequested: []string{"erick", "ellen"},
		},
		{
			name:              "2 files, 2 leaf+parent reviewers, 1 parent reviewer, 1 parent approver, request 4",
			filesChanged:      []string{"e.go", "ee.go"},
			reviewerCount:     4,
			expectedRequested: []string{"erick", "ellen", "evan", "larry"},
		},
	}
	for _, tc := range testcases {
		pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}}
		repo := github.Repo{Owner: github.User{Login: "org"}, Name: "repo"}
		fghc := newFakeGitHubClient(&pr, tc.filesChanged)
		if err := handle(
			fghc, froc, logrus.WithField("plugin", PluginName),
			&tc.reviewerCount, nil, tc.maxReviewerCount, false, false, &repo, &pr,
		); err != nil {
			t.Errorf("[%s] unexpected error from handle: %v", tc.name, err)
			continue
		}

		sort.Strings(fghc.requested)
		sort.Strings(tc.expectedRequested)
		sort.Strings(tc.alternateExpectedRequested)
		if !reflect.DeepEqual(fghc.requested, tc.expectedRequested) {
			if len(tc.alternateExpectedRequested) > 0 {
				if !reflect.DeepEqual(fghc.requested, tc.alternateExpectedRequested) {
					t.Errorf("[%s] expected the requested reviewers to be %q or %q, but got %q.", tc.name, tc.expectedRequested, tc.alternateExpectedRequested, fghc.requested)
				}
				continue
			}
			t.Errorf("[%s] expected the requested reviewers to be %q, but got %q.", tc.name, tc.expectedRequested, fghc.requested)
		}
	}
}

func TestHandleOld(t *testing.T) {
	froc := &fakeRepoownersClient{
		foc: &fakeOwnersClient{
			reviewers: map[string]sets.String{
				"c.go": sets.NewString("charles"),
				"d.go": sets.NewString("dan"),
				"e.go": sets.NewString("erick", "evan"),
				"f.go": sets.NewString("author", "non-author"),
			},
			leafReviewers: map[string]sets.String{
				"a.go": sets.NewString("alice"),
				"b.go": sets.NewString("bob"),
				"c.go": sets.NewString("cole", "carl", "chad"),
				"e.go": sets.NewString("erick"),
				"f.go": sets.NewString("author"),
			},
		},
	}

	var testcases = []struct {
		name              string
		filesChanged      []string
		reviewerCount     int
		expectedRequested []string
	}{
		{
			name:              "one file, 3 leaf reviewers, request 3",
			filesChanged:      []string{"c.go"},
			reviewerCount:     3,
			expectedRequested: []string{"cole", "carl", "chad"},
		},
		{
			name:              "one file, 3 leaf reviewers, 1 parent reviewer, request 4",
			filesChanged:      []string{"c.go"},
			reviewerCount:     4,
			expectedRequested: []string{"cole", "carl", "chad", "charles"},
		},
		{
			name:              "two files, 2 leaf reviewers, request 2",
			filesChanged:      []string{"a.go", "b.go"},
			reviewerCount:     2,
			expectedRequested: []string{"alice", "bob"},
		},
		{
			name:              "one files, 1 leaf reviewers, request 1",
			filesChanged:      []string{"a.go"},
			reviewerCount:     1,
			expectedRequested: []string{"alice"},
		},
		{
			name:              "one file, 0 leaf reviewers, 1 parent reviewer, request 1",
			filesChanged:      []string{"d.go"},
			reviewerCount:     1,
			expectedRequested: []string{"dan"},
		},
		{
			name:              "one file, 0 leaf reviewers, 1 parent reviewer, request 2",
			filesChanged:      []string{"d.go"},
			reviewerCount:     2,
			expectedRequested: []string{"dan"},
		},
		{
			name:              "one file, 1 leaf reviewers, 2 parent reviewers (1 dup), request 2",
			filesChanged:      []string{"e.go"},
			reviewerCount:     2,
			expectedRequested: []string{"erick", "evan"},
		},
		{
			name:              "exclude author",
			filesChanged:      []string{"f.go"},
			reviewerCount:     1,
			expectedRequested: []string{"non-author"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}}
			repo := github.Repo{Owner: github.User{Login: "org"}, Name: "repo"}
			fghc := newFakeGitHubClient(&pr, tc.filesChanged)

			err := handle(
				fghc, froc, logrus.WithField("plugin", PluginName),
				nil, &tc.reviewerCount, 0, false, false, &repo, &pr,
			)
			if err != nil {
				t.Fatalf("unexpected error from handle: %v", err)
			}

			sort.Strings(fghc.requested)
			sort.Strings(tc.expectedRequested)
			if !reflect.DeepEqual(fghc.requested, tc.expectedRequested) {
				t.Fatalf("expected the requested reviewers to be %q, but got %q.", tc.expectedRequested, fghc.requested)
			}
		})
	}
}

func TestHandlePullRequest(t *testing.T) {
	froc := &fakeRepoownersClient{
		foc: &fakeOwnersClient{
			owners: map[string]string{
				"a.go": "1",
			},
			leafReviewers: map[string]sets.String{
				"a.go": sets.NewString("al"),
			},
		},
	}

	var testcases = []struct {
		name              string
		action            github.PullRequestEventAction
		body              string
		filesChanged      []string
		reviewerCount     int
		expectedRequested []string
	}{
		{
			name:              "PR opened",
			action:            github.PullRequestActionOpened,
			body:              "/auto-cc",
			filesChanged:      []string{"a.go"},
			expectedRequested: []string{"al"},
		},
		{
			name:         "PR opened with /cc command",
			action:       github.PullRequestActionOpened,
			body:         "/cc",
			filesChanged: []string{"a.go"},
		},
		{
			name:         "PR closed",
			action:       github.PullRequestActionClosed,
			body:         "/auto-cc",
			filesChanged: []string{"a.go"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}, Body: tc.body}
			repo := github.Repo{Owner: github.User{Login: "org"}, Name: "repo"}
			fghc := newFakeGitHubClient(&pr, tc.filesChanged)
			config := plugins.Blunderbuss{
				ReviewerCount:    &tc.reviewerCount,
				FileWeightCount:  nil,
				MaxReviewerCount: 0,
				ExcludeApprovers: false,
			}

			if err := handlePullRequest(
				fghc, froc, logrus.WithField("plugin", PluginName),
				config, tc.action, &pr, &repo,
			); err != nil {
				t.Fatalf("unexpected error from handle: %v", err)
			}

			sort.Strings(fghc.requested)
			sort.Strings(tc.expectedRequested)
			if !reflect.DeepEqual(fghc.requested, tc.expectedRequested) {
				t.Fatalf("expected the requested reviewers to be %q, but got %q.", tc.expectedRequested, fghc.requested)
			}
		})
	}
}

func TestHandleGenericComment(t *testing.T) {
	froc := &fakeRepoownersClient{
		foc: &fakeOwnersClient{
			owners: map[string]string{
				"a.go": "1",
			},
			leafReviewers: map[string]sets.String{
				"a.go": sets.NewString("al"),
			},
		},
	}

	var testcases = []struct {
		name              string
		action            github.GenericCommentEventAction
		issueState        string
		isPR              bool
		body              string
		filesChanged      []string
		reviewerCount     int
		expectedRequested []string
	}{
		{
			name:              "comment with a valid command in an open PR triggers auto-assignment",
			action:            github.GenericCommentActionCreated,
			issueState:        "open",
			isPR:              true,
			body:              "/auto-cc",
			filesChanged:      []string{"a.go"},
			expectedRequested: []string{"al"},
		},
		{
			name:         "comment with an invalid command in an open PR will not trigger auto-assignment",
			action:       github.GenericCommentActionCreated,
			issueState:   "open",
			isPR:         true,
			body:         "/automatic-review",
			filesChanged: []string{"a.go"},
		},
		{
			name:         "comment with a valid command in a closed PR will not trigger auto-assignment",
			action:       github.GenericCommentActionCreated,
			issueState:   "closed",
			isPR:         true,
			body:         "/auto-cc",
			filesChanged: []string{"a.go"},
		},
		{
			name:         "comment deleted from an open PR will not trigger auto-assignment",
			action:       github.GenericCommentActionDeleted,
			issueState:   "open",
			isPR:         true,
			body:         "/auto-cc",
			filesChanged: []string{"a.go"},
		},
		{
			name:       "comment with valid command in an open issue will not trigger auto-assignment",
			action:     github.GenericCommentActionCreated,
			issueState: "open",
			isPR:       false,
			body:       "/auto-cc",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}}
			fghc := newFakeGitHubClient(&pr, tc.filesChanged)
			repo := github.Repo{Owner: github.User{Login: "org"}, Name: "repo"}
			config := plugins.Blunderbuss{
				ReviewerCount:    &tc.reviewerCount,
				FileWeightCount:  nil,
				MaxReviewerCount: 0,
				ExcludeApprovers: false,
			}

			if err := handleGenericComment(
				fghc, froc, logrus.WithField("plugin", PluginName), config,
				tc.action, tc.isPR, pr.Number, tc.issueState, &repo, tc.body,
			); err != nil {
				t.Fatalf("unexpected error from handle: %v", err)
			}

			sort.Strings(fghc.requested)
			sort.Strings(tc.expectedRequested)
			if !reflect.DeepEqual(fghc.requested, tc.expectedRequested) {
				t.Fatalf("expected the requested reviewers to be %q, but got %q.", tc.expectedRequested, fghc.requested)
			}
		})
	}
}

func TestHandleGenericCommentEvent(t *testing.T) {
	pc := plugins.Agent{
		PluginConfig: &plugins.Configuration{},
	}
	ce := github.GenericCommentEvent{}
	handleGenericCommentEvent(pc, ce)
}

func TestHandlePullRequestEvent(t *testing.T) {
	pc := plugins.Agent{
		PluginConfig: &plugins.Configuration{},
	}
	pre := github.PullRequestEvent{}
	handlePullRequestEvent(pc, pre)
}

func TestHelpProvider(t *testing.T) {
	cases := []struct {
		name               string
		config             *plugins.Configuration
		enabledRepos       []string
		err                bool
		configInfoIncludes []string
	}{
		{
			name:               "Empty config",
			config:             &plugins.Configuration{},
			enabledRepos:       []string{"org1", "org2/repo"},
			configInfoIncludes: []string{configString(0)},
		},
		{
			name:               "Overlapping org and org/repo",
			config:             &plugins.Configuration{},
			enabledRepos:       []string{"org2", "org2/repo"},
			configInfoIncludes: []string{configString(0)},
		},
		{
			name:               "Invalid enabledRepos",
			config:             &plugins.Configuration{},
			enabledRepos:       []string{"org1", "org2/repo/extra"},
			err:                true,
			configInfoIncludes: []string{configString(0)},
		},
		{
			name: "ReviewerCount specified",
			config: &plugins.Configuration{
				Blunderbuss: plugins.Blunderbuss{
					ReviewerCount: &[]int{2}[0],
				},
			},
			enabledRepos:       []string{"org1", "org2/repo"},
			configInfoIncludes: []string{configString(2)},
		},
		{
			name: "FileWeightCount specified",
			config: &plugins.Configuration{
				Blunderbuss: plugins.Blunderbuss{
					FileWeightCount: &[]int{2}[0],
				},
			},
			enabledRepos:       []string{"org1", "org2/repo"},
			configInfoIncludes: []string{configString(2)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pluginHelp, err := helpProvider(c.config, c.enabledRepos)
			if err != nil && !c.err {
				t.Fatalf("helpProvider error: %v", err)
			}
			for _, msg := range c.configInfoIncludes {
				if !strings.Contains(pluginHelp.Config[""], msg) {
					t.Fatalf("helpProvider.Config error mismatch: didn't get %v, but wanted it", msg)
				}
			}
		})
	}
}

// TestPopActiveReviewer checks to ensure that no matter how hard we try, we
// never assign a user that has their availability marked as busy.
func TestPopActiveReviewer(t *testing.T) {
	froc := &fakeRepoownersClient{
		foc: &fakeOwnersClient{
			owners: map[string]string{
				"a.go":  "1",
				"b.go":  "2",
				"bb.go": "3",
				"c.go":  "4",
			},
			approvers: map[string]sets.String{
				"a.go": sets.NewString("alice"),
				"b.go": sets.NewString("brad"),
				"c.go": sets.NewString("busy-user"),
			},
			leafApprovers: map[string]sets.String{
				"a.go": sets.NewString("alice"),
				"b.go": sets.NewString("brad"),
				"c.go": sets.NewString("busy-user"),
			},
			reviewers: map[string]sets.String{
				"a.go": sets.NewString("alice"),
				"b.go": sets.NewString("brad"),
				"c.go": sets.NewString("busy-user"),
			},
			leafReviewers: map[string]sets.String{
				"a.go": sets.NewString("alice"),
				"b.go": sets.NewString("brad"),
				"c.go": sets.NewString("busy-user"),
			},
		},
	}

	var testcases = []struct {
		name                       string
		filesChanged               []string
		reviewerCount              int
		maxReviewerCount           int
		expectedRequested          []string
		alternateExpectedRequested []string
	}{
		{
			name:              "request three reviewers, only receive two, never get the busy user",
			filesChanged:      []string{"a.go", "b.go", "c.go"},
			reviewerCount:     3,
			expectedRequested: []string{"alice", "brad"},
		},
	}
	for _, tc := range testcases {
		pr := github.PullRequest{Number: 5, User: github.User{Login: "author"}}
		repo := github.Repo{Owner: github.User{Login: "org"}, Name: "repo"}
		fghc := newFakeGitHubClient(&pr, tc.filesChanged)
		if err := handle(
			fghc, froc, logrus.WithField("plugin", PluginName),
			&tc.reviewerCount, nil, tc.maxReviewerCount, false, true, &repo, &pr,
		); err != nil {
			t.Errorf("[%s] unexpected error from handle: %v", tc.name, err)
			continue
		}

		sort.Strings(fghc.requested)
		sort.Strings(tc.expectedRequested)
		sort.Strings(tc.alternateExpectedRequested)
		if !reflect.DeepEqual(fghc.requested, tc.expectedRequested) {
			if len(tc.alternateExpectedRequested) > 0 {
				if !reflect.DeepEqual(fghc.requested, tc.alternateExpectedRequested) {
					t.Errorf("[%s] expected the requested reviewers to be %q or %q, but got %q.", tc.name, tc.expectedRequested, tc.alternateExpectedRequested, fghc.requested)
				}
				continue
			}
			t.Errorf("[%s] expected the requested reviewers to be %q, but got %q.", tc.name, tc.expectedRequested, fghc.requested)
		}
	}
}
