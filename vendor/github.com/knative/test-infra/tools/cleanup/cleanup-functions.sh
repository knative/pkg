#!/bin/bash

# Copyright 2018 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Functions for cleaning up GCRs.
# It doesn't do anything when called from command line.

source $(dirname $0)/../../scripts/library.sh

# Delete old images in the given GCR.
# Parameters: $1 - gcr to be cleaned up (e.g. gcr.io/fooProj)
#             $2 - days to keep images
function delete_old_images_from_gcr() {
  [[ -z $1 ]] && abort "missing gcr name"
  [[ -z $2 ]] && abort "missing days to keep images"

  is_protected_gcr $1 && \
    abort "Target GCR set to $1, which is forbidden"

  for image in $(gcloud --format='value(name)' container images list --repository=$1); do
      echo "Checking ${image} for removal"

      delete_old_images_from_gcr ${image} $2

      local target_date=$(date -d "`date`-$2days" +%Y-%m-%d)
      for digest in $(gcloud --format='get(digest)' container images list-tags ${image} \
          --filter="timestamp.datetime<${target_date}" --limit=99999); do
        local full_image="${image}@${digest}"
        echo "Deleting image: ${full_image}"
        if (( DRY_RUN )); then
          echo "[DRY RUN] gcloud container images delete -q --force-delete-tags ${full_image}"
        else
          gcloud container images delete -q --force-delete-tags ${full_image}
        fi
      done
  done
}

# Delete old images in the given GCP projects
# Parameters: $1 - array of projects names
#             $2 - days to keep images
function delete_old_gcr_images() {
  [[ -z $1 ]] && abort "missing project names"
  [[ -z $2 ]] && abort "missing days to keep images"

  for project in $1; do
    echo "Start deleting images from ${project}"
    delete_old_images_from_gcr "gcr.io/${project}" $2
  done
}

# Delete old clusters in the given GCP projects
# Parameters: $1 - array of projects names
#             $2 - hours to keep images
function delete_old_test_clusters() {
  [[ -z $1 ]] && abort "missing project names"
  [[ -z $2 ]] && abort "missing hours to keep clusters"

  for project in $1; do
    echo "Start deleting clusters from ${project}"

    is_protected_project $project && \
      abort "Target project set to $project, which is forbidden"

    local current_time=$(date +%s)
    local target_time=$(date -d "`date -d @${current_time}`-$2hours" +%s)
    # Fail if the difference of current time and target time is not 3600 times hours to keep
    if (( ! DRY_RUN )); then # Don't check on dry runs, as dry run is used for unit testing
      [[ "$((3600*$2))" -eq "$(($current_time-$target_time))" ]] || abort "date operation failed"
    fi

    gcloud --format='get(name,createTime,zone)' container clusters list --project=$project --limit=99999 | \
    while read cluster_name cluster_createtime cluster_zone; do
      [[ -n "${cluster_name}" ]]  && [[ -z "${cluster_zone}" ]] && abort "list cluster output missing cluster zone"
      echo "Checking ${cluster_name} for removal"
      local create_time=$(date -d "$cluster_createtime" +%s)
      [[ $create_time -gt $current_time ]] && abort "cluster creation time shouldn't be newer than current time"
      [[ $create_time -gt $target_time ]] && echo "skip deleting as it's created within $2 hours" && continue
      if (( DRY_RUN )); then
        echo "[DRY RUN] gcloud container clusters delete -q ${cluster_name} -zone ${cluster_zone}"
      else
        gcloud container clusters delete -q ${cluster_name} -zone ${cluster_zone}
      fi
    done
  done
}
