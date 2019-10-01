# Building, Testing, and Updating Prow

This guide is directed at Prow developers and maintainers who want to build/test individual components or deploy changes to an existing Prow cluster. [`getting_started_deploy.md`](/prow/getting_started_deploy.md) is a better reference for deploying a new Prow cluster.

## How to build and test Prow

You can build, test, and deploy Prow’s binaries, container images, and cluster resources using [`bazel`](https://bazel.build).

Build with:
```shell
bazel build //prow/...
```
Test with:
```shell
bazel test --features=race //prow/...
```
Individual packages and components can be built and tested like:
```shell
bazel build //prow/cmd/hook
bazel test //prow/plugins/lgtm:go_default_test
```

### How to test a plugin

If you are making changes to a Prow plugin you can test the new behavior by sending fake webhooks to [`hook`](/prow/cmd/hook) with [`phony`](/prow/cmd/phony#phony).

## How to update the cluster

Any modifications to Go code will require redeploying the affected binaries.
Assuming your prow components have multiple replicas, this will result in no downtime.

Update your deployment (optionally build/pushing the image) to a new image with:
```shell
# export PROW_REPO_OVERRIDE=gcr.io/k8s-prow  # optionally override project
push.sh  # Build and push the current repo state.
bump.sh --list  # Choose a recent published version
bump.sh v20181002-deadbeef # Use a specific version
```

Once your deployment files are updated, please update these resources on your cluster:

```shell
# Set the kubectl context you want to use
export PROW_CLUSTER_OVERRIDE=my-k8s-cluster-context # or whatever the correct value is
export BUILD_CLUSTER_OVERRIDE=my-k8s-job-cluster-context # or whatever the correct value is

# Generally just do
bazel run //prow/cluster:production.apply # deploy everything

# In case of an emergency hook update
bazel run //prow/cluster:hook.apply # just update hook

# This is equivalent to doing the following with kubectl directly:
kubectl config use-context my-k8s-cluster-context
kubectl apply -f prow/cluster/*.yaml
kubectl apply -f prow/cluster/hook_deployment.yaml
```

## How to test a ProwJob

The best way to go about testing a new ProwJob depends on the job itself. If the
job's test container can be run locally that is typically the best way to
initially test the job because local debugging is easier than debugging a job in
CI.

Actually running the job on Prow is the next step. Before Prow can run your job,
you'll need to supply the job's config. Typically, new presubmit jobs
are configured to `skip_report`ing to GitHub and may not be configured to 
automatically run on every PR with `always_run: true`. Once the job is stable
these values can be changed to make the job run everywhere and become visible
to users by posting results to GitHub (if desired).
Changes to existing jobs can be trialed on canary jobs.

### Running a ProwJob Locally

[Phaino](/prow/cmd/phaino) lets you interactively mock and run the job locally on your workstation
in a docker container. Detailed instructions can be found in Phaino's [Readme](/prow/cmd/phaino/README.md)

### How to manually run a given job on Prow

If the normal job triggering mechanisms (`/test foo` comments, PR changes, PR
merges, cron schedule) are not sufficient for your testing you can use `mkpj` to
manually trigger new ProwJob runs.
To manually trigger any ProwJob, run the following, specifying `JOB_NAME`:

For K8S Prow, you can trigger a job by running
```shell
bazel run //config:mkpj -- --job=JOB_NAME
```

For your own prow instance, you can either define your own bazel rule, or
just go run mkpj like:
```shell
go run k8s.io/test-infra/prow/cmd/mkpj --job=JOB_NAME --config-path=path/to/config.yaml
```

Alternatively, if you have jobs defined in a separate `job-config`, you can
specify the config by adding the flag `--job-config-path=path/to/job/config.yaml`.

This will print the ProwJob YAML to stdout. You may pipe it into `kubectl`.
Depending on the job, you will need to specify more information such as PR
number.

NOTE: It is dangerous to create ProwJobs from handcrafted YAML. Please use `mkpj`
to generate ProwJob YAML.
