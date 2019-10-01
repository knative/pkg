#!/usr/bin/env bash

# Copyright 2019 The Knative Authors
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

set -e

readonly NUMBER=${1:?"First argument is the number of new projects to create."}
readonly BILLING_ACCOUNT=${2:?"Second argument must be the billing account."}

readonly RESOURCE_FILE="resources.yaml"

if [[ ! -f ${RESOURCE_FILE} || ! -w ${RESOURCE_FILE} ]]; then
  echo "${RESOURCE_FILE} does not exist or is not writable"
  exit 1
fi

# Get the index of the last boskos project from the resources file
LAST_INDEX=$(grep "knative-boskos-" ${RESOURCE_FILE} | grep -o "[0-9]\+" | sort -nr | head -1)
for (( i=1; i<=${NUMBER}; i++ )); do
  PROJECT="knative-boskos-$(( ${LAST_INDEX} + i ))"
  # This Folder ID is google.com/google-default
  # If this needs to be changed for any reason, GCP project settings must be updated.
  # Details are available in Google's internal issue 137963841.
  gcloud projects create ${PROJECT} --folder=396521612403
  gcloud beta billing projects link ${PROJECT} --billing-account=${BILLING_ACCOUNT}

  # Set permissions for this project
  "$(dirname $0)/set_permissions.sh" ${PROJECT}

  LAST_PROJECT=$(grep "knative-boskos-" ${RESOURCE_FILE} | tail -1)
  sed "/${LAST_PROJECT}/a\ \ -\ ${PROJECT}" -i ${RESOURCE_FILE}
done
