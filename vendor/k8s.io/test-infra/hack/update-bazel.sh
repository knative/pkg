#!/usr/bin/env bash
# Copyright 2016 The Kubernetes Authors.
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

if [[ -n "${BUILD_WORKSPACE_DIRECTORY:-}" ]]; then # Running inside bazel
  echo "Updating bazel rules..." >&2
elif ! command -v bazel &>/dev/null; then
  echo "Install bazel at https://bazel.build" >&2
  exit 1
elif ! bazel query @io_k8s_test_infra//vendor/github.com/bazelbuild/bazel-gazelle/cmd/gazelle &>/dev/null; then
  (
    set -o xtrace
    bazel run @io_k8s_test_infra//hack:bootstrap-testinfra
    bazel run @io_k8s_test_infra//hack:update-bazel
  )
  exit 0
else
  (
    set -o xtrace
    bazel run @io_k8s_test_infra//hack:update-bazel
  )
  exit 0
fi

gazelle=$(realpath "$1")
kazel=$(realpath "$2")

cd "$BUILD_WORKSPACE_DIRECTORY"

if [[ ! -f go.mod ]]; then
    echo "No module defined, see https://github.com/golang/go/wiki/Modules#how-to-define-a-module" >&2
    exit 1
fi

"$gazelle" fix --external=vendored
"$kazel" --cfg-path=./hack/.kazelcfg.json
"$gazelle" fix --external=vendored # TODO(fejta): remove this
