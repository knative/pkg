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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/logrusutil"
	"k8s.io/test-infra/prow/metrics"
	"k8s.io/test-infra/prow/pjutil"
)

type options struct {
	runOnce       bool
	configPath    string
	jobConfigPath string
	dryRun        flagutil.Bool
	kubernetes    flagutil.ExperimentalKubernetesOptions
	namespace     string
}

const (
	reasonPodAged     = "aged"
	reasonPodOrphaned = "orphaned"

	reasonProwJobAged         = "aged"
	reasonProwJobAgedPeriodic = "aged-periodic"
)

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	o := options{}
	fs.BoolVar(&o.runOnce, "run-once", false, "If true, run only once then quit.")
	fs.StringVar(&o.configPath, "config-path", "", "Path to config.yaml.")
	fs.StringVar(&o.jobConfigPath, "job-config-path", "", "Path to prow job configs.")
	fs.StringVar(&o.namespace, "namespace", "default", "The namespae sinker runs in. Will be used for leader election")

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
	logrusutil.ComponentInit("sinker")

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	pjutil.ServePProf()

	if !o.dryRun.Explicit {
		logrus.Warning("Sinker requires --dry-run=false to function correctly in production.")
		logrus.Warning("--dry-run will soon default to true. Set --dry-run=false by March 15.")
	}

	configAgent := &config.Agent{}
	if err := configAgent.Start(o.configPath, o.jobConfigPath); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}
	cfg := configAgent.Config

	metrics.ExposeMetrics("sinker", cfg().PushGateway)

	// Enabling debug logging has the unfortunate side-effect of making the log
	// unstructured
	// https://github.com/kubernetes-sigs/controller-runtime/issues/442
	ctrlruntimelog.SetLogger(ctrlruntimelog.ZapLogger(cfg().LogLevel == "debug"))

	infrastructureClusterConfig, err := o.kubernetes.InfrastructureClusterConfig(o.dryRun.Value)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting config for infastructure cluster")
	}
	opts := manager.Options{
		MetricsBindAddress:      "0",
		Namespace:               cfg().ProwJobNamespace,
		LeaderElection:          true,
		LeaderElectionNamespace: o.namespace,
		LeaderElectionID:        "prow-sinker-leaderlock",
	}
	mgr, err := manager.New(infrastructureClusterConfig, opts)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating manager")
	}
	if err := prowapi.AddToScheme(mgr.GetScheme()); err != nil {
		logrus.WithError(err).Fatal("Error adding prow api to scheme")
	}

	buildClusterClients, err := o.kubernetes.BuildClusterClients(cfg().PodNamespace, o.dryRun.Value)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating build cluster clients.")
	}

	var podClients []corev1.PodInterface
	for _, client := range buildClusterClients {
		// sinker doesn't care about build cluster aliases
		podClients = append(podClients, client)
	}

	c := controller{
		ctx:           context.Background(),
		logger:        logrus.NewEntry(logrus.StandardLogger()),
		prowJobClient: mgr.GetClient(),
		podClients:    podClients,
		config:        cfg,
		runOnce:       o.runOnce,
	}
	if err := mgr.Add(&c); err != nil {
		logrus.WithError(err).Fatal("failed to add controller to manager")
	}
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logrus.WithError(err).Fatal("failed to start manager")
	}
}

type controller struct {
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *logrus.Entry
	prowJobClient ctrlruntimeclient.Client
	podClients    []corev1.PodInterface
	config        config.Getter
	runOnce       bool
}

func (c *controller) Start(stopChan <-chan struct{}) error {
	runChan := make(chan struct{})

	// We want to be able to dynamically adjust to changed config values, hence we cant use a time.Ticker
	go func() {
		for {
			runChan <- struct{}{}
			time.Sleep(c.config().Sinker.ResyncPeriod.Duration)
		}
	}()

	for {
		select {
		case <-stopChan:
			c.logger.Info("stop signal received, quitting")
			return nil
		case <-runChan:
			start := time.Now()
			c.clean()
			c.logger.Infof("Sync time: %v", time.Since(start))
			if c.runOnce {
				return nil
			}
		}
	}
}

type sinkerReconciliationMetrics struct {
	podsCreated            int
	startAt                time.Time
	finishedAt             time.Time
	podsRemoved            map[string]int
	podRemovalErrors       map[string]int
	prowJobsCreated        int
	prowJobsCleaned        map[string]int
	prowJobsCleaningErrors map[string]int
}

