/*
Copyright 2016 The Kubernetes Authors.

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

package trigger

import (
	"encoding/json"
	"fmt"
	"net/url"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/errorutil"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/labels"
	"k8s.io/test-infra/prow/pjutil"
	"k8s.io/test-infra/prow/plugins"
)

func handlePR(c Client, trigger plugins.Trigger, pr github.PullRequestEvent) error {
	if len(c.Config.Presubmits[pr.PullRequest.Base.Repo.FullName]) == 0 {
		return nil
	}

	org, repo, a := orgRepoAuthor(pr.PullRequest)
	author := string(a)
	num := pr.PullRequest.Number
	switch pr.Action {
	case github.PullRequestActionOpened:
		// When a PR is opened, if the author is in the org then build it.
		// Otherwise, ask for "/ok-to-test". There's no need to look for previous
		// "/ok-to-test" comments since the PR was just opened!
		member, err := TrustedUser(c.GitHubClient, trigger.OnlyOrgMembers, trigger.TrustedOrg, author, org, repo)
		if err != nil {
			return fmt.Errorf("could not check membership: %s", err)
		}
		if member {
			c.Logger.Info("Starting all jobs for new PR.")
			return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
		}
		c.Logger.Infof("Welcome message to PR author %q.", author)
		if err := welcomeMsg(c.GitHubClient, trigger, pr.PullRequest); err != nil {
			return fmt.Errorf("could not welcome non-org member %q: %v", author, err)
		}
	case github.PullRequestActionReopened:
		// When a PR is reopened, check that the user is in the org or that an org
		// member had said "/ok-to-test" before building, resulting in label ok-to-test.
		l, trusted, err := TrustedPullRequest(c.GitHubClient, trigger, author, org, repo, num, nil)
		if err != nil {
			return fmt.Errorf("could not validate PR: %s", err)
		} else if trusted {
			// Eventually remove need-ok-to-test
			// Does not work for TrustedUser() == true since labels are not fetched in this case
			if github.HasLabel(labels.NeedsOkToTest, l) {
				if err := c.GitHubClient.RemoveLabel(org, repo, num, labels.NeedsOkToTest); err != nil {
					return err
				}
			}
			c.Logger.Info("Starting all jobs for updated PR.")
			return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
		}
	case github.PullRequestActionEdited:
		// if someone changes the base of their PR, we will get this
		// event and the changes field will list that the base SHA and
		// ref changes so we can detect such a case and retrigger tests
		var changes struct {
			Base struct {
				Ref struct {
					From string `json:"from"`
				} `json:"ref"`
				Sha struct {
					From string `json:"from"`
				} `json:"sha"`
			} `json:"base"`
		}
		if err := json.Unmarshal(pr.Changes, &changes); err != nil {
			// we're detecting this best-effort so we can forget about
			// the event
			return nil
		} else if changes.Base.Ref.From != "" || changes.Base.Sha.From != "" {
			// the base of the PR changed and we need to re-test it
			return buildAllIfTrusted(c, trigger, pr)
		}
	case github.PullRequestActionSynchronize:
		return buildAllIfTrusted(c, trigger, pr)
	case github.PullRequestActionLabeled:
		// When a PR is LGTMd, if it is untrusted then build it once.
		if pr.Label.Name == labels.LGTM {
			_, trusted, err := TrustedPullRequest(c.GitHubClient, trigger, author, org, repo, num, nil)
			if err != nil {
				return fmt.Errorf("could not validate PR: %s", err)
			} else if !trusted {
				c.Logger.Info("Starting all jobs for untrusted PR with LGTM.")
				return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
			}
		}
	}
	return nil
}

type login string

func orgRepoAuthor(pr github.PullRequest) (string, string, login) {
	org := pr.Base.Repo.Owner.Login
	repo := pr.Base.Repo.Name
	author := pr.User.Login
	return org, repo, login(author)
}

func buildAllIfTrusted(c Client, trigger plugins.Trigger, pr github.PullRequestEvent) error {
	// When a PR is updated, check that the user is in the org or that an org
	// member has said "/ok-to-test" before building. There's no need to ask
	// for "/ok-to-test" because we do that once when the PR is created.
	org, repo, a := orgRepoAuthor(pr.PullRequest)
	author := string(a)
	num := pr.PullRequest.Number
	l, trusted, err := TrustedPullRequest(c.GitHubClient, trigger, author, org, repo, num, nil)
	if err != nil {
		return fmt.Errorf("could not validate PR: %s", err)
	} else if trusted {
		// Eventually remove needs-ok-to-test
		// Will not work for org members since labels are not fetched in this case
		if github.HasLabel(labels.NeedsOkToTest, l) {
			if err := c.GitHubClient.RemoveLabel(org, repo, num, labels.NeedsOkToTest); err != nil {
				return err
			}
		}
		c.Logger.Info("Starting all jobs for updated PR.")
		return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
	}
	return nil
}

func welcomeMsg(ghc githubClient, trigger plugins.Trigger, pr github.PullRequest) error {
	var errors []error
	org, repo, a := orgRepoAuthor(pr)
	author := string(a)
	encodedRepoFullName := url.QueryEscape(pr.Base.Repo.FullName)
	var more string
	if trigger.TrustedOrg != "" && trigger.TrustedOrg != org {
		more = fmt.Sprintf("or [%s](https://github.com/orgs/%s/people) ", trigger.TrustedOrg, trigger.TrustedOrg)
	}

	var joinOrgURL string
	if trigger.JoinOrgURL != "" {
		joinOrgURL = trigger.JoinOrgURL
	} else {
		joinOrgURL = fmt.Sprintf("https://github.com/orgs/%s/people", org)
	}

	var comment string
	if trigger.IgnoreOkToTest {
		comment = fmt.Sprintf(`Hi @%s. Thanks for your PR.

PRs from untrusted users cannot be marked as trusted with `+"`/ok-to-test`"+` in this repo meaning untrusted PR authors can never trigger tests themselves. Collaborators can still trigger tests on the PR using `+"`/test all`"+`.

I understand the commands that are listed [here](https://go.k8s.io/bot-commands?repo=%s).

<details>

%s
</details>
`, author, encodedRepoFullName, plugins.AboutThisBotWithoutCommands)
	} else {
		comment = fmt.Sprintf(`Hi @%s. Thanks for your PR.

I'm waiting for a [%s](https://github.com/orgs/%s/people) %smember to verify that this patch is reasonable to test. If it is, they should reply with `+"`/ok-to-test`"+` on its own line. Until that is done, I will not automatically test new commits in this PR, but the usual testing commands by org members will still work. Regular contributors should [join the org](%s) to skip this step.

Once the patch is verified, the new status will be reflected by the `+"`%s`"+` label.

I understand the commands that are listed [here](https://go.k8s.io/bot-commands?repo=%s).

<details>

%s
</details>
`, author, org, org, more, joinOrgURL, labels.OkToTest, encodedRepoFullName, plugins.AboutThisBotWithoutCommands)
		if err := ghc.AddLabel(org, repo, pr.Number, labels.NeedsOkToTest); err != nil {
			errors = append(errors, err)
		}
	}

	if err := ghc.CreateComment(org, repo, pr.Number, comment); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return errorutil.NewAggregate(errors...)
	}
	return nil
}

// TrustedPullRequest returns whether or not the given PR should be tested.
// It first checks if the author is in the org, then looks for "ok-to-test" label.
// If already known, GitHub labels should be provided to save tokens. Otherwise, it fetches them.
func TrustedPullRequest(tprc trustedPullRequestClient, trigger plugins.Trigger, author, org, repo string, num int, l []github.Label) ([]github.Label, bool, error) {
	// First check if the author is a member of the org.
	if orgMember, err := TrustedUser(tprc, trigger.OnlyOrgMembers, trigger.TrustedOrg, author, org, repo); err != nil {
		return l, false, fmt.Errorf("error checking %s for trust: %v", author, err)
	} else if orgMember {
		return l, true, nil
	}
	// Then check if PR has ok-to-test label
	if l == nil {
		var err error
		l, err = tprc.GetIssueLabels(org, repo, num)
		if err != nil {
			return l, false, err
		}
	}
	return l, github.HasLabel(labels.OkToTest, l), nil
}

// buildAll ensures that all builds that should run and will be required are built
func buildAll(c Client, pr *github.PullRequest, eventGUID string, elideSkippedContexts bool) error {
	org, repo, number, branch := pr.Base.Repo.Owner.Login, pr.Base.Repo.Name, pr.Number, pr.Base.Ref
	changes := config.NewGitHubDeferredChangedFilesProvider(c.GitHubClient, org, repo, number)
	toTest, toSkip, err := pjutil.FilterPresubmits(pjutil.TestAllFilter(), changes, branch, c.Config.Presubmits[pr.Base.Repo.FullName], c.Logger)
	if err != nil {
		return err
	}
	return RunAndSkipJobs(c, pr, toTest, toSkip, eventGUID, elideSkippedContexts)
}
