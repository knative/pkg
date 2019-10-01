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

package repoowners

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/sets"
	prowConf "k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/localgit"
	"k8s.io/test-infra/prow/github"
)

var (
	testFiles = map[string][]byte{
		"foo": []byte(`approvers:
- bob`),
		"OWNERS": []byte(`approvers:
- cjwagner
reviewers:
- Alice
- bob
required_reviewers:
- chris
labels:
- EVERYTHING`),
		"src/OWNERS": []byte(`approvers:
- Best-Approvers`),
		"src/dir/OWNERS": []byte(`approvers:
- bob
reviewers:
- alice
- "@CJWagner"
- jakub
required_reviewers:
- ben
labels:
- src-code`),
		"src/dir/subdir/OWNERS": []byte(`approvers:
- bob
- alice
reviewers:
- bob
- alice`),
		"src/dir/conformance/OWNERS": []byte(`options:
  no_parent_owners: true
approvers:
- mml`),
		"docs/file.md": []byte(`---
approvers:
- ALICE

labels:
- docs
---`),
		"vendor/OWNERS": []byte(`approvers:
- alice`),
		"vendor/k8s.io/client-go/OWNERS": []byte(`approvers:
- bob`),
	}

	testFilesRe = map[string][]byte{
		// regexp filtered
		"re/OWNERS": []byte(`filters:
  ".*":
    labels:
    - re/all
  "\\.go$":
    labels:
    - re/go`),
		"re/a/OWNERS": []byte(`filters:
  "\\.md$":
    labels:
    - re/md-in-a
  "\\.go$":
    labels:
    - re/go-in-a`),
	}
)

// regexpAll is used to construct a default {regexp -> values} mapping for ".*"
func regexpAll(values ...string) map[*regexp.Regexp]sets.String {
	return map[*regexp.Regexp]sets.String{nil: sets.NewString(values...)}
}

// patternAll is used to construct a default {regexp string -> values} mapping for ".*"
func patternAll(values ...string) map[string]sets.String {
	// use "" to represent nil and distinguish it from a ".*" regexp (which shouldn't exist).
	return map[string]sets.String{"": sets.NewString(values...)}
}

type cacheOptions struct {
	hasAliases bool

	mdYaml                   bool
	commonFileChanged        bool
	mdFileChanged            bool
	ownersAliasesFileChanged bool
	ownersFileChanged        bool
}

type fakeGitHubClient struct {
	Collaborators []string
	ref           string
}

func (f *fakeGitHubClient) ListCollaborators(org, repo string) ([]github.User, error) {
	result := make([]github.User, 0, len(f.Collaborators))
	for _, login := range f.Collaborators {
		result = append(result, github.User{Login: login})
	}
	return result, nil
}

func (f *fakeGitHubClient) GetRef(org, repo, ref string) (string, error) {
	return f.ref, nil
}

