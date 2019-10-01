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

// Package dco implements a DCO (https://developercertificate.org/) checker plugin
package dco

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
	"k8s.io/test-infra/prow/plugins/trigger"
)

const (
	pluginName               = "dco"
	dcoContextName           = "dco"
	dcoContextMessageFailed  = "Commits in PR missing Signed-off-by"
	dcoContextMessageSuccess = "All commits have Signed-off-by"

	dcoYesLabel        = "dco-signoff: yes"
	dcoNoLabel         = "dco-signoff: no"
	dcoMsgPruneMatch   = "Thanks for your pull request. Before we can look at it, you'll need to add a 'DCO signoff' to your commits."
	dcoNotFoundMessage = `Thanks for your pull request. Before we can look at it, you'll need to add a 'DCO signoff' to your commits.

:memo: **Please follow instructions in the [contributing guide](%s) to update your commits with the DCO**

Full details of the Developer Certificate of Origin can be found at [developercertificate.org](https://developercertificate.org/).

**The list of commits missing DCO signoff**:

%s

<details>

%s
</details>
`
)

var (
	checkDCORe = regexp.MustCompile(`(?mi)^/check-dco\s*$`)
	testRe     = regexp.MustCompile(`(?mi)^signed-off-by:`)
)

func init() {
	plugins.RegisterPullRequestHandler(pluginName, handlePullRequestEvent, helpProvider)
	plugins.RegisterGenericCommentHandler(pluginName, handleCommentEvent, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	configInfo := map[string]string{}
	for _, orgRepo := range enabledRepos {
		parts := strings.Split(orgRepo, "/")
		var opts *plugins.Dco
		switch len(parts) {
		case 1:
			opts = config.DcoFor(parts[0], "")
		case 2:
			opts = config.DcoFor(parts[0], parts[1])
		default:
			return nil, fmt.Errorf("invalid repo in enabledRepos: %q", orgRepo)
		}

		if opts.SkipDCOCheckForMembers || opts.SkipDCOCheckForCollaborators {
			configInfo[orgRepo] = fmt.Sprintf("The trusted GitHub organization for this repository is %q.", orgRepo)
		}
	}

	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The dco plugin checks pull request commits for 'DCO sign off' and maintains the '" + dcoContextName + "' status context, as well as the 'dco' label.",
		Config:      configInfo,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/check-dco",
		Description: "Forces rechecking of the DCO status.",
		Featured:    true,
		WhoCanUse:   "Anyone",
		Examples:    []string{"/check-dco"},
	})
	return pluginHelp, nil
}

type gitHubClient interface {
	BotName() (string, error)
	IsMember(org, user string) (bool, error)
	IsCollaborator(org, repo, user string) (bool, error)
	CreateComment(owner, repo string, number int, comment string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	AddLabel(owner, repo string, number int, label string) error
	RemoveLabel(owner, repo string, number int, label string) error
	CreateStatus(owner, repo, ref string, status github.Status) error
	ListPRCommits(org, repo string, number int) ([]github.RepositoryCommit, error)
	GetPullRequest(owner, repo string, number int) (*github.PullRequest, error)
	GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error)
}

type commentPruner interface {
	PruneComments(shouldPrune func(github.IssueComment) bool)
}

// checkTrustedUser checks are all commits from a trusted user
func checkTrustedUser(gc gitHubClient, l *logrus.Entry, skipDCOCheckForCollaborators bool, trustedOrg, org, repo string, number int) (bool, error) {
	allCommits, err := gc.ListPRCommits(org, repo, number)
	if err != nil {
		return false, fmt.Errorf("error listing commits for pull request: %v", err)
	}

	for _, commit := range allCommits {
		trusted, err := trigger.TrustedUser(gc, !skipDCOCheckForCollaborators, trustedOrg, commit.Author.Login, org, repo)
		if err != nil {
			return false, fmt.Errorf("Error checking is member trusted: %v", err)
		}
		if !trusted {
			l.Debugf("Member %s is not trusted", commit.Author.Login)
			return false, nil
		}
	}

	return true, nil
}

// checkCommitMessages will perform the actual DCO check by retrieving all
// commits contained within the PR with the given number.
// *All* commits in the pull request *must* match the 'testRe' in order to pass.
func checkCommitMessages(gc gitHubClient, l *logrus.Entry, org, repo string, number int) ([]github.GitCommit, error) {
	allCommits, err := gc.ListPRCommits(org, repo, number)
	if err != nil {
		return nil, fmt.Errorf("error listing commits for pull request: %v", err)
	}
	l.Debugf("Found %d commits in PR", len(allCommits))

	var commitsMissingDCO []github.GitCommit
	for _, commit := range allCommits {
		if !testRe.MatchString(commit.Commit.Message) {
			c := commit.Commit
			c.SHA = commit.SHA
			commitsMissingDCO = append(commitsMissingDCO, c)
		}
	}

	l.Debugf("All commits in PR have DCO signoff: %t", len(commitsMissingDCO) == 0)
	return commitsMissingDCO, nil
}

