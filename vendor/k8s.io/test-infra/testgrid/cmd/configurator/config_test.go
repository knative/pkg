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
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"

	prow_config "k8s.io/test-infra/prow/config"
	config_pb "k8s.io/test-infra/testgrid/config"
)

type SQConfig struct {
	Data map[string]string `yaml:"data,omitempty"`
}

var (
	companies = []string{
		"canonical",
		"cos",
		"cri-o",
		"istio",
		"google",
		"kopeio",
		"redhat",
		"vmware",
		"gardener",
	}
	orgs = []string{
		"conformance",
		"presubmits",
		"sig",
		"wg",
	}
	dashboardPrefixes = [][]string{orgs, companies}

	// gcs prefixes populated by the kubernetes prow instance
	prowGcsPrefixes = []string{
		"kubernetes-jenkins/logs/",
		"kubernetes-jenkins/pr-logs/directory/",
	}
)

const (
	prowPath    = "../../../prow/config.yaml"
	jobPath     = "../../../config/jobs"
	defaultPath = "../../../config/testgrids/default.yaml"
)

// Shared testgrid config, loaded at TestMain.
var cfg *config_pb.Configuration

func TestMain(m *testing.M) {
	//make sure we can parse configurations
	yamlFiles := []string{"../../../config/testgrids"}

	var c Config
	b, err := ioutil.ReadFile(defaultPath)
	if err != nil {
		fmt.Printf("Could not read default config: %v", err)
	}
	if err := c.UpdateDefaults(b); err != nil {
		fmt.Printf("Could not update defaults: %v", err)
	}

	if err := readToConfig(&c, yamlFiles); err != nil {
		fmt.Printf("Could not read testgrid config: %v", err)
		os.Exit(1)
	}

	pca := &prow_config.Agent{}
	if err := pca.Start(prowPath, jobPath); err != nil {
		fmt.Printf("Prow config agent error: %v", err)
		os.Exit(1)
	}

	if err := applyProwjobAnnotations(&c, pca); err != nil {
		fmt.Printf("Couldn't apply prowjob annotations: %v", err)
		os.Exit(1)
	}

	cfg, err = c.Raw()
	if err != nil {
		fmt.Printf("Error validating config: %v", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestConfig(t *testing.T) {
	// testgroup - occurrence map, validate testgroups
	testgroupMap := make(map[string]int32)

	for testgroupidx, testgroup := range cfg.TestGroups {
		// All testgroup must have a name and a query
		if testgroup.Name == "" || testgroup.GcsPrefix == "" {
			t.Errorf("Testgroup #%v (Name: '%v', GcsPrefix: '%v'): - Must have a name and gcs_prefix",
				testgroupidx, testgroup.Name, testgroup.GcsPrefix)
		}

		// All testgroup must not have duplicated names
		if testgroupMap[testgroup.Name] > 0 {
			t.Errorf("Duplicated Testgroup: %v", testgroup.Name)
		} else {
			testgroupMap[testgroup.Name] = 1
		}

		if !testgroup.IsExternal {
			t.Errorf("Testgroup %v: IsExternal should always be true!", testgroup.Name)
		}

		if !testgroup.UseKubernetesClient {
			t.Errorf("Testgroup %v: UseKubernetesClient should always be true!", testgroup.Name)
		}

		for _, prowGcsPrefix := range prowGcsPrefixes {
			if strings.Contains(testgroup.GcsPrefix, prowGcsPrefix) {
				// The expectation is that testgroup.Name is the name of a Prow job and the GCSPrefix
				// follows the convention kubernetes-jenkins/logs/.../jobName
				// The final part of the prefix should be the job name.
				expected := filepath.Join(filepath.Dir(testgroup.GcsPrefix), testgroup.Name)
				if expected != testgroup.GcsPrefix {
					t.Errorf("Kubernetes Testgroup %v GcsPrefix; Got %v; Want %v", testgroup.Name, testgroup.GcsPrefix, expected)
				}
				break // out of prowGcsPrefix for loop
			}
		}

		if testgroup.TestNameConfig != nil {
			if testgroup.TestNameConfig.NameFormat == "" {
				t.Errorf("Testgroup %v: NameFormat must not be empty!", testgroup.Name)
			}

			if len(testgroup.TestNameConfig.NameElements) != strings.Count(testgroup.TestNameConfig.NameFormat, "%") {
				t.Errorf("Testgroup %v: TestNameConfig must have number NameElement equal to format count in NameFormat!", testgroup.Name)
			}
		}

		// All PR testgroup has num_columns_recent equals 20
		if strings.HasPrefix(testgroup.GcsPrefix, "kubernetes-jenkins/pr-logs/directory/") {
			if testgroup.NumColumnsRecent < 20 {
				t.Errorf("Testgroup %v: num_columns_recent: must be greater than 20 for presubmit jobs!", testgroup.Name)
			}
		}
	}

	// dashboard name set
	dashboardmap := make(map[string]bool)

	for dashboardidx, dashboard := range cfg.Dashboards {
		// All dashboard must have a name
		if dashboard.Name == "" {
			t.Errorf("Dashboard %v: - Must have a name", dashboardidx)
		}

		found := false
		for _, kind := range dashboardPrefixes {
			for _, prefix := range kind {
				if strings.HasPrefix(dashboard.Name, prefix+"-") || dashboard.Name == prefix {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("Dashboard %v: must prefix with one of: %v", dashboard.Name, dashboardPrefixes)
		}

		// All dashboard must not have duplicated names
		if dashboardmap[dashboard.Name] {
			t.Errorf("Duplicated dashboard: %v", dashboard.Name)
		} else {
			dashboardmap[dashboard.Name] = true
		}

		// All dashboard must have at least one tab
		if len(dashboard.DashboardTab) == 0 {
			t.Errorf("Dashboard %v: - Must have more than one dashboardtab", dashboard.Name)
		}

		// dashboardtabSet is a set that checks duplicate tab name within each dashboard
		dashboardtabSet := sets.NewString()

		// dashboardtestgroupSet is a set that checks duplicate testgroups within each dashboard
		dashboardtestgroupSet := sets.NewString()

		// All notifications in dashboard must have a summary
		if len(dashboard.Notifications) != 0 {
			for notificationindex, notification := range dashboard.Notifications {
				if notification.Summary == "" {
					t.Errorf("Notification %v in dashboard %v: - Must have a summary", notificationindex, dashboard.Name)
				}
			}
		}

		for tabindex, dashboardtab := range dashboard.DashboardTab {

			// All dashboardtab must have a name and a testgroup
			if dashboardtab.Name == "" || dashboardtab.TestGroupName == "" {
				t.Errorf("Dashboard %v, tab %v: - Must have a name and a testgroup name", dashboard.Name, tabindex)
			}

			// All dashboardtab within a dashboard must not have duplicated names
			if dashboardtabSet.Has(dashboardtab.Name) {
				t.Errorf("Duplicated name in dashboard %s: %v", dashboard.Name, dashboardtab.Name)
			} else {
				dashboardtabSet.Insert(dashboardtab.Name)
			}

			// All dashboardtab within a dashboard must not have duplicated testgroupnames
			if dashboardtestgroupSet.Has(dashboardtab.TestGroupName) {
				t.Errorf("Duplicated testgroupnames in dashboard %s: %v", dashboard.Name, dashboardtab.TestGroupName)
			} else {
				dashboardtestgroupSet.Insert(dashboardtab.TestGroupName)
			}

			// All testgroup in dashboard must be defined in testgroups
			if testgroupMap[dashboardtab.TestGroupName] == 0 {
				t.Errorf("Dashboard %v, tab %v: - Testgroup %v must be defined first",
					dashboard.Name, dashboardtab.Name, dashboardtab.TestGroupName)
			} else {
				testgroupMap[dashboardtab.TestGroupName]++
			}

			if dashboardtab.AlertOptions != nil && (dashboardtab.AlertOptions.AlertStaleResultsHours != 0 || dashboardtab.AlertOptions.NumFailuresToAlert != 0) {
				for _, testgroup := range cfg.TestGroups {
					// Disallow alert options in tab but not group.
					// Disallow different alert options in tab vs. group.
					if testgroup.Name == dashboardtab.TestGroupName {
						if testgroup.AlertStaleResultsHours == 0 {
							t.Errorf("Cannot define alert_stale_results_hours in DashboardTab %v and not TestGroup %v.", dashboardtab.Name, dashboardtab.TestGroupName)
						}
						if testgroup.NumFailuresToAlert == 0 {
							t.Errorf("Cannot define num_failures_to_alert in DashboardTab %v and not TestGroup %v.", dashboardtab.Name, dashboardtab.TestGroupName)
						}
						if testgroup.AlertStaleResultsHours != dashboardtab.AlertOptions.AlertStaleResultsHours {
							t.Errorf("alert_stale_results_hours for DashboardTab %v must match TestGroup %v.", dashboardtab.Name, dashboardtab.TestGroupName)
						}
						if testgroup.NumFailuresToAlert != dashboardtab.AlertOptions.NumFailuresToAlert {
							t.Errorf("num_failures_to_alert for DashboardTab %v must match TestGroup %v.", dashboardtab.Name, dashboardtab.TestGroupName)
						}
					}
				}
			}
		}
	}

	// No dup of dashboard groups, and no dup dashboard in a dashboard group
	groups := make(map[string]bool)
	tabs := make(map[string]string)

	for idx, dashboardGroup := range cfg.DashboardGroups {
		// All dashboard must have a name
		if dashboardGroup.Name == "" {
			t.Errorf("DashboardGroup %v: - DashboardGroup must have a name", idx)
		}

		found := false
		for _, kind := range dashboardPrefixes {
			for _, prefix := range kind {
				if strings.HasPrefix(dashboardGroup.Name, prefix+"-") || prefix == dashboardGroup.Name {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("Dashboard group %v: must prefix with one of: %v", dashboardGroup.Name, dashboardPrefixes)
		}

		// All dashboardgroup must not have duplicated names
		if _, ok := groups[dashboardGroup.Name]; ok {
			t.Errorf("Duplicated dashboard: %v", dashboardGroup.Name)
		} else {
			groups[dashboardGroup.Name] = true
		}

		if _, ok := dashboardmap[dashboardGroup.Name]; ok {
			t.Errorf("%v is both a dashboard and dashboard group name.", dashboardGroup.Name)
		}

		for _, dashboard := range dashboardGroup.DashboardNames {
			// All dashboard must not have duplicated names
			if exist, ok := tabs[dashboard]; ok {
				t.Errorf("Duplicated dashboard %v in dashboard group %v and %v", dashboard, exist, dashboardGroup.Name)
			} else {
				tabs[dashboard] = dashboardGroup.Name
			}

			if _, ok := dashboardmap[dashboard]; !ok {
				t.Errorf("Dashboard %v needs to be defined before adding to a dashboard group!", dashboard)
			}

			if !strings.HasPrefix(dashboard, dashboardGroup.Name+"-") {
				t.Errorf("Dashboard %v in group %v must have the group name as a prefix", dashboard, dashboardGroup.Name)
			}
		}

		// Dashboards that match this dashboard group's prefix should be a part of it
		for dashboard := range dashboardmap {
			if strings.HasPrefix(dashboard, dashboardGroup.Name+"-") {
				group, ok := tabs[dashboard]
				if !ok {
					t.Errorf("Dashboard %v should be in dashboard_group %v", dashboard, dashboardGroup.Name)
				} else if group != dashboardGroup.Name {
					t.Errorf("Dashboard %v should be in dashboard_group %v instead of dashboard_group %v", dashboard, dashboardGroup.Name, group)
				}
			}
		}
	}

	// All Testgroup should be mapped to one or more tabs
	missedTestgroups := false
	for testgroupname, occurrence := range testgroupMap {
		if occurrence == 1 {
			t.Errorf("Testgroup %v - defined but not used in any dashboards.", testgroupname)
			missedTestgroups = true
		}
	}
	if missedTestgroups {
		t.Logf("Note: Testgroups are automatically defined for postsubmits and periodics.")
		t.Logf("Testgroups can be added to a dashboard either by using the `testgrid-dashboards` annotation on the prowjob, or by adding them to testgrid/config.yaml")
	}
}

// TODO(spiffxp): These are all repos that don't have their presubmits in testgrid.
// Convince sig leads or subproject owners this is a bad idea and whittle this down
// to just kubernetes-security/
var noPresubmitsInTestgridPrefixes = []string{
	"containerd/cri",
	"GoogleCloudPlatform/k8s-multicluster-ingress",
	"kubeflow/pipelines",
	"kubernetes-csi/csi-driver-host-path",
	"kubernetes-csi/csi-driver-flex",
	"kubernetes-csi/csi-lib-utils",
	"kubernetes-csi/csi-release-tools",
	"kubernetes-csi/csi-test",
	"kubernetes-csi/external-attacher",
	"kubernetes-csi/external-provisioner",
	"kubernetes-csi/external-snapshotter",
	"kubernetes-csi/livenessprobe",
	"kubernetes-csi/node-driver-registrar",
	"kubernetes-sigs/gcp-compute-persistent-disk-csi-driver",
	"kubernetes-sigs/gcp-filestore-csi-driver",
	"kubernetes-sigs/kind",
	"kubernetes-sigs/kubebuilder-declarative-pattern",
	"kubernetes-sigs/service-catalog",
	"kubernetes-sigs/sig-storage-local-static-provisioner",
	"kubernetes-sigs/slack-infra",
	"kubernetes-sigs/testing_frameworks",
	"kubernetes/client-go",
	"kubernetes/cloud-provider-gcp",
	"kubernetes/cloud-provider-openstack",
	"kubernetes/dns",
	"kubernetes/enhancements",
	"kubernetes/ingress-gce",
	"kubernetes/k8s.io",
	"kubernetes/kubeadm",
	"kubernetes/minikube",
	// This is the one entry that should be here
	"kubernetes-security/",
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func TestKubernetesProwInstanceJobsMustHaveMatchingTestgridEntries(t *testing.T) {
	jobs := make(map[string]bool)

	prowConfig, err := prow_config.Load(prowPath, jobPath)
	if err != nil {
		t.Fatalf("Could not load prow configs: %v\n", err)
	}

	for repo, presubmits := range prowConfig.Presubmits {
		if hasAnyPrefix(repo, noPresubmitsInTestgridPrefixes) {
			continue
		}
		for _, job := range presubmits {
			jobs[job.Name] = false
		}
	}

	for _, job := range prowConfig.AllPostsubmits([]string{}) {
		jobs[job.Name] = false
	}

	for _, job := range prowConfig.AllPeriodics() {
		jobs[job.Name] = false
	}

	// Ignore any test groups that get their results from a gcs prefix
	// that is not populated by the kubernetes prow instance
	testgroups := make(map[string]bool)
	for _, testgroup := range cfg.TestGroups {
		for _, prowGcsPrefix := range prowGcsPrefixes {
			if strings.Contains(testgroup.GcsPrefix, prowGcsPrefix) {
				// The convention is that the job name is the final part of the GcsPrefix
				job := filepath.Base(testgroup.GcsPrefix)
				testgroups[job] = false
				break // to next testgroup
			}
		}
	}

	// Each job running in the kubernetes prow instance must have an
	// identically named test_groups entry in the kubernetes testgrid config
	for job := range jobs {
		if _, ok := testgroups[job]; ok {
			testgroups[job] = true
			jobs[job] = true
		}
	}

	// Conclusion
	badjobs := []string{}
	for job, valid := range jobs {
		if !valid {
			badjobs = append(badjobs, job)
			t.Errorf("Job %v does not have a matching testgrid testgroup", job)
		}
	}

	badconfigs := []string{}
	for testgroup, valid := range testgroups {
		if !valid {
			badconfigs = append(badconfigs, testgroup)
			t.Errorf("Testgrid group %v does not have a matching jenkins or prow job", testgroup)
		}
	}
}

func TestReleaseBlockingJobsMustHaveTestgridDescriptions(t *testing.T) {
	// TODO(spiffxp): start with master, enforce for all release branches
	re := regexp.MustCompile("^sig-release-master-(blocking|informing)$")
	for _, dashboard := range cfg.Dashboards {
		if !re.MatchString(dashboard.Name) {
			continue
		}
		suffix := re.FindStringSubmatch(dashboard.Name)[1]
		for _, dashboardtab := range dashboard.DashboardTab {
			intro := fmt.Sprintf("dashboard_tab %v/%v is release-%v", dashboard.Name, dashboardtab.Name, suffix)
			if dashboardtab.Name == "" {
				t.Errorf("%v: - Must have a name", intro)
			}
			if dashboardtab.TestGroupName == "" {
				t.Errorf("%v: - Must have a test_group_name", intro)
			}
			if dashboardtab.Description == "" {
				t.Errorf("%v: - Must have a description", intro)
			}
			// TODO(spiffxp): enforce for informing as well
			if suffix == "informing" {
				if !strings.HasPrefix(dashboardtab.Description, "OWNER: ") {
					t.Logf("NOTICE: %v: - Must have a description that starts with OWNER: ", intro)
				}
				if dashboardtab.AlertOptions == nil {
					t.Logf("NOTICE: %v: - Must have alert_options", intro)
				} else if dashboardtab.AlertOptions.AlertMailToAddresses == "" {
					t.Logf("NOTICE: %v: - Must have alert_options.alert_mail_to_addresses", intro)
				}
			} else {
				if dashboardtab.AlertOptions == nil {
					t.Errorf("%v: - Must have alert_options", intro)
				} else if dashboardtab.AlertOptions.AlertMailToAddresses == "" {
					t.Errorf("%v: - Must have alert_options.alert_mail_to_addresses", intro)
				}
			}
		}
	}
}

func TestNoEmpyMailToAddresses(t *testing.T) {
	for _, dashboard := range cfg.Dashboards {
		for _, dashboardtab := range dashboard.DashboardTab {
			intro := fmt.Sprintf("dashboard_tab %v/%v", dashboard.Name, dashboardtab.Name)
			if dashboardtab.AlertOptions != nil {
				mails := strings.Split(dashboardtab.AlertOptions.AlertMailToAddresses, ",")
				for _, m := range mails {
					_, err := mail.ParseAddress(m)
					if err != nil {
						t.Errorf("%v: - invalid alert_mail_to_address '%v': %v", intro, m, err)
					}
				}
			}
		}
	}
}
