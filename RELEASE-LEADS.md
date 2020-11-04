# Release Leads

For each release cycle, we dedicate a team of two individuals, one from Eventing
and one from Serving, to shepherd the release process. Participation is
voluntary and based on good faith. We are only expected to participate during
our local office hour.

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
| v0.20   | 2021-01-12   | nak3           | slinkydeveloper | 2020-11-11  | 2020-01-07 |
| v0.21   | 2021-02-23   | mattmoor       | lionelvillard   | 2021-01-13  | 2021-02-16 |
| v0.22   | 2021-04-06   | markusthoemmes | evankanderson   | 2021-02-24  | 2021-03-30 |
| v0.23   | 2021-05-18   | tcnghia        | vaikas          | 2021-04-07  | 2021-05-11 |
| v0.24   | 2021-06-29   | dprotaso       | matzew          | 2021-05-19  | 2021-06-22 |
| v0.25   | 2021-08-10   | vagababov      | grantr          | 2021-06-30  | 2021-08-03 |
| v0.26   | 2021-09-21   | JRBANCEL       | ...             | 2021-08-11  | 2021-09-14 |

**NOTE:** v0.20 is moved by 3 weeks for end of year holidays

# Release instruction

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

### Revert all pins to pin master branches again

Revert all pins in all repositories to pin the **master** branches again, run
`hack/update-deps.sh --upgrade` and PR the changes.

You should only need to do this for
`knative/{serving,eventing-contrib,eventing}` and
`knative-sandbox/net-{istio,contour,kourier,http01,certmanager}`. However, you
may want to double check `knative/{pkg,caching,networking}` as well in case the
previous release leads missed a step during their last rotation.

Example PRs:

