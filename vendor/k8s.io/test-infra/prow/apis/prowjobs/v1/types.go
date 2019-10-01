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

package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"strings"
	"time"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	prowgithub "k8s.io/test-infra/prow/github"
)

// ProwJobType specifies how the job is triggered.
type ProwJobType string

// Various job types.
const (
	// PresubmitJob means it runs on unmerged PRs.
	PresubmitJob ProwJobType = "presubmit"
	// PostsubmitJob means it runs on each new commit.
	PostsubmitJob ProwJobType = "postsubmit"
	// Periodic job means it runs on a time-basis, unrelated to git changes.
	PeriodicJob ProwJobType = "periodic"
	// BatchJob tests multiple unmerged PRs at the same time.
	BatchJob ProwJobType = "batch"
)

// ProwJobState specifies whether the job is running
type ProwJobState string

// Various job states.
const (
	// TriggeredState means the job has been created but not yet scheduled.
	TriggeredState ProwJobState = "triggered"
	// PendingState means the job is scheduled but not yet running.
	PendingState ProwJobState = "pending"
	// SuccessState means the job completed without error (exit 0)
	SuccessState ProwJobState = "success"
	// FailureState means the job completed with errors (exit non-zero)
	FailureState ProwJobState = "failure"
	// AbortedState means prow killed the job early (new commit pushed, perhaps).
	AbortedState ProwJobState = "aborted"
	// ErrorState means the job could not schedule (bad config, perhaps).
	ErrorState ProwJobState = "error"
)

// ProwJobAgent specifies the controller (such as plank or jenkins-agent) that runs the job.
type ProwJobAgent string

const (
	// KubernetesAgent means prow will create a pod to run this job.
	KubernetesAgent ProwJobAgent = "kubernetes"
	// JenkinsAgent means prow will schedule the job on jenkins.
	JenkinsAgent ProwJobAgent = "jenkins"
	// KnativeBuildAgent means prow will schedule the job via a build-crd resource.
	KnativeBuildAgent ProwJobAgent = "knative-build"
	// TektonAgent means prow will schedule the job via a tekton PipelineRun CRD resource.
	TektonAgent = "tekton-pipeline"
)

const (
	// DefaultClusterAlias specifies the default cluster key to schedule jobs.
	DefaultClusterAlias = "default"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProwJob contains the spec as well as runtime metadata.
type ProwJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProwJobSpec   `json:"spec,omitempty"`
	Status ProwJobStatus `json:"status,omitempty"`
}

