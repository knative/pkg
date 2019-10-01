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

// checkconfig loads configuration for Prow to validate it
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"

	v1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/errorutil"
	needsrebase "k8s.io/test-infra/prow/external-plugins/needs-rebase/plugin"
	"k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
	_ "k8s.io/test-infra/prow/hook"
	"k8s.io/test-infra/prow/labels"
	"k8s.io/test-infra/prow/logrusutil"
	"k8s.io/test-infra/prow/plugins"
	"k8s.io/test-infra/prow/plugins/approve"
	"k8s.io/test-infra/prow/plugins/blockade"
	"k8s.io/test-infra/prow/plugins/blunderbuss"
	"k8s.io/test-infra/prow/plugins/bugzilla"
	"k8s.io/test-infra/prow/plugins/cherrypickunapproved"
	"k8s.io/test-infra/prow/plugins/hold"
	"k8s.io/test-infra/prow/plugins/lgtm"
	ownerslabel "k8s.io/test-infra/prow/plugins/owners-label"
	"k8s.io/test-infra/prow/plugins/releasenote"
	"k8s.io/test-infra/prow/plugins/trigger"
	verifyowners "k8s.io/test-infra/prow/plugins/verify-owners"
	"k8s.io/test-infra/prow/plugins/wip"
)

type options struct {
	configPath    string
	jobConfigPath string
	pluginConfig  string

	warnings        flagutil.Strings
	excludeWarnings flagutil.Strings
	strict          bool
	expensive       bool

	github flagutil.GitHubOptions
}

func reportWarning(strict bool, errs errorutil.Aggregate) {
	for _, item := range errs.Strings() {
		logrus.Warn(item)
	}
	if strict {
		logrus.Fatal("Strict is set and there were warnings")
	}
}

func (o *options) warningEnabled(warning string) bool {
	return sets.NewString(o.warnings.Strings()...).Difference(sets.NewString(o.excludeWarnings.Strings()...)).Has(warning)
}

const (
	mismatchedTideWarning        = "mismatched-tide"
	mismatchedTideLenientWarning = "mismatched-tide-lenient"
	tideStrictBranchWarning      = "tide-strict-branch"
	nonDecoratedJobsWarning      = "non-decorated-jobs"
	jobNameLengthWarning         = "long-job-names"
	needsOkToTestWarning         = "needs-ok-to-test"
	validateOwnersWarning        = "validate-owners"
	missingTriggerWarning        = "missing-trigger"
	validateURLsWarning          = "validate-urls"
	unknownFieldsWarning         = "unknown-fields"
	verifyOwnersFilePresence     = "verify-owners-presence"
)

var defaultWarnings = []string{
	mismatchedTideWarning,
	tideStrictBranchWarning,
	mismatchedTideLenientWarning,
	nonDecoratedJobsWarning,
	jobNameLengthWarning,
	needsOkToTestWarning,
	validateOwnersWarning,
	missingTriggerWarning,
	validateURLsWarning,
	unknownFieldsWarning,
}

var expensiveWarnings = []string{
	verifyOwnersFilePresence,
}

func getAllWarnings() []string {
	var all []string
	all = append(all, defaultWarnings...)
	all = append(all, expensiveWarnings...)

	return all
}

func (o *options) Validate() error {
	allWarnings := getAllWarnings()
	if o.configPath == "" {
		return errors.New("required flag --config-path was unset")
	}
	for _, warning := range o.warnings.Strings() {
		found := false
		for _, registeredWarning := range allWarnings {
			if warning == registeredWarning {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no such warning %q, valid warnings: %v", warning, allWarnings)
		}
	}
	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&o.configPath, "config-path", "", "Path to config.yaml.")
	fs.StringVar(&o.jobConfigPath, "job-config-path", "", "Path to prow job configs.")
	fs.StringVar(&o.pluginConfig, "plugin-config", "", "Path to plugin config file.")
	fs.Var(&o.warnings, "warnings", "Warnings to validate. Use repeatedly to provide a list of warnings")
	fs.Var(&o.excludeWarnings, "exclude-warning", "Warnings to exclude. Use repeatedly to provide a list of warnings to exclude")
	fs.BoolVar(&o.expensive, "expensive-checks", false, "If set, additional expensive warnings will be enabled")
	fs.BoolVar(&o.strict, "strict", false, "If set, consider all warnings as errors.")
	o.github.AddFlagsWithoutDefaultGitHubTokenPath(fs)
	fs.Parse(os.Args[1:])
	return o
}

