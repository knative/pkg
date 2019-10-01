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

package prstatus

import (
	"context"
	"encoding/gob"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
)

type MockQueryHandler struct {
	prs        []PullRequest
	contextMap map[int][]Context
}

func (mh *MockQueryHandler) QueryPullRequests(ctx context.Context, ghc githubClient, query string) ([]PullRequest, error) {
	return mh.prs, nil
}

func (mh *MockQueryHandler) GetHeadContexts(ghc githubClient, pr PullRequest) ([]Context, error) {
	return mh.contextMap[int(pr.Number)], nil
}

func (mh *MockQueryHandler) BotName(github.Client) (*github.User, error) {
	login := "random_user"
	return &github.User{
		Login: login,
	}, nil
}

type fgc struct {
	combinedStatus *github.CombinedStatus
}

func (c *fgc) Query(context.Context, interface{}, map[string]interface{}) error {
	return nil
}

func (c *fgc) GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error) {
	return c.combinedStatus, nil
}

func newMockQueryHandler(prs []PullRequest, contextMap map[int][]Context) *MockQueryHandler {
	return &MockQueryHandler{
		prs:        prs,
		contextMap: contextMap,
	}
}

func createMockAgent(repos []string, config *config.GitHubOAuthConfig) *DashboardAgent {
	return &DashboardAgent{
		repos: repos,
		goac:  config,
		log:   logrus.WithField("unit-test", "dashboard-agent"),
	}
}

func TestHandlePrStatusWithoutLogin(t *testing.T) {
	repos := []string{"mock/repo", "kubernetes/test-infra", "foo/bar"}
	mockCookieStore := sessions.NewCookieStore([]byte("secret-key"))
	mockConfig := &config.GitHubOAuthConfig{
		CookieStore: mockCookieStore,
	}
	mockAgent := createMockAgent(repos, mockConfig)
	mockData := UserData{
		Login: false,
	}

	rr := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/pr-data.js", nil)

	mockQueryHandler := newMockQueryHandler(nil, nil)
	prHandler := mockAgent.HandlePrStatus(mockQueryHandler)
	prHandler.ServeHTTP(rr, request)
	if rr.Code != http.StatusOK {
		t.Fatalf("Bad status code: %d", rr.Code)
	}
	response := rr.Result()
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Error with reading response body: %v", err)
	}
	var dataReturned UserData
	if err := yaml.Unmarshal(body, &dataReturned); err != nil {
		t.Errorf("Error with unmarshaling response: %v", err)
	}
	if !reflect.DeepEqual(dataReturned, mockData) {
		t.Errorf("Invalid user data. Got %v, expected %v", dataReturned, mockData)
	}
}

func TestHandlePrStatusWithInvalidToken(t *testing.T) {
	logrus.SetLevel(logrus.ErrorLevel)
	repos := []string{"mock/repo", "kubernetes/test-infra", "foo/bar"}
	mockCookieStore := sessions.NewCookieStore([]byte("secret-key"))
	mockConfig := &config.GitHubOAuthConfig{
		CookieStore: mockCookieStore,
	}
	mockAgent := createMockAgent(repos, mockConfig)
	mockQueryHandler := newMockQueryHandler([]PullRequest{}, map[int][]Context{})

	rr := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/pr-data.js", nil)
	request.AddCookie(&http.Cookie{Name: tokenSession, Value: "garbage"})
	prHandler := mockAgent.HandlePrStatus(mockQueryHandler)
	prHandler.ServeHTTP(rr, request)
	if rr.Code != http.StatusOK {
		t.Fatalf("Bad status code: %d", rr.Code)
	}
	response := rr.Result()
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Error with reading response body: %v", err)
	}

	var dataReturned UserData
	if err := yaml.Unmarshal(body, &dataReturned); err != nil {
		t.Errorf("Error with unmarshaling response: %v", err)
	}

	expectedData := UserData{Login: false}
	if !reflect.DeepEqual(dataReturned, expectedData) {
		t.Fatalf("Invalid user data. Got %v, expected %v.", dataReturned, expectedData)
	}
}

