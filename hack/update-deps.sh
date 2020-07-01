#!/usr/bin/env bash

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

readonly ROOT_DIR=$(dirname $0)/..
source ${ROOT_DIR}/vendor/knative.dev/test-infra/scripts/library.sh

set -o errexit
set -o nounset
set -o pipefail

cd ${ROOT_DIR}

# The list of dependencies that we track at HEAD and periodically
# float forward in this repository.
FLOATING_DEPS=(
  "knative.dev/test-infra@master"
)

readonly ALLOWED_UPDATE_OPTIONS=("all" "knative" "none")
# Parse flags to determine any we should pass to dep.
UPDATE="none"
while [[ $# -ne 0 ]]; do
  parameter=$1
  case ${parameter} in
    --upgrade=*) UPDATE="${parameter#*=}";;
    --upgrade) UPDATE="knative" ;;
    *) abort "unknown option ${parameter}" ;;
  esac
  shift
done
readonly UPDATE

if [[ ! " ${ALLOWED_UPDATE_OPTIONS[@]} " =~ " ${UPDATE} " ]]; then
  echo "Option '${UPDATE}' is not supported. Supported update types are all, knative, and none."
  exit 1
fi

if [[ "${UPDATE?}" != "none" ]]; then
  go get -d ${FLOATING_DEPS[@]}
  go get -u ./...
fi

if [[ "${UPDATE?}" == "all" ]]; then
  go get -u ./...
fi

# Prune modules.
go mod tidy
go mod vendor

rm -rf $(find vendor/ -name 'OWNERS')
rm -rf $(find vendor/ -name '*_test.go')