// ProwJobSpec configures the details of the prow job.
//
// Details include the podspec, code to clone, the cluster it runs
// any child jobs, concurrency limitations, etc.
type ProwJobSpec struct {
	// Type is the type of job and informs how
	// the jobs is triggered
	Type ProwJobType `json:"type,omitempty"`
	// Agent determines which controller fulfills
	// this specific ProwJobSpec and runs the job
	Agent ProwJobAgent `json:"agent,omitempty"`
	// Cluster is which Kubernetes cluster is used
	// to run the job, only applicable for that
	// specific agent
	Cluster string `json:"cluster,omitempty"`
	// Namespace defines where to create pods/resources.
	Namespace string `json:"namespace,omitempty"`
	// Job is the name of the job
	Job string `json:"job,omitempty"`
	// Refs is the code under test, determined at
	// runtime by Prow itself
	Refs *Refs `json:"refs,omitempty"`
	// ExtraRefs are auxiliary repositories that
	// need to be cloned, determined from config
	ExtraRefs []Refs `json:"extra_refs,omitempty"`
	// Report determines if the result of this job should
	// be posted as a status on GitHub
	Report bool `json:"report,omitempty"`
	// Context is the name of the status context used to
	// report back to GitHub
	Context string `json:"context,omitempty"`
	// RerunCommand is the command a user would write to
	// trigger this job on their pull request
	RerunCommand string `json:"rerun_command,omitempty"`
	// MaxConcurrency restricts the total number of instances
	// of this job that can run in parallel at once
	MaxConcurrency int `json:"max_concurrency,omitempty"`
	// ErrorOnEviction indicates that the ProwJob should be completed and given
	// the ErrorState status if the pod that is executing the job is evicted.
	// If this field is unspecified or false, a new pod will be created to replace
	// the evicted one.
	ErrorOnEviction bool `json:"error_on_eviction,omitempty"`

	// PodSpec provides the basis for running the test under
	// a Kubernetes agent
	PodSpec *corev1.PodSpec `json:"pod_spec,omitempty"`

	// BuildSpec provides the basis for running the test as
	// a build-crd resource
	// https://github.com/knative/build
	BuildSpec *buildv1alpha1.BuildSpec `json:"build_spec,omitempty"`

	// JenkinsSpec holds configuration specific to Jenkins jobs
	JenkinsSpec *JenkinsSpec `json:"jenkins_spec,omitempty"`

	// PipelineRunSpec provides the basis for running the test as
	// a pipeline-crd resource
	// https://github.com/tektoncd/pipeline
	PipelineRunSpec *pipelinev1alpha1.PipelineRunSpec `json:"pipeline_run_spec,omitempty"`

	// DecorationConfig holds configuration options for
	// decorating PodSpecs that users provide
	DecorationConfig *DecorationConfig `json:"decoration_config,omitempty"`

	// ReporterConfig holds reporter-specific configuration
	ReporterConfig *ReporterConfig `json:"reporter_config,omitempty"`

	// RerunAuthConfig holds information about which users can rerun the job
	RerunAuthConfig RerunAuthConfig `json:"rerun_auth_config,omitempty"`
}

type GitHubTeamSlug struct {
	Slug string `json:"slug"`
	Org  string `json:"org"`
}

type RerunAuthConfig struct {
	// If AllowAnyone is set to true, any user can rerun the job
	AllowAnyone bool `json:"allow_anyone,omitempty"`
	// GitHubTeams contains IDs of GitHub teams of users who can rerun the job
	// If you know the name of a team and the org it belongs to,
	// you can look up its ID using this command, where the team slug is the hyphenated name:
	// curl -H "Authorization: token <token>" "https://api.github.com/orgs/<org-name>/teams/<team slug>"
	// or, to list all teams in a given org, use
	// curl -H "Authorization: token <token>" "https://api.github.com/orgs/<org-name>/teams"
	GitHubTeamIDs []int `json:"github_team_ids,omitempty"`
	// GitHubTeamSlugs contains slugs and orgs of teams of users who can rerun the job
	GitHubTeamSlugs []GitHubTeamSlug `json:"github_team_slugs,omitempty"`
	// GitHubUsers contains names of individual users who can rerun the job
	GitHubUsers []string `json:"github_users,omitempty"`
}

// IsSpecifiedUser returns true if AllowAnyone is set to true or if the given user is
// specified as a permitted GitHubUser
func (rac *RerunAuthConfig) IsAuthorized(user string, cli prowgithub.RerunClient) (bool, error) {
	if rac.AllowAnyone {
		return true, nil
	}
	for _, u := range rac.GitHubUsers {
		if prowgithub.NormLogin(u) == prowgithub.NormLogin(user) {
			return true, nil
		}
	}
	// if there is no client, no token was provided, so we cannot access the teams
	if cli == nil {
		return false, nil
	}
	for _, ght := range rac.GitHubTeamIDs {
		member, err := cli.TeamHasMember(ght, user)
		if err != nil {
			return false, fmt.Errorf("GitHub failed to fetch members of team %v, verify that you have the correct team number and access token: %v", ght, err)
		}
		if member {
			return true, nil
		}
	}
	for _, ghts := range rac.GitHubTeamSlugs {
		team, err := cli.GetTeamBySlug(ghts.Slug, ghts.Org)
		if err != nil {
			return false, fmt.Errorf("GitHub failed to fetch team with slug %s and org %s: %v", ghts.Slug, ghts.Org, err)
		}
		member, err := cli.TeamHasMember(team.ID, user)
		if err != nil {
			return false, fmt.Errorf("GitHub failed to fetch members of team %v: %v", team, err)
		}
		if member {
			return true, nil
		}
	}
	return false, nil
}

