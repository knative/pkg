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

// data definitions that are used for the config file generation of periodic prow jobs

package main

import (
	"fmt"
	"log"
	"path"

	yaml "gopkg.in/yaml.v2"
)

const (
	// Template for periodic test/release jobs.
	periodicTestJob = "prow_periodic_test_job.yaml"

	// Template for periodic custom jobs.
	periodicCustomJob = "prow_periodic_custom_job.yaml"

	// Cron strings for key jobs
	goCoveragePeriodicJobCron           = "0 1 * * *"    // Run at 01:00 every day
	cleanupPeriodicJobCron              = "0 19 * * 1"   // Run at 11:00PST/12:00PST every Monday (19:00 UTC)
	flakesReporterPeriodicJobCron       = "0 12 * * *"   // Run at 4:00PST/5:00PST every day (12:00 UTC)
	flakesResultRecorderPeriodicJobCron = "0 * * * *"    // Run every hour
	prowversionbumperPeriodicJobCron    = "0 20 * * 1"   // Run at 12:00PST/13:00PST every Monday (20:00 UTC)
	backupPeriodicJobCron               = "15 9 * * *"   // Run at 02:15PST every day (09:15 UTC)
	perfPeriodicJobCron                 = "0 */3 * * *"  // Run every 3 hours
	clearAlertsPeriodicJobCron          = "0,30 * * * *" // Run every 30 minutes
	recreatePerfClusterPeriodicJobCron  = "30 07 * * *"  // Run at 00:30PST every day (07:30 UTC)
	updatePerfClusterPeriodicJobCron    = "5 * * * *"    // Run every an hour

	// Perf job constants
	perfTimeout = 120 // Job timeout in minutes
	perfNodes   = "4" // Number of nodes needed to run perf tests. Needs to be string
)

// periodicJobTemplateData contains data about a periodic Prow job.
type periodicJobTemplateData struct {
	Base            baseProwJobTemplateData
	PeriodicJobName string
	CronString      string
	PeriodicCommand []string
}

