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

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/cron"
	"k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/logrusutil"
	"k8s.io/test-infra/prow/pjutil"
)

type options struct {
	configPath    string
	jobConfigPath string

	kubernetes flagutil.ExperimentalKubernetesOptions
	dryRun     flagutil.Bool
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options
	fs.StringVar(&o.configPath, "config-path", "", "Path to config.yaml.")
	fs.StringVar(&o.jobConfigPath, "job-config-path", "", "Path to prow job configs.")

	// TODO(fejta): switch dryRun to be a bool, defaulting to true after March 15, 2019.
	fs.Var(&o.dryRun, "dry-run", "Whether or not to make mutating API calls to Kubernetes.")
	o.kubernetes.AddFlags(fs)

	fs.Parse(args)
	o.configPath = config.ConfigPath(o.configPath)
	return o
}

func (o *options) Validate() error {
	if err := o.kubernetes.Validate(o.dryRun.Value); err != nil {
		return err
	}

	if o.configPath == "" {
		return errors.New("--config-path is required")
	}

	return nil
}

func main() {
	logrusutil.ComponentInit("horologium")

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	pjutil.ServePProf()

	if !o.dryRun.Explicit {
		logrus.Warning("Horologium requires --dry-run=false to function correctly in production.")
		logrus.Warning("--dry-run will soon default to true. Set --dry-run=false by March 15.")
	}

	configAgent := config.Agent{}
	if err := configAgent.Start(o.configPath, o.jobConfigPath); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}

	prowJobClient, err := o.kubernetes.ProwJobClient(configAgent.Config().ProwJobNamespace, o.dryRun.Value)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting Kubernetes client.")
	}

	// start a cron
	cr := cron.New()
	cr.Start()

	for now := range time.Tick(1 * time.Minute) {
		start := time.Now()
		if err := sync(prowJobClient, configAgent.Config(), cr, now); err != nil {
			logrus.WithError(err).Error("Error syncing periodic jobs.")
		}
		logrus.Infof("Sync time: %v", time.Since(start))
	}
}

type prowJobClient interface {
	Create(*prowapi.ProwJob) (*prowapi.ProwJob, error)
	List(opts metav1.ListOptions) (*prowapi.ProwJobList, error)
}

type cronClient interface {
	SyncConfig(cfg *config.Config) error
	QueuedJobs() []string
}

func sync(prowJobClient prowJobClient, cfg *config.Config, cr cronClient, now time.Time) error {
	jobs, err := prowJobClient.List(metav1.ListOptions{LabelSelector: labels.Everything().String()})
	if err != nil {
		return fmt.Errorf("error listing prow jobs: %v", err)
	}
	latestJobs := pjutil.GetLatestProwJobs(jobs.Items, prowapi.PeriodicJob)

	if err := cr.SyncConfig(cfg); err != nil {
		logrus.WithError(err).Error("Error syncing cron jobs.")
	}

	cronTriggers := sets.NewString()
	for _, job := range cr.QueuedJobs() {
		cronTriggers.Insert(job)
	}

	var errs []error
	for _, p := range cfg.Periodics {
		j, previousFound := latestJobs[p.Name]
		logger := logrus.WithFields(logrus.Fields{
			"job":            p.Name,
			"previous-found": previousFound,
		})

		if p.Cron == "" {
			shouldTrigger := j.Complete() && now.Sub(j.Status.StartTime.Time) > p.GetInterval()
			logger = logger.WithField("should-trigger", shouldTrigger)
			if !previousFound || shouldTrigger {
				prowJob := pjutil.NewProwJob(pjutil.PeriodicSpec(p), p.Labels, p.Annotations)
				logger.WithFields(pjutil.ProwJobFields(&prowJob)).Info("Triggering new run of interval periodic.")
				if _, err := prowJobClient.Create(&prowJob); err != nil {
					errs = append(errs, err)
				}
			}
		} else if cronTriggers.Has(p.Name) {
			shouldTrigger := j.Complete()
			logger = logger.WithField("should-trigger", shouldTrigger)
			if !previousFound || shouldTrigger {
				prowJob := pjutil.NewProwJob(pjutil.PeriodicSpec(p), p.Labels, p.Annotations)
				logger.WithFields(pjutil.ProwJobFields(&prowJob)).Info("Triggering new run of cron periodic.")
				if _, err := prowJobClient.Create(&prowJob); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to create %d prowjobs: %v", len(errs), errs)
	}

	return nil
}