func main() {
	logrusutil.ComponentInit("checkconfig")

	o := gatherOptions()
	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	// use all warnings by default
	if len(o.warnings.Strings()) == 0 {
		if o.expensive {
			o.warnings = flagutil.NewStrings(getAllWarnings()...)
		} else {
			o.warnings = flagutil.NewStrings(defaultWarnings...)
		}
	}

	configAgent := config.Agent{}
	if err := configAgent.Start(o.configPath, o.jobConfigPath); err != nil {
		logrus.WithError(err).Fatal("Error loading Prow config.")
	}
	cfg := configAgent.Config()

	pluginAgent := plugins.ConfigAgent{}
	var pcfg *plugins.Configuration
	if o.pluginConfig != "" {
		if err := pluginAgent.Load(o.pluginConfig); err != nil {
			logrus.WithError(err).Fatal("Error loading Prow plugin config.")
		}
		pcfg = pluginAgent.Config()
	}

	// the following checks are useful in finding user errors but their
	// presence won't lead to strictly incorrect behavior, so we can
	// detect them here but don't necessarily want to stop config re-load
	// in all components on their failure.
	var errs []error
	if pcfg != nil && o.warningEnabled(verifyOwnersFilePresence) {
		if o.github.TokenPath == "" {
			logrus.Fatal("Cannot verify OWNERS file presence without a GitHub token")
		}
		secretAgent := &secret.Agent{}
		if o.github.TokenPath != "" {
			if err := secretAgent.Start([]string{o.github.TokenPath}); err != nil {
				logrus.WithError(err).Fatal("Error starting secrets agent.")
			}
		}

		githubClient, err := o.github.GitHubClient(secretAgent, false)
		if err != nil {
			logrus.WithError(err).Fatal("Error getting GitHub client.")
		}
		githubClient.Throttle(3000, 100) // 300 hourly tokens, bursts of 100
		// 404s are expected to happen, no point in retrying
		githubClient.SetMax404Retries(0)

		if err := verifyOwnersPresence(pcfg, githubClient); err != nil {
			errs = append(errs, err)
		}
	}
	if pcfg != nil && o.warningEnabled(mismatchedTideWarning) {
		if err := validateTideRequirements(cfg, pcfg, true); err != nil {
			errs = append(errs, err)
		}
	} else if pcfg != nil && o.warningEnabled(mismatchedTideLenientWarning) {
		if err := validateTideRequirements(cfg, pcfg, false); err != nil {
			errs = append(errs, err)
		}
	}
	if o.warningEnabled(nonDecoratedJobsWarning) {
		if err := validateDecoratedJobs(cfg); err != nil {
			errs = append(errs, err)
		}
	}
	if o.warningEnabled(jobNameLengthWarning) {
		if err := validateJobRequirements(cfg.JobConfig); err != nil {
			errs = append(errs, err)
		}
	}
	if o.warningEnabled(needsOkToTestWarning) {
		if err := validateNeedsOkToTestLabel(cfg); err != nil {
			errs = append(errs, err)
		}
	}
	if pcfg != nil && o.warningEnabled(validateOwnersWarning) {
		if err := verifyOwnersPlugin(pcfg); err != nil {
			errs = append(errs, err)
		}
	}
	if pcfg != nil && o.warningEnabled(missingTriggerWarning) {
		if err := validateTriggers(cfg, pcfg); err != nil {
			errs = append(errs, err)
		}
	}
	if pcfg != nil && o.warningEnabled(validateURLsWarning) {
		if err := validateURLs(cfg.ProwConfig); err != nil {
			errs = append(errs, err)
		}
	}
	if o.warningEnabled(unknownFieldsWarning) {
		cfgBytes, err := ioutil.ReadFile(o.configPath)
		if err != nil {
			logrus.WithError(err).Fatal("Error loading Prow config for validation.")
		}
		if err := validateUnknownFields(cfg, cfgBytes, o.configPath); err != nil {
			errs = append(errs, err)
		}
	}
	if pcfg != nil && o.warningEnabled(unknownFieldsWarning) {
		pcfgBytes, err := ioutil.ReadFile(o.pluginConfig)
		if err != nil {
			logrus.WithError(err).Fatal("Error loading Prow plugin config for validation.")
		}
		if err := validateUnknownFields(pcfg, pcfgBytes, o.pluginConfig); err != nil {
			errs = append(errs, err)
		}
	}
	if o.warningEnabled(tideStrictBranchWarning) {
		if err := validateStrictBranches(cfg.ProwConfig); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		reportWarning(o.strict, errorutil.NewAggregate(errs...))
		return
	}

	logrus.Info("checkconfig passes without any error!")
}
func policyIsStrict(p config.Policy) bool {
	if p.Protect == nil || !*p.Protect {
		return false
	}
	if p.RequiredStatusChecks == nil || p.RequiredStatusChecks.Strict == nil {
		return false
	}
	return *p.RequiredStatusChecks.Strict
}

