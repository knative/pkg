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

source $(dirname $0)/test-helper.sh
source $(dirname $0)/../../tools/cleanup/cleanup-functions.sh

readonly _FAKE_NIGHTLY_PROJECT_NAME="gcr.io/knative-nightly"
readonly _FAKE_BOSKOS_PROJECT_NAME="gcr.io/fake-boskos-project"
readonly _PROJECT_RESOURCE_YAML="ci/prow/boskos/resources.yaml"
readonly _RE_PROJECT_NAME="knative-boskos-[a-zA-Z0-9]+"

# Call "cleanup.sh" function with given parameters
# Parameters: $1..$n - parameters passed to "cleanup.sh" script
function cleanup_script() {
  "./tools/cleanup/cleanup.sh" $@
}

set -e

cd ${REPO_ROOT_DIR}

echo ">> Testing directly invoking cleanup script"

test_function ${FAILURE} "error: missing parameter" cleanup_script

test_function ${FAILURE} "error: expecting value following" cleanup_script --project-resource-yaml --dry-run
test_function ${FAILURE} "error: expecting value following" cleanup_script --re-project-name --dry-run
test_function ${FAILURE} "error: expecting value following" cleanup_script --gcr-to-cleanup --dry-run
test_function ${FAILURE} "error: expecting value following" cleanup_script --days-to-keep-images --dry-run
test_function ${FAILURE} "error: expecting value following" cleanup_script --hours-to-keep-clusters --dry-run
test_function ${FAILURE} "error: expecting value following" cleanup_script --artifacts --dry-run

test_function ${FAILURE} "error: days to keep" cleanup_script --days-to-keep-images "a" --dry-run
test_function ${FAILURE} "error: hours to keep" cleanup_script --hours-to-keep-clusters "a" --dry-run

# Test individual functions
echo ">> Testing deleting images from single project"

test_function ${FAILURE} "error: missing gcr" delete_old_images_from_gcr
test_function ${FAILURE} "error: missing days" delete_old_images_from_gcr ${_FAKE_BOSKOS_PROJECT_NAME}
test_function ${SUCCESS} "" mock_gcloud_function delete_old_images_from_gcr ${_FAKE_BOSKOS_PROJECT_NAME} 1

echo ">> Testing deleting images from multiple projects"

test_function ${FAILURE} "error: missing project names" delete_old_gcr_images
test_function ${FAILURE} "error: missing days" delete_old_gcr_images "${_FAKE_BOSKOS_PROJECT_NAME}1"
test_function ${SUCCESS} "Start" mock_gcloud_function delete_old_gcr_images "${_PROJECT_RESOURCE_YAML}1 ${_PROJECT_RESOURCE_YAML}2" 99

echo ">> Testing deleting clusters from multiple projects"

test_function ${FAILURE} "error: missing project names" delete_old_test_clusters
test_function ${FAILURE} "error: missing hours" delete_old_test_clusters "${_FAKE_BOSKOS_PROJECT_NAME}1"
test_function ${SUCCESS} "Start" mock_gcloud_function delete_old_test_clusters "${_FAKE_BOSKOS_PROJECT_NAME}1 ${_FAKE_BOSKOS_PROJECT_NAME}2" 99

echo ">> All tests passed"
