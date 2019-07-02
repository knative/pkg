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

# This is a script to clean up stale resources

source $(dirname $0)/cleanup-functions.sh

# Global variables
DAYS_TO_KEEP_IMAGES=365 # Keep images up to 1 year by default
HOURS_TO_KEEP_CLUSTERS=720 # keep clusters up to 30 days by default
RE_PROJECT_NAME="knative-boskos-[a-zA-Z0-9]+"
PROJECT_RESOURCE_YAML=""
ARTIFACTS_DIR=""
DRY_RUN=0

function parse_args() {
  while [[ $# -ne 0 ]]; do
    local parameter=$1
    case ${parameter} in
      --dry-run) DRY_RUN=1 ;;
      *)
        [[ -z $2 || $2 =~ ^-- ]] && abort "expecting value following $1"
        shift
        case ${parameter} in
          --project-resource-yaml) PROJECT_RESOURCE_YAML=$1 ;;
          --re-project-name) RE_PROJECT_NAME=$1 ;;
          --days-to-keep-images) DAYS_TO_KEEP_IMAGES=$1 ;;
          --hours-to-keep-clusters) HOURS_TO_KEEP_CLUSTERS=$1 ;;
          --artifacts) ARTIFACTS_DIR=$1 ;;
          --service-account)
            gcloud auth activate-service-account --key-file=$1 || exit 1
            ;;
          *) abort "unknown option ${parameter}" ;;
        esac
    esac
    shift
  done

  is_int ${DAYS_TO_KEEP_IMAGES} || abort "days to keep has to be integer"
  is_int ${HOURS_TO_KEEP_CLUSTERS} || abort "hours to keep clusters has to be integer"

  readonly DAYS_TO_KEEP_IMAGES
  readonly HOURS_TO_KEEP_CLUSTERS
  readonly PROJECT_RESOURCE_YAML
  readonly RE_PROJECT_NAME
  readonly ARTIFACTS_DIR
  readonly DRY_RUN
}

# Script entry point

cd ${REPO_ROOT_DIR}

if [[ -z $1 ]]; then
  abort "missing parameters to the tool"
fi

parse_args $@

(( DRY_RUN )) && echo "-- Running in dry-run mode, no resource deletion --"
echo "Iterating over projects defined in '${PROJECT_RESOURCE_YAML}', matching '${RE_PROJECT_NAME}"
target_projects="$(grep -Eio "${RE_PROJECT_NAME}" "${PROJECT_RESOURCE_YAML}")"
[[ $? -eq 0 ]] || abort "no project found in $PROJECT_RESOURCE_YAML"

# delete old gcr images
echo "Removing images with following rules:"
echo "- older than ${DAYS_TO_KEEP_IMAGES} days"
delete_old_gcr_images "${target_projects}" "${DAYS_TO_KEEP_IMAGES}"
# delete old clusters
echo "Removing clusters with following rules:"
echo "- older than ${HOURS_TO_KEEP_CLUSTERS} hours"
delete_old_test_clusters "${target_projects}" "${HOURS_TO_KEEP_CLUSTERS}"

# Gubernator considers job failure if "junit_*.xml" not found under artifact,
#   create a placeholder file to make this job succeed
if [[ ! -z ${ARTIFACTS_DIR} ]]; then
  echo "<testsuite time='0'/>" > "${ARTIFACTS_DIR}/junit_knative.xml"
fi