func strictBranchesConfig(c config.ProwConfig) (*orgRepoConfig, error) {
	strictOrgExceptions := make(map[string]sets.String)
	strictRepos := sets.NewString()
	for orgName := range c.BranchProtection.Orgs {
		org := c.BranchProtection.GetOrg(orgName)
		// First find explicitly configured repos and partition based on strictness.
		// If any branch in a repo is strict we assume that the whole repo is to
		// simplify this validation.
		strictExplicitRepos, nonStrictExplicitRepos := sets.NewString(), sets.NewString()
		for repoName := range org.Repos {
			repo := org.GetRepo(repoName)
			strict := policyIsStrict(repo.Policy)
			if !strict {
				for branchName := range repo.Branches {
					branch, err := repo.GetBranch(branchName)
					if err != nil {
						return nil, err
					}
					if policyIsStrict(branch.Policy) {
						strict = true
						break
					}
				}
			}
			fullRepoName := fmt.Sprintf("%s/%s", orgName, repoName)
			if strict {
				strictExplicitRepos.Insert(fullRepoName)
			} else {
				nonStrictExplicitRepos.Insert(fullRepoName)
			}
		}
		// Done partitioning the repos.

		if policyIsStrict(org.Policy) {
			// This org is strict, record with repo exceptions (blacklist).
			strictOrgExceptions[orgName] = nonStrictExplicitRepos
		} else {
			// The org is not strict, record member repos that are (whitelist).
			strictRepos.Insert(strictExplicitRepos.UnsortedList()...)
		}
	}
	return newOrgRepoConfig(strictOrgExceptions, strictRepos), nil
}

func validateStrictBranches(c config.ProwConfig) error {
	const explanation = "See #5: https://github.com/kubernetes/test-infra/blob/master/prow/cmd/tide/maintainers.md#best-practices Also note that this validation is imperfect, see the check-config code for details"
	if len(c.Tide.Queries) == 0 {
		// Short circuit here so that we can allow global level branchprotector
		// 'strict: true' if Tide is not enabled.
		// Ignoring the case where Tide is enabled only on orgs/repos specifically
		// exempted from the global setting simplifies validation immensely.
		return nil
	}
	if policyIsStrict(c.BranchProtection.Policy) {
		return fmt.Errorf("strict branchprotection context requirements cannot be globally enabled when Tide is configured for use. %s", explanation)
	}
	// The two assumptions below are not necessarily true, but they hold for all
	// known instances and make this validation much simpler.

	// Assumes if any branch is managed by Tide, the whole repo is.
	overallTideConfig := newOrgRepoConfig(c.Tide.Queries.OrgExceptionsAndRepos())
	// Assumes if any branch is strict the repo is strict.
	strictBranchConfig, err := strictBranchesConfig(c)
	if err != nil {
		return err
	}

	conflicts := overallTideConfig.intersection(strictBranchConfig).items()
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf(
		"the following enable strict branchprotection context requirements even though Tide handles merges: [%s]. %s",
		strings.Join(conflicts, "; "),
		explanation,
	)
}

func validateURLs(c config.ProwConfig) error {
	var validationErrs []error

	if _, err := url.Parse(c.StatusErrorLink); err != nil {
		validationErrs = append(validationErrs, fmt.Errorf("status_error_link is not a valid url: %s", c.StatusErrorLink))
	}

	return errorutil.NewAggregate(validationErrs...)
}