// Prometheus Metrics
var (
	sinkerMetrics = struct {
		podsCreated            prometheus.Gauge
		timeUsed               prometheus.Gauge
		podsRemoved            *prometheus.GaugeVec
		podRemovalErrors       *prometheus.GaugeVec
		prowJobsCreated        prometheus.Gauge
		prowJobsCleaned        *prometheus.GaugeVec
		prowJobsCleaningErrors *prometheus.GaugeVec
	}{
		podsCreated: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sinker_pods_existing",
			Help: "Number of the existing pods in each sinker cleaning.",
		}),
		timeUsed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sinker_loop_duration_seconds",
			Help: "Time used in each sinker cleaning.",
		}),
		podsRemoved: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sinker_pods_removed",
			Help: "Number of pods removed in each sinker cleaning.",
		}, []string{
			"reason",
		}),
		podRemovalErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sinker_pod_removal_errors",
			Help: "Number of errors which occurred in each sinker pod cleaning.",
		}, []string{
			"reason",
		}),
		prowJobsCreated: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sinker_prow_jobs_existing",
			Help: "Number of the existing prow jobs in each sinker cleaning.",
		}),
		prowJobsCleaned: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sinker_prow_jobs_cleaned",
			Help: "Number of prow jobs cleaned in each sinker cleaning.",
		}, []string{
			"reason",
		}),
		prowJobsCleaningErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sinker_prow_jobs_cleaning_errors",
			Help: "Number of errors which occurred in each sinker prow job cleaning.",
		}, []string{
			"reason",
		}),
	}
)

func init() {
	prometheus.MustRegister(sinkerMetrics.podsCreated)
	prometheus.MustRegister(sinkerMetrics.timeUsed)
	prometheus.MustRegister(sinkerMetrics.podsRemoved)
	prometheus.MustRegister(sinkerMetrics.podRemovalErrors)
	prometheus.MustRegister(sinkerMetrics.prowJobsCreated)
	prometheus.MustRegister(sinkerMetrics.prowJobsCleaned)
	prometheus.MustRegister(sinkerMetrics.prowJobsCleaningErrors)
}

func (m *sinkerReconciliationMetrics) getPodsTotalRemoved() int {
	result := 0
	for _, v := range m.podsRemoved {
		result += v
	}
	return result
}

func (m *sinkerReconciliationMetrics) getTimeUsed() time.Duration {
	return m.finishedAt.Sub(m.startAt)
}

