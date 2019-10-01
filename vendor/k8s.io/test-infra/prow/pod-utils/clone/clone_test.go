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

package clone

import (
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
)

func TestPathForRefs(t *testing.T) {
	var testCases = []struct {
		name     string
		refs     prowapi.Refs
		expected string
	}{
		{
			name: "literal override",
			refs: prowapi.Refs{
				PathAlias: "alias",
			},
			expected: "base/src/alias",
		},
		{
			name: "default generated",
			refs: prowapi.Refs{
				Org:  "org",
				Repo: "repo",
			},
			expected: "base/src/github.com/org/repo",
		},
	}

	for _, testCase := range testCases {
		if actual, expected := PathForRefs("base", testCase.refs), testCase.expected; actual != expected {
			t.Errorf("%s: expected path %q, got %q", testCase.name, expected, actual)
		}
	}
}

func TestCommandsForRefs(t *testing.T) {
	fakeTimestamp := 100200300
	var testCases = []struct {
		name                                       string
		refs                                       prowapi.Refs
		dir, gitUserName, gitUserEmail, cookiePath string
		env                                        []string
		expectedBase                               []cloneCommand
		expectedPull                               []cloneCommand
	}{
		{
			name: "simplest case, minimal refs",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "minimal refs with git user name",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
			},
			gitUserName: "user",
			dir:         "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"config", "user.name", "user"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "minimal refs with git user email",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
			},
			gitUserEmail: "user@go.com",
			dir:          "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"config", "user.email", "user@go.com"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "minimal refs with http cookie file",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
			},
			cookiePath: "/cookie.txt",
			dir:        "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"config", "http.cookiefile", "/cookie.txt"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "minimal refs with no submodules",
			refs: prowapi.Refs{
				Org:            "org",
				Repo:           "repo",
				BaseRef:        "master",
				SkipSubmodules: true,
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: nil,
		},
		{
			name: "refs with clone URI override",
			refs: prowapi.Refs{
				Org:      "org",
				Repo:     "repo",
				BaseRef:  "master",
				CloneURI: "internet.com",
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "internet.com", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "internet.com", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "refs with path alias",
			refs: prowapi.Refs{
				Org:       "org",
				Repo:      "repo",
				BaseRef:   "master",
				PathAlias: "my/favorite/dir",
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/my/favorite/dir"}},
				{dir: "/go/src/my/favorite/dir", command: "git", args: []string{"init"}},
				{dir: "/go/src/my/favorite/dir", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/my/favorite/dir", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/my/favorite/dir", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/my/favorite/dir", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/my/favorite/dir", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/my/favorite/dir", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "refs with specific base sha",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
				BaseSHA: "abcdef",
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "abcdef"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "abcdef"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "refs with simple pr ref",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
				Pulls: []prowapi.Pull{
					{Number: 1},
				},
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "pull/1/head"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"merge", "--no-ff", "FETCH_HEAD"}, env: gitTimestampEnvs(fakeTimestamp + 1)},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "refs with pr ref override",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
				Pulls: []prowapi.Pull{
					{Number: 1, Ref: "pull-me"},
				},
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "pull-me"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"merge", "--no-ff", "FETCH_HEAD"}, env: gitTimestampEnvs(fakeTimestamp + 1)},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "refs with pr ref with specific sha",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
				Pulls: []prowapi.Pull{
					{Number: 1, SHA: "abcdef"},
				},
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "pull/1/head"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"merge", "--no-ff", "abcdef"}, env: gitTimestampEnvs(fakeTimestamp + 1)},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
		{
			name: "refs with multiple simple pr refs",
			refs: prowapi.Refs{
				Org:     "org",
				Repo:    "repo",
				BaseRef: "master",
				Pulls: []prowapi.Pull{
					{Number: 1},
					{Number: 2},
				},
			},
			dir: "/go",
			expectedBase: []cloneCommand{
				{dir: "/", command: "mkdir", args: []string{"-p", "/go/src/github.com/org/repo"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"init"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "--tags", "--prune"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "master"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"branch", "--force", "master", "FETCH_HEAD"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"checkout", "master"}},
			},
			expectedPull: []cloneCommand{
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "pull/1/head"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"merge", "--no-ff", "FETCH_HEAD"}, env: gitTimestampEnvs(fakeTimestamp + 1)},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"fetch", "https://github.com/org/repo.git", "pull/2/head"}},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"merge", "--no-ff", "FETCH_HEAD"}, env: gitTimestampEnvs(fakeTimestamp + 2)},
				{dir: "/go/src/github.com/org/repo", command: "git", args: []string{"submodule", "update", "--init", "--recursive"}},
			},
		},
	}

	for _, testCase := range testCases {
		g := gitCtxForRefs(testCase.refs, testCase.dir, testCase.env)
		actualBase := g.commandsForBaseRef(testCase.refs, testCase.gitUserName, testCase.gitUserEmail, testCase.cookiePath)
		if !reflect.DeepEqual(actualBase, testCase.expectedBase) {
			t.Errorf("%s: generated incorrect commands: %v", testCase.name, diff.ObjectGoPrintDiff(testCase.expectedBase, actualBase))
		}
		actualPull := g.commandsForPullRefs(testCase.refs, fakeTimestamp)
		if !reflect.DeepEqual(actualPull, testCase.expectedPull) {
			t.Errorf("%s: generated incorrect commands: %v", testCase.name, diff.ObjectGoPrintDiff(testCase.expectedPull, actualPull))
		}
	}
}

