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

readonly PROJECT=${1:?"First argument must be the boskos project name."}

# APIs, Permissions and accounts to be set.
# * Resources with API names will be enabled.
# * Resources starting with "role/" indicates that the next accounts will be added with such role.
# * Resources named as emails are added to the project using the last role defined.
#   - @google.com addresses are assumed to be groups.
#   - @googlegroups.com addresses are assumed to be groups.
#   - @...gserviceaccount.com addresses are assumed to be service accounts.
readonly RESOURCES=(
    "roles/owner"
    "prime-engprod-sea@google.com"

    "roles/editor"
    "knative-productivity-admins@googlegroups.com"
    "knative-tests@appspot.gserviceaccount.com"
    "prow-job@knative-tests.iam.gserviceaccount.com"
    "prow-job@knative-nightly.iam.gserviceaccount.com"
    "prow-job@knative-releases.iam.gserviceaccount.com"

    "roles/viewer"
    "knative-dev@googlegroups.com"

    # APIs to enable
    "cloudresourcemanager.googleapis.com"
    "compute.googleapis.com"
    "container.googleapis.com"
)

# Loop through the list of resources and add them.

# Start with a non-existing role, so gcloud clearly fails if resources are set incorrectly.
role="unknown"
for res in ${RESOURCES[@]}; do
  if [[ ${res} == roles/* ]]; then
    role=${res}
    continue
  fi
  if [[ ${res} == *.googleapis.com ]]; then
    echo "NOTE: Enabling API ${res}"
    gcloud services enable ${res} --project=${PROJECT}
    continue
  fi
  type="user"
  [[ ${res} == *@googlegroups.com || ${res} == *@google.com ]] && type="group"
  [[ ${res} == *.gserviceaccount.com ]] && type="serviceAccount"
  echo "NOTE: Adding ${res} as ${role}"
  gcloud projects add-iam-policy-binding ${PROJECT} --member ${type}:${res} --role ${role}
done
