# Creating a new Knative release using Prow

**Note:** The user starting the release job must have privileged access to the
Prow cluster, otherwise the job won't run.

This is the preferred method of manually building a new release.

## Creating an arbitrary release

1. Create a temporary config file like the one below. Replace `MODULE` with the
   right Knative module name (e.g., `serving` or `build`). Replace `VERSION`
   with the desired release version number (in the X.Y.Z format), and `BRANCH`
   with the branch version number (in the X.Y format). For the rest of this doc,
   it's assumed this file is `/tmp/release.yaml`.

   ```
   periodics:
     knative/MODULE:
       - auto-release: true
         cron: "* * * * *"
         args:
         - "--publish"
         - "--tag-release"
         - "--github-token /etc/hub-token/token"
         - "--release-gcr gcr.io/knative-releases"
         - "--release-gcs knative-releases/MODULE"
         - "--version VERSION"
         - "--branch release-BRANCH"
   ```

1. Generate the full config from the file above. For the rest of this doc, it's
   assumed this file is `/tmp/release_config.yaml`. Replace `MODULE` with the
   right Knative module name (e.g., `serving` or `build`).

   ```
   cd ci/prow
   go run *_config.go \
     --job-filter=ci-knative-MODULE-auto-release \
     --generate-testgrid-config=false \
     --prow-config-output=/tmp/release_config.yaml \
     /tmp/release.yaml
   ```

1. Generate the job config from the full config. For the rest of this doc, it's
   assumed this file is `/tmp/release_job.yaml`. Replace `MODULE` with the right
   Knative module name (e.g., `serving` or `build`).

   ```
   bazel run @k8s//prow/cmd/mkpj -- --job=ci-knative-MODULE-auto-release \
     --config-path=/tmp/release_config.yaml \
     > /tmp/release_job.yaml
   ```

1. Start the job on Prow. Make sure you get the credentials first.

   ```
   make get-cluster-credentials
   kubectl apply -f /tmp/release_job.yaml
   ```

1. Monitor the new job through [Prow UI](https://prow.knative.dev).

## Creating a "dot" release on demand

1. Use the `run_job.sh` script to start the dot release job for the module you
   want, like in the example below. Replace `MODULE` with the right Knative
   module name (e.g., `serving` or `build`).

   ```
   cd ci/prow
   ./run_job.sh ci-knative-MODULE-dot-release
   ```

1. Monitor the new job through [Prow UI](https://prow.knative.dev).

## Creating a nightly release on demand

1. Use the `run_job.sh` script to start the nightly release job for the module
   you want, like in the example below. Replace `MODULE` with the right Knative
   module name (e.g., `serving` or `build`).

   ```
   cd ci/prow
   ./run_job.sh ci-knative-MODULE-nightly-release
   ```

1. Monitor the new job through [Prow UI](https://prow.knative.dev).