func TestHandlePrStatusWithLogin(t *testing.T) {
	repos := []string{"mock/repo", "kubernetes/test-infra", "foo/bar"}
	mockCookieStore := sessions.NewCookieStore([]byte("secret-key"))
	mockConfig := &config.GitHubOAuthConfig{
		CookieStore: mockCookieStore,
	}
	mockAgent := createMockAgent(repos, mockConfig)

	testCases := []struct {
		prs          []PullRequest
		contextMap   map[int][]Context
		expectedData UserData
	}{
		{
			prs:        []PullRequest{},
			contextMap: map[int][]Context{},
			expectedData: UserData{
				Login: true,
			},
		},
		{
			prs: []PullRequest{
				{
					Number: 0,
					Title:  "random pull request",
				},
				{
					Number: 1,
					Title:  "This is a test",
				},
				{
					Number: 2,
					Title:  "test pull request",
				},
			},
			contextMap: map[int][]Context{
				0: {
					{
						Context:     "gofmt-job",
						Description: "job succeed",
						State:       "SUCCESS",
					},
				},
				1: {
					{
						Context:     "verify-bazel-job",
						Description: "job failed",
						State:       "FAILURE",
					},
				},
				2: {
					{
						Context:     "gofmt-job",
						Description: "job succeed",
						State:       "SUCCESS",
					},
					{
						Context:     "verify-bazel-job",
						Description: "job failed",
						State:       "FAILURE",
					},
				},
			},
			expectedData: UserData{
				Login: true,
				PullRequestsWithContexts: []PullRequestWithContexts{
					{
						PullRequest: PullRequest{
							Number: 0,
							Title:  "random pull request",
						},
						Contexts: []Context{
							{
								Context:     "gofmt-job",
								Description: "job succeed",
								State:       "SUCCESS",
							},
						},
					},
					{
						PullRequest: PullRequest{
							Number: 1,
							Title:  "This is a test",
						},
						Contexts: []Context{
							{
								Context:     "verify-bazel-job",
								Description: "job failed",
								State:       "FAILURE",
							},
						},
					},
					{
						PullRequest: PullRequest{
							Number: 2,
							Title:  "test pull request",
						},
						Contexts: []Context{
							{
								Context:     "gofmt-job",
								Description: "job succeed",
								State:       "SUCCESS",
							},
							{
								Context:     "verify-bazel-job",
								Description: "job failed",
								State:       "FAILURE",
							},
						},
					},
				},
			},
		},
	}
	for id, testcase := range testCases {
		t.Logf("Test %d:", id)
		rr := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/pr-data.js", nil)
		mockSession, err := sessions.GetRegistry(request).Get(mockCookieStore, tokenSession)
		if err != nil {
			t.Errorf("Error with creating mock session: %v", err)
		}
		gob.Register(oauth2.Token{})
		token := &oauth2.Token{AccessToken: "secret-token", Expiry: time.Now().Add(time.Duration(24*365) * time.Hour)}
		mockSession.Values[tokenKey] = token
		mockSession.Values[loginKey] = "random_user"
		mockQueryHandler := newMockQueryHandler(testcase.prs, testcase.contextMap)
		prHandler := mockAgent.HandlePrStatus(mockQueryHandler)
		prHandler.ServeHTTP(rr, request)
		if rr.Code != http.StatusOK {
			t.Fatalf("Bad status code: %d", rr.Code)
		}
		response := rr.Result()
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("Error with reading response body: %v", err)
		}
		var dataReturned UserData
		if err := yaml.Unmarshal(body, &dataReturned); err != nil {
			t.Errorf("Error with unmarshaling response: %v", err)
		}
		if !reflect.DeepEqual(dataReturned, testcase.expectedData) {
			t.Fatalf("Invalid user data. Got %v, expected %v.", dataReturned, testcase.expectedData)
		}
		t.Logf("Passed")
		response.Body.Close()
	}
}

func TestGetHeadContexts(t *testing.T) {
	repos := []string{"mock/repo", "kubernetes/test-infra", "foo/bar"}
	mockCookieStore := sessions.NewCookieStore([]byte("secret-key"))
	mockConfig := &config.GitHubOAuthConfig{
		CookieStore: mockCookieStore,
	}
	mockAgent := createMockAgent(repos, mockConfig)
	testCases := []struct {
		combinedStatus   *github.CombinedStatus
		pr               PullRequest
		expectedContexts []Context
	}{
		{
			combinedStatus:   &github.CombinedStatus{},
			pr:               PullRequest{},
			expectedContexts: []Context{},
		},
		{
			combinedStatus: &github.CombinedStatus{
				Statuses: []github.Status{
					{
						State:       "FAILURE",
						Description: "job failed",
						Context:     "gofmt-job",
					},
					{
						State:       "SUCCESS",
						Description: "job succeed",
						Context:     "k8s-job",
					},
					{
						State:       "PENDING",
						Description: "triggered",
						Context:     "test-job",
					},
				},
			},
			pr: PullRequest{},
			expectedContexts: []Context{
				{
					Context:     "gofmt-job",
					Description: "job failed",
					State:       "FAILURE",
				},
				{
					State:       "SUCCESS",
					Description: "job succeed",
					Context:     "k8s-job",
				},
				{
					State:       "PENDING",
					Description: "triggered",
					Context:     "test-job",
				},
			},
		},
	}
	for id, testcase := range testCases {
		t.Logf("Test %d:", id)
		contexts, err := mockAgent.GetHeadContexts(&fgc{
			combinedStatus: testcase.combinedStatus,
		}, testcase.pr)
		if err != nil {
			t.Fatalf("Error with getting head contexts")
		}
		if !reflect.DeepEqual(contexts, testcase.expectedContexts) {
			t.Fatalf("Invalid user data. Got %v, expected %v.", contexts, testcase.expectedContexts)
		}
		t.Logf("Passed")
	}
}

func TestConstructSearchQuery(t *testing.T) {
	repos := []string{"mock/repo", "kubernetes/test-infra", "foo/bar"}
	mockCookieStore := sessions.NewCookieStore([]byte("secret-key"))
	mockConfig := &config.GitHubOAuthConfig{
		CookieStore: mockCookieStore,
	}
	mockAgent := createMockAgent(repos, mockConfig)
	query := mockAgent.ConstructSearchQuery("random_username")
	mockQuery := "is:pr state:open author:random_username repo:\"mock/repo\" repo:\"kubernetes/test-infra\" repo:\"foo/bar\""
	if query != mockQuery {
		t.Errorf("Invalid query. Got: %v, expected %v", query, mockQuery)
	}
}