func getTestClient(
	files map[string][]byte,
	enableMdYaml,
	skipCollab,
	includeAliases bool,
	ignorePreconfiguredDefaults bool,
	ownersDirBlacklistDefault []string,
	ownersDirBlacklistByRepo map[string][]string,
	extraBranchesAndFiles map[string]map[string][]byte,
	cacheOptions *cacheOptions,
) (*Client, func(), error) {
	testAliasesFile := map[string][]byte{
		"OWNERS_ALIASES": []byte("aliases:\n  Best-approvers:\n  - carl\n  - cjwagner\n  best-reviewers:\n  - Carl\n  - BOB"),
	}

	localGit, git, err := localgit.New()
	if err != nil {
		return nil, nil, err
	}
	if err := localGit.MakeFakeRepo("org", "repo"); err != nil {
		return nil, nil, fmt.Errorf("cannot make fake repo: %v", err)
	}
	if err := localGit.AddCommit("org", "repo", files); err != nil {
		return nil, nil, fmt.Errorf("cannot add initial commit: %v", err)
	}
	if includeAliases {
		if err := localGit.AddCommit("org", "repo", testAliasesFile); err != nil {
			return nil, nil, fmt.Errorf("cannot add OWNERS_ALIASES commit: %v", err)
		}
	}
	if len(extraBranchesAndFiles) > 0 {
		for branch, extraFiles := range extraBranchesAndFiles {
			if err := localGit.CheckoutNewBranch("org", "repo", branch); err != nil {
				return nil, nil, err
			}
			if len(extraFiles) > 0 {
				if err := localGit.AddCommit("org", "repo", extraFiles); err != nil {
					return nil, nil, fmt.Errorf("cannot add commit: %v", err)
				}
			}
		}
		if err := localGit.Checkout("org", "repo", "master"); err != nil {
			return nil, nil, err
		}
	}
	cache := make(map[string]cacheEntry)
	if cacheOptions != nil {
		var entry cacheEntry
		entry.sha, err = localGit.RevParse("org", "repo", "HEAD")
		if err != nil {
			return nil, nil, fmt.Errorf("cannot get commit SHA: %v", err)
		}
		if cacheOptions.hasAliases {
			entry.aliases = make(map[string]sets.String)
		}
		entry.owners = &RepoOwners{
			enableMDYAML: cacheOptions.mdYaml,
		}
		if cacheOptions.commonFileChanged {
			md := map[string][]byte{"common": []byte(`---
This file could be anything
---`)}
			if err := localGit.AddCommit("org", "repo", md); err != nil {
				return nil, nil, fmt.Errorf("cannot add commit: %v", err)
			}
		}
		if cacheOptions.mdFileChanged {
			md := map[string][]byte{"docs/file.md": []byte(`---
approvers:
- ALICE


labels:
- docs
---`)}
			if err := localGit.AddCommit("org", "repo", md); err != nil {
				return nil, nil, fmt.Errorf("cannot add commit: %v", err)
			}
		}
		if cacheOptions.ownersAliasesFileChanged {
			testAliasesFile = map[string][]byte{
				"OWNERS_ALIASES": []byte("aliases:\n  Best-approvers:\n\n  - carl\n  - cjwagner\n  best-reviewers:\n  - Carl\n  - BOB"),
			}
			if err := localGit.AddCommit("org", "repo", testAliasesFile); err != nil {
				return nil, nil, fmt.Errorf("cannot add commit: %v", err)
			}
		}
		if cacheOptions.ownersFileChanged {
			owners := map[string][]byte{
				"OWNERS": []byte(`approvers:
- cjwagner
reviewers:
- "@Alice"
- bob

required_reviewers:
- chris
labels:
- EVERYTHING`),
			}
			if err := localGit.AddCommit("org", "repo", owners); err != nil {
				return nil, nil, fmt.Errorf("cannot add commit: %v", err)
			}
		}
		cache["org"+"/"+"repo:master"] = entry
		// mark this entry is cache
		entry.owners.baseDir = "cache"
	}
	ghc := &fakeGitHubClient{Collaborators: []string{"cjwagner", "k8s-ci-robot", "alice", "bob", "carl", "mml", "maggie"}}
	ghc.ref, err = localGit.RevParse("org", "repo", "HEAD")
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get commit SHA: %v", err)
	}
	return &Client{
			git:    git,
			ghc:    ghc,
			logger: logrus.WithField("client", "repoowners"),
			cache:  cache,

			mdYAMLEnabled: func(org, repo string) bool {
				return enableMdYaml
			},
			skipCollaborators: func(org, repo string) bool {
				return skipCollab
			},
			ownersDirBlacklist: func() prowConf.OwnersDirBlacklist {
				return prowConf.OwnersDirBlacklist{
					Repos:                       ownersDirBlacklistByRepo,
					Default:                     ownersDirBlacklistDefault,
					IgnorePreconfiguredDefaults: ignorePreconfiguredDefaults,
				}
			},
		},
		// Clean up function
		func() {
			git.Clean()
			localGit.Clean()
		},
		nil
}