// generatePeriodic generates all periodic job configs for the given repo and configuration.
func generatePeriodic(title string, repoName string, periodicConfig yaml.MapSlice) {
	var data periodicJobTemplateData
	data.Base = newbaseProwJobTemplateData(repoName)
	jobNameSuffix := ""
	jobTemplate := readTemplate(periodicTestJob)
	jobType := ""
	isMonitoredJob := false

	for i, item := range periodicConfig {
		switch item.Key {
		case "continuous":
			if !getBool(item.Value) {
				return
			}
			jobType = getString(item.Key)
			jobNameSuffix = "continuous"
			isMonitoredJob = true
			// Use default command and arguments if none given.
			if data.Base.Command == "" {
				data.Base.Command = presubmitScript
			}
			if len(data.Base.Args) == 0 {
				data.Base.Args = allPresubmitTests
			}
		case "nightly":
			if !getBool(item.Value) {
				return
			}
			jobType = getString(item.Key)
			jobNameSuffix = "nightly-release"
			data.Base.ServiceAccount = nightlyAccount
			data.Base.Command = releaseScript
			data.Base.Args = releaseNightly
			data.Base.Timeout = 90
			isMonitoredJob = true
		case "branch-ci":
			if !getBool(item.Value) {
				return
			}
			jobType = getString(item.Key)
			jobNameSuffix = "continuous"
			data.Base.Command = releaseScript
			data.Base.Args = releaseLocal
			setupDockerInDockerForJob(&data.Base)
			// TODO(adrcunha): Consider reducing the timeout in the future.
			data.Base.Timeout = 180
			isMonitoredJob = true
		case "dot-release", "auto-release":
			if !getBool(item.Value) {
				return
			}
			jobType = getString(item.Key)
			jobNameSuffix = getString(item.Key)
			data.Base.ServiceAccount = releaseAccount
			data.Base.Command = releaseScript
			data.Base.Args = []string{
				"--" + jobNameSuffix,
				"--release-gcs " + data.Base.ReleaseGcs,
				"--release-gcr gcr.io/knative-releases",
				"--github-token /etc/hub-token/token"}
			addVolumeToJob(&data.Base, "/etc/hub-token", "hub-token", true, "")
			data.Base.Timeout = 90
			isMonitoredJob = true
		case "performance", "performance-mesh":
			if !getBool(item.Value) {
				return
			}
			jobType = getString(item.Key)
			jobNameSuffix = getString(item.Key)
			data.Base.Command = performanceScript
			data.CronString = perfPeriodicJobCron
			// We need a larger cluster of at least 16 nodes for perf tests
			addEnvToJob(&data.Base, "E2E_MIN_CLUSTER_NODES", perfNodes)
			addEnvToJob(&data.Base, "E2E_MAX_CLUSTER_NODES", perfNodes)
			data.Base.Timeout = perfTimeout
			isMonitoredJob = true
		case "latency":
			if !getBool(item.Value) {
				return
			}
			jobType = getString(item.Key)
			jobTemplate = readTemplate(periodicCustomJob)
			jobNameSuffix = "latency"
			data.Base.Image = "gcr.io/knative-tests/test-infra/metrics:latest"
			data.Base.Command = "/metrics"
			data.Base.Args = []string{
				fmt.Sprintf("--source-directory=ci-%s-continuous", data.Base.RepoNameForJob),
				"--artifacts-dir=$(ARTIFACTS)",
				"--service-account=" + data.Base.ServiceAccount}
			isMonitoredJob = true
		case "custom-job":
			jobType = getString(item.Key)
			jobNameSuffix = getString(item.Value)
			data.Base.Timeout = 100
		case "cron":
			data.CronString = getString(item.Value)
		case "release":
			version := getString(item.Value)
			jobNameSuffix = version + "-" + jobNameSuffix
			data.Base.RepoBranch = "release-" + version
			isMonitoredJob = true
		case "webhook-apicoverage":
			if !getBool(item.Value) {
				return
			}
			jobType = getString(item.Key)
			jobNameSuffix = "webhook-apicoverage"
			data.Base.Command = webhookAPICoverageScript
			addEnvToJob(&data.Base, "SYSTEM_NAMESPACE", data.Base.RepoNameForJob)
		default:
			continue
		}
		// Knock-out the item, signalling it was already parsed.
		periodicConfig[i] = yaml.MapItem{}
	}
	parseBasicJobConfigOverrides(&data.Base, periodicConfig)
	data.PeriodicJobName = fmt.Sprintf("ci-%s", data.Base.RepoNameForJob)
	if jobNameSuffix != "" {
		data.PeriodicJobName += "-" + jobNameSuffix
	}
	if isMonitoredJob {
		addMonitoringPubsubLabelsToJob(&data.Base, data.PeriodicJobName)
	}
	if data.CronString == "" {
		data.CronString = generateCron(jobType, data.PeriodicJobName, data.Base.Timeout)
	}
	// Ensure required data exist.
	if data.CronString == "" {
		log.Fatalf("Job %q is missing cron string", data.PeriodicJobName)
	}
	if len(data.Base.Args) == 0 && data.Base.Command == "" {
		log.Fatalf("Job %q is missing command", data.PeriodicJobName)
	}
	if jobType == "branch-ci" && data.Base.RepoBranch == "" {
		log.Fatalf("%q jobs are intended to be used on release branches", jobType)
	}
	// Generate config itself.
	data.PeriodicCommand = createCommand(data.Base)
	if data.Base.ServiceAccount != "" {
		addEnvToJob(&data.Base, "GOOGLE_APPLICATION_CREDENTIALS", data.Base.ServiceAccount)
		addEnvToJob(&data.Base, "E2E_CLUSTER_REGION", "us-central1")
	}
	if data.Base.RepoBranch != "" && data.Base.RepoBranch != "master" {
		// If it's a release version, add env var PULL_BASE_REF as ref name of the base branch.
		// The reason for having it is in https://github.com/knative/test-infra/issues/780.
		addEnvToJob(&data.Base, "PULL_BASE_REF", data.Base.RepoBranch)
	}
	addExtraEnvVarsToJob(extraEnvVars, &data.Base)
	configureServiceAccountForJob(&data.Base)
	executeJobTemplate("periodic", jobTemplate, title, repoName, data.PeriodicJobName, false, data)
}

// generateCleanupPeriodicJob generates the cleanup job config.
func generateCleanupPeriodicJob() {
	var data periodicJobTemplateData
	data.Base = newbaseProwJobTemplateData("knative/test-infra")
	data.PeriodicJobName = "ci-knative-cleanup"
	data.CronString = cleanupPeriodicJobCron
	data.Base.DecorationConfig = []string{"timeout: 432000000000000"} // 120 hours
	data.Base.Command = cleanupScript
	data.Base.Args = []string{
		"--project-resource-yaml ci/prow/boskos/resources.yaml",
		"--days-to-keep-images 30",
		"--hours-to-keep-clusters 24",
		"--service-account " + data.Base.ServiceAccount,
		"--artifacts $(ARTIFACTS)"}
	data.Base.ExtraRefs = append(data.Base.ExtraRefs, "  base_ref: "+data.Base.RepoBranch)
	addExtraEnvVarsToJob(extraEnvVars, &data.Base)
	configureServiceAccountForJob(&data.Base)
	addMonitoringPubsubLabelsToJob(&data.Base, data.PeriodicJobName)
	executeJobTemplate("periodic cleanup", readTemplate(periodicCustomJob), "presubmits", "", data.PeriodicJobName, false, data)
}