type ReporterConfig struct {
	Slack *SlackReporterConfig `json:"slack,omitempty"`
}

type SlackReporterConfig struct {
	Channel string `json:"channel"`
}

// Duration is a wrapper around time.Duration that parses times in either
// 'integer number of nanoseconds' or 'duration string' formats and serializes
// to 'duration string' format.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &d.Duration); err == nil {
		// b was an integer number of nanoseconds.
		return nil
	}
	// b was not an integer. Assume that it is a duration string.

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	pd, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	d.Duration = pd
	return nil
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

// DecorationConfig specifies how to augment pods.
//
// This is primarily used to provide automatic integration with gubernator
// and testgrid.
type DecorationConfig struct {
	// Timeout is how long the pod utilities will wait
	// before aborting a job with SIGINT.
	Timeout *Duration `json:"timeout,omitempty"`
	// GracePeriod is how long the pod utilities will wait
	// after sending SIGINT to send SIGKILL when aborting
	// a job. Only applicable if decorating the PodSpec.
	GracePeriod *Duration `json:"grace_period,omitempty"`

	// UtilityImages holds pull specs for utility container
	// images used to decorate a PodSpec.
	UtilityImages *UtilityImages `json:"utility_images,omitempty"`
	// GCSConfiguration holds options for pushing logs and
	// artifacts to GCS from a job.
	GCSConfiguration *GCSConfiguration `json:"gcs_configuration,omitempty"`
	// GCSCredentialsSecret is the name of the Kubernetes secret
	// that holds GCS push credentials.
	GCSCredentialsSecret string `json:"gcs_credentials_secret,omitempty"`
	// SSHKeySecrets are the names of Kubernetes secrets that contain
	// SSK keys which should be used during the cloning process.
	SSHKeySecrets []string `json:"ssh_key_secrets,omitempty"`
	// SSHHostFingerprints are the fingerprints of known SSH hosts
	// that the cloning process can trust.
	// Create with ssh-keyscan [-t rsa] host
	SSHHostFingerprints []string `json:"ssh_host_fingerprints,omitempty"`
	// SkipCloning determines if we should clone source code in the
	// initcontainers for jobs that specify refs
	SkipCloning *bool `json:"skip_cloning,omitempty"`
	// CookieFileSecret is the name of a kubernetes secret that contains
	// a git http.cookiefile, which should be used during the cloning process.
	CookiefileSecret string `json:"cookiefile_secret,omitempty"`
}

// ApplyDefault applies the defaults for the ProwJob decoration. If a field has a zero value, it
// replaces that with the value set in def.
func (d *DecorationConfig) ApplyDefault(def *DecorationConfig) *DecorationConfig {
	if d == nil && def == nil {
		return nil
	}
	var merged DecorationConfig
	if d != nil {
		merged = *d
	} else {
		merged = *def
	}
	if d == nil || def == nil {
		return &merged
	}
	merged.UtilityImages = merged.UtilityImages.ApplyDefault(def.UtilityImages)
	merged.GCSConfiguration = merged.GCSConfiguration.ApplyDefault(def.GCSConfiguration)

	if merged.Timeout == nil {
		merged.Timeout = def.Timeout
	}
	if merged.GracePeriod == nil {
		merged.GracePeriod = def.GracePeriod
	}
	if merged.GCSCredentialsSecret == "" {
		merged.GCSCredentialsSecret = def.GCSCredentialsSecret
	}
	if len(merged.SSHKeySecrets) == 0 {
		merged.SSHKeySecrets = def.SSHKeySecrets
	}
	if len(merged.SSHHostFingerprints) == 0 {
		merged.SSHHostFingerprints = def.SSHHostFingerprints
	}
	if merged.SkipCloning == nil {
		merged.SkipCloning = def.SkipCloning
	}
	if merged.CookiefileSecret == "" {
		merged.CookiefileSecret = def.CookiefileSecret
	}

	return &merged
}