func validateUnknownFields(cfg interface{}, cfgBytes []byte, filePath string) error {
	var obj interface{}
	err := yaml.Unmarshal(cfgBytes, &obj)
	if err != nil {
		return fmt.Errorf("error while unmarshaling yaml: %v", err)
	}
	unknownFields := checkUnknownFields("", obj, reflect.ValueOf(cfg))
	if len(unknownFields) > 0 {
		sort.Strings(unknownFields)
		return fmt.Errorf("unknown fields present in %s: %v", filePath, strings.Join(unknownFields, ", "))
	}
	return nil
}

func checkUnknownFields(keyPref string, obj interface{}, cfg reflect.Value) []string {
	var uf []string
	switch concreteVal := obj.(type) {
	case map[string]interface{}:
		// Iterate over map and check every value
		for key, val := range concreteVal {
			fullKey := fmt.Sprintf("%s.%s", keyPref, key)
			subCfg := getSubCfg(key, cfg)
			if !subCfg.IsValid() {
				// Append fullKey without leading "."
				uf = append(uf, fullKey[1:])
			} else {
				subUf := checkUnknownFields(fullKey, val, subCfg)
				uf = append(uf, subUf...)
			}
		}
	case []interface{}:
		for i, val := range concreteVal {
			fullKey := fmt.Sprintf("%s[%v]", keyPref, i)
			var subCfg reflect.Value
			if cfg.Kind() == reflect.Ptr {
				subCfg = cfg.Elem().Index(i)
			} else {
				subCfg = cfg.Index(i)
			}
			uf = append(uf, checkUnknownFields(fullKey, val, subCfg)...)
		}
	}
	return uf
}

func getSubCfg(key string, cfg reflect.Value) reflect.Value {
	cfgElem := cfg
	if cfg.Kind() == reflect.Interface || cfg.Kind() == reflect.Ptr {
		cfgElem = cfg.Elem()
	}
	switch cfgElem.Kind() {
	case reflect.Map:
		for _, k := range cfgElem.MapKeys() {
			strK := fmt.Sprintf("%v", k.Interface())
			if strK == key {
				return cfgElem.MapIndex(k)
			}
		}
	case reflect.Struct:
		for i := 0; i < cfgElem.NumField(); i++ {
			structField := cfgElem.Type().Field(i)
			// Check if field is embedded struct
			if structField.Anonymous {
				subStruct := getSubCfg(key, cfgElem.Field(i))
				if subStruct.IsValid() {
					return subStruct
				}
			} else {
				field := getJSONTagName(structField)
				if field == key {
					return cfgElem.Field(i)
				}
			}
		}
	}
	return reflect.Value{}
}

func getJSONTagName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		if commaIdx := strings.Index(jsonTag, ","); commaIdx > 0 {
			return jsonTag[:commaIdx]
		}
		return jsonTag
	}
	return ""
}

func validateJobRequirements(c config.JobConfig) error {
	var validationErrs []error
	for repo, jobs := range c.Presubmits {
		for _, job := range jobs {
			validationErrs = append(validationErrs, validatePresubmitJob(repo, job))
		}
	}
	for repo, jobs := range c.Postsubmits {
		for _, job := range jobs {
			validationErrs = append(validationErrs, validatePostsubmitJob(repo, job))
		}
	}
	for _, job := range c.Periodics {
		validationErrs = append(validationErrs, validatePeriodicJob(job))
	}

	return errorutil.NewAggregate(validationErrs...)
}

func validatePresubmitJob(repo string, job config.Presubmit) error {
	var validationErrs []error
	// Prow labels k8s resources with job names. Labels are capped at 63 chars.
	if job.Agent == string(v1.KubernetesAgent) && len(job.Name) > validation.LabelValueMaxLength {
		validationErrs = append(validationErrs, fmt.Errorf("name of Presubmit job %q (for repo %q) too long (should be at most 63 characters)", job.Name, repo))
	}
	return errorutil.NewAggregate(validationErrs...)
}

func validatePostsubmitJob(repo string, job config.Postsubmit) error {
	var validationErrs []error
	// Prow labels k8s resources with job names. Labels are capped at 63 chars.
	if job.Agent == string(v1.KubernetesAgent) && len(job.Name) > validation.LabelValueMaxLength {
		validationErrs = append(validationErrs, fmt.Errorf("name of Postsubmit job %q (for repo %q) too long (should be at most 63 characters)", job.Name, repo))
	}
	return errorutil.NewAggregate(validationErrs...)
}

