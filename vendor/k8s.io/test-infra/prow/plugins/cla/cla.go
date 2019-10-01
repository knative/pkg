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

package cla

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"regexp"

	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/labels"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
)

const (
	pluginName             = "cla"
	claContextName         = "cla/linuxfoundation"
	cncfclaNotFoundMessage = `Thanks for your pull request. Before we can look at your pull request, you'll need to sign a Contributor License Agreement (CLA).

:memo: **Please follow instructions at <https://git.k8s.io/community/CLA.md#the-contributor-license-agreement> to sign the CLA.**

It may take a couple minutes for the CLA signature to be fully registered; after that, please reply here with a new comment and we'll verify.  Thanks.

---

- If you've already signed a CLA, it's possible we don't have your GitHub username or you're using a different email address.  Check your existing CLA data and verify that your [email is set on your git commits](https://help.github.com/articles/setting-your-email-in-git/).
- If you signed the CLA as a corporation, please sign in with your organization's credentials at <https://identity.linuxfoundation.org/projects/cncf> to be authorized.
- If you have done the above and are still having issues with the CLA being reported as unsigned, please log a ticket with the Linux Foundation Helpdesk: <https://support.linuxfoundation.org/>
- Should you encounter any issues with the Linux Foundation Helpdesk, send a message to the backup e-mail support address at: login-issues@jira.linuxfoundation.org

<!-- need_sender_cla -->

<details>

%s
</details>
	`
	maxRetries = 5
)

var (
	checkCLARe = regexp.MustCompile(`(?mi)^/check-cla\s*$`)
)

func init() {
	plugins.RegisterStatusEventHandler(pluginName, handleStatusEvent, helpProvider)
	plugins.RegisterGenericCommentHandler(pluginName, handleCommentEvent, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	// The {WhoCanUse, Usage, Examples, Config} fields are omitted because this plugin cannot be
	// manually triggered and is not configurable.
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The cla plugin manages the application and removal of the 'cncf-cla' prefixed labels on pull requests as a reaction to the " + claContextName + " github status context. It is also responsible for warning unauthorized PR authors that they need to sign the CNCF CLA before their PR will be merged.",
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/check-cla",
		Description: "Forces rechecking of the CLA status.",
		Featured:    true,
		WhoCanUse:   "Anyone",
		Examples:    []string{"/check-cla"},
	})
	return pluginHelp, nil
}

type gitHubClient interface {
	CreateComment(owner, repo string, number int, comment string) error
	AddLabel(owner, repo string, number int, label string) error
	RemoveLabel(owner, repo string, number int, label string) error
	GetPullRequest(owner, repo string, number int) (*github.PullRequest, error)
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error)
}

func handleStatusEvent(pc plugins.Agent, se github.StatusEvent) error {
	return handle(pc.GitHubClient, pc.Logger, se)
}

// 1. Check that the status event received from the webhook is for the CNCF-CLA.
// 2. Use the github search API to search for the PRs which match the commit hash corresponding to the status event.
// 3. For each issue that matches, check that the PR's HEAD commit hash against the commit hash for which the status
//    was received. This is because we only care about the status associated with the last (latest) commit in a PR.
// 4. Set the corresponding CLA label if needed.
func handle(gc gitHubClient, log *logrus.Entry, se github.StatusEvent) error {
	if se.State == "" || se.Context == "" {
		return fmt.Errorf("invalid status event delivered with empty state/context")
	}

	if se.Context != claContextName {
		// Not the CNCF CLA context, do not process this.
		return nil
	}

	if se.State == github.StatusPending {
		// do nothing and wait for state to be updated.
		return nil
	}

	org := se.Repo.Owner.Login
	repo := se.Repo.Name
	log.Info("Searching for PRs matching the commit.")

	var issues []github.Issue
	var err error
	for i := 0; i < maxRetries; i++ {
		issues, err = gc.FindIssues(fmt.Sprintf("%s repo:%s/%s type:pr state:open", se.SHA, org, repo), "", false)
		if err != nil {
			return fmt.Errorf("error searching for issues matching commit: %v", err)
		}
		if len(issues) > 0 {
			break
		}
		time.Sleep(10 * time.Second)
	}
	log.Infof("Found %d PRs matching commit.", len(issues))

	for _, issue := range issues {
		l := log.WithField("pr", issue.Number)
		hasCncfYes := issue.HasLabel(labels.ClaYes)
		hasCncfNo := issue.HasLabel(labels.ClaNo)
		if hasCncfYes && se.State == github.StatusSuccess {
			// Nothing to update.
			l.Infof("PR has up-to-date %s label.", labels.ClaYes)
			continue
		}

		if hasCncfNo && (se.State == github.StatusFailure || se.State == github.StatusError) {
			// Nothing to update.
			l.Infof("PR has up-to-date %s label.", labels.ClaNo)
			continue
		}

		l.Info("PR labels may be out of date. Getting pull request info.")
		pr, err := gc.GetPullRequest(org, repo, issue.Number)
		if err != nil {
			l.WithError(err).Warningf("Unable to fetch PR-%d from %s/%s.", issue.Number, org, repo)
			continue
		}

		// Check if this is the latest commit in the PR.
		if pr.Head.SHA != se.SHA {
			l.Info("Event is not for PR HEAD, skipping.")
			continue
		}

		number := pr.Number
		if se.State == github.StatusSuccess {
			if hasCncfNo {
				if err := gc.RemoveLabel(org, repo, number, labels.ClaNo); err != nil {
					l.WithError(err).Warningf("Could not remove %s label.", labels.ClaNo)
				}
			}
			if err := gc.AddLabel(org, repo, number, labels.ClaYes); err != nil {
				l.WithError(err).Warningf("Could not add %s label.", labels.ClaYes)
			}
			continue
		}

		// If we end up here, the status is a failure/error.
		if hasCncfYes {
			if err := gc.RemoveLabel(org, repo, number, labels.ClaYes); err != nil {
				l.WithError(err).Warningf("Could not remove %s label.", labels.ClaYes)
			}
		}
		if err := gc.CreateComment(org, repo, number, fmt.Sprintf(cncfclaNotFoundMessage, plugins.AboutThisBot)); err != nil {
			l.WithError(err).Warning("Could not create CLA not found comment.")
		}
		if err := gc.AddLabel(org, repo, number, labels.ClaNo); err != nil {
			l.WithError(err).Warningf("Could not add %s label.", labels.ClaNo)
		}
	}
	return nil
}

