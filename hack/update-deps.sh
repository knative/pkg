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

set -o errexit
set -o nounset
set -o pipefail

source $(dirname "$0")/../vendor/knative.dev/hack/library.sh

go_update_deps "$@"

# Match the SDK's resource schema, not the newest available semconv package.
log.step "Syncing semconv imports with the OTel SDK"
SEMCONV_IMPORT=$(go list -mod=mod -f '{{join .Imports "\n"}}' \
  go.opentelemetry.io/otel/sdk/resource | grep -F 'go.opentelemetry.io/otel/semconv/')

SEMCONV_CHANGED=false
while IFS= read -r -d '' file; do
  sed -i.semconv-backup -E \
    "s|go\\.opentelemetry\\.io/otel/semconv/v[0-9]+\\.[0-9]+\\.[0-9]+|${SEMCONV_IMPORT}|g" "$file"
  if ! cmp -s "$file" "$file.semconv-backup"; then
    SEMCONV_CHANGED=true
  fi
  rm "$file.semconv-backup"
done < <(find observability -type f -name '*.go' -print0)

# Refresh dependencies and vendoring only when imports changed.
if [ "$SEMCONV_CHANGED" = true ]; then
  go_update_deps
fi
