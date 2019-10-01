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

package main

import (
	"errors"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/pjutil"

	v1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
	prowjobinformer "k8s.io/test-infra/prow/client/informers/externalversions"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/crier"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	gerritclient "k8s.io/test-infra/prow/gerrit/client"
	gerritreporter "k8s.io/test-infra/prow/gerrit/reporter"
	githubreporter "k8s.io/test-infra/prow/github/reporter"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/logrusutil"
	pubsubreporter "k8s.io/test-infra/prow/pubsub/reporter"
	slackreporter "k8s.io/test-infra/prow/slack/reporter"
)

const (
	resync         = 0 * time.Minute
	controllerName = "prow-crier"
)

type options struct {
	client         prowflagutil.ExperimentalKubernetesOptions
	cookiefilePath string
	gerritProjects gerritclient.ProjectsFlag
	github         prowflagutil.GitHubOptions

	configPath    string
	jobConfigPath string

	gerritWorkers int
	pubsubWorkers int
	githubWorkers int
	slackWorkers  int

	slackTokenFile string

	dryrun      bool
	reportAgent string
}

func (o *options) validate() error {
	if o.configPath == "" {
		return errors.New("required flag --config-path was unset")
	}

	// TODO(krzyzacy): gerrit && github report are actually stateful..
	// Need a better design to re-enable parallel reporting
	if o.gerritWorkers > 1 {
		logrus.Warn("gerrit reporter only supports one worker")
		o.gerritWorkers = 1
	}

	if o.githubWorkers > 1 {
		logrus.Warn("github reporter only supports one worker (https://github.com/kubernetes/test-infra/issues/13306)")
		o.githubWorkers = 1
	}

	if o.gerritWorkers+o.pubsubWorkers+o.githubWorkers+o.slackWorkers <= 0 {
		return errors.New("crier need to have at least one report worker to start")
	}

	if o.gerritWorkers > 0 {
		if len(o.gerritProjects) == 0 {
			return errors.New("--gerrit-projects must be set")
		}

		if o.cookiefilePath == "" {
			logrus.Info("--cookiefile is not set, using anonymous authentication")
		}
	}

	if o.githubWorkers > 0 {
		if err := o.github.Validate(o.dryrun); err != nil {
			return err
		}
	}

	if o.slackWorkers > 0 {
		if o.slackTokenFile == "" {
			return errors.New("--slack-token-file must be set")
		}
	}

	if err := o.client.Validate(o.dryrun); err != nil {
		return err
	}

	return nil
}

func (o *options) parseArgs(fs *flag.FlagSet, args []string) error {

	o.gerritProjects = gerritclient.ProjectsFlag{}

	fs.StringVar(&o.cookiefilePath, "cookiefile", "", "Path to git http.cookiefile, leave empty for anonymous")
	fs.Var(&o.gerritProjects, "gerrit-projects", "Set of gerrit repos to monitor on a host example: --gerrit-host=https://android.googlesource.com=platform/build,toolchain/llvm, repeat flag for each host")
	fs.IntVar(&o.gerritWorkers, "gerrit-workers", 0, "Number of gerrit report workers (0 means disabled)")
	fs.IntVar(&o.pubsubWorkers, "pubsub-workers", 0, "Number of pubsub report workers (0 means disabled)")
	fs.IntVar(&o.githubWorkers, "github-workers", 0, "Number of github report workers (0 means disabled)")
	fs.IntVar(&o.slackWorkers, "slack-workers", 0, "Number of Slack report workers (0 means disabled)")
	fs.StringVar(&o.slackTokenFile, "slack-token-file", "", "Path to a Slack token file")
	fs.StringVar(&o.reportAgent, "report-agent", "", "Only report specified agent - empty means report to all agents (effective for github and Slack only)")

	fs.StringVar(&o.configPath, "config-path", "", "Path to config.yaml.")
	fs.StringVar(&o.jobConfigPath, "job-config-path", "", "Path to prow job configs.")

	// TODO(krzyzacy): implement dryrun for gerrit/pubsub
	fs.BoolVar(&o.dryrun, "dry-run", false, "Run in dry-run mode, not doing actual report (effective for github and Slack only)")

	o.github.AddFlags(fs)
	o.client.AddFlags(fs)

	fs.Parse(args)

	return o.validate()
}