// generateFlakytoolPeriodicJob generates the flaky tests reporting job config.
func generateFlakytoolPeriodicJob() {
	var data periodicJobTemplateData
	data.Base = newbaseProwJobTemplateData("knative/test-infra")
	data.Base.Image = flakesreporterDockerImage
	data.PeriodicJobName = "ci-knative-flakes-reporter"
	data.CronString = flakesReporterPeriodicJobCron
	data.Base.Command = "/flaky-test-reporter"
	data.Base.Args = []string{
		"--service-account=" + data.Base.ServiceAccount,
		"--github-account=/etc/flaky-test-reporter-github-token/token",
		"--slack-account=/etc/flaky-test-reporter-slack-token/token"}
	data.Base.ExtraRefs = append(data.Base.ExtraRefs, "  base_ref: "+data.Base.RepoBranch)
	addExtraEnvVarsToJob(extraEnvVars, &data.Base)
	configureServiceAccountForJob(&data.Base)
	addVolumeToJob(&data.Base, "/etc/flaky-test-reporter-github-token", "flaky-test-reporter-github-token", true, "")
	addVolumeToJob(&data.Base, "/etc/flaky-test-reporter-slack-token", "flaky-test-reporter-slack-token", true, "")
	addMonitoringPubsubLabelsToJob(&data.Base, data.PeriodicJobName)
	executeJobTemplate("periodic flakesreporter", readTemplate(periodicCustomJob), "presubmits", "", data.PeriodicJobName, false, data)

	// Generate another job that runs more frequently but not reporting to
	// Github or Slack
	data.PeriodicJobName = "ci-knative-flakes-resultsrecorder"
	data.CronString = flakesResultRecorderPeriodicJobCron
	data.Base.Args = []string{
		"--service-account=" + data.Base.ServiceAccount,
		"--skip-report",
		"--build-count=20"}
	executeJobTemplate("periodic flakesresultrecorder", readTemplate(periodicCustomJob), "presubmits", "", data.PeriodicJobName, false, data)
}

// generateVersionBumpertoolPeriodicJob generates the Prow version bumper job config.
func generateVersionBumpertoolPeriodicJob() {
	var data periodicJobTemplateData
	data.Base = newbaseProwJobTemplateData("knative/test-infra")
	data.Base.Image = prowversionbumperDockerImage
	data.PeriodicJobName = "ci-knative-prow-auto-bumper"
	data.CronString = prowversionbumperPeriodicJobCron
	data.Base.Command = "/prow-auto-bumper"
	data.Base.Args = []string{
		"--github-account=/etc/prow-auto-bumper-github-token/token",
		"--git-userid=knative-prow-updater-robot",
		"--git-username='Knative Prow Updater Robot'",
		"--git-email=knative-prow-updater-robot@google.com"}
	data.Base.ExtraRefs = append(data.Base.ExtraRefs, "  base_ref: "+data.Base.RepoBranch)
	addExtraEnvVarsToJob(extraEnvVars, &data.Base)
	configureServiceAccountForJob(&data.Base)
	addVolumeToJob(&data.Base, "/etc/prow-auto-bumper-github-token", "prow-auto-bumper-github-token", true, "")
	addVolumeToJob(&data.Base, "/root/.ssh", "prow-updater-robot-ssh-key", true, "0400")
	executeJobTemplate("periodic versionbumper", readTemplate(periodicCustomJob), "presubmits", "", data.PeriodicJobName, false, data)
}

// generateBackupPeriodicJob generates the backup job config.
func generateBackupPeriodicJob() {
	var data periodicJobTemplateData
	data.Base = newbaseProwJobTemplateData("none/unused")
	data.Base.ServiceAccount = "/etc/backup-account/service-account.json"
	data.Base.Image = "gcr.io/knative-tests/test-infra/backups:latest"
	data.PeriodicJobName = "ci-knative-backup-artifacts"
	data.CronString = backupPeriodicJobCron
	data.Base.Command = "/backup.sh"
	data.Base.Args = []string{data.Base.ServiceAccount}
	data.Base.ExtraRefs = []string{} // no repo clone required
	addExtraEnvVarsToJob(extraEnvVars, &data.Base)
	configureServiceAccountForJob(&data.Base)
	executeJobTemplate("periodic backup", readTemplate(periodicCustomJob), "presubmits", "", data.PeriodicJobName, false, data)
}

