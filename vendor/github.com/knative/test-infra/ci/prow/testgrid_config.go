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

// data definitions that are used for the testgrid config file generation

package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

const (
	// baseOptions setting for testgrid dashboard tabs
	testgridTabGroupByDir    = "exclude-filter-by-regex=Overall$&group-by-directory=&expand-groups=&sort-by-name="
	testgridTabGroupByTarget = "exclude-filter-by-regex=Overall$&group-by-target=&expand-groups=&sort-by-name="
	testgridTabSortByName    = "sort-by-name="

	// generalTestgridConfig contains config-wide definitions.
	generalTestgridConfig = "testgrid_config_header.yaml"

	// testGroupTemplate is the template for the test group config
	testGroupTemplate = "testgrid_testgroup.yaml"

	// dashboardTabTemplate is the template for the dashboard tab config
	dashboardTabTemplate = "testgrid_dashboardtab.yaml"

	// dashboardGroupTemplate is the template for the dashboard tab config
	dashboardGroupTemplate = "testgrid_dashboardgroup.yaml"
)

var (
	// goCoverageMap keep track of which repo has go code coverage when parsing the simple config file
	goCoverageMap map[string]bool
	// projNames save the proj names in a list when parsing the simple config file, for the purpose of maintaining the output sequence
	projNames []string
	// repoNames save the repo names in a list when parsing the simple config file, for the purpose of maintaining the output sequence
	repoNames []string

	// metaData saves the meta data needed to generate the final config file.
	// key is the main project version, value is another map containing job details
	//     for the job detail map, key is the repo name, value is the list of job types, like continuous, latency, nightly, and etc.
	metaData = make(map[string]map[string][]string)

	// templatesCache caches templates in memory to avoid I/O
	templatesCache = make(map[string]string)
)

// baseTestgridTemplateData contains basic data about the testgrid config file.
// TODO(Fredy-Z): remove this structure and use baseProwJobTemplateData instead
type baseTestgridTemplateData struct {
	TestGroupName string
	Year          int
}

// testGroupTemplateData contains data about a test group
type testGroupTemplateData struct {
	Base baseTestgridTemplateData
	// TODO(Fredy-Z): use baseProwJobTemplateData then this attribute can be removed
	GcsLogDir string
	Extras    map[string]string
}

// dashboardTabTemplateData contains data about a dashboard tab
type dashboardTabTemplateData struct {
	Base        baseTestgridTemplateData
	Name        string
	BaseOptions string
	Extras      map[string]string
}

// dashboardGroupTemplateData contains data about a dashboard group
type dashboardGroupTemplateData struct {
	Name      string
	RepoNames []string
}

// testgridEntityGenerator is a function that generates the entity given the repo name and job names
type testgridEntityGenerator func(string, string, []string)

// newBaseTestgridTemplateData returns a testgridTemplateData type with its initial, default values.
func newBaseTestgridTemplateData(testGroupName string) baseTestgridTemplateData {
	var data baseTestgridTemplateData
	data.Year = time.Now().Year()
	data.TestGroupName = testGroupName
	return data
}

// generateTestGridSection generates the configs for a TestGrid section using the given generator
func generateTestGridSection(sectionName string, generator testgridEntityGenerator, skipReleasedProj bool) {
	outputConfig(sectionName + ":")
	for _, projName := range projNames {
		// Do not handle the project if it is released and we want to skip it.
		if skipReleasedProj && isReleased(projName) {
			continue
		}
		repos := metaData[projName]
		for _, repoName := range repoNames {
			if jobNames, exists := repos[repoName]; exists {
				generator(projName, repoName, jobNames)
			}
		}
	}
}

// generateTestGroup generates the test group configuration
func generateTestGroup(projName string, repoName string, jobNames []string) {
	projRepoStr := buildProjRepoStr(projName, repoName)
	for _, jobName := range jobNames {
		testGroupName := getTestGroupName(projRepoStr, jobName)
		gcsLogDir := fmt.Sprintf("%s/%s/%s", gcsBucket, logsDir, testGroupName)
		extras := make(map[string]string)
		switch jobName {
		case "continuous", "dot-release", "auto-release", "performance", "performance-mesh", "latency", "nightly":
			isDailyBranch := regexp.MustCompile(`-[0-9\.]+-continuous`).FindString(testGroupName) != ""
			if !isDailyBranch && (jobName == "continuous" || jobName == "auto-release") {
				// TODO(Fredy-Z): this value should be derived from the cron string
				extras["alert_stale_results_hours"] = "3"
				if jobName == "continuous" {
					// For continuous flows, alert after 3 failures due to flakiness
					extras["num_failures_to_alert"] = "3"
				}
			}
			if jobName == "dot-release" {
				// TODO(Fredy-Z): this value should be derived from the cron string
				extras["alert_stale_results_hours"] = "170" // 1 week + 2h
			}
			if jobName == "latency" {
				extras["short_text_metric"] = "latency"
			}
			if jobName == "performance" || jobName == "performance-mesh" {
				extras["short_text_metric"] = "perf_latency"
			}
		case "test-coverage":
			gcsLogDir = strings.ToLower(fmt.Sprintf("%s/%s/ci-%s-%s", gcsBucket, logsDir, projRepoStr, "go-coverage"))
			extras["short_text_metric"] = "coverage"
			// Do not alert on coverage failures (i.e., coverage below threshold)
			extras["num_failures_to_alert"] = "9999"
		case "istio-1.0-mesh", "istio-1.0-no-mesh", "istio-1.1-mesh", "istio-1.1-no-mesh", "istio-1.2-mesh", "istio-1.2-no-mesh":
			extras["alert_stale_results_hours"] = "3"
			extras["num_failures_to_alert"] = "3"
		default:
			log.Fatalf("Unknown jobName for generateTestGroup: %s", jobName)
		}
		executeTestGroupTemplate(testGroupName, gcsLogDir, extras)
	}
}