func (c *controller) clean() {

	metrics := sinkerReconciliationMetrics{
		startAt:                time.Now(),
		podsRemoved:            map[string]int{},
		podRemovalErrors:       map[string]int{},
		prowJobsCleaned:        map[string]int{},
		prowJobsCleaningErrors: map[string]int{}}

	// Clean up old prow jobs first.
	prowJobs := &prowapi.ProwJobList{}
	if err := c.prowJobClient.List(c.ctx, prowJobs, ctrlruntimeclient.InNamespace(c.config().ProwJobNamespace)); err != nil {
		c.logger.WithError(err).Error("Error listing prow jobs.")
		return
	}
	metrics.prowJobsCreated = len(prowJobs.Items)

	// Only delete pod if its prowjob is marked as finished
	isExist := sets.NewString()
	isFinished := sets.NewString()

	maxProwJobAge := c.config().Sinker.MaxProwJobAge.Duration
	for _, prowJob := range prowJobs.Items {
		isExist.Insert(prowJob.ObjectMeta.Name)
		// Handle periodics separately.
		if prowJob.Spec.Type == prowapi.PeriodicJob {
			continue
		}
		if !prowJob.Complete() {
			continue
		}
		isFinished.Insert(prowJob.ObjectMeta.Name)
		if time.Since(prowJob.Status.StartTime.Time) <= maxProwJobAge {
			continue
		}
		if err := c.prowJobClient.Delete(c.ctx, &prowJob); err == nil {
			c.logger.WithFields(pjutil.ProwJobFields(&prowJob)).Info("Deleted prowjob.")
			metrics.prowJobsCleaned[reasonProwJobAged]++
		} else {
			c.logger.WithFields(pjutil.ProwJobFields(&prowJob)).WithError(err).Error("Error deleting prowjob.")
			metrics.prowJobsCleaningErrors[string(k8serrors.ReasonForError(err))]++
		}
	}

	// Keep track of what periodic jobs are in the config so we will
	// not clean up their last prowjob.
	isActivePeriodic := make(map[string]bool)
	for _, p := range c.config().Periodics {
		isActivePeriodic[p.Name] = true
	}

	// Get the jobs that we need to retain so horologium can continue working
	// as intended.
	latestPeriodics := pjutil.GetLatestProwJobs(prowJobs.Items, prowapi.PeriodicJob)
	for _, prowJob := range prowJobs.Items {
		if prowJob.Spec.Type != prowapi.PeriodicJob {
			continue
		}

		latestPJ := latestPeriodics[prowJob.Spec.Job]
		if isActivePeriodic[prowJob.Spec.Job] && prowJob.ObjectMeta.Name == latestPJ.ObjectMeta.Name {
			// Ignore deleting this one.
			continue
		}
		if !prowJob.Complete() {
			continue
		}
		isFinished.Insert(prowJob.ObjectMeta.Name)
		if time.Since(prowJob.Status.StartTime.Time) <= maxProwJobAge {
			continue
		}
		if err := c.prowJobClient.Delete(c.ctx, &prowJob); err == nil {
			c.logger.WithFields(pjutil.ProwJobFields(&prowJob)).Info("Deleted prowjob.")
			metrics.prowJobsCleaned[reasonProwJobAgedPeriodic]++
		} else {
			c.logger.WithFields(pjutil.ProwJobFields(&prowJob)).WithError(err).Error("Error deleting prowjob.")
			metrics.prowJobsCleaningErrors[string(k8serrors.ReasonForError(err))]++
		}
	}

	// Now clean up old pods.
	selector := fmt.Sprintf("%s = %s", kube.CreatedByProw, "true")
	for _, client := range c.podClients {
		pods, err := client.List(metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			c.logger.WithError(err).Error("Error listing pods.")
			return
		}
		metrics.podsCreated += len(pods.Items)
		maxPodAge := c.config().Sinker.MaxPodAge.Duration
		for _, pod := range pods.Items {
			clean := !pod.Status.StartTime.IsZero() && time.Since(pod.Status.StartTime.Time) > maxPodAge
			reason := reasonPodAged
			if !isFinished.Has(pod.ObjectMeta.Name) {
				// prowjob exists and is not marked as completed yet
				// deleting the pod now will result in plank creating a brand new pod
				clean = false
			}
			if !isExist.Has(pod.ObjectMeta.Name) {
				// prowjob has gone, we want to clean orphan pods regardless of the state
				reason = reasonPodOrphaned
				clean = true
			}

			if !clean {
				continue
			}

			// Delete old finished or orphan pods. Don't quit if we fail to delete one.
			if err := client.Delete(pod.ObjectMeta.Name, &metav1.DeleteOptions{}); err == nil {
				c.logger.WithField("pod", pod.ObjectMeta.Name).Info("Deleted old completed pod.")
				metrics.podsRemoved[reason]++
			} else {
				c.logger.WithField("pod", pod.ObjectMeta.Name).WithError(err).Error("Error deleting pod.")
				metrics.podRemovalErrors[string(k8serrors.ReasonForError(err))]++
			}
		}
	}

	metrics.finishedAt = time.Now()
	sinkerMetrics.podsCreated.Set(float64(metrics.podsCreated))
	sinkerMetrics.timeUsed.Set(float64(metrics.getTimeUsed().Seconds()))
	for k, v := range metrics.podsRemoved {
		sinkerMetrics.podsRemoved.WithLabelValues(k).Set(float64(v))
	}
	for k, v := range metrics.podRemovalErrors {
		sinkerMetrics.podRemovalErrors.WithLabelValues(k).Set(float64(v))
	}
	sinkerMetrics.prowJobsCreated.Set(float64(metrics.prowJobsCreated))
	for k, v := range metrics.prowJobsCleaned {
		sinkerMetrics.prowJobsCleaned.WithLabelValues(k).Set(float64(v))
	}
	for k, v := range metrics.prowJobsCleaningErrors {
		sinkerMetrics.prowJobsCleaningErrors.WithLabelValues(k).Set(float64(v))
	}
	c.logger.Info("Sinker reconciliation complete.")
}
