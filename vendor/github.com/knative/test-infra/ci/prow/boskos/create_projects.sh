#!/bin/bash

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

readonly FIRST=${1:?"First argument is the first number of the new project(s)."}
readonly NUMBER=${2:?"Second argument is the number of new projects."}
readonly BILLING_ACCOUNT=${3:?"Third argument must be the billing account."}
readonly OUTPUT_FILE=${4:?"Fourth argument should be a file name all project names will be appended to in a resources.yaml format."}

for (( i=0; i<${NUMBER}; i++ )); do
  PROJECT="knative-boskos-$(( i + ${FIRST} ))"
  # This Folder ID is google.com/google-default
  gcloud projects create ${PROJECT} --folder=396521612403
  gcloud beta billing projects link ${PROJECT} --billing-account=${BILLING_ACCOUNT}
  "$(dirname $0)/set_permissions.sh" ${PROJECT}
  echo "  - ${PROJECT}" >> ${OUTPUT_FILE}
done
