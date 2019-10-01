# Announcements

New features added to each component:
 - *July 30, 2019* `authorized_users` in `rerun_auth_config` for deck will become `github_users`. 
 - *July 19, 2019* deck will soon remove its default value for `--cookie-secret-file`.
   If you set `--oauth-url` but not `--cookie-secret-file`, add
   `--cookie-secret-file=/etc/cookie-secret` to your deck instance. The default value
   will be removed at the end of October 2019.
 - *July 2, 2019* prow defaults to report status for both presubmit and postsubmit
   jobs on GitHub now.
 - *June 17, 2019* It is now possible to configure the channel for the Slack reporter
   directly on jobs via the `.reporter_config.slack.channel` config option
 - *May 13, 2019* New `plank` config `pod_running_timeout` is added and
   defaulted to two days to allow plank abort pods stuck in running state.
 - *April 25, 2019* `--job-config` in `peribolos` has never been used; it is
   deprecated and will be removed in July 2019. Remove the flag from any calls
   to the tool.
 - *April 24, 2019* `file_weight_count` in blunderbuss is being deprecated in
   favour of the more current `max_request_count` functionality. Please ensure
   your configuration is up to date before the end of May 2019.
 - *March 12, 2019* tide now records a history of its actions and exposes a
   filterable view of these actions at the `/tide-history` deck path.
 - *March 9, 2019* prow components now support reading gzipped config files
 - *February 13, 2019* prow (both plank and crier) can set status on the commit
   for postsubmit jobs on github now!
   Type of jobs can be reported to github is gated by a config field like
   ```yaml
   github_reporter:
     job_types_to_report:
     - presubmit
     - postsubmit
   ```
   now and default to report for presubmit only.
   *** The default will change in April to include postsubmit jobs as well ***
   You can also add `skip_report: true` to your post-submit jobs to skip reporting
    if you enable postsubmit reporting on.
 - *January 15, 2019* `approve` now considers self-approval and github review
   state by default. Configure with `require_self_approval` and
   `ignore_review_state`. Temporarily revert to old defaults with `use_deprecated_2018_implicit_self_approve_default_migrate_before_july_2019` and `use_deprecated_2018_review_acts_as_approve_default_migrate_before_july_2019`.
 - *January 12, 2019* `blunderbluss` plugin now provides a new command, `/auto-cc`,
   that triggers automatic review requests.
 - *January 7, 2019* `implicit_self_approve` will become `require_self_approval` in
   the second half of this year.
 - *January 7, 2019* `review_acts_as_approve` will become `ignore_review_state` in
   the second half of this year.
 - *October 10, 2018* `tide` now supports the `-repo:foo/bar` tag in queries via
   the `excludedRepos` YAML field.
 - *October 3, 2018* `welcome` now supports a configurable message on a per-org,
   or per-repo basis. Please note that this includes a config schema change that
   will break previous deployments of this plugin.
 - *August 22, 2018* `spyglass` is a pluggable viewing framework for artifacts
   produced by Prowjobs. See a demo [here](https://prow.k8s.io/view/gcs/kubernetes-jenkins/logs/ci-kubernetes-e2e-gce-large-performance/121)!
 - *July 13, 2018* `blunderbluss` plugin will now support `required_reviewers` in
   OWNERS file to specify a person or github team to be cc'd on every PR that
   touches the corresponding part of the code.
 - *June 25, 2018* `updateconfig` plugin will now support update/remove keys
   from a glob match.
 - *June 05, 2018* `blunderbuss` plugin may now suggest approvers in addition
   to reviewers. Use `exclude_approvers: true` to revert to previous behavior.
 - *April 10, 2018* `cla` plugin now supports `/check-cla` command
   to force rechecking of the CLA status.
 - *February 1, 2018* `updateconfig` will now update any configmap on merge
 - *November 14, 2017* `jenkins-operator:0.58` exposes prometheus metrics.
 - *November 8, 2017* `horologium:0.14` prow periodic job now support cron
   triggers. See https://godoc.org/gopkg.in/robfig/cron.v2 for doc to the
   cron library we are using.

Breaking changes to external APIs (labels, GitHub interactions, configuration
or deployment) will be documented in this section. Prow is in a pre-release
state and no claims of backwards compatibility are made for any external API.
Note: versions specified in these announcements may not include bug fixes made
in more recent versions so it is recommended that the most recent versions are
used when updating deployments.

 - *May 2, 2019* All components exposing Prometheus metrics will now either push them
   to the Prometheus PushGateway, if configured, or serve them locally on port 9090 at
   `/metrics`, if not configured (the default).
 - *April 26, 2019* `blunderbuss`, `approve`, and other plugins that read OWNERS
   now treat `owners_dir_blacklist` as a list of regular expressions matched
   against the entire (repository-relative) directory path of the OWNERS file rather
   than as a list of strings matched exactly against the basename only of the directory
   containing the OWNERS file.
 - *April 2, 2019* `hook`, `deck`, `horologium`, `tide`, `plank` and `sinker` will no
   longer provide a default value for the `--config-path` flag.
   It is required to explicitly provide `--config-path` when upgrading to a new version of
   these components that were previously relying on the default `--config-path=/etc/config/config.yaml`.
 - *March 29, 2019* Custom logos should be provided as full paths in the configuration
   under `deck.branding.logos` and will not implicitly be assumed to be under the static
   assets directory.
 - *February 26, 2019* The `job_url_prefix` option from `plank` has been deprecated in
    favor of the new `job_url_prefix_config` option which allows configuration on a global,
    organization or repo level. `job_url_prefix` will be removed in September 2019.
 - *February 13, 2019* `horologium` and `sinker` deployments will soon require `--dry-run=false` in production, please set this before March 15. At that time flag will default to --dry-run=true instead of --dry-run=false.
 - *February 1, 2019* Now that `hook` and `tide` will no longer post "Skipped" statuses
   for jobs that do not need to run, it is not possible to require those statuses with
   branch protection. Therefore, it is necessary to run the `branchprotector` from at
   least version `510db59` before upgrading `tide` to that version.
 - *February 1, 2019* `horologium` and `sinker` now support the `--dry-run` flag,
   so you must pass `--dry-run=false` to keep the previous behavior (see Feb 13 update).
 - *January 31, 2019* `sub` no longer supports the `--masterurl` flag for connecting
   to the infrastructure cluster. Use `--kubeconfig` with `--context` for this.
 - *January 31, 2019* `crier` no longer supports the `--masterurl` flag for connecting
   to the infrastructure cluster. Use `--kubeconfig` with `--context` for this.
 - *January 27, 2019* Jobs that do not run will no longer post "Skipped" statuses.
 - *January 27, 2019* Jobs that do not run always will no longer be required by
   branch protection as they will not always produce a status. They will continue
   to be required for merge by `tide` if they are configured as required.
 - *January 27, 2019* All support for `run_after_success` jobs has been removed.
   Configuration of these jobs will continue to parse but will ignore the field.
 - *January 27, 2019* `hook` will now correctly honor the `run_always` field on Gerrit
   presubmits. Previously, if this field was unset it would have defaulted to `true`; now,
   it will correctly default to `false`.
 - *January 22, 2019* `sinker` prefers `.kube/config` instead of the custom `Clusters`
   file to specify credentials for external build clusters. The flag name has changed
   from `--build-cluster` to `--kubeconfig`. Migrate before June 2019.
 - *November 29, 2018* `plank` will no longer default jobs with `decorate: true`
   to have `automountServiceAccountToken: false` in their PodSpec if unset, if the
   job explicitly sets `serviceAccountName`
 - *November 26, 2018* job names must now match `^[A-Za-z0-9-._]+$`. Jobs that did not
   match this before were allowed but did not provide a good user experience.
 - *November 15, 2018* the `hook` service account now requires RBAC privileges
   to create `ConfigMaps` to support new functionality in the `updateconfig` plugin.
 - *November 9, 2018* Prow gerrit client label/annotations now have a `prow.k8s.io/` namespace
    prefix, if you have a gerrit deployment, please bump both cmd/gerrit and cmd/crier.
 - *November 8, 2018* `plank` now defaults jobs with `decorate: true` to have
   `automountServiceAccountToken: false` in their PodSpec if unset. Jobs that used the default
   service account should explicitly set this field to maintain functionality.
 - *October 16, 2018* Prow tls-cert management has been migrated from kube-lego to cert-manager.
 - *October 12, 2018* Removed deprecated `buildId` environment variable from prow jobs. Use `BUILD_ID.`
 - *October 3, 2018* `-github-token-file` replaced with
    `-github-token-path` for consistency with `branchprotector` and
    `peribolos` which were already using `-github-token-path`.
    `-github-token-file` will continue to work through the remainder
    of 2018, but it will be removed in early 2019.  The following
    commands are affected: `cherrypicker`, `hook`, `jenkins-operator`,
    `needs-rebase`, `phony`, `plank`, `refresh`, and `tide`.
 - *October 1, 2018* bazel is the one official way to build container images.
    Please use prow/bump.sh and/or bazel run //prow:release-push
 - *Sep 27, 2018* If you are setting explicit decorate configs, the format has changed from
    ```yaml
    - name: job-foo
      decorate: true
      timeout: 1
    ```
    to
    ```yaml
    - name: job-foo
      decorate: true
      decoration_config:
        timeout: 1
    ```
 - *September 24, 2018* the `splice` component has been deleted following the
   deletion of mungegithub.
 - *July 9, 2018* `milestone` format has changed from
    ```yaml
    milestone:
     maintainers_id: <some_team_id>
     maintainers_team: <some_team_name>
    ```
    to `repo_milestone`
    ```yaml
    repo_milestone:
     <some_repo_name>:
       maintainers_id: <some_team_id>
       maintainers_team: <some_team_name>
    ```
 - *July 2, 2018* the `trigger` plugin will now trust PRs from repo
   collaborators. Use `only_org_members: true` in the trigger config to
   temporarily disable this behavior.
 - *June 14, 2018* the `updateconfig` plugin will only add data to your `ConfigMaps`
   using the basename of the updated file, instead of using that and also duplicating
   the data using the name of the `ConfigMap` as a key
 - *June 1, 2018* all unquoted `boolean` fields in config.yaml that were unmarshall
   into type `string` now need to be quoted to avoid unmarshalling error.
 - *May 9, 2018* `deck` logs for jobs run as `Pods` will now return logs for the
   `"test"` container only.
 - *April 2, 2018* `updateconfig` format has been changed from
   ```yaml
   path/to/some/other/thing: configName
   ```
   to
   ```yaml
   path/to/some/other/thing:
     Name: configName
     # If unspecified, Namespace default to the value of ProwJobNamespace.
     Namespace: myNamespace
   ```
 - *March 15, 2018* `jenkins_operator` is removed from the config in favor of
   `jenkins_operators`.
 - *March 1, 2018* `MilestoneStatus` has been removed from the plugins Configuration
   in favor of the `Milestone` which is shared between two plugins: 1) `milestonestatus`
   and 2) `milestone`.  The milestonestatus plugin now uses the `Milestone` object to
   get the maintainers team ID
 - *February 27, 2018* `jenkins-operator` does not use `$BUILD_ID` as a fallback
   to `$PROW_JOB_ID` anymore.
 - *February 15, 2018* `jenkins-operator` can now accept the `--tot-url` flag
   and will use the connection to `tot` to vend build identifiers as `plank`
   does, giving control over where in GCS artifacts land to Prow and away from
   Jenkins. Furthermore, the `$BUILD_ID` variable in Jenkins jobs will now
   correctly point to the build identifier vended by `tot` and a new variable,
   `$PROW_JOB_ID`, points to the identifier used to link ProwJobs to Jenkins builds.
   `$PROW_JOB_ID` fallbacks to `$BUILD_ID` for backwards-compatibility, ie. to
   not break in-flight jobs during the time of the jenkins-operator update.
 - *February 1, 2018* The `config_updater` section in `plugins.yaml`
 now uses a `maps` object instead of `config_file`, `plugin_file` strings.
 Please switch over before July.
 - *November 30, 2017* If you use tide, you'll need to switch your query format
 and bump all prow component versions to reflect the changes in #5754.
 - *November 14, 2017* `horologium:0.17` fixes cron job not being scheduled.
 - *November 10, 2017* If you want to use cron feature in prow, you need to bump to:
 `hook:0.181`, `sinker:0.23`, `deck:0.62`, `splice:0.32`, `horologium:0.15`
 `plank:0.60`, `jenkins-operator:0.57` and `tide:0.12` to avoid error spamming from
 the config parser.
 - *November 7, 2017* `plank:0.56` fixes bug introduced in `plank:0.53` that
   affects controllers using an empty kubernetes selector.
 - *November 7, 2017* `jenkins-operator:0.51` provides jobs with the `$BUILD_ID`
   variable as well as the `$buildId` variable. The latter is deprecated and will
   be removed in a future version.
 - *November 6, 2017* `plank:0.55` provides `Pods` with the `$BUILD_ID` variable
   as well as the `$BUILD_NUMBER` variable. The latter is deprecated and will be
   removed in a future version.
 - *November 3, 2017* Added `EmptyDir` volume type. To update to `hook:0.176+`
   or `horologium:0.11+` the following components must have the associated
   minimum versions: `deck:0.58+`, `plank:0.54+`, `jenkins-operator:0.50+`.
 - *November 2, 2017* `plank:0.53` changes the `type` label key to `prow.k8s.io/type`
   and the `job` annotation key to `prow.k8s.io/job` added in pods.
 - *October 14, 2017* `deck:0:53+` needs to be updated in conjunction with
   `jenkins-operator:0:48+` since Jenkins logs are now exposed from the
   operator and `deck` needs to use the `external_agent_logs` option in order
   to redirect requests to the location `jenkins-operator` exposes logs.
 - *October 13, 2017* `hook:0.174`, `plank:0.50`, and `jenkins-operator:0.47`
   drop the deprecated `github-bot-name` flag.
 - *October 2, 2017* `hook:0.171`: The label plugin was split into three
   plugins (label, sigmention, milestonestatus). Breaking changes:
   - The configuration key for the milestone maintainer team's ID has been
   changed. Previously the team ID was stored in the plugins config at key
   `label`>>`milestone_maintainers_id`. Now that the milestone status labels are
   handled in the `milestonestatus` plugin instead of the `label` plugin, the
   team ID is stored at key `milestonestatus`>>`maintainers_id`.
   - The sigmention and milestonestatus plugins must be enabled on any repos
   that require them since their functionality is no longer included in the
   label plugin.
 - *September 3, 2017* `sinker:0.17` now deletes pods labeled by `plank:0.42` in
   order to avoid cleaning up unrelated pods that happen to be found in the
   same namespace prow runs pods. If you run other pods in the same namespace,
   you will have to manually delete or label the prow-owned pods, otherwise you
   can bulk-label all of them with the following command and let sinker collect
   them normally:
   ```
   kubectl label pods --all -n pod_namespace created-by-prow=true
   ```
 - *September 1, 2017* `deck:0.44` and `jenkins-operator:0.41` controllers
   no longer provide a default value for the `--jenkins-token-file` flag.
   Cluster administrators should provide `--jenkins-token-file=/etc/jenkins/jenkins`
   explicitly when upgrading to a new version of these components if they were
   previously relying on the default. For more context, please see
   [this pull request.](https://github.com/kubernetes/test-infra/pull/4210)
 - *August 29, 2017* Configuration specific to plugins is now held in the
   `plugins` `ConfigMap` and serialized in this repo in the `plugins.yaml` file.
   Cluster administrators upgrading to `hook:0.148` or newer should move
   plugin configuration from the main `ConfigMap`. For more context, please see
   [this pull request.](https://github.com/kubernetes/test-infra/pull/4213)