// Validate ensures all the values set in the DecorationConfig are valid.
func (d *DecorationConfig) Validate() error {
	if d.UtilityImages == nil {
		return errors.New("utility image config is not specified")
	}
	var missing []string
	if d.UtilityImages.CloneRefs == "" {
		missing = append(missing, "clonerefs")
	}
	if d.UtilityImages.InitUpload == "" {
		missing = append(missing, "initupload")
	}
	if d.UtilityImages.Entrypoint == "" {
		missing = append(missing, "entrypoint")
	}
	if d.UtilityImages.Sidecar == "" {
		missing = append(missing, "sidecar")
	}
	if len(missing) > 0 {
		return fmt.Errorf("the following utility images are not specified: %q", missing)
	}

	if d.GCSConfiguration == nil {
		return errors.New("GCS upload configuration is not specified")
	}
	if d.GCSCredentialsSecret == "" {
		return errors.New("GCS upload credential secret is not specified")
	}
	if err := d.GCSConfiguration.Validate(); err != nil {
		return fmt.Errorf("GCS configuration is invalid: %v", err)
	}
	return nil
}

func (d *Duration) Get() time.Duration {
	if d == nil {
		return 0
	}
	return d.Duration
}

// UtilityImages holds pull specs for the utility images
// to be used for a job
type UtilityImages struct {
	// CloneRefs is the pull spec used for the clonerefs utility
	CloneRefs string `json:"clonerefs,omitempty"`
	// InitUpload is the pull spec used for the initupload utility
	InitUpload string `json:"initupload,omitempty"`
	// Entrypoint is the pull spec used for the entrypoint utility
	Entrypoint string `json:"entrypoint,omitempty"`
	// sidecar is the pull spec used for the sidecar utility
	Sidecar string `json:"sidecar,omitempty"`
}

// ApplyDefault applies the defaults for the UtilityImages decorations. If a field has a zero value,
// it replaces that with the value set in def.
func (u *UtilityImages) ApplyDefault(def *UtilityImages) *UtilityImages {
	if u == nil {
		return def
	} else if def == nil {
		return u
	}

	merged := *u
	if merged.CloneRefs == "" {
		merged.CloneRefs = def.CloneRefs
	}
	if merged.InitUpload == "" {
		merged.InitUpload = def.InitUpload
	}
	if merged.Entrypoint == "" {
		merged.Entrypoint = def.Entrypoint
	}
	if merged.Sidecar == "" {
		merged.Sidecar = def.Sidecar
	}
	return &merged
}

// PathStrategy specifies minutia about how to construct the url.
// Usually consumed by gubernator/testgrid.
const (
	PathStrategyLegacy   = "legacy"
	PathStrategySingle   = "single"
	PathStrategyExplicit = "explicit"
)

// GCSConfiguration holds options for pushing logs and
// artifacts to GCS from a job.
type GCSConfiguration struct {
	// Bucket is the GCS bucket to upload to
	Bucket string `json:"bucket,omitempty"`
	// PathPrefix is an optional path that follows the
	// bucket name and comes before any structure
	PathPrefix string `json:"path_prefix,omitempty"`
	// PathStrategy dictates how the org and repo are used
	// when calculating the full path to an artifact in GCS
	PathStrategy string `json:"path_strategy,omitempty"`
	// DefaultOrg is omitted from GCS paths when using the
	// legacy or simple strategy
	DefaultOrg string `json:"default_org,omitempty"`
	// DefaultRepo is omitted from GCS paths when using the
	// legacy or simple strategy
	DefaultRepo string `json:"default_repo,omitempty"`
	// MediaTypes holds additional extension media types to add to Go's
	// builtin's and the local system's defaults.  This maps extensions
	// to media types, for example: MediaTypes["log"] = "text/plain"
	MediaTypes map[string]string `json:"mediaTypes,omitempty"`

	// LocalOutputDir specifies a directory where files should be copied INSTEAD of uploading to GCS.
	// This option is useful for testing jobs that use the pod-utilities without actually uploading.
	LocalOutputDir string `json:"local_output_dir,omitempty"`
}