func handleCommentEvent(pc plugins.Agent, ce github.GenericCommentEvent) error {
	return handleComment(pc.GitHubClient, pc.Logger, &ce)
}

func handleComment(gc gitHubClient, log *logrus.Entry, e *github.GenericCommentEvent) error {
	// Only consider open PRs and new comments.
	if e.IssueState != "open" || e.Action != github.GenericCommentActionCreated {
		return nil
	}
	// Only consider "/check-cla" comments.
	if !checkCLARe.MatchString(e.Body) {
		return nil
	}

	org := e.Repo.Owner.Login
	repo := e.Repo.Name
	number := e.Number
	hasCLAYes := false
	hasCLANo := false

	// Check for existing cla labels.
	issueLabels, err := gc.GetIssueLabels(org, repo, number)
	if err != nil {
		log.WithError(err).Errorf("Failed to get the labels on %s/%s#%d.", org, repo, number)
	}
	for _, candidate := range issueLabels {
		if candidate.Name == labels.ClaYes {
			hasCLAYes = true
		}
		// Could theoretically have both yes/no labels.
		if candidate.Name == labels.ClaNo {
			hasCLANo = true
		}
	}

	pr, err := gc.GetPullRequest(org, repo, e.Number)
	if err != nil {
		log.WithError(err).Errorf("Unable to fetch PR-%d from %s/%s.", e.Number, org, repo)
	}

	// Check for the cla in past commit statuses, and add/remove corresponding cla label if necessary.
	ref := pr.Head.SHA
	combined, err := gc.GetCombinedStatus(org, repo, ref)
	if err != nil {
		log.WithError(err).Errorf("Failed to get statuses on %s/%s#%d", org, repo, number)
	}

	for _, status := range combined.Statuses {

		// Only consider "cla/linuxfoundation" status.
		if status.Context == claContextName {

			// Success state implies that the cla exists, so label should be cncf-cla:yes.
			if status.State == github.StatusSuccess {

				// Remove cncf-cla:no (if label exists).
				if hasCLANo {
					if err := gc.RemoveLabel(org, repo, number, labels.ClaNo); err != nil {
						log.WithError(err).Warningf("Could not remove %s label.", labels.ClaNo)
					}
				}

				// Add cncf-cla:yes (if label doesn't exist).
				if !hasCLAYes {
					if err := gc.AddLabel(org, repo, number, labels.ClaYes); err != nil {
						log.WithError(err).Warningf("Could not add %s label.", labels.ClaYes)
					}
				}

				// Failure state implies that the cla does not exist, so label should be cncf-cla:no.
			} else if status.State == github.StatusFailure {

				// Remove cncf-cla:yes (if label exists).
				if hasCLAYes {
					if err := gc.RemoveLabel(org, repo, number, labels.ClaYes); err != nil {
						log.WithError(err).Warningf("Could not remove %s label.", labels.ClaYes)
					}
				}

				// Add cncf-cla:no (if label doesn't exist).
				if !hasCLANo {
					if err := gc.AddLabel(org, repo, number, labels.ClaNo); err != nil {
						log.WithError(err).Warningf("Could not add %s label.", labels.ClaNo)
					}
				}
			}

			// No need to consider other contexts once you find the one you need.
			break
		}
	}
	return nil
}
