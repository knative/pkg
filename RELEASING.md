# Releasing Knative

We release the components of Knative every 6 weeks. All of these components must
be moved to the latest "release" of all shared dependencies prior to each
release.

---

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

Both `pkg` and `test-infra` also need to pin each other's release branch. To do
that, edit `hack/update-deps.sh` in the respective repo **on the newly created
branch** to pin the respective branch. Then run
`./hack/update-deps.sh --upgrade` and commit the changes.

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

Similar to how the pin between `pkg` and `test-infra` themselves work, all
downstream users must be pinned to the newly cut `release-x.y` branches on those
libraries. The changes to `hack/update-deps.sh` look similar to above, but in
most cases both dependencies will need to be pinned.

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
- [knative/eventing-contrib](https://github.com/knative/eventing-contrib)
- [knative/eventing](https://github.com/knative/eventing)
- [knative/net-certmanager](https://github.com/knative/net-certmanager)
- [knative/net-contour](https://github.com/knative/net-contour)
- [knative/net-http01](https://github.com/knative/net-http01)
- [knative/net-istio](https://github.com/knative/net-istio)
- [knative/net-kourier](https://github.com/knative/net-kourier)
- [knative/operator](https://github.com/knative/operator)
- [knative/serving](https://github.com/knative/serving)

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

### Pin `serving` and `eventing` releases in dependent repositories

**After** the tags for `serving` and `eventing` are created, their version needs
to be pinned in all repositories that depend on them.

For **serving** that is:

- [knative/eventing-contrib](https://github.com/knative/eventing-contrib)
- [knative/net-certmanager](https://github.com/knative/net-certmanager)
- [knative/net-contour](https://github.com/knative/net-contour)
- [knative/net-http01](https://github.com/knative/net-http01)
- [knative/net-istio](https://github.com/knative/net-istio)
- [knative/net-kourier](https://github.com/knative/net-kourier)

For **eventing** that is:

- [knative/eventing-contrib](https://github.com/knative/eventing-contrib)

The pins are similar to step 5 above, but now we're pinning `serving` and
`eventing` respectively. Again, the pin PRs are sent against the **master**
branch of each repository respectively.

### Cut `release-x.y` branches of all remaining repositories

After the pin PRs are merged, cut the `release-x.y` branch in each of the
remaining repositories (except `operator`):

- [knative/eventing-contrib](https://github.com/knative/eventing-contrib)
- [knative/net-certmanager](https://github.com/knative/net-certmanager)
- [knative/net-contour](https://github.com/knative/net-contour)
- [knative/net-http01](https://github.com/knative/net-http01)
- [knative/net-istio](https://github.com/knative/net-istio)
- [knative/net-kourier](https://github.com/knative/net-kourier)

Release automation will automatically pick up the branches and will likewise
create the respective tags.

---

## After the release

### Revert all pins to pin master branches again

Revert all pins in all repositories to pin the **master** branches again, run
`hack/update-deps.sh --upgrade` and PR the changes.

### Change this file to reflect new modification to the release if applicable

---
