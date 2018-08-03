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

set -o errexit
set -o nounset
set -o pipefail

readonly PKG_ROOT_DIR="$(git rev-parse --show-toplevel)"
readonly TMP_DIFFROOT="$(mktemp -d -p ${PKG_ROOT_DIR})"

cleanup() {
  rm -rf "${TMP_DIFFROOT}"
}

trap "cleanup" EXIT SIGINT

cleanup

# Save working tree state
mkdir -p "${TMP_DIFFROOT}/apis"
mkdir -p "${TMP_DIFFROOT}/client"
cp -aR "${PKG_ROOT_DIR}/Gopkg.lock" "${PKG_ROOT_DIR}/apis" "${PKG_ROOT_DIR}/client" "${PKG_ROOT_DIR}/vendor" "${TMP_DIFFROOT}"

# TODO(mattmoor): We should be able to rm -rf pkg/client/ and vendor/

"${PKG_ROOT_DIR}/hack/update-codegen.sh"
echo "Diffing ${PKG_ROOT_DIR} against freshly generated codegen"
ret=0
diff -Naupr "${PKG_ROOT_DIR}/apis" "${TMP_DIFFROOT}/apis" || ret=1
diff -Naupr "${PKG_ROOT_DIR}/client" "${TMP_DIFFROOT}/client" || ret=1

# Restore working tree state
rm -fr "${PKG_ROOT_DIR}/Gopkg.lock" "${PKG_ROOT_DIR}/apis" "${PKG_ROOT_DIR}/client" "${PKG_ROOT_DIR}/vendor"
cp -aR "${TMP_DIFFROOT}"/* "${PKG_ROOT_DIR}"

if [[ $ret -eq 0 ]]
then
  echo "${PKG_ROOT_DIR} up to date."
else
  echo "ERROR: ${PKG_ROOT_DIR} is out of date. Please run ./hack/update-codegen.sh"
  exit 1
fi