// generateGoCoveragePeriodic generates the go coverage periodic job config for the given repo (configuration is ignored).
func generateGoCoveragePeriodic(title string, repoName string, _ yaml.MapSlice) {
	for i, repo := range repositories {
		if repoName != repo.Name || !repo.EnableGoCoverage {
			continue
		}
		repositories[i].Processed = true
		var data periodicJobTemplateData
		data.Base = newbaseProwJobTemplateData(repoName)
		data.Base.Image = coverageDockerImage
		data.PeriodicJobName = fmt.Sprintf("ci-%s-go-coverage", data.Base.RepoNameForJob)
		data.CronString = goCoveragePeriodicJobCron
		data.Base.GoCoverageThreshold = repo.GoCoverageThreshold
		data.Base.Command = "/coverage"
		data.Base.Args = []string{
			"--artifacts=$(ARTIFACTS)",
			fmt.Sprintf("--cov-threshold-percentage=%d", data.Base.GoCoverageThreshold)}
		data.Base.ServiceAccount = ""
		data.Base.ExtraRefs = append(data.Base.ExtraRefs, "  base_ref: "+data.Base.RepoBranch)
		if repositories[i].DotDev {
			data.Base.ExtraRefs = append(data.Base.ExtraRefs, "  path_alias: knative.dev/"+path.Base(repoName))
		}
		addExtraEnvVarsToJob(extraEnvVars, &data.Base)
		addMonitoringPubsubLabelsToJob(&data.Base, data.PeriodicJobName)
		configureServiceAccountForJob(&data.Base)
		executeJobTemplate("periodic go coverage", readTemplate(periodicCustomJob), title, repoName, data.PeriodicJobName, false, data)
		return
	}
}

// generatePerfClusterUpdatePeriodicJobs generates periodic jobs to update serving clusters
// that run performance testing benchmarks
func generatePerfClusterUpdatePeriodicJobs() {
	// Generate periodic performance jobs for serving
	perfClusterUpdatePeriodicJob(
		"ci-knative-serving-recreate-clusters",
		recreatePerfClusterPeriodicJobCron,
		"./test/performance/tools/recreate_clusters.sh",
		"serving",
		"performance-test",
	)
	perfClusterUpdatePeriodicJob(
		"ci-knative-serving-update-clusters",
		updatePerfClusterPeriodicJobCron,
		"./test/performance/tools/update_clusters.sh",
		"serving",
		"performance-test",
	)

	// Generate periodic performance jobs for eventing
	perfClusterUpdatePeriodicJob(
		"ci-knative-eventing-recreate-clusters",
		recreatePerfClusterPeriodicJobCron,
		"./test/performance/tools/recreate_clusters.sh",
		"eventing",
		"eventing-performance-test",
	)
	perfClusterUpdatePeriodicJob(
		"ci-knative-eventing-update-clusters",
		updatePerfClusterPeriodicJobCron,
		"./test/performance/tools/update_clusters.sh",
		"eventing",
		"eventing-performance-test",
	)
}

func perfClusterUpdatePeriodicJob(jobName, cronString, command, repo, sa string) {
	var data periodicJobTemplateData
	data.Base = newbaseProwJobTemplateData("knative/" + repo)
	data.Base.ExtraRefs = append(data.Base.ExtraRefs, "  base_ref: "+data.Base.RepoBranch)
	data.Base.ExtraRefs = append(data.Base.ExtraRefs, "  path_alias: knative.dev/"+repo)
	data.Base.Command = command
	data.PeriodicJobName = jobName
	data.CronString = cronString
	data.PeriodicCommand = createCommand(data.Base)
	configureServiceAccountForJob(&data.Base)
	addEnvToJob(&data.Base, "GOOGLE_APPLICATION_CREDENTIALS", data.Base.ServiceAccount)
	addVolumeToJob(&data.Base, "/etc/performance-test", sa, true, "")
	addEnvToJob(&data.Base, "PERF_TEST_GOOGLE_APPLICATION_CREDENTIALS", "/etc/performance-test/service-account.json")
	addEnvToJob(&data.Base, "GITHUB_TOKEN", "/etc/performance-test/github-token")
	addEnvToJob(&data.Base, "SLACK_READ_TOKEN", "/etc/performance-test/slack-read-token")
	addEnvToJob(&data.Base, "SLACK_WRITE_TOKEN", "/etc/performance-test/slack-write-token")
	addMonitoringPubsubLabelsToJob(&data.Base, data.PeriodicJobName)
	executeJobTemplate(jobName, readTemplate(periodicTestJob), "presubmits", "", data.PeriodicJobName, false, data)
}