func validatePeriodicJob(job config.Periodic) error {
	var validationErrs []error
	// Prow labels k8s resources with job names. Labels are capped at 63 chars.
	if job.Agent == string(v1.KubernetesAgent) && len(job.Name) > validation.LabelValueMaxLength {
		validationErrs = append(validationErrs, fmt.Errorf("name of Periodic job %q too long (should be at most 63 characters)", job.Name))
	}
	return errorutil.NewAggregate(validationErrs...)
}

func validateTideRequirements(cfg *config.Config, pcfg *plugins.Configuration, includeForbidden bool) error {
	type matcher struct {
		// matches determines if the tide query appropriately honors the
		// label in question -- whether by requiring it or forbidding it
		matches func(label string, query config.TideQuery) bool
		// verb is used in forming error messages
		verb string
	}
	requires := matcher{
		matches: func(label string, query config.TideQuery) bool {
			return sets.NewString(query.Labels...).Has(label)
		},
		verb: "require",
	}
	forbids := matcher{
		matches: func(label string, query config.TideQuery) bool {
			return sets.NewString(query.MissingLabels...).Has(label)
		},
		verb: "forbid",
	}

	type plugin struct {
		// name and label identify the relationship we are validating
		name, label string
		// external indicates plugin is external or not
		external bool
		// matcher determines if the tide query appropriately honors the
		// label in question -- whether by requiring it or forbidding it
		matcher matcher
		// config holds the orgs and repos for which tide does honor the
		// label; this container is populated conditionally from queries
		// using the matcher
		config *orgRepoConfig
	}
	// configs list relationships between tide config
	// and plugin enablement that we want to validate
	configs := []plugin{
		{name: lgtm.PluginName, label: labels.LGTM, matcher: requires},
		{name: approve.PluginName, label: labels.Approved, matcher: requires},
	}
	if includeForbidden {
		configs = append(configs,
			plugin{name: hold.PluginName, label: labels.Hold, matcher: forbids},
			plugin{name: wip.PluginName, label: labels.WorkInProgress, matcher: forbids},
			plugin{name: bugzilla.PluginName, label: labels.InvalidBug, matcher: forbids},
			plugin{name: verifyowners.PluginName, label: labels.InvalidOwners, matcher: forbids},
			plugin{name: releasenote.PluginName, label: releasenote.ReleaseNoteLabelNeeded, matcher: forbids},
			plugin{name: cherrypickunapproved.PluginName, label: labels.CpUnapproved, matcher: forbids},
			plugin{name: blockade.PluginName, label: labels.BlockedPaths, matcher: forbids},
			plugin{name: needsrebase.PluginName, label: labels.NeedsRebase, external: true, matcher: forbids},
		)
	}

	for i := range configs {
		// For each plugin determine the subset of tide queries that match and then
		// the orgs and repos that the subset matches.
		var matchingQueries config.TideQueries
		for _, query := range cfg.Tide.Queries {
			if configs[i].matcher.matches(configs[i].label, query) {
				matchingQueries = append(matchingQueries, query)
			}
		}
		configs[i].config = newOrgRepoConfig(matchingQueries.OrgExceptionsAndRepos())
	}

	overallTideConfig := newOrgRepoConfig(cfg.Tide.Queries.OrgExceptionsAndRepos())

	// Now actually execute the checks we just configured.
	var validationErrs []error
	for _, pluginConfig := range configs {
		err := ensureValidConfiguration(
			pluginConfig.name,
			pluginConfig.label,
			pluginConfig.matcher.verb,
			pluginConfig.config,
			overallTideConfig,
			enabledOrgReposForPlugin(pcfg, pluginConfig.name, pluginConfig.external),
		)
		validationErrs = append(validationErrs, err)
	}

	return errorutil.NewAggregate(validationErrs...)
}

func newOrgRepoConfig(orgExceptions map[string]sets.String, repos sets.String) *orgRepoConfig {
	return &orgRepoConfig{
		orgExceptions: orgExceptions,
		repos:         repos,
	}
}

// orgRepoConfig describes a set of repositories with an explicit
// whitelist and a mapping of blacklists for owning orgs
type orgRepoConfig struct {
	// orgExceptions holds explicit blacklists of repos for owning orgs
	orgExceptions map[string]sets.String
	// repos is a whitelist of repos
	repos sets.String
}

