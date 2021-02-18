# Release Leads

This document includes the roster, instructions and timetable to perform a
Knative release.

For each release cycle, we dedicate a team of two individuals, one from Eventing
and one from Serving, to shepherd the release process. Participation is
voluntary and based on good faith. We are only expected to participate during
our local office hour.

---

# Roster

We seed this rotation with all approvers from all the Serving and Eventing
workgroups, excluding productivity. If you are no longer active in Knative, or
if you are contributing on personal capacity and do not have time to contribute
in the rotation, feel free to send a PR to remove yourself.

## Serving roster

This roster is seeded with all approvers from Serving workgroups.

- dprotaso
- julz
- JRBANCEL
- markusthoemmes
- mattmoor
- nak3
- tcnghia
- vagababov
- yanweiguo
- ZhiminXiang

## Eventing roster

This roster is seeded with all approvers from Eventing workgroups.

- evankanderson
- grantr
- Harwayne
- lionelvillard
- matzew
- n3wscott
- nachocano
- slinkydeveloper
- vaikas

## Schedule

| Release | Release Date | Serving        | Eventing        | Unpin repos | PKG cut    |
| ------- | ------------ | -------------- | --------------- | ----------- | ---------- |
| v0.17   | 2020-08-18   | yanweiguo      | Harwayne        | -           | 2020-08-11 |
| v0.18   | 2020-09-29   | ZhiminXiang    | n3wscott        | 2020-08-19  | 2020-09-22 |
| v0.19   | 2020-11-10   | julz           | n3wscott        | 2020-09-30  | 2020-11-03 |
| v0.20   | 2021-01-12   | nak3           | slinkydeveloper | 2020-11-11  | 2021-01-07 |
| v0.21   | 2021-02-23   | mattmoor       | lionelvillard   | 2021-01-13  | 2021-02-16 |
| v0.22   | 2021-04-06   | markusthoemmes | evankanderson   | 2021-02-24  | 2021-03-30 |
| v0.23   | 2021-05-18   | tcnghia        | vaikas          | 2021-04-07  | 2021-05-11 |
| v0.24   | 2021-06-29   | dprotaso       | matzew          | 2021-05-19  | 2021-06-22 |
| v0.25   | 2021-08-10   | vagababov      | grantr          | 2021-06-30  | 2021-08-03 |
| v0.26   | 2021-09-21   | JRBANCEL       | ...             | 2021-08-11  | 2021-09-14 |

**NOTE:** v0.20 is moved by 3 weeks for end of year holidays

---

# Instructions

Below you'll find the instructions to release a `knative.dev` repository.

