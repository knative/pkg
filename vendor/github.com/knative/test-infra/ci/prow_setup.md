# Prow setup

## Creating the cluster

1. Create the GKE cluster, the role bindings and the GitHub secrets. You might
   need to update [Makefile](./prow/Makefile). For details, see
   <https://github.com/kubernetes/test-infra/blob/master/prow/getting_started_deploy.md>.

1. Ensure the GCP projects listed in
   [resources.yaml](./prow/boskos/resources.yaml) are created.

1. Apply [config_start.yaml](./prow/config_start.yaml) to the cluster.

1. Apply Boskos [config_start.yaml](./prow/boskos/config_start.yaml) to the
   cluster.

1. Run `make update-cluster`, `make update-boskos`, `make update-config`,
   `make update-plugins` and `make update-boskos-config`.

1. If SSL needs to be reconfigured, promote your ingress IP to static in Cloud
   Console, and
   [create the TLS secret](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls).

## Expanding Boskos pool

1. All projects and permissions can be created by running `./ci/prow/boskos/create_projects.sh`.
   For example, to create 10 projects named `knative-boskos-51`,
   `knative-boskos-52`, ... `knative-boskos-60`, run:
   `./ci/prow/boskos/create_projects 51 10 0X0X0X-0X0X0X-0X0X0X /tmp/successful.out`.
   You will need to substitute the actual billing ID for the third argument.

1. Edit [resources.yaml](./prow/boskos/resources.yaml) with the new projects.
   Conveniently ready for cut-and-paste from the output file in the previous
   step.

1. Get the commit reviewed.

1. Run `make update-boskos-config` to update the Boskos config.

1. Increase the compute CPU quota for the project to 200. Go to
   `https://console.cloud.google.com/iam-admin/quotas?project=<project_name>&service=compute.googleapis.com&metric=CPUs`
   and click `Edit Quota`. Select at least five regions to increase the quota
   (`us-central1, us-west1, us-east1, europe-west1, asia-east1`). This needs
   to be done manually and should get automatically approved once the request
   is submitted. The request asks for a phone number and a reason. You can
   add any number and a reason _Need more resources for running tests_

In the event the create_projects fails, it is a script you should easily be
follow along with in the GUI or run on the CLI. The gcloud billing command is
still in alpha/beta, so it's probably the section most likely to give you
trouble.

## Setting up Prow for a new organization

1. In GitHub, add the following
   [webhooks](https://developer.github.com/webhooks/) to the org (or repo), in
   `application/json` format and for all events. Ask one of the owners of
   knative/test-infra for the webhook secrets.

   1. <http://prow.knative.dev/hook> (for Prow)
   1. <https://github-dot-knative-tests.appspot.com/webhook> (for Gubernator PR
      Dashboard)

1. Create a team called _Knative Prow Robots_, and make it an Admin of the org
   (or repo).

1. Invite at least [knative-prow-robot](https://github.com/knative-prow-robot)
   for your org. Add it to the robots team you created. For automated releases
   and metrics reporting (e.g., code coverage) you'll need to also add
   [knative-prow-releaser-robot](https://github.com/knative-prow-releaser-robot)
   and [knative-metrics-robot](https://github.com/knative-metrics-robot).

1. Add the org (and/or repo) to the [plugins.yaml](./prow/plugins.yaml) file, at
   least to the `approve` and `plugins` sections. Create a PR with the changes
   and once it's merged ask one of the owners of _knative/test-infra_ to deploy
   the new config.

## Setting up Prow for a new repo (reviewers assignment and auto merge)

1. Create the appropriate `OWNERS` files (at least one for the root dir).

1. Make sure that _Knative Robots_ is an Admin of the repo.

1. Add the repo to the
   [tide section](https://github.com/knative/test-infra/blob/b2cd02e6836ea2744ad838522b9eaf46385990f7/ci/prow/templates/prow_config_header.yaml#L72)
   in the [Prow config template](./prow/templates/prow_config_header.yaml) and
   run `make config`. Create a PR with the changes to the template and generated
   [config.yaml](./prow/config.yaml) file. Once the PR is merged, ask one of the
   owners of _knative/test-infra_ to deploy the new config.

1. Wait a few minutes, check that Prow is working by entering `/woof` as a
   comment in any PR in the new repo.

1. Set **tide** as a required status check for the master branch.

   ![Branch Checks](branch_checks.png)

### Setting up jobs for a new repo

1. Have the test infrastructure in place (usually this means having at least
   `//test/presubmit-tests.sh` working, and optionally `//hack/release.sh`
   working for automated nightly releases).

1. Merge a pull request that:

   1. Updates [config_knative.yaml](./prow/config_knative.yaml), the Prow config
      file (usually, copy and update the existing configuration from another
      repository). Run `make config` to regenerate
      [config.yaml](./prow/config.yaml), otherwise the presubmit test will fail.

   1. Updates the Testgrid config with the new buckets, tabs and dashboard.

1. Ask one of the owners of _knative/test-infra_ to:

   1. Run `make update-config` in `ci/prow`.

   1. Run `make update-config` in `ci/testgrid`.

1. Wait a few minutes, enter `/retest` as a comment in any PR in the repo and
   ensure the test jobs are executed.

1. Set the new test jobs as required status checks for the master branch.

   ![Branch Checks](branch_checks.png)