// ApplyDefault applies the defaults for GCSConfiguration decorations. If a field has a zero value,
// it replaces that with the value set in def.
func (g *GCSConfiguration) ApplyDefault(def *GCSConfiguration) *GCSConfiguration {
	if g == nil && def == nil {
		return nil
	}
	var merged GCSConfiguration
	if g != nil {
		merged = *g
	} else {
		merged = *def
	}
	if g == nil || def == nil {
		return &merged
	}

	if merged.Bucket == "" {
		merged.Bucket = def.Bucket
	}
	if merged.PathPrefix == "" {
		merged.PathPrefix = def.PathPrefix
	}
	if merged.PathStrategy == "" {
		merged.PathStrategy = def.PathStrategy
	}
	if merged.DefaultOrg == "" {
		merged.DefaultOrg = def.DefaultOrg
	}
	if merged.DefaultRepo == "" {
		merged.DefaultRepo = def.DefaultRepo
	}

	for extension, mediaType := range def.MediaTypes {
		merged.MediaTypes[extension] = mediaType
	}
	for extension, mediaType := range g.MediaTypes {
		merged.MediaTypes[extension] = mediaType
	}

	if merged.LocalOutputDir == "" {
		merged.LocalOutputDir = def.LocalOutputDir
	}
	return &merged
}

// Validate ensures all the values set in the GCSConfiguration are valid.
func (g *GCSConfiguration) Validate() error {
	for _, mediaType := range g.MediaTypes {
		if _, _, err := mime.ParseMediaType(mediaType); err != nil {
			return fmt.Errorf("invalid extension media type %q: %v", mediaType, err)
		}
	}
	if g.PathStrategy != PathStrategyLegacy && g.PathStrategy != PathStrategyExplicit && g.PathStrategy != PathStrategySingle {
		return fmt.Errorf("gcs_path_strategy must be one of %q, %q, or %q", PathStrategyLegacy, PathStrategyExplicit, PathStrategySingle)
	}
	if g.PathStrategy != PathStrategyExplicit && (g.DefaultOrg == "" || g.DefaultRepo == "") {
		return fmt.Errorf("default org and repo must be provided for GCS strategy %q", g.PathStrategy)
	}
	return nil
}

// ProwJobStatus provides runtime metadata, such as when it finished, whether it is running, etc.
type ProwJobStatus struct {
	StartTime      metav1.Time  `json:"startTime,omitempty"`
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	State          ProwJobState `json:"state,omitempty"`
	Description    string       `json:"description,omitempty"`
	URL            string       `json:"url,omitempty"`

	// PodName applies only to ProwJobs fulfilled by
	// plank. This field should always be the same as
	// the ProwJob.ObjectMeta.Name field.
	PodName string `json:"pod_name,omitempty"`

	// BuildID is the build identifier vended either by tot
	// or the snowflake library for this job and used as an
	// identifier for grouping artifacts in GCS for views in
	// TestGrid and Gubernator. Idenitifiers vended by tot
	// are monotonically increasing whereas identifiers vended
	// by the snowflake library are not.
	BuildID string `json:"build_id,omitempty"`

	// JenkinsBuildID applies only to ProwJobs fulfilled
	// by the jenkins-operator. This field is the build
	// identifier that Jenkins gave to the build for this
	// ProwJob.
	JenkinsBuildID string `json:"jenkins_build_id,omitempty"`

	// PrevReportStates stores the previous reported prowjob state per reporter
	// So crier won't make duplicated report attempt
	PrevReportStates map[string]ProwJobState `json:"prev_report_states,omitempty"`
}