func TestGitHeadTimestamp(t *testing.T) {
	fakeTimestamp := 987654321
	fakeGitDir, err := makeFakeGitRepo(fakeTimestamp)
	if err != nil {
		t.Errorf("error creating fake git dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(fakeGitDir); err != nil {
			t.Errorf("error cleaning up fake git dir: %v", err)
		}
	}()

	var testCases = []struct {
		name        string
		dir         string
		noPath      bool
		expected    int
		expectError bool
	}{
		{
			name:        "root - no git",
			dir:         "/",
			expected:    0,
			expectError: true,
		},
		{
			name:        "fake git repo",
			dir:         fakeGitDir,
			expected:    fakeTimestamp,
			expectError: false,
		},
		{
			name:        "fake git repo but no git binary",
			dir:         fakeGitDir,
			noPath:      true,
			expected:    0,
			expectError: true,
		},
	}
	origCwd, err := os.Getwd()
	if err != nil {
		t.Errorf("failed getting cwd: %v", err)
	}
	origPath := os.Getenv("PATH")
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := os.Chdir(testCase.dir); err != nil {
				t.Errorf("%s: failed to chdir to %s: %v", testCase.name, testCase.dir, err)
			}
			if testCase.noPath {
				if err := os.Unsetenv("PATH"); err != nil {
					t.Errorf("%s: failed to unset PATH: %v", testCase.name, err)
				}
			}
			g := gitCtx{
				cloneDir: testCase.dir,
			}
			timestamp, err := g.gitHeadTimestamp()
			if timestamp != testCase.expected {
				t.Errorf("%s: timestamp %d does not match expected timestamp %d", testCase.name, timestamp, testCase.expected)
			}
			if (err == nil && testCase.expectError) || (err != nil && !testCase.expectError) {
				t.Errorf("%s: expect error is %v but received error %v", testCase.name, testCase.expectError, err)
			}
			if err := os.Chdir(origCwd); err != nil {
				t.Errorf("%s: failed to chdir to original cwd %s: %v", testCase.name, origCwd, err)
			}
			if testCase.noPath {
				if err := os.Setenv("PATH", origPath); err != nil {
					t.Errorf("%s: failed to set PATH to original: %v", testCase.name, err)
				}
			}

		})
	}
}

// makeFakeGitRepo creates a fake git repo with a constant digest and timestamp.
func makeFakeGitRepo(fakeTimestamp int) (string, error) {
	fakeGitDir, err := ioutil.TempDir("", "fakegit")
	if err != nil {
		return "", err
	}
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.test"},
		{"git", "config", "user.name", "test test"},
		{"touch", "a_file"},
		{"git", "add", "a_file"},
		{"git", "commit", "-m", "adding a_file"},
	}
	for _, cmd := range cmds {
		c := exec.Command(cmd[0], cmd[1:]...)
		c.Dir = fakeGitDir
		c.Env = append(os.Environ(), gitTimestampEnvs(fakeTimestamp)...)
		if err := c.Run(); err != nil {
			return fakeGitDir, err
		}
	}
	return fakeGitDir, nil
}
