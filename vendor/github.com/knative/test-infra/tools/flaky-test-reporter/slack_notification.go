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

// slack_notification.go sends notifications to slack channels

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"net/http"
	"net/url"

	"github.com/knative/test-infra/shared/testgrid"
)

const (
	knativeBotName          = "Knative Testgrid Robot"
	slackChatPostMessageURL = "https://slack.com/api/chat.postMessage"
	// default filter for testgrid link
	testgridFilter = "exclude-non-failed-tests=20"
)

// SlackClient contains Slack bot related information
type SlackClient struct {
	userName  string
	tokenStr  string
	iconEmoji *string
}

// slackChannel contains channel logical name and Slack identity
type slackChannel struct {
	name, identity string
}

// newSlackClient reads token file and stores it for later authentication
func newSlackClient(slackTokenPath string) (*SlackClient, error) {
	b, err := ioutil.ReadFile(slackTokenPath)
	if err != nil {
		return nil, err
	}
	return &SlackClient{
		userName: knativeBotName,
		tokenStr: strings.TrimSpace(string(b)),
	}, nil
}

// postMessage does http post
func (c *SlackClient) postMessage(url string, uv *url.Values) error {
	resp, err := http.PostForm(url, *uv)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	t, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("http response code is not 200 '%s'", string(t))
	}
	// response code could also be 200 if channel doesn't exist, parse response body to find out
	var b struct {
		OK bool `json:"ok"`
	}
	if err = json.Unmarshal(t, &b); nil != err || !b.OK {
		return fmt.Errorf("response not ok '%s'", string(t))
	}
	return nil
}

// writeSlackMessage posts text to channel
func (c *SlackClient) writeSlackMessage(text, channel string) error {
	uv := &url.Values{}
	uv.Add("username", c.userName)
	uv.Add("token", c.tokenStr)
	if nil != c.iconEmoji {
		uv.Add("icon_emoji", *c.iconEmoji)
	}
	uv.Add("channel", channel)
	uv.Add("text", text)

	return c.postMessage(slackChatPostMessageURL, uv)
}

// createSlackMessageForRepo creates slack message layout from RepoData
func createSlackMessageForRepo(rd *RepoData, flakyIssuesMap map[string][]*flakyIssue) string {
	flakyTests := getFlakyTests(rd)
	message := fmt.Sprintf("As of %s, there are %d flaky tests in '%s' from repo '%s'",
		time.Unix(*rd.LastBuildStartTime, 0).String(), len(flakyTests), rd.Config.Name, rd.Config.Repo)
	if "" == rd.Config.IssueRepo {
		message += fmt.Sprintf("\n(Job is marked to not create GitHub issues)")
	}
	if flakyRateAboveThreshold(rd) { // Don't list each test as this can be huge
		flakyRate := getFlakyRate(rd)
		message += fmt.Sprintf("\n>- skip displaying all tests as flaky rate above threshold")
		if flakyIssues, ok := flakyIssuesMap[getBulkIssueIdentity(rd, flakyRate)]; ok && "" != rd.Config.IssueRepo {
			// When flaky rate is above threshold, there is only one issue created,
			// so there is only one element in flakyIssues
			for _, fi := range flakyIssues {
				message += fmt.Sprintf("\t%s", fi.issue.GetHTMLURL())
			}
		}
	} else {
		for _, testFullName := range flakyTests {
			message += fmt.Sprintf("\n>- %s", testFullName)
			if flakyIssues, ok := flakyIssuesMap[getIdentityForTest(testFullName, rd.Config.Repo)]; ok && "" != rd.Config.IssueRepo {
				for _, fi := range flakyIssues {
					message += fmt.Sprintf("\t%s", fi.issue.GetHTMLURL())
				}
			}
		}
	}

	if testgridTabURL, err := testgrid.GetTestgridTabURL(rd.Config.Name, []string{testgridFilter}); nil != err {
		log.Println(err) // don't fail as this could be optional
	} else {
		message += fmt.Sprintf("\nSee Testgrid for up-to-date flaky tests information: %s", testgridTabURL)
	}
	return message
}

func sendSlackNotifications(repoDataAll []*RepoData, c *SlackClient, ghi *GithubIssue, dryrun bool) error {
	var allErrs []error
	flakyIssuesMap, err := ghi.getFlakyIssues()
	if nil != err { // failure here will make message missing Github issues link, which should not prevent notification
		allErrs = append(allErrs, err)
		log.Println("Warning: cannot get flaky Github issues: ", err)
	}
	for _, rd := range repoDataAll {
		channels := rd.Config.SlackChannels
		if len(channels) == 0 {
			log.Printf("cannot find Slack channel for job '%s' in repo '%s', skipping Slack notification", rd.Config.Name, rd.Config.Repo)
			continue
		}
		ch := make(chan bool, len(channels))
		wg := sync.WaitGroup{}
		for _, channel := range channels {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				message := createSlackMessageForRepo(rd, flakyIssuesMap)
				if err := run(
					fmt.Sprintf("post Slack message for job '%s' from repo '%s' in channel '%s'", rd.Config.Name, rd.Config.Repo, channel.Name),
					func() error {
						return c.writeSlackMessage(message, channel.Identity)
					},
					dryrun); nil != err {
					allErrs = append(allErrs, err)
					log.Printf("failed sending notification to Slack channel '%s': '%v'", channel.Name, err)
				}
				if dryrun {
					log.Printf("[dry run] Slack message not sent. See it below:\n%s\n\n", message)
				}
				ch <- true
				wg.Done()
			}(&wg)
		}
		wg.Wait()
		close(ch)
	}
	return combineErrors(allErrs)
}