func TestOwnersDirBlacklist(t *testing.T) {
	getRepoOwnersWithBlacklist := func(t *testing.T, defaults []string, byRepo map[string][]string, ignorePreconfiguredDefaults bool) *RepoOwners {
		client, cleanup, err := getTestClient(testFiles, true, false, true, ignorePreconfiguredDefaults, defaults, byRepo, nil, nil)
		if err != nil {
			t.Fatalf("Error creating test client: %v.", err)
		}
		defer cleanup()

		ro, err := client.LoadRepoOwners("org", "repo", "master")
		if err != nil {
			t.Fatalf("Unexpected error loading RepoOwners: %v.", err)
		}

		return ro.(*RepoOwners)
	}

	type testConf struct {
		blacklistDefault            []string
		blacklistByRepo             map[string][]string
		ignorePreconfiguredDefaults bool
		includeDirs                 []string
		excludeDirs                 []string
	}

	tests := map[string]testConf{}

	tests["blacklist by org"] = testConf{
		blacklistByRepo: map[string][]string{
			"org": {"src"},
		},
		includeDirs: []string{""},
		excludeDirs: []string{"src", "src/dir", "src/dir/conformance", "src/dir/subdir"},
	}
	tests["blacklist by org/repo"] = testConf{
		blacklistByRepo: map[string][]string{
			"org/repo": {"src"},
		},
		includeDirs: []string{""},
		excludeDirs: []string{"src", "src/dir", "src/dir/conformance", "src/dir/subdir"},
	}
	tests["blacklist by default"] = testConf{
		blacklistDefault: []string{"src"},
		includeDirs:      []string{""},
		excludeDirs:      []string{"src", "src/dir", "src/dir/conformance", "src/dir/subdir"},
	}
	tests["subdir blacklist"] = testConf{
		blacklistDefault: []string{"dir"},
		includeDirs:      []string{"", "src"},
		excludeDirs:      []string{"src/dir", "src/dir/conformance", "src/dir/subdir"},
	}
	tests["no blacklist setup"] = testConf{
		includeDirs: []string{"", "src", "src/dir", "src/dir/conformance", "src/dir/subdir"},
	}
	tests["blacklist setup but not matching this repo"] = testConf{
		blacklistByRepo: map[string][]string{
			"not_org/not_repo": {"src"},
			"not_org":          {"src"},
		},
		includeDirs: []string{"", "src", "src/dir", "src/dir/conformance", "src/dir/subdir"},
	}
	tests["non-matching blacklist"] = testConf{
		blacklistDefault: []string{"sr$"},
		includeDirs:      []string{"", "src", "src/dir", "src/dir/conformance", "src/dir/subdir"},
	}
	tests["path blacklist"] = testConf{
		blacklistDefault: []string{"src/dir"},
		includeDirs:      []string{"", "src"},
		excludeDirs:      []string{"src/dir", "src/dir/conformance", "src/dir/subdir"},
	}
	tests["regexp blacklist path"] = testConf{
		blacklistDefault: []string{"src/dir/."},
		includeDirs:      []string{"", "src", "src/dir"},
		excludeDirs:      []string{"src/dir/conformance", "src/dir/subdir"},
	}
	tests["path substring"] = testConf{
		blacklistDefault: []string{"/c"},
		includeDirs:      []string{"", "src", "src/dir", "src/dir/subdir"},
		excludeDirs:      []string{"src/dir/conformance"},
	}
	tests["exclude preconfigured defaults"] = testConf{
		includeDirs: []string{"", "src", "src/dir", "src/dir/subdir", "vendor"},
		excludeDirs: []string{"vendor/k8s.io/client-go"},
	}
	tests["ignore preconfigured defaults"] = testConf{
		includeDirs:                 []string{"", "src", "src/dir", "src/dir/subdir", "vendor", "vendor/k8s.io/client-go"},
		ignorePreconfiguredDefaults: true,
	}

	for name, conf := range tests {
		t.Run(name, func(t *testing.T) {
			ro := getRepoOwnersWithBlacklist(t, conf.blacklistDefault, conf.blacklistByRepo, conf.ignorePreconfiguredDefaults)

			includeDirs := sets.NewString(conf.includeDirs...)
			excludeDirs := sets.NewString(conf.excludeDirs...)
			for dir := range ro.approvers {
				if excludeDirs.Has(dir) {
					t.Errorf("Expected directory %s to be excluded from the approvers map", dir)
				}
				includeDirs.Delete(dir)
			}
			for dir := range ro.reviewers {
				if excludeDirs.Has(dir) {
					t.Errorf("Expected directory %s to be excluded from the reviewers map", dir)
				}
				includeDirs.Delete(dir)
			}

			for _, dir := range includeDirs.List() {
				t.Errorf("Expected to find approvers or reviewers for directory %s", dir)
			}
		})
	}
}