- [knative/serving](https://github.com/knative/serving/pull/8579)
- [knative/eventing](https://github.com/knative/eventing/pull/3546)
- [knative/eventing-contrib](https://github.com/knative/eventing-contrib/pull/1272)
- [knative-sandbox/net-istio](https://github.com/knative-sandbox/net-istio/pull/172)
- [knative-sandbox/net-contour](https://github.com/knative-sandbox/net-contour/pull/154)
- [knative-sandbox/net-kourier](https://github.com/knative-sandbox/net-kourier/pull/84)
- [knative-sandbox/net-http01](https://github.com/knative-sandbox/net-http01/pull/42)
- [knative-sandbox/net-certmanager](https://github.com/knative-sandbox/net-certmanager/pull/39)

## 14 days prior to the release

### Announce the imminent `pkg` cut

Announce on **#general** that `pkg` will be cut in a week.

---

## 7 days prior to the release

### Announce the imminent release cut

Announce on **#general** that the release will be cut in a week and that
additional caution should be used when merging big changes.

### Collect release-notes

Make a copy of the
[last release notes document](https://docs.google.com/document/d/1FTL_fMXU2hv2POh9uN_8IJe9FbqEIzFdRZZRXI0WaT4/edit),
empty it out and send it to the WG leads of the respective project (serving or
eventing) to fill in. Coordinate with both serving and eventing leads.

### Cut `release-x.y` in `test-infra`, `pkg`, `caching`, and `networking` libraries

Shared dependencies like `knative/{test-infra, pkg, caching, networking}` are
kept up-to-date nightly in each of the releasing repositories. To stabilize
things shortly before the release we cut the `release-x.y` branches on those 7
days prior to the main release.

First, create a release branch for `test-infra` named `release-x.y`.

Next, `pkg` needs to pin to `test-infra`'s release branch. To do that, edit
`hack/update-deps.sh` in `pkg` **on the newly created branch** to pin the
branch. Then run `./hack/update-deps.sh --upgrade` and commit the changes.

The change to `hack/update-deps.sh` will look like this:

```diff
diff --git a/hack/update-deps.sh b/hack/update-deps.sh
index a39fc858..0634362f 100755
--- a/hack/update-deps.sh
+++ b/hack/update-deps.sh
@@ -26,7 +26,7 @@ cd ${ROOT_DIR}
 # The list of dependencies that we track at HEAD and periodically
 # float forward in this repository.
 FLOATING_DEPS=(
-  "knative.dev/test-infra@master"
+  "knative.dev/test-infra@release-x.y"
 )

 # Parse flags to determine any we should pass to dep.
```

PR the changes to each repository respectively, prepending the PR title with
`[RELEASE]`.

After `test-infra` and `pkg` are pinned, change `caching` and `networking`'s
`update-deps.sh` to use `release-x.y` branch of `test-infra` and `pkg`.
Following that, cut new `release-x.y` branches for `caching` and `networking`.

### Pin `test-infra`, `pkg`, `caching`, `networking` in downstream repositories

Similar to how we pin `pkg` to `test-infra`, all downstream users must be pinned
to the newly cut `release-x.y` branches on those libraries. The changes to
`hack/update-deps.sh` look similar to above, but in most cases both dependencies
will need to be pinned.

```diff
diff --git a/hack/update-deps.sh b/hack/update-deps.sh
index b277dd3ff..1989885ce 100755
--- a/hack/update-deps.sh
+++ b/hack/update-deps.sh
@@ -32,8 +32,8 @@ VERSION="master"
 # The list of dependencies that we track at HEAD and periodically
 # float forward in this repository.
 FLOATING_DEPS=(
-  "knative.dev/test-infra@${VERSION}"
-  "knative.dev/pkg@${VERSION}"
-  "knative.dev/caching@${VERSION}"
-  "knative.dev/networking@${VERSION}"
+  "knative.dev/test-infra@release-x.y"
+  "knative.dev/pkg@release-x.y"
+  "knative.dev/caching@release-x.y"
+  "knative.dev/networking@release-x.y"
 )
```

The downstream repositories this needs to happen on are:

- [knative/client](https://github.com/knative/client)

- [knative/operator](https://github.com/knative/operator)

- [knative/serving](https://github.com/knative/serving)
- [knative-sandbox/net-certmanager](https://github.com/knative-sandbox/net-certmanager)
- [knative-sandbox/net-contour](https://github.com/knative-sandbox/net-contour)
- [knative-sandbox/net-http01](https://github.com/knative-sandbox/net-http01)
- [knative-sandbox/net-istio](https://github.com/knative-sandbox/net-istio)
- [knative-sandbox/net-kourier](https://github.com/knative-sandbox/net-kourier)

- [knative/eventing](https://github.com/knative/eventing)
- [knative/eventing-contrib](https://github.com/knative/eventing-contrib)
- [knative-sandbox/eventing-kafka-broker](https://github.com/knative-sandbox/eventing-kafka-broker)
- [knative-sandbox/discovery](https://github.com/knative-sandbox/discovery)
- [knative-sandbox/eventing-autoscaler-keda](https://github.com/knative-sandbox/eventing-autoscaler-keda)
- [knative-sandbox/eventing-awssqs](https://github.com/knative-sandbox/eventing-awssqs)
- [knative-sandbox/eventing-camel](https://github.com/knative-sandbox/eventing-camel)
- [knative-sandbox/eventing-ceph](https://github.com/knative-sandbox/eventing-ceph)
- [knative-sandbox/eventing-couchdb](https://github.com/knative-sandbox/eventing-couchdb)
- [knative-sandbox/eventing-github](https://github.com/knative-sandbox/eventing-github)
- [knative-sandbox/eventing-gitlab](https://github.com/knative-sandbox/eventing-gitlab)
- [knative-sandbox/eventing-kafka](https://github.com/knative-sandbox/eventing-kafka)
- [knative-sandbox/eventing-natss](https://github.com/knative-sandbox/eventing-natss)
- [knative-sandbox/eventing-prometheus](https://github.com/knative-sandbox/eventing-prometheus)
- [knative-sandbox/eventing-rabbitmq](https://github.com/knative-sandbox/eventing-rabbitmq)

Apply the changes the the **master branches**, run
`hack/update-deps.sh --upgrade` (and potentially `hack/update-codegen.sh` if
necessary) and PR the changes to the **master branch**. Don't cut the release
branch yet.

### Verify nightly release automation is intact

The automation used to cut the actual releases is the very same as the
automation used to cut nightly releases. Verify via testgrid that all relevant
nightly releases are passing. If they are not coordinate with the relevant WG
leads to fix them.

---

## 1 day prior to the release

### Confirm readiness

Confirm with the respective WG leads that the release is imminent and obtain
green light.

---

## Day of the release

### Cut `release-x.y` branches of `serving` and `eventing`

Create a `release-x.y` branch from master in both repositories. Wait for release
automation to kick in (runs on a 2 hour interval). Once the release automation
passed, it will create a release tag in both repositories. Enhance the
respective tags with the collected release-notes using the Github UI.

### Cut `release-x.y` branches of `net-*`

Cut a `release-x.y` branch in each of the following repositories which do not
depend on `serving` or `eventing`:

- [knative-sandbox/net-certmanager](https://github.com/knative-sandbox/net-certmanager)
- [knative-sandbox/net-contour](https://github.com/knative-sandbox/net-contour)
- [knative-sandbox/net-http01](https://github.com/knative-sandbox/net-http01)
- [knative-sandbox/net-istio](https://github.com/knative-sandbox/net-istio)

### Pin `serving` and `eventing` releases in dependent repositories

**After** the tags for `serving` and `eventing` are created, their version needs
to be pinned in all repositories that depend on them.

For **serving** that is:

- [knative/client](https://github.com/knative/client)
- [knative-sandbox/net-kourier](https://github.com/knative-sandbox/net-kourier)
- [knative/eventing-contrib](https://github.com/knative/eventing-contrib)

For **eventing** that is:

- [knative/client](https://github.com/knative/client)
- [knative/eventing-contrib](https://github.com/knative/eventing-contrib)
- [knative-sandbox/eventing-kafka-broker](https://github.com/knative-sandbox/eventing-kafka-broker)

The pins are similar to step 5 above, but now we're pinning `serving` and
`eventing` respectively. Again, the pin PRs are sent against the **master**
branch of each repository respectively.

### Cut `release-x.y` branches of all remaining repositories

After the pin PRs are merged, cut the `release-x.y` branch in each of the
remaining repositories (except `operator` and `client` as they are cut
separately by the respective working group):

- [knative-sandbox/net-kourier](https://github.com/knative-sandbox/net-kourier)
- [knative/eventing-contrib](https://github.com/knative/eventing-contrib)
- [knative-sandbox/eventing-kafka-broker](https://github.com/knative-sandbox/eventing-kafka-broker)

Release automation will automatically pick up the branches and will likewise
create the respective tags.

## Right after the release

Send a PR like [this one](https://github.com/knative/community/pull/209) to
grant ACLs for the next release leads, and to remove yourself from the rotation.
Include the next release leads in the PR as a reminder.

---