func parseOptions() options {
	var o options

	if err := o.parseArgs(flag.CommandLine, os.Args[1:]); err != nil {
		logrus.WithError(err).Fatal("Invalid flag options")
	}

	return o
}

func main() {
	logrusutil.ComponentInit("crier")

	o := parseOptions()

	pjutil.ServePProf()

	configAgent := &config.Agent{}
	if err := configAgent.Start(o.configPath, o.jobConfigPath); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}
	cfg := configAgent.Config

	prowjobClientset, err := o.client.ProwJobClientset(cfg().ProwJobNamespace, o.dryrun)
	if err != nil {
		logrus.WithError(err).Fatal("unable to create prow job client")
	}

	prowjobInformerFactory := prowjobinformer.NewSharedInformerFactory(prowjobClientset, resync)

	var controllers []*crier.Controller

	// track all worker status before shutdown
	wg := &sync.WaitGroup{}

	if o.slackWorkers > 0 {
		if cfg().SlackReporter == nil {
			logrus.Fatal("slackreporter is enabled but has no config")
		}
		slackConfig := func() *config.SlackReporter {
			return cfg().SlackReporter
		}
		slackReporter, err := slackreporter.New(slackConfig, o.dryrun, o.slackTokenFile)
		if err != nil {
			logrus.WithError(err).Fatal("failed to create slackreporter")
		}
		controllers = append(
			controllers,
			crier.NewController(
				prowjobClientset,
				kube.RateLimiter(slackReporter.GetName()),
				prowjobInformerFactory.Prow().V1().ProwJobs(),
				slackReporter,
				o.slackWorkers,
				wg))
	}

	if o.gerritWorkers > 0 {
		informer := prowjobInformerFactory.Prow().V1().ProwJobs()
		gerritReporter, err := gerritreporter.NewReporter(o.cookiefilePath, o.gerritProjects, informer.Lister())
		if err != nil {
			logrus.WithError(err).Fatal("Error starting gerrit reporter")
		}

		controllers = append(
			controllers,
			crier.NewController(
				prowjobClientset,
				kube.RateLimiter(gerritReporter.GetName()),
				informer,
				gerritReporter,
				o.gerritWorkers,
				wg))
	}

	if o.pubsubWorkers > 0 {
		pubsubReporter := pubsubreporter.NewReporter(cfg)
		controllers = append(
			controllers,
			crier.NewController(
				prowjobClientset,
				kube.RateLimiter(pubsubReporter.GetName()),
				prowjobInformerFactory.Prow().V1().ProwJobs(),
				pubsubReporter,
				o.pubsubWorkers,
				wg))
	}

	if o.githubWorkers > 0 {
		secretAgent := &secret.Agent{}
		if o.github.TokenPath != "" {
			if err := secretAgent.Start([]string{o.github.TokenPath}); err != nil {
				logrus.WithError(err).Fatal("Error starting secrets agent")
			}
		}

		githubClient, err := o.github.GitHubClient(secretAgent, o.dryrun)
		if err != nil {
			logrus.WithError(err).Fatal("Error getting GitHub client.")
		}

		githubReporter := githubreporter.NewReporter(githubClient, cfg, v1.ProwJobAgent(o.reportAgent))
		controllers = append(
			controllers,
			crier.NewController(
				prowjobClientset,
				kube.RateLimiter(githubReporter.GetName()),
				prowjobInformerFactory.Prow().V1().ProwJobs(),
				githubReporter,
				o.githubWorkers,
				wg))
	}

	if len(controllers) == 0 {
		logrus.Fatalf("should have at least one controller to start crier.")
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	// run the controller loop to process items
	prowjobInformerFactory.Start(stopCh)
	for _, controller := range controllers {
		go controller.Run(stopCh)
	}

	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	signal.Notify(sigTerm, syscall.SIGINT)

	<-sigTerm
	logrus.Info("Crier received a termination signal and is shutting down...")
	for range controllers {
		stopCh <- struct{}{}
	}

	// waiting for all crier worker to finish
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		logrus.Info("All worker finished, exiting crier")
	case <-time.After(10 * time.Second):
		logrus.Info("timed out waiting for all worker to finish")
	}
}