// checkExistingStatus will retrieve the current status of the DCO context for
// the provided SHA.
func checkExistingStatus(gc gitHubClient, l *logrus.Entry, org, repo, sha string) (string, error) {
	combinedStatus, err := gc.GetCombinedStatus(org, repo, sha)
	if err != nil {
		return "", fmt.Errorf("error listing pull request combined statuses: %v", err)
	}

	existingStatus := ""
	for _, status := range combinedStatus.Statuses {
		if status.Context != dcoContextName {
			continue
		}
		existingStatus = status.State
		break
	}
	l.Debugf("Existing DCO status context status is %q", existingStatus)
	return existingStatus, nil
}

// checkExistingLabels will check the provided PR for the dco sign off labels,
// returning bool's indicating whether the 'yes' and the 'no' label are present.
func checkExistingLabels(gc gitHubClient, l *logrus.Entry, org, repo string, number int) (hasYesLabel, hasNoLabel bool, err error) {
	labels, err := gc.GetIssueLabels(org, repo, number)
	if err != nil {
		return false, false, fmt.Errorf("error getting pull request labels: %v", err)
	}

	for _, l := range labels {
		if l.Name == dcoYesLabel {
			hasYesLabel = true
		}
		if l.Name == dcoNoLabel {
			hasNoLabel = true
		}
	}

	return hasYesLabel, hasNoLabel, nil
}

// takeAction will take appropriate action on the pull request according to its
// current state.
func takeAction(gc gitHubClient, cp commentPruner, l *logrus.Entry, org, repo string, pr github.PullRequest, commitsMissingDCO []github.GitCommit, existingStatus string, hasYesLabel, hasNoLabel, addComment, trustedUser bool) error {
	targetURL := fmt.Sprintf("https://github.com/%s/%s/blob/master/CONTRIBUTING.md", org, repo)

	signedOff := len(commitsMissingDCO) == 0

	// handle the 'all commits signed off' case by adding appropriate labels
	// TODO: clean-up old comments?
	if signedOff || trustedUser {
		if hasNoLabel {
			l.Debugf("Removing %q label", dcoNoLabel)
			// remove 'dco-signoff: no' label
			if err := gc.RemoveLabel(org, repo, pr.Number, dcoNoLabel); err != nil {
				return fmt.Errorf("error removing label: %v", err)
			}
		}
		if !hasYesLabel {
			l.Debugf("Adding %q label", dcoYesLabel)
			// add 'dco-signoff: yes' label
			if err := gc.AddLabel(org, repo, pr.Number, dcoYesLabel); err != nil {
				return fmt.Errorf("error adding label: %v", err)
			}
		}
		if existingStatus != github.StatusSuccess {
			l.Debugf("Setting DCO status context to succeeded")
			if err := gc.CreateStatus(org, repo, pr.Head.SHA, github.Status{
				Context:     dcoContextName,
				State:       github.StatusSuccess,
				TargetURL:   targetURL,
				Description: dcoContextMessageSuccess,
			}); err != nil {
				return fmt.Errorf("error setting pull request status: %v", err)
			}
		}

		cp.PruneComments(shouldPrune(l))
		return nil
	}

	// handle the 'not all commits signed off' case
	if !hasNoLabel {
		l.Debugf("Adding %q label", dcoNoLabel)
		// add 'dco-signoff: no' label
		if err := gc.AddLabel(org, repo, pr.Number, dcoNoLabel); err != nil {
			return fmt.Errorf("error adding label: %v", err)
		}
	}
	if hasYesLabel {
		l.Debugf("Removing %q label", dcoYesLabel)
		// remove 'dco-signoff: yes' label
		if err := gc.RemoveLabel(org, repo, pr.Number, dcoYesLabel); err != nil {
			return fmt.Errorf("error removing label: %v", err)
		}
	}
	if existingStatus != github.StatusFailure {
		l.Debugf("Setting DCO status context to failed")
		if err := gc.CreateStatus(org, repo, pr.Head.SHA, github.Status{
			Context:     dcoContextName,
			State:       github.StatusFailure,
			TargetURL:   targetURL,
			Description: dcoContextMessageFailed,
		}); err != nil {
			return fmt.Errorf("error setting pull request status: %v", err)
		}
	}

	if addComment {
		// prune any old comments and add a new one with the latest list of
		// failing commits
		cp.PruneComments(shouldPrune(l))
		l.Debugf("Commenting on PR to advise users of DCO check")
		if err := gc.CreateComment(org, repo, pr.Number, fmt.Sprintf(dcoNotFoundMessage, targetURL, MarkdownSHAList(org, repo, commitsMissingDCO), plugins.AboutThisBot)); err != nil {
			l.WithError(err).Warning("Could not create DCO not found comment.")
		}
	}

	return nil
}