// executeTestGroupTemplate outputs the given test group config template with the given data
func executeTestGroupTemplate(testGroupName string, gcsLogDir string, extras map[string]string) {
	var data testGroupTemplateData
	data.Base.TestGroupName = testGroupName
	data.GcsLogDir = gcsLogDir
	data.Extras = extras
	executeTemplate("test group", readTemplate(testGroupTemplate), data)
}

// generateDashboard generates the dashboard configuration
func generateDashboard(projName string, repoName string, jobNames []string) {
	projRepoStr := buildProjRepoStr(projName, repoName)
	outputConfig("- name: " + strings.ToLower(repoName) + "\n" + baseIndent + "dashboard_tab:")
	noExtras := make(map[string]string)
	for _, jobName := range jobNames {
		testGroupName := getTestGroupName(projRepoStr, jobName)
		switch jobName {
		case "continuous":
			executeDashboardTabTemplate("continuous", testGroupName, testgridTabSortByName, noExtras)
			// This is a special case for knative/serving, as conformance tab is just a filtered view of the continuous tab.
			if projRepoStr == "knative-serving" {
				executeDashboardTabTemplate("conformance", testGroupName, "include-filter-by-regex=test/conformance/&sort-by-name=", noExtras)
			}
		case "dot-release", "auto-release", "performance", "performance-mesh", "latency":
			extras := make(map[string]string)
			baseOptions := testgridTabSortByName
			if jobName == "performance" || jobName == "performance-mesh" {
				baseOptions = testgridTabGroupByTarget
			}
			if jobName == "latency" {
				baseOptions = testgridTabGroupByDir
				extras["description"] = "95% latency in ms"
			}
			executeDashboardTabTemplate(jobName, testGroupName, baseOptions, extras)
		case "nightly":
			executeDashboardTabTemplate("nightly", testGroupName, testgridTabSortByName, noExtras)
		case "test-coverage":
			executeDashboardTabTemplate("coverage", testGroupName, testgridTabGroupByDir, noExtras)
		case "istio-1.0-mesh", "istio-1.0-no-mesh", "istio-1.1-mesh", "istio-1.1-no-mesh", "istio-1.2-mesh", "istio-1.2-no-mesh":
			executeDashboardTabTemplate(jobName, testGroupName, testgridTabSortByName, noExtras)
		default:
			log.Fatalf("Unknown job name %q", jobName)
		}
	}
}

// executeTestGroupTemplate outputs the given dashboard tab config template with the given data
func executeDashboardTabTemplate(dashboardTabName string, testGroupName string, baseOptions string, extras map[string]string) {
	var data dashboardTabTemplateData
	data.Name = dashboardTabName
	data.Base.TestGroupName = testGroupName
	data.BaseOptions = baseOptions
	data.Extras = extras
	executeTemplate("dashboard tab", readTemplate(dashboardTabTemplate), data)
}

// getTestGroupName get the testGroupName from the given repoName and jobName
func getTestGroupName(repoName string, jobName string) string {
	switch jobName {
	case "continuous", "dot-release", "auto-release", "performance", "performance-mesh", "latency":
		return strings.ToLower(fmt.Sprintf("ci-%s-%s", repoName, jobName))
	case "nightly":
		return strings.ToLower(fmt.Sprintf("ci-%s-%s-release", repoName, jobName))
	case "test-coverage":
		return strings.ToLower(fmt.Sprintf("pull-%s-%s", repoName, jobName))
	case "istio-1.0-mesh", "istio-1.0-no-mesh", "istio-1.1-mesh", "istio-1.1-no-mesh", "istio-1.2-mesh", "istio-1.2-no-mesh":
		return strings.ToLower(fmt.Sprintf("ci-%s-%s", repoName, jobName))
	}
	log.Fatalf("Unknown jobName for getTestGroupName: %s", jobName)
	return ""
}

func generateDashboardsForReleases() {
	for _, projName := range projNames {
		// Do not handle the project if it is not released.
		if !isReleased(projName) {
			continue
		}
		repos := metaData[projName]
		outputConfig("- name: " + projName + "\n" + baseIndent + "dashboard_tab:")
		noExtras := make(map[string]string)
		for _, repoName := range repoNames {
			if _, exists := repos[repoName]; exists {
				testGroupName := getTestGroupName(buildProjRepoStr(projName, repoName), "continuous")
				executeDashboardTabTemplate(repoName, testGroupName, testgridTabSortByName, noExtras)
			}
		}
	}
}

// generateDashboardGroups generates the dashboard groups configuration
func generateDashboardGroups() {
	outputConfig("dashboard_groups:")
	for _, projName := range projNames {
		// there is only one dashboard for each released project, so we do not need to group them
		if isReleased(projName) {
			continue
		}

		dashboardRepoNames := make([]string, 0)
		repos := metaData[projName]
		for _, repoName := range repoNames {
			if _, exists := repos[repoName]; exists {
				dashboardRepoNames = append(dashboardRepoNames, repoName)
			}
		}
		executeDashboardGroupTemplate(projName, dashboardRepoNames)
	}
}

// executeDashboardGroupTemplate outputs the given dashboard group config template with the given data
func executeDashboardGroupTemplate(dashboardGroupName string, dashboardRepoNames []string) {
	var data dashboardGroupTemplateData
	data.Name = dashboardGroupName
	data.RepoNames = dashboardRepoNames
	executeTemplate("dashboard group", readTemplate(dashboardGroupTemplate), data)
}