For more information on the timetable, jump to the [Timetable](#timetable)
paragraph.

## Release a repository

Releasing a repository includes:

- Aligning the `knative.dev` dependencies to the other release versions on
  master
- Creating a new branch starting from master for the release (e.g.
  `release-0.20`)
- Execute the job on Prow that builds the code from the release branch, tags the
  revision, publishes the images, publishes the yaml artifacts and generates the
  Github release.

Most of the above steps are automated, although in some situations it might be
necessary to perform some of them manually.

### Check the build on master pass

Before beginning, check if the repository is in a good shape and the builds pass
consistently. **This is required** because the Prow job that builds the release
artifacts will execute all the various tests (build, unit, e2e) and, if
something goes wrong, you will probably need to restart this process from the
beginning.

For any problems in a specific repo, get in touch with the relevant WG leads to
fix them.

### Aligning the dependencies

In order to align the `knative.dev` dependencies, knobots will perform PRs like
[this](https://github.com/knative/eventing/pull/4713) for each repo, executing
the command `./hack/update-deps.sh --upgrade --release 0.20` and committing all
the content.

If no dependency bump PR is available, you can:

- Manually trigger the generation of these PRs starting the
  [Knobots Auto Updates workflow](https://github.com/knative-sandbox/knobots/actions?query=workflow%3A%22Auto+Updates%22)
  and wait for the PR to pop in the repo you need.
- Execute the script below on your machine and PR the result to master:

```shell
RELEASE=0.20
REPO=git@github.com:knative/example.git

tmpdir=$(dirname $(mktemp -u))
cd ${tmpdir}
git clone ${REPO}
cd "$(basename "${REPO}" .git)"

./hack/update-deps.sh --upgrade --release ${RELEASE}
./hack/update-codegen.sh

# If there are no changes, you can go to the next step without committing any change.
# Otherwise, commit all the changes
git status
```

### Releasability

At this point, you can proceed with the releasability check. A releasability
check is executed periodically and posts the result on the Slack release channel
and it fails if the dependencies are not properly aligned. If you don't want to
wait, you can manually execute the
[Releasability workflow](https://github.com/knative/serving/actions?query=workflow%3AReleasability).

If the releasability reports NO-GO, probably there is some deps misalignment,
hence you need to go back to the previous step and check the dependencies,
otherwise, you're ready to proceed.

You can execute the releasability check locally using
[**buoy**](https://github.com/knative/test-infra/tree/master/buoy):

```bash
RELEASE=0.20
REPO=git@github.com:knative/example.git

tmpdir=$(dirname $(mktemp -u))
cd ${tmpdir}
git clone ${REPO}
cd "$(basename "${REPO}" .git)"

if buoy check go.mod --domain knative.dev --release ${RELEASE} --verbose; then
  git checkout -b release-${RELEASE}
  ./hack/update-deps.sh --upgrade --release ${RELEASE}
  git status
fi
```

If there are changes, then it's NO-GO, otherwise it's GO

### Just one last check before cutting

After the dependencies are aligned and releasability is ready to GO, perform one
last check manually that every `knative.dev` in the `go.mod` file is properly
configured:

- For the _support_ repos (`hack`, `test-infra`, `pkg`, etc) you should see the
  dependency version pointing at a revision which should match the `HEAD` of the
  release branch. E.g. `knative.dev/pkg v0.0.0-20210112143930-acbf2af596cf`
  points at the revision `acbf2af596cf`, which is the `HEAD` of the
  `release-0.20` branch in `pkg` repo.
- For the _release_ repos, you should see the dependency version pointing at the
  version tag. E.g. `knative.dev/eventing v0.20.0` points at the tag `v0.20.0`
  in the `eventing` repo.

### Cut the branch

Now you're ready to create the `release-v.y` branch. This can be done by using
the GitHub UI:

1. Click on the branch selection box at the top level page of the repository.

   ![Click the branch selection box](images/github-branch.png)

1. Search for the correct `release-x.y` branch name for the release.

   ![Search for the expected release branch name](images/github-branch-search.png)

1. Click "Create branch: release-x.y".

   ![Click create branch: release-x.y](images/github-branch-create.png)

Otherwise, you can do it by hand on your local machine.

### The Prow job

After a `release-x.y` branch exists, a 4 hourly prow job will build the code
from the release branch, tag the revision, publish the images, publish the yaml
artifacts and generate the Github release. Update the description of the release
with the release notes collected.

You can manually trigger the release:

1. Navigate to https://prow.knative.dev/

   ![Prow homepage](images/prow-home.png)

1. Search for the `*-auto-release` job for the repository.

   ![Search Prow for the repo and select the auto-release](images/prow-search.png)

1. Rerun the auto-release job.

   ![Rerun Prow Auto Release](images/prow-rerun.png)

### Verify nightly release automation is intact

The automation used to cut the actual releases is the very same as the
automation used to cut nightly releases. Verify via testgrid that all relevant
nightly releases are passing. If they are not coordinate with the relevant WG
leads to fix them.

### What could go wrong?

In case you cut a branch before it was ready (e.g. some deps misalignment, a
failing test, etc), you can try to restart this process. But first, clean up the
repo in this order:

1. Remove the Github release (if any)
1. Remove the git tag (if any) using `git push --delete origin v0.20.0`
   (assuming `origin` is the `knative.dev` repo)
1. Remove the git branch (if any) from the Github UI

---

# Timetable

We release the components of Knative every 6 weeks. All of these components must
be moved to the latest "release" of all shared dependencies prior to each
release.

## First week of the rotation

### Make sure you have the right permission

Check to make sure you already are in the "Knative Release Leads" team in
https://github.com/knative/community/blob/master/peribolos/knative.yaml and
https://github.com/knative/community/blob/master/peribolos/knative-sandbox.yaml
. If not, send a PR like
[this one](https://github.com/knative/community/pull/209) to grant yourself some
super power.

### Create a release Slack channel

Ask someone from the TOC to create a **release-`#`** Slack channel that will be
used to help manage this release.

## 14 days prior to the release

### Update the Knative releasability defaults

Update the defaults in
[knative-releasability.yaml](https://github.com/knative-sandbox/.github/blob/1e4e31edfb2181220db744ad0fcb135629e1cb8e/workflow-templates/knative-releasability.yaml#L37-L41)
to this release. These changes will be propagated to the rest of Knative in the
next round of workflow syncs.

### Announce the imminent `pkg` cut

Announce on **#general** that `pkg` will be cut in a week.

### Cut release branches of supporting repos

The supporting repos are the base repos where we have common code and common
scripts. For these repos, we follow the same release process as explained in
[release a repository](#release-a-repository), but no prow job is executed,
hence no git tag and Github release are produced.

Follow the [release a repository](#release-a-repository) guide, skipping the
prow job part, starting with the **hack** repo:

- [knative/hack](https://github.com/knative/hack)

After **hack**:

| Repo                                                            | Releasability                                                                             |
| --------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| [knative.dev/pkg](https://github.com/knative/pkg)               | ![Releasability](https://github.com/knative/pkg/workflows/Releasability/badge.svg)        |
| [knative.dev/test-infra](https://github.com/knative/test-infra) | ![Releasability](https://github.com/knative/test-infra/workflows/Releasability/badge.svg) |

After **pkg**:

| Repo                                                                              | Releasability                                                                                          |
| --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| [knative.dev/networking](https://github.com/knative/networking)                   | ![Releasability](https://github.com/knative/networking/workflows/Releasability/badge.svg)              |
| [knative.dev/caching](https://github.com/knative/caching)                         | ![Releasability](https://github.com/knative/caching/workflows/Releasability/badge.svg)                 |
| [knative.dev/reconciler-test](https://github.com/knative-sandbox/reconciler-test) | ![Releasability](https://github.com/knative-sandbox/reconciler-test/workflows/Releasability/badge.svg) |

Automation will propagate these updates to all the downstream repos in the next
few cycles. The goal is to have the first wave of repo releases (**serving**,
**eventing**, etc) to become "releasabile" by the end of the week. This is
signaled via the Slack report of releasability posted to the **release-`#`**
channel every morning (5am PST, M-F).

## 7 days prior to the release

### Announce the imminent release cut

Announce on **#general** that the release will be cut in a week and that
additional caution should be used when merging big changes.

### Collect release-notes

Make a new HackMD release notes document.
[last release notes document](https://hackmd.io/cJwvzJ4eRVeqqiXzOPtxsA), empty
it out and send it to the WG leads of the respective project (serving or
eventing) to fill in. Coordinate with both serving and eventing leads.

Each repo has a `Release Notes` GitHub Action workflow. This can be used to
generate the starting point for the release notes. See an example in
[Eventing](https://github.com/knative/eventing/actions?query=workflow%3A%22Release+Notes%22).
The default starting and ending SHAs will work if running out of the `master`
branch, or you can determine the correct starting and ending SHAs for the script
to run.

## 1 day prior to the release

### Confirm readiness

Confirm with the respective WG leads that the release is imminent and obtain
green light.

## Day of the release

Follow the [release a repository](#release-a-repository) instructions for each
repo. Wait for release automation to kick in (runs on a 2 hour interval). Once
the release automation passed, it will create a release tag in the repository.
Enhance the respective tags with the collected release-notes using the GitHub
UI.

In general the release dependency order is something like the following (as of
v0.20). Note: `buoy check` will fail if the dependencies are not yet ready.

First:

| Repo                                                                                  | Releasability                                                                                            | Nightly |
| ------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- | ------- |
| [knative.dev/serving](https://github.com/knative/serving)                             | ![Releasability](https://github.com/knative/serving/workflows/Releasability/badge.svg)                   | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-serving-nightly-release) |
| [knative.dev/net-certmanager](https://github.com/knative-sandbox/net-certmanager)     | ![Releasability](https://github.com/knative-sandbox/net-certmanager/workflows/Releasability/badge.svg)   | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-net-certmanager-nightly-release) |
| [knative.dev/net-contour](https://github.com/knative-sandbox/net-contour)             | ![Releasability](https://github.com/knative-sandbox/net-contour/workflows/Releasability/badge.svg)       | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-net-contour-nightly-release) |
| [knative.dev/net-http01](https://github.com/knative-sandbox/net-http01)               | ![Releasability](https://github.com/knative-sandbox/net-http01/workflows/Releasability/badge.svg)        | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-net-http01-nightly-release) |
| [knative.dev/net-istio](https://github.com/knative-sandbox/net-istio)                 | ![Releasability](https://github.com/knative-sandbox/net-istio/workflows/Releasability/badge.svg)         | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-net-istio-nightly-release) |
| [knative.dev/net-kourier](https://github.com/knative-sandbox/net-kourier)             | ![Releasability](https://github.com/knative-sandbox/net-kourier/workflows/Releasability/badge.svg)       | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-net-kourier-nightly-release) |
| [knative.dev/eventing](https://github.com/knative/eventing)                           | ![Releasability](https://github.com/knative/eventing/workflows/Releasability/badge.svg)                  | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-eventing-nightly-release) |
| [knative.dev/discovery](https://github.com/knative-sandbox/discovery)                 | ![Releasability](https://github.com/knative-sandbox/discovery/workflows/Releasability/badge.svg)         | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-discovery-nightly-release) |
| [knative.dev/sample-controller](https://github.com/knative-sandbox/sample-controller) | ![Releasability](https://github.com/knative-sandbox/sample-controller/workflows/Releasability/badge.svg) | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-sample-controller-nightly-release) |

After **eventing**:

| Repo                                                                                          | Releasability                                                                                                | Nightly |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ------- |
| [knative.dev/eventing-awssqs](https://github.com/knative-sandbox/eventing-awssqs)             | ![Releasability](https://github.com/knative-sandbox/eventing-awssqs/workflows/Releasability/badge.svg)       | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-awssqs-nightly-release) |
| [knative.dev/eventing-camel](https://github.com/knative-sandbox/eventing-camel)               | ![Releasability](https://github.com/knative-sandbox/eventing-camel/workflows/Releasability/badge.svg)        | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-camel-nightly-release) |
| [knative.dev/eventing-ceph](https://github.com/knative-sandbox/eventing-ceph)                 | ![Releasability](https://github.com/knative-sandbox/eventing-ceph/workflows/Releasability/badge.svg)         | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-ceph-nightly-release) |
| [knative.dev/eventing-couchdb](https://github.com/knative-sandbox/eventing-couchdb)           | ![Releasability](https://github.com/knative-sandbox/eventing-couchdb/workflows/Releasability/badge.svg)      | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-couchdb-nightly-release) |
| [knative.dev/eventing-kafka](https://github.com/knative-sandbox/eventing-kafka)               | ![Releasability](https://github.com/knative-sandbox/eventing-kafka/workflows/Releasability/badge.svg)        | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-kafka-nightly-release) |
| [knative.dev/eventing-kafka-broker](https://github.com/knative-sandbox/eventing-kafka-broker) | ![Releasability](https://github.com/knative-sandbox/eventing-kafka-broker/workflows/Releasability/badge.svg) | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-kafka-broker-nightly-release) |
| [knative.dev/eventing-natss](https://github.com/knative-sandbox/eventing-natss)               | ![Releasability](https://github.com/knative-sandbox/eventing-natss/workflows/Releasability/badge.svg)        | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-natss-nightly-release) |
| [knative.dev/eventing-prometheus](https://github.com/knative-sandbox/eventing-prometheus)     | ![Releasability](https://github.com/knative-sandbox/eventing-prometheus/workflows/Releasability/badge.svg)   | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-prometheus-nightly-release) |
| [knative.dev/eventing-rabbitmq](https://github.com/knative-sandbox/eventing-rabbitmq)         | ![Releasability](https://github.com/knative-sandbox/eventing-rabbitmq/workflows/Releasability/badge.svg)     | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-rabbitmq-nightly-release) |
| [knative.dev/sample-source](https://github.com/knative-sandbox/sample-source)                 | ![Releasability](https://github.com/knative-sandbox/sample-source/workflows/Releasability/badge.svg)         | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-sample-source-nightly-release) |

After both **eventing** and **serving**:

| Repo                                                                              | Releasability                                                                                          | Nightly |
| --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ | ------- |
| [knative.dev/eventing-redis](https://github.com/knative-sandbox/eventing-redis)   | ![Releasability](https://github.com/knative-sandbox/eventing-redis/workflows/Releasability/badge.svg)  | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-redis-nightly-release) |
| [knative.dev/eventing-github](https://github.com/knative-sandbox/eventing-github) | ![Releasability](https://github.com/knative-sandbox/eventing-github/workflows/Releasability/badge.svg) | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-github-nightly-release) |
| [knative.dev/eventing-gitlab](https://github.com/knative-sandbox/eventing-gitlab) | ![Releasability](https://github.com/knative-sandbox/eventing-gitlab/workflows/Releasability/badge.svg) | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-gitlab-nightly-release) |

Lastly:

| Repo                                                                                                | Releasability                                                                                                   | Nightly |
| --------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- | ------- |
| [knative.dev/eventing-autoscaler-keda](https://github.com/knative-sandbox/eventing-autoscaler-keda) | ![Releasability](https://github.com/knative-sandbox/eventing-autoscaler-keda/workflows/Releasability/badge.svg) | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-eventing-autoscaler-keda-nightly-release) |

We have a few repos inside of Knative that are not handled in the standard
process at the moment. They might have additional dependencies or depend on the
releases existing. **Skip these**. Special cases are:

| Repo                                                        | Releasability                                                                           | Nightly |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------- | ------- |
| [knative.dev/client](https://github.com/knative/client)     | ![Releasability](https://github.com/knative/client/workflows/Releasability/badge.svg)   | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-client-nightly-release) |
| [knative.dev/docs](https://github.com/knative/docs)         | ![Releasability](https://github.com/knative/docs/workflows/Releasability/badge.svg)     | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-docs-nightly-release) |
| [knative.dev/website](https://github.com/knative/website)   | ![Releasability](https://github.com/knative/website/workflows/Releasability/badge.svg)  | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-website-nightly-release) |
| [knative.dev/operator](https://github.com/knative/operator) | ![Releasability](https://github.com/knative/operator/workflows/Releasability/badge.svg) | ![Nightly](https://prow.knative.dev/badge.svg?jobs=ci-knative-sandbox-operator-nightly-release) |

## After the release

Send a PR like [this one](https://github.com/knative/community/pull/209) to
grant ACLs for the next release leads, and to remove yourself from the rotation.
Include the next release leads in the PR as a reminder.

Send a PR like [this one](https://github.com/knative-sandbox/knobots/pull/18) to
bump knobots auto release workflow to the next release.