func (c *orgRepoConfig) items() []string {
	items := make([]string, 0, len(c.orgExceptions)+len(c.repos))
	for org, excepts := range c.orgExceptions {
		item := fmt.Sprintf("org: %s", org)
		if excepts.Len() > 0 {
			item = fmt.Sprintf("%s without repo(s) %s", item, strings.Join(excepts.List(), ", "))
			for _, repo := range excepts.List() {
				item = fmt.Sprintf("%s '%s'", item, repo)
			}
		}
		items = append(items, item)
	}
	for _, repo := range c.repos.List() {
		items = append(items, fmt.Sprintf("repo: %s", repo))
	}
	return items
}

// difference returns a new orgRepoConfig that represents the set difference of
// the repos specified by the receiver and the parameter orgRepoConfigs.
func (c *orgRepoConfig) difference(c2 *orgRepoConfig) *orgRepoConfig {
	res := &orgRepoConfig{
		orgExceptions: make(map[string]sets.String),
		repos:         sets.NewString().Union(c.repos),
	}
	for org, excepts1 := range c.orgExceptions {
		if excepts2, ok := c2.orgExceptions[org]; ok {
			res.repos.Insert(excepts2.Difference(excepts1).UnsortedList()...)
		} else {
			excepts := sets.NewString().Union(excepts1)
			// Add any applicable repos in repos2 to excepts
			for _, repo := range c2.repos.UnsortedList() {
				if parts := strings.SplitN(repo, "/", 2); len(parts) == 2 && parts[0] == org {
					excepts.Insert(repo)
				}
			}
			res.orgExceptions[org] = excepts
		}
	}

	res.repos = res.repos.Difference(c2.repos)

	for _, repo := range res.repos.UnsortedList() {
		if parts := strings.SplitN(repo, "/", 2); len(parts) == 2 {
			if excepts2, ok := c2.orgExceptions[parts[0]]; ok && !excepts2.Has(repo) {
				res.repos.Delete(repo)
			}
		}
	}
	return res
}

// intersection returns a new orgRepoConfig that represents the set intersection
// of the repos specified by the receiver and the parameter orgRepoConfigs.
func (c *orgRepoConfig) intersection(c2 *orgRepoConfig) *orgRepoConfig {
	res := &orgRepoConfig{
		orgExceptions: make(map[string]sets.String),
		repos:         sets.NewString(),
	}
	for org, excepts1 := range c.orgExceptions {
		// Include common orgs, but union exceptions.
		if excepts2, ok := c2.orgExceptions[org]; ok {
			res.orgExceptions[org] = excepts1.Union(excepts2)
		} else {
			// Include right side repos that match left side org.
			for _, repo := range c2.repos.UnsortedList() {
				if parts := strings.SplitN(repo, "/", 2); len(parts) == 2 && parts[0] == org && !excepts1.Has(repo) {
					res.repos.Insert(repo)
				}
			}
		}
	}
	for _, repo := range c.repos.UnsortedList() {
		if c2.repos.Has(repo) {
			res.repos.Insert(repo)
		} else if parts := strings.SplitN(repo, "/", 2); len(parts) == 2 {
			// Include left side repos that match right side org.
			if excepts2, ok := c2.orgExceptions[parts[0]]; ok && !excepts2.Has(repo) {
				res.repos.Insert(repo)
			}
		}
	}
	return res
}

// union returns a new orgRepoConfig that represents the set union of the
// repos specified by the receiver and the parameter orgRepoConfigs
func (c *orgRepoConfig) union(c2 *orgRepoConfig) *orgRepoConfig {
	res := &orgRepoConfig{
		orgExceptions: make(map[string]sets.String),
		repos:         sets.NewString(),
	}

	for org, excepts1 := range c.orgExceptions {
		// keep only items in both blacklists that are not in the
		// explicit repo whitelists for the other configuration;
		// we know from how the orgRepoConfigs are constructed that
		// a org blacklist won't intersect it's own repo whitelist
		pruned := excepts1.Difference(c2.repos)
		if excepts2, ok := c2.orgExceptions[org]; ok {
			res.orgExceptions[org] = pruned.Intersection(excepts2.Difference(c.repos))
		} else {
			res.orgExceptions[org] = pruned
		}
	}

	for org, excepts2 := range c2.orgExceptions {
		// update any blacklists not previously updated
		if _, exists := res.orgExceptions[org]; !exists {
			res.orgExceptions[org] = excepts2.Difference(c.repos)
		}
	}

	// we need to prune out repos in the whitelists which are
	// covered by an org already; we know from above that no
	// org blacklist in the result will contain a repo whitelist
	for _, repo := range c.repos.Union(c2.repos).UnsortedList() {
		parts := strings.SplitN(repo, "/", 2)
		if len(parts) != 2 {
			logrus.Warnf("org/repo %q is formatted incorrectly", repo)
			continue
		}
		if _, exists := res.orgExceptions[parts[0]]; !exists {
			res.repos.Insert(repo)
		}
	}
	return res
}