// Complete returns true if the prow job has finished
func (j *ProwJob) Complete() bool {
	// TODO(fejta): support a timeout?
	return j.Status.CompletionTime != nil
}

// SetComplete marks the job as completed (at time now).
func (j *ProwJob) SetComplete() {
	j.Status.CompletionTime = new(metav1.Time)
	*j.Status.CompletionTime = metav1.Now()
}

// ClusterAlias specifies the key in the clusters map to use.
//
// This allows scheduling a prow job somewhere aside from the default build cluster.
func (j *ProwJob) ClusterAlias() string {
	if j.Spec.Cluster == "" {
		return DefaultClusterAlias
	}
	return j.Spec.Cluster
}

// Pull describes a pull request at a particular point in time.
type Pull struct {
	Number int    `json:"number"`
	Author string `json:"author"`
	SHA    string `json:"sha"`
	Title  string `json:"title,omitempty"`

	// Ref is git ref can be checked out for a change
	// for example,
	// github: pull/123/head
	// gerrit: refs/changes/00/123/1
	Ref string `json:"ref,omitempty"`
	// Link links to the pull request itself.
	Link string `json:"link,omitempty"`
	// CommitLink links to the commit identified by the SHA.
	CommitLink string `json:"commit_link,omitempty"`
	// AuthorLink links to the author of the pull request.
	AuthorLink string `json:"author_link,omitempty"`
}

// Refs describes how the repo was constructed.
type Refs struct {
	// Org is something like kubernetes or k8s.io
	Org string `json:"org"`
	// Repo is something like test-infra
	Repo string `json:"repo"`
	// RepoLink links to the source for Repo.
	RepoLink string `json:"repo_link,omitempty"`

	BaseRef string `json:"base_ref,omitempty"`
	BaseSHA string `json:"base_sha,omitempty"`
	// BaseLink is a link to the commit identified by BaseSHA.
	BaseLink string `json:"base_link,omitempty"`

	Pulls []Pull `json:"pulls,omitempty"`

	// PathAlias is the location under <root-dir>/src
	// where this repository is cloned. If this is not
	// set, <root-dir>/src/github.com/org/repo will be
	// used as the default.
	PathAlias string `json:"path_alias,omitempty"`

	// WorkDir defines if the location of the cloned
	// repository will be used as the default working
	// directory.
	WorkDir bool `json:"workdir,omitempty"`

	// CloneURI is the URI that is used to clone the
	// repository. If unset, will default to
	// `https://github.com/org/repo.git`.
	CloneURI string `json:"clone_uri,omitempty"`
	// SkipSubmodules determines if submodules should be
	// cloned when the job is run. Defaults to true.
	SkipSubmodules bool `json:"skip_submodules,omitempty"`
	// CloneDepth is the depth of the clone that will be used.
	// A depth of zero will do a full clone.
	CloneDepth int `json:"clone_depth,omitempty"`
}

func (r Refs) String() string {
	rs := []string{}
	if r.BaseSHA != "" {
		rs = append(rs, fmt.Sprintf("%s:%s", r.BaseRef, r.BaseSHA))
	} else {
		rs = append(rs, r.BaseRef)
	}

	for _, pull := range r.Pulls {
		ref := fmt.Sprintf("%d:%s", pull.Number, pull.SHA)

		if pull.Ref != "" {
			ref = fmt.Sprintf("%s:%s", ref, pull.Ref)
		}

		rs = append(rs, ref)
	}
	return strings.Join(rs, ",")
}

// JenkinsSpec is optional parameters for Jenkins jobs.
// Currently, the only parameter supported is for telling
// jenkins-operator that the job is generated by the https://go.cloudbees.com/docs/plugins/github-branch-source/#github-branch-source plugin
type JenkinsSpec struct {
	GitHubBranchSourceJob bool `json:"github_branch_source_job,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProwJobList is a list of ProwJob resources
type ProwJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ProwJob `json:"items"`
}