func TestOwnersRegexpFiltering(t *testing.T) {
	tests := map[string]sets.String{
		"re/a/go.go":   sets.NewString("re/all", "re/go", "re/go-in-a"),
		"re/a/md.md":   sets.NewString("re/all", "re/md-in-a"),
		"re/a/txt.txt": sets.NewString("re/all"),
		"re/go.go":     sets.NewString("re/all", "re/go"),
		"re/txt.txt":   sets.NewString("re/all"),
		"re/b/md.md":   sets.NewString("re/all"),
	}

	client, cleanup, err := getTestClient(testFilesRe, true, false, true, false, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Error creating test client: %v.", err)
	}
	defer cleanup()

	r, err := client.LoadRepoOwners("org", "repo", "master")
	if err != nil {
		t.Fatalf("Unexpected error loading RepoOwners: %v.", err)
	}
	ro := r.(*RepoOwners)
	t.Logf("labels: %#v\n\n", ro.labels)
	for file, expected := range tests {
		if got := ro.FindLabelsForFile(file); !got.Equal(expected) {
			t.Errorf("For file %q expected labels %q, but got %q.", file, expected.List(), got.List())
		}
	}
}

func strP(str string) *string {
	return &str
}

func TestLoadRepoOwners(t *testing.T) {
	tests := []struct {
		name              string
		mdEnabled         bool
		aliasesFileExists bool
		skipCollaborators bool
		// used for testing OWNERS from a branch different from master
		branch                *string
		extraBranchesAndFiles map[string]map[string][]byte

		expectedApprovers, expectedReviewers, expectedRequiredReviewers, expectedLabels map[string]map[string]sets.String

		expectedOptions  map[string]dirOptions
		cacheOptions     *cacheOptions
		expectedReusable bool
	}{
		{
			name: "no alias, no md",
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll(),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":        patternAll("EVERYTHING"),
				"src/dir": patternAll("src-code"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
		},
		{
			name:              "alias, no md",
			aliasesFileExists: true,
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll("carl", "cjwagner"),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":        patternAll("EVERYTHING"),
				"src/dir": patternAll("src-code"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
		},
		{
			name:              "alias, md",
			aliasesFileExists: true,
			mdEnabled:         true,
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll("carl", "cjwagner"),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"docs/file.md":        patternAll("alice"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":             patternAll("EVERYTHING"),
				"src/dir":      patternAll("src-code"),
				"docs/file.md": patternAll("docs"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
		},
		{
			name:   "OWNERS from non-default branch",
			branch: strP("release-1.10"),
			extraBranchesAndFiles: map[string]map[string][]byte{
				"release-1.10": {
					"src/doc/OWNERS": []byte("approvers:\n - maggie\n"),
				},
			},
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll(),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"src/doc":             patternAll("maggie"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":        patternAll("EVERYTHING"),
				"src/dir": patternAll("src-code"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
		},
		{
			name:   "OWNERS from master branch while release branch diverges",
			branch: strP("master"),
			extraBranchesAndFiles: map[string]map[string][]byte{
				"release-1.10": {
					"src/doc/OWNERS": []byte("approvers:\n - maggie\n"),
				},
			},
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll(),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":        patternAll("EVERYTHING"),
				"src/dir": patternAll("src-code"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
		},
		{
			name:              "Skip collaborator checks, use only OWNERS files",
			skipCollaborators: true,
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll("best-approvers"),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner", "jakub"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":        patternAll("EVERYTHING"),
				"src/dir": patternAll("src-code"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
		},
		{
			name:              "cache reuses, base sha equals to cache sha",
			skipCollaborators: true,
			cacheOptions: &cacheOptions{
				hasAliases: true,
			},
			expectedReusable: true,
		},
		{
			name:              "cache reuses, only change common files",
			skipCollaborators: true,
			cacheOptions: &cacheOptions{
				hasAliases:        true,
				commonFileChanged: true,
			},
			expectedReusable: true,
		},
		{
			name:              "cache does not reuse, mdYaml changed",
			aliasesFileExists: true,
			mdEnabled:         true,
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll("carl", "cjwagner"),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"docs/file.md":        patternAll("alice"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":             patternAll("EVERYTHING"),
				"src/dir":      patternAll("src-code"),
				"docs/file.md": patternAll("docs"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
			cacheOptions: &cacheOptions{},
		},
		{
			name:              "cache does not reuse, aliases is nil",
			aliasesFileExists: true,
			mdEnabled:         true,
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll("carl", "cjwagner"),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"docs/file.md":        patternAll("alice"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":             patternAll("EVERYTHING"),
				"src/dir":      patternAll("src-code"),
				"docs/file.md": patternAll("docs"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
			cacheOptions: &cacheOptions{
				commonFileChanged: true,
			},
		},
		{
			name:              "cache does not reuse, changes files contains OWNERS",
			aliasesFileExists: true,
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll("carl", "cjwagner"),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":        patternAll("EVERYTHING"),
				"src/dir": patternAll("src-code"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
			cacheOptions: &cacheOptions{
				hasAliases:        true,
				ownersFileChanged: true,
			},
		},
		{
			name:              "cache does not reuse, changes files contains OWNERS_ALIASES",
			aliasesFileExists: true,
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll("carl", "cjwagner"),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":        patternAll("EVERYTHING"),
				"src/dir": patternAll("src-code"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
			cacheOptions: &cacheOptions{
				hasAliases:               true,
				ownersAliasesFileChanged: true,
			},
		},
		{
			name:              "cache reuses, changes files contains .md, but mdYaml is false",
			skipCollaborators: true,
			cacheOptions: &cacheOptions{
				hasAliases:    true,
				mdFileChanged: true,
			},
			expectedReusable: true,
		},
		{
			name:              "cache does not reuse, changes files contains .md, and mdYaml is true",
			aliasesFileExists: true,
			mdEnabled:         true,
			expectedApprovers: map[string]map[string]sets.String{
				"":                    patternAll("cjwagner"),
				"src":                 patternAll("carl", "cjwagner"),
				"src/dir":             patternAll("bob"),
				"src/dir/conformance": patternAll("mml"),
				"src/dir/subdir":      patternAll("alice", "bob"),
				"docs/file.md":        patternAll("alice"),
				"vendor":              patternAll("alice"),
			},
			expectedReviewers: map[string]map[string]sets.String{
				"":               patternAll("alice", "bob"),
				"src/dir":        patternAll("alice", "cjwagner"),
				"src/dir/subdir": patternAll("alice", "bob"),
			},
			expectedRequiredReviewers: map[string]map[string]sets.String{
				"":        patternAll("chris"),
				"src/dir": patternAll("ben"),
			},
			expectedLabels: map[string]map[string]sets.String{
				"":             patternAll("EVERYTHING"),
				"src/dir":      patternAll("src-code"),
				"docs/file.md": patternAll("docs"),
			},
			expectedOptions: map[string]dirOptions{
				"src/dir/conformance": {
					NoParentOwners: true,
				},
			},
			cacheOptions: &cacheOptions{
				hasAliases:    true,
				mdYaml:        true,
				mdFileChanged: true,
			},
		},
	}

	for _, test := range tests {
		t.Logf("Running scenario %q", test.name)
		client, cleanup, err := getTestClient(testFiles, test.mdEnabled, test.skipCollaborators, test.aliasesFileExists, false, nil, nil, test.extraBranchesAndFiles, test.cacheOptions)
		if err != nil {
			t.Errorf("Error creating test client: %v.", err)
			continue
		}
		defer cleanup()

		base := "master"
		if test.branch != nil {
			base = *test.branch
		}
		r, err := client.LoadRepoOwners("org", "repo", base)
		if err != nil {
			t.Errorf("Unexpected error loading RepoOwners: %v.", err)
			continue
		}
		ro := r.(*RepoOwners)
		if test.expectedReusable {
			if ro.baseDir != "cache" {
				t.Error("expected cache must be reused, but not")
			}
			continue
		} else {
			if ro.baseDir == "cache" {
				t.Error("expected cache should not be reused, but reused")
				continue
			}
		}
		if ro.baseDir == "" {
			t.Errorf("Expected 'baseDir' to be populated.")
			continue
		}
		if (ro.RepoAliases != nil) != test.aliasesFileExists {
			t.Errorf("Expected 'RepoAliases' to be poplulated: %t, but got %t.", test.aliasesFileExists, ro.RepoAliases != nil)
			continue
		}
		if ro.enableMDYAML != test.mdEnabled {
			t.Errorf("Expected 'enableMdYaml' to be: %t, but got %t.", test.mdEnabled, ro.enableMDYAML)
			continue
		}

		check := func(field string, expected map[string]map[string]sets.String, got map[string]map[*regexp.Regexp]sets.String) {
			converted := map[string]map[string]sets.String{}
			for path, m := range got {
				converted[path] = map[string]sets.String{}
				for re, s := range m {
					var pattern string
					if re != nil {
						pattern = re.String()
					}
					converted[path][pattern] = s
				}
			}
			if !reflect.DeepEqual(expected, converted) {
				t.Errorf("Expected %s to be:\n%+v\ngot:\n%+v.", field, expected, converted)
			}
		}
		check("approvers", test.expectedApprovers, ro.approvers)
		check("reviewers", test.expectedReviewers, ro.reviewers)
		check("required_reviewers", test.expectedRequiredReviewers, ro.requiredReviewers)
		check("labels", test.expectedLabels, ro.labels)
		if !reflect.DeepEqual(test.expectedOptions, ro.options) {
			t.Errorf("Expected options to be:\n%#v\ngot:\n%#v.", test.expectedOptions, ro.options)
		}
	}
}

func TestLoadRepoAliases(t *testing.T) {
	tests := []struct {
		name string

		aliasFileExists       bool
		branch                *string
		extraBranchesAndFiles map[string]map[string][]byte

		expectedRepoAliases RepoAliases
	}{
		{
			name:                "No aliases file",
			aliasFileExists:     false,
			expectedRepoAliases: nil,
		},
		{
			name:            "Normal aliases file",
			aliasFileExists: true,
			expectedRepoAliases: RepoAliases{
				"best-approvers": sets.NewString("carl", "cjwagner"),
				"best-reviewers": sets.NewString("carl", "bob"),
			},
		},
		{
			name: "Aliases file from non-default branch",

			aliasFileExists: true,
			branch:          strP("release-1.10"),
			extraBranchesAndFiles: map[string]map[string][]byte{
				"release-1.10": {
					"OWNERS_ALIASES": []byte("aliases:\n  Best-approvers:\n  - carl\n  - cjwagner\n  best-reviewers:\n  - Carl\n  - BOB\n  - maggie"),
				},
			},

			expectedRepoAliases: RepoAliases{
				"best-approvers": sets.NewString("carl", "cjwagner"),
				"best-reviewers": sets.NewString("carl", "bob", "maggie"),
			},
		},
	}
	for _, test := range tests {
		client, cleanup, err := getTestClient(testFiles, false, false, test.aliasFileExists, false, nil, nil, test.extraBranchesAndFiles, nil)
		if err != nil {
			t.Errorf("[%s] Error creating test client: %v.", test.name, err)
			continue
		}

		branch := "master"
		if test.branch != nil {
			branch = *test.branch
		}
		got, err := client.LoadRepoAliases("org", "repo", branch)
		if err != nil {
			t.Errorf("[%s] Unexpected error loading RepoAliases: %v.", test.name, err)
			cleanup()
			continue
		}
		if !reflect.DeepEqual(got, test.expectedRepoAliases) {
			t.Errorf("[%s] Expected RepoAliases: %#v, but got: %#v.", test.name, test.expectedRepoAliases, got)
		}
		cleanup()
	}
}

const (
	baseDir        = ""
	leafDir        = "a/b/c"
	noParentsDir   = "d"
	nonExistentDir = "DELETED_DIR"
)

func TestGetApprovers(t *testing.T) {
	ro := &RepoOwners{
		approvers: map[string]map[*regexp.Regexp]sets.String{
			baseDir:      regexpAll("alice", "bob"),
			leafDir:      regexpAll("carl", "dave"),
			noParentsDir: regexpAll("mml"),
		},
		options: map[string]dirOptions{
			noParentsDir: {
				NoParentOwners: true,
			},
		},
	}
	tests := []struct {
		name               string
		filePath           string
		expectedOwnersPath string
		expectedLeafOwners sets.String
		expectedAllOwners  sets.String
	}{
		{
			name:               "Modified Base Dir Only",
			filePath:           filepath.Join(baseDir, "testFile.md"),
			expectedOwnersPath: baseDir,
			expectedLeafOwners: ro.approvers[baseDir][nil],
			expectedAllOwners:  ro.approvers[baseDir][nil],
		},
		{
			name:               "Modified Leaf Dir Only",
			filePath:           filepath.Join(leafDir, "testFile.md"),
			expectedOwnersPath: leafDir,
			expectedLeafOwners: ro.approvers[leafDir][nil],
			expectedAllOwners:  ro.approvers[baseDir][nil].Union(ro.approvers[leafDir][nil]),
		},
		{
			name:               "Modified NoParentOwners Dir Only",
			filePath:           filepath.Join(noParentsDir, "testFile.go"),
			expectedOwnersPath: noParentsDir,
			expectedLeafOwners: ro.approvers[noParentsDir][nil],
			expectedAllOwners:  ro.approvers[noParentsDir][nil],
		},
		{
			name:               "Modified Nonexistent Dir (Default to Base)",
			filePath:           filepath.Join(nonExistentDir, "testFile.md"),
			expectedOwnersPath: baseDir,
			expectedLeafOwners: ro.approvers[baseDir][nil],
			expectedAllOwners:  ro.approvers[baseDir][nil],
		},
	}
	for testNum, test := range tests {
		foundLeafApprovers := ro.LeafApprovers(test.filePath)
		foundApprovers := ro.Approvers(test.filePath)
		foundOwnersPath := ro.FindApproverOwnersForFile(test.filePath)
		if !foundLeafApprovers.Equal(test.expectedLeafOwners) {
			t.Errorf("The Leaf Approvers Found Do Not Match Expected For Test %d: %s", testNum, test.name)
			t.Errorf("\tExpected Owners: %v\tFound Owners: %v ", test.expectedLeafOwners, foundLeafApprovers)
		}
		if !foundApprovers.Equal(test.expectedAllOwners) {
			t.Errorf("The Approvers Found Do Not Match Expected For Test %d: %s", testNum, test.name)
			t.Errorf("\tExpected Owners: %v\tFound Owners: %v ", test.expectedAllOwners, foundApprovers)
		}
		if foundOwnersPath != test.expectedOwnersPath {
			t.Errorf("The Owners Path Found Does Not Match Expected For Test %d: %s", testNum, test.name)
			t.Errorf("\tExpected Owners: %v\tFound Owners: %v ", test.expectedOwnersPath, foundOwnersPath)
		}
	}
}

func TestFindLabelsForPath(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedLabels sets.String
	}{
		{
			name:           "base 1",
			path:           "foo.txt",
			expectedLabels: sets.NewString("sig/godzilla"),
		}, {
			name:           "base 2",
			path:           "./foo.txt",
			expectedLabels: sets.NewString("sig/godzilla"),
		}, {
			name:           "base 3",
			path:           "",
			expectedLabels: sets.NewString("sig/godzilla"),
		}, {
			name:           "base 4",
			path:           ".",
			expectedLabels: sets.NewString("sig/godzilla"),
		}, {
			name:           "leaf 1",
			path:           "a/b/c/foo.txt",
			expectedLabels: sets.NewString("sig/godzilla", "wg/save-tokyo"),
		}, {
			name:           "leaf 2",
			path:           "a/b/foo.txt",
			expectedLabels: sets.NewString("sig/godzilla"),
		},
	}

	testOwners := &RepoOwners{
		labels: map[string]map[*regexp.Regexp]sets.String{
			baseDir: regexpAll("sig/godzilla"),
			leafDir: regexpAll("wg/save-tokyo"),
		},
	}
	for _, test := range tests {
		got := testOwners.FindLabelsForFile(test.path)
		if !got.Equal(test.expectedLabels) {
			t.Errorf(
				"[%s] Expected labels %q for path %q, but got %q.",
				test.name,
				test.expectedLabels.List(),
				test.path,
				got.List(),
			)
		}
	}
}

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedPath string
	}{
		{
			name:         "Empty String",
			path:         "",
			expectedPath: "",
		},
		{
			name:         "Dot (.) as Path",
			path:         ".",
			expectedPath: "",
		},
		{
			name:         "GitHub Style Input (No Root)",
			path:         "a/b/c/d.txt",
			expectedPath: "a/b/c/d.txt",
		},
		{
			name:         "Preceding Slash and Trailing Slash",
			path:         "/a/b/",
			expectedPath: "/a/b",
		},
		{
			name:         "Trailing Slash",
			path:         "foo/bar/baz/",
			expectedPath: "foo/bar/baz",
		},
	}
	for _, test := range tests {
		if got := canonicalize(test.path); test.expectedPath != got {
			t.Errorf(
				"[%s] Expected the canonical path for %v to be %v.  Found %v instead",
				test.name,
				test.path,
				test.expectedPath,
				got,
			)
		}
	}
}

var (
	lowerCaseAliases = []byte(`
aliases:
  team/t1:
    - u1
    - u2
  team/t2:
    - u1
    - u3`)
	mixedCaseAliases = []byte(`
aliases:
  TEAM/T1:
    - U1
    - U2`)
)

func TestExpandAliases(t *testing.T) {
	testAliases := RepoAliases{
		"team/t1": sets.NewString("u1", "u2"),
		"team/t2": sets.NewString("u1", "u3"),
	}
	tests := []struct {
		name             string
		unexpanded       sets.String
		expectedExpanded sets.String
	}{
		{
			name:             "No expansions.",
			unexpanded:       sets.NewString("abc", "def"),
			expectedExpanded: sets.NewString("abc", "def"),
		},
		{
			name:             "One alias to be expanded",
			unexpanded:       sets.NewString("abc", "team/t1"),
			expectedExpanded: sets.NewString("abc", "u1", "u2"),
		},
		{
			name:             "Duplicates inside and outside alias.",
			unexpanded:       sets.NewString("u1", "team/t1"),
			expectedExpanded: sets.NewString("u1", "u2"),
		},
		{
			name:             "Duplicates in multiple aliases.",
			unexpanded:       sets.NewString("u1", "team/t1", "team/t2"),
			expectedExpanded: sets.NewString("u1", "u2", "u3"),
		},
		{
			name:             "Mixed casing in aliases.",
			unexpanded:       sets.NewString("Team/T1"),
			expectedExpanded: sets.NewString("u1", "u2"),
		},
	}

	for _, test := range tests {
		if got := testAliases.ExpandAliases(test.unexpanded); !test.expectedExpanded.Equal(got) {
			t.Errorf(
				"[%s] Expected %q to expand to %q, but got %q.",
				test.name,
				test.unexpanded.List(),
				test.expectedExpanded.List(),
				got.List(),
			)
		}
	}
}