func enabledOrgReposForPlugin(c *plugins.Configuration, plugin string, external bool) *orgRepoConfig {
	var (
		orgs  []string
		repos []string
	)
	if external {
		orgs, repos = c.EnabledReposForExternalPlugin(plugin)
	} else {
		orgs, repos = c.EnabledReposForPlugin(plugin)
	}
	orgMap := make(map[string]sets.String, len(orgs))
	for _, org := range orgs {
		orgMap[org] = nil
	}
	return newOrgRepoConfig(orgMap, sets.NewString(repos...))
}

// ensureValidConfiguration enforces rules about tide and plugin config.
// In this context, a subset is the set of repos or orgs for which a specific
// plugin is either enabled (for plugins) or required for merge (for tide). The
// tide superset is every org or repo that has any configuration at all in tide.
// Specifically:
//   - every item in the tide subset must also be in the plugins subset
//   - every item in the plugins subset that is in the tide superset must also be in the tide subset
// For example:
//   - if org/repo is configured in tide to require lgtm, it must have the lgtm plugin enabled
//   - if org/repo is configured in tide, the tide configuration must require the same set of
//     plugins as are configured. If the repository has LGTM and approve enabled, the tide query
//     must require both labels
func ensureValidConfiguration(plugin, label, verb string, tideSubSet, tideSuperSet, pluginsSubSet *orgRepoConfig) error {
	notEnabled := tideSubSet.difference(pluginsSubSet).items()
	notRequired := pluginsSubSet.intersection(tideSuperSet).difference(tideSubSet).items()

	var configErrors []error
	if len(notEnabled) > 0 {
		configErrors = append(configErrors, fmt.Errorf("the following orgs or repos %s the %s label for merging but do not enable the %s plugin: %v", verb, label, plugin, notEnabled))
	}
	if len(notRequired) > 0 {
		configErrors = append(configErrors, fmt.Errorf("the following orgs or repos enable the %s plugin but do not %s the %s label for merging: %v", plugin, verb, label, notRequired))
	}

	return errorutil.NewAggregate(configErrors...)
}

func validateDecoratedJobs(cfg *config.Config) error {
	var nonDecoratedJobs []string
	for _, presubmit := range cfg.AllPresubmits([]string{}) {
		if presubmit.Agent == string(v1.KubernetesAgent) && !presubmit.Decorate {
			nonDecoratedJobs = append(nonDecoratedJobs, presubmit.Name)
		}
	}

	for _, postsubmit := range cfg.AllPostsubmits([]string{}) {
		if postsubmit.Agent == string(v1.KubernetesAgent) && !postsubmit.Decorate {
			nonDecoratedJobs = append(nonDecoratedJobs, postsubmit.Name)
		}
	}

	for _, periodic := range cfg.AllPeriodics() {
		if periodic.Agent == string(v1.KubernetesAgent) && !periodic.Decorate {
			nonDecoratedJobs = append(nonDecoratedJobs, periodic.Name)
		}
	}

	if len(nonDecoratedJobs) > 0 {
		return fmt.Errorf("the following jobs use the kubernetes provider but do not use the pod utilities: %v", nonDecoratedJobs)
	}
	return nil
}

func validateNeedsOkToTestLabel(cfg *config.Config) error {
	var queryErrors []error
	for i, query := range cfg.Tide.Queries {
		for _, label := range query.Labels {
			if label == lgtm.LGTMLabel {
				for _, label := range query.MissingLabels {
					if label == labels.NeedsOkToTest {
						queryErrors = append(queryErrors, fmt.Errorf(
							"the tide query at position %d"+
								"forbids the %q label and requires the %q label, "+
								"which is not recommended; "+
								"see https://github.com/kubernetes/test-infra/blob/master/prow/cmd/tide/maintainers.md#best-practices "+
								"for more information",
							i, labels.NeedsOkToTest, lgtm.LGTMLabel),
						)
					}
				}
			}
		}
	}
	return errorutil.NewAggregate(queryErrors...)
}