// 1. Check should commit messages from trusted users be checked
// 2. Check commit messages in the pull request for the sign-off string
// 3. Check the existing status context value
// 4. Check the existing PR labels
// 5. If signed off, apply appropriate labels and status context.
// 6. If not signed off, apply appropriate labels and status context and add a comment.
func handle(config plugins.Dco, gc gitHubClient, cp commentPruner, log *logrus.Entry, org, repo string, pr github.PullRequest, addComment bool) error {
	l := log.WithField("pr", pr.Number)

	var err error
	var trustedUser bool
	if config.SkipDCOCheckForMembers || config.SkipDCOCheckForCollaborators {
		trustedUser, err = checkTrustedUser(gc, l, config.SkipDCOCheckForCollaborators, config.TrustedOrg, org, repo, pr.Number)
		if err != nil {
			l.WithError(err).Infof("Error running trusted org member check against commits in PR")
			return err
		}
	}

	commitsMissingDCO, err := checkCommitMessages(gc, l, org, repo, pr.Number)
	if err != nil {
		l.WithError(err).Infof("Error running DCO check against commits in PR")
		return err
	}

	existingStatus, err := checkExistingStatus(gc, l, org, repo, pr.Head.SHA)
	if err != nil {
		l.WithError(err).Infof("Error checking existing PR status")
		return err
	}

	hasYesLabel, hasNoLabel, err := checkExistingLabels(gc, l, org, repo, pr.Number)
	if err != nil {
		l.WithError(err).Infof("Error checking existing PR labels")
		return err
	}

	return takeAction(gc, cp, l, org, repo, pr, commitsMissingDCO, existingStatus, hasYesLabel, hasNoLabel, addComment, trustedUser)
}

// MardkownSHAList prints the list of commits in a markdown-friendly way.
func MarkdownSHAList(org, repo string, list []github.GitCommit) string {
	lines := make([]string, len(list))
	lineFmt := "- [%s](https://github.com/%s/%s/commits/%s) %s"
	for i, commit := range list {
		if commit.SHA == "" {
			continue
		}
		// if we somehow encounter a SHA that's less than 7 characters, we will
		// just use it as is.
		shortSHA := commit.SHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}

		// get the first line of the commit
		message := strings.Split(commit.Message, "\n")[0]

		lines[i] = fmt.Sprintf(lineFmt, shortSHA, org, repo, commit.SHA, message)
	}
	return strings.Join(lines, "\n")
}

// shouldPrune finds comments left by this plugin.
func shouldPrune(log *logrus.Entry) func(github.IssueComment) bool {
	return func(comment github.IssueComment) bool {
		return strings.Contains(comment.Body, dcoMsgPruneMatch)
	}
}

func handlePullRequestEvent(pc plugins.Agent, pe github.PullRequestEvent) error {
	config := pc.PluginConfig.DcoFor(pe.Repo.Owner.Login, pe.Repo.Name)

	cp, err := pc.CommentPruner()
	if err != nil {
		return err
	}

	return handlePullRequest(*config, pc.GitHubClient, cp, pc.Logger, pe)
}

func handlePullRequest(config plugins.Dco, gc gitHubClient, cp commentPruner, log *logrus.Entry, pe github.PullRequestEvent) error {
	org := pe.Repo.Owner.Login
	repo := pe.Repo.Name

	// we only reprocess on label, unlabel, open, reopen and synchronize events
	// this will reduce our API token usage and save processing of unrelated events
	switch pe.Action {
	case github.PullRequestActionOpened,
		github.PullRequestActionReopened,
		github.PullRequestActionSynchronize:
	default:
		return nil
	}

	shouldComment := pe.Action == github.PullRequestActionSynchronize ||
		pe.Action == github.PullRequestActionOpened

	return handle(config, gc, cp, log, org, repo, pe.PullRequest, shouldComment)
}

func handleCommentEvent(pc plugins.Agent, ce github.GenericCommentEvent) error {
	config := pc.PluginConfig.DcoFor(ce.Repo.Owner.Login, ce.Repo.Name)

	cp, err := pc.CommentPruner()
	if err != nil {
		return err
	}

	return handleComment(*config, pc.GitHubClient, cp, pc.Logger, ce)
}

func handleComment(config plugins.Dco, gc gitHubClient, cp commentPruner, log *logrus.Entry, ce github.GenericCommentEvent) error {
	// Only consider open PRs and new comments.
	if ce.IssueState != "open" || ce.Action != github.GenericCommentActionCreated || !ce.IsPR {
		return nil
	}
	// Only consider "/check-dco" comments.
	if !checkDCORe.MatchString(ce.Body) {
		return nil
	}

	org := ce.Repo.Owner.Login
	repo := ce.Repo.Name

	pr, err := gc.GetPullRequest(org, repo, ce.Number)
	if err != nil {
		return fmt.Errorf("error getting pull request for comment: %v", err)
	}

	return handle(config, gc, cp, log, org, repo, *pr, true)
}
