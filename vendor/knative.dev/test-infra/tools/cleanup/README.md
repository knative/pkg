# Resources Clean Up Tool

This tool is designed to clean up stale test resources. For now it deletes GCR
images and GKE clusters created during testing.

## Basic Usage

Directly invoke [cleanup.sh](cleanup.sh) script with certain flags, but don't
source this script.

By default the current gcloud credentials are used to delete the images. If
necessary, use the flag `--service-account _key-file.json_` to specify a service
account that will be performing the access to the gcr.

Projects to be cleaned up are expected to be defined in a `resources.yaml` file.
To remove old images and clusters from them, call [cleanup.sh](cleanup.sh) with
following flags:

- "--project-resource-yaml" as path of `resources.yaml` file - Mandatory
- "--re-project-name" for regex matching projects names - Optional, defaults to
  `knative-boskos-[a-zA-Z0-9]+`
- "--days-to-keep-images" - Optional, defaults to `365` as 1 year
- "--hours-to-keep-clusters" - Optional, defaults to `720` as 30 days
- "--dry-run" - Optional, performs dryrun for all gcloud functions, defaults to
  false

Example:

`./cleanup.sh --project-resource-yaml "ci/prow/boskos/resources.yaml" --days-to-keep-images 90 --days-to-keep-clusters 24`
This command deletes test images older than 90 days and test clusters created
more than 24 hours ago.

## Prow Job

There is a weekly prow job that triggers this tool runs at 11:00/12:00PM(Day
light saving) PST every Monday. This tool scans all gcr projects defined in
[ci/prow/boskos/resources.yaml](/ci/prow/boskos/resources.yaml) and deletes
images older than 90 days and clusters older than 24 hours.