func pluginsWithOwnersFile() string {
	return strings.Join([]string{approve.PluginName, blunderbuss.PluginName, ownerslabel.PluginName}, ", ")
}

func orgReposUsingOwnersFile(cfg *plugins.Configuration) *orgRepoConfig {
	// we do not know the set of repos that use OWNERS, but we
	// can get a reasonable proxy for this by looking at where
	// the `approve', `blunderbuss' and `owners-label' plugins
	// are enabled
	approveConfig := enabledOrgReposForPlugin(cfg, approve.PluginName, false)
	blunderbussConfig := enabledOrgReposForPlugin(cfg, blunderbuss.PluginName, false)
	ownersLabelConfig := enabledOrgReposForPlugin(cfg, ownerslabel.PluginName, false)
	return approveConfig.union(blunderbussConfig).union(ownersLabelConfig)
}

type FileInRepoExistsChecker interface {
	GetRepos(org string, isUser bool) ([]github.Repo, error)
	GetFile(org, repo, filepath, commit string) ([]byte, error)
}

func verifyOwnersPresence(cfg *plugins.Configuration, rc FileInRepoExistsChecker) error {
	ownersConfig := orgReposUsingOwnersFile(cfg)

	var missing []string
	for org, excluded := range ownersConfig.orgExceptions {
		repos, err := rc.GetRepos(org, false)
		if err != nil {
			return err
		}

		for _, repo := range repos {
			if excluded.Has(repo.FullName) || repo.Archived {
				continue
			}
			if _, err := rc.GetFile(repo.Owner.Login, repo.Name, "OWNERS", ""); err != nil {
				if _, nf := err.(*github.FileNotFound); nf {
					missing = append(missing, repo.FullName)
				} else {
					return fmt.Errorf("got error: %v", err)
				}
			}
		}
	}

	for repo := range ownersConfig.repos {
		items := strings.Split(repo, "/")
		if len(items) != 2 {
			return fmt.Errorf("bad repository '%s', expected org/repo format", repo)
		}
		if _, err := rc.GetFile(items[0], items[1], "OWNERS", ""); err != nil {
			if _, nf := err.(*github.FileNotFound); nf {
				missing = append(missing, repo)
			} else {
				return fmt.Errorf("got error: %v", err)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("the following orgs or repos enable at least one"+
			" plugin that uses OWNERS files (%s), but its master branch does not contain"+
			" a root level OWNERS file: %v", pluginsWithOwnersFile(), missing)
	}
	return nil
}

func verifyOwnersPlugin(cfg *plugins.Configuration) error {
	ownersConfig := orgReposUsingOwnersFile(cfg)
	validateOwnersConfig := enabledOrgReposForPlugin(cfg, verifyowners.PluginName, false)

	invalid := ownersConfig.difference(validateOwnersConfig).items()
	if len(invalid) > 0 {
		return fmt.Errorf("the following orgs or repos "+
			"enable at least one plugin that uses OWNERS files (%s) "+
			"but do not enable the %s plugin to ensure validity of OWNERS files: %v",
			pluginsWithOwnersFile(), verifyowners.PluginName, invalid,
		)
	}
	return nil
}

func validateTriggers(cfg *config.Config, pcfg *plugins.Configuration) error {
	configuredRepos := sets.NewString()
	for orgRepo := range cfg.JobConfig.Presubmits {
		configuredRepos.Insert(orgRepo)
	}
	for orgRepo := range cfg.JobConfig.Postsubmits {
		configuredRepos.Insert(orgRepo)
	}

	configured := newOrgRepoConfig(map[string]sets.String{}, configuredRepos)
	enabled := enabledOrgReposForPlugin(pcfg, trigger.PluginName, false)

	if missing := configured.difference(enabled).items(); len(missing) > 0 {
		return fmt.Errorf("the following repos have jobs configured but do not have the %s plugin enabled: %s", trigger.PluginName, strings.Join(missing, ", "))
	}
	return nil
}
