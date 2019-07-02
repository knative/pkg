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

package alert

import (
	"context"
	"log"
	"strings"

	"github.com/knative/test-infra/shared/gcs"
	"github.com/knative/test-infra/tools/monitoring/config"
	"github.com/knative/test-infra/tools/monitoring/log_parser"
	"github.com/knative/test-infra/tools/monitoring/mail"
	"github.com/knative/test-infra/tools/monitoring/mysql"
	"github.com/knative/test-infra/tools/monitoring/prowapi"
	"github.com/knative/test-infra/tools/monitoring/subscriber"
)

const YamlURL = "https://raw.githubusercontent.com/knative/test-infra/master/tools/monitoring/config/config.yaml"

// Client holds all the resources required to run alerting
type Client struct {
	*subscriber.Client
	*MailConfig
	db *mysql.DB
}

type MailConfig struct {
	*mail.Config
	Recipients []string
}

// Setup sets up the client required to run alerting workflow
func Setup(psClient *subscriber.Client, db *mysql.DB, mc *MailConfig) *Client {
	return &Client{psClient, mc, db}
}

// RunAlerting start the alerting workflow
func (c *Client) RunAlerting() {
	log.Println("Starting alerting workflow")
	go func() {
		err := c.ReceiveMessageAckAll(context.Background(), c.handleReportMessage)
		if err != nil {
			log.Printf("Failed to retrieve messages due to %v", err)
		}
	}()
}

func (c *Client) handleReportMessage(rmsg *prowapi.ReportMessage) {
	if rmsg.Status == prowapi.SuccessState || rmsg.Status == prowapi.FailureState || rmsg.Status == prowapi.AbortedState {
		config, err := config.ParseYaml(YamlURL)
		if err != nil {
			log.Printf("Failed to config yaml (%v): %v\n", config, err)
			return
		}

		rmsg.GCSPath = toGcsLink(rmsg.GCSPath)
		blPath, err := gcs.BuildLogPath(rmsg.GCSPath)
		if err != nil {
			log.Printf("Failed to construct build log url from gcs path %s. Error: %v\n", rmsg.GCSPath, err)
			return
		}
		buildLog, err := gcs.ReadURL(context.Background(), blPath)
		if err != nil {
			log.Printf("Failed to read from url %s. Error: %v\n", blPath, err)
			return
		}
		log.Printf("Build Log Content: %s\n", buildLog)

		errorLogs, err := log_parser.ParseLog(buildLog, config.CollectErrorPatterns())
		if err != nil {
			log.Printf("Failed to parse content %v. Error: %v\n", string(buildLog), err)
			return
		}

		log.Printf("Parsed errorLogs: %v\n", errorLogs)

		for _, el := range errorLogs {
			c.handleSingleError(config, rmsg, &el)
		}
	}
}

func (c *Client) handleSingleError(config *config.Config, rmsg *prowapi.ReportMessage, el *mysql.ErrorLog) {
	log.Println("Handling single error")
	var err error

	// Add the PR number if it is a pull request job
	log.Println("Adding Error Log to the table")
	if len(rmsg.Refs) <= 0 || len(rmsg.Refs[0].Pulls) <= 0 {
		err = c.db.AddErrorLog(el.Pattern, el.Msg, rmsg.JobName, 0, rmsg.GCSPath)
	} else {
		err = c.db.AddErrorLog(el.Pattern, el.Msg, rmsg.JobName, rmsg.Refs[0].Pulls[0].Number, rmsg.GCSPath)
	}
	if err != nil {
		log.Printf("Failed to insert error to db %+v\n", err)
		return
	}

	log.Println("Selecting the config")
	sc, noMatchErr := config.Select(el.Pattern, rmsg.JobName)
	if noMatchErr != nil {
		log.Printf("No matching config found for pattern (%v) job(%v): %v", el.Pattern, rmsg.JobName, noMatchErr)
		return
	}

	log.Println("Sending the alert")
	_, err = c.Alert(el.Pattern, sc, c.db)
	if err != nil {
		log.Printf("Failed to Alert %v", err)
	}
}

// TODO(yt3liu): Remove this hack after the gcs path does not contain extra link prefix
func toGcsLink(link string) string {
	return strings.Replace(strings.Replace(link, "https://gubernator.knative.dev/build/", "", 1),
		"https://prow.knative.dev/view/gcs/", "", 1)
}

func (m *MailConfig) sendAlert(c *mailContent) error {
	log.Printf("sending alert...")
	return m.Send(m.Recipients, c.subject(), c.body())
}

// Alert checks alert condition and alerts table and send alert mail conditionally
func (m *MailConfig) Alert(errorPattern string, s *config.SelectedConfig, db *mysql.DB) (bool, error) {
	log.Println("Fetcing error logs")
	errorLogs, err := db.ListErrorLogs(errorPattern, s.Duration())
	if err != nil {
		return false, err
	}

	log.Println("Building Report and checking alert conditions")
	report := newReport(errorLogs)
	if !report.CheckAlertCondition(s) {
		return false, nil
	}

	log.Println("checking if the alert is a fresh alert pattern")
	ok, err := db.IsFreshAlertPattern(errorPattern, s.Duration())
	if err != nil || !ok {
		return false, err
	}

	log.Println("Adding the new alert pattern to the database")
	if err := db.AddAlert(errorPattern); err != nil {
		return false, err
	}

	log.Println("Generating and sending the alert email")
	content := mailContent{*report, errorPattern, s.Hint, s.Duration()}
	err = m.sendAlert(&content)
	return err == nil, err
}
