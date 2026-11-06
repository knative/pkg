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

# Sync semconv imports with the resolved OTEL module.
log.step "Syncing semconv imports"
OTEL_VERSION=$(go list -m -f '{{.Version}}' go.opentelemetry.io/otel 2>/dev/null || true)
if [ -n "$OTEL_VERSION" ]; then
  group "Detected go.opentelemetry.io/otel ${OTEL_VERSION}"

  # Semconv versions are package directories inside the OTEL module.
  OTEL_DIR=$(GOFLAGS=-mod=mod go list -m -f '{{.Dir}}' go.opentelemetry.io/otel 2>/dev/null || true)
  SEMCONV_VERSION=""
  if [ -d "${OTEL_DIR}/semconv" ]; then
    SEMCONV_VERSION=$(find "${OTEL_DIR}/semconv" -mindepth 1 -maxdepth 1 \
      -type d -name 'v[0-9]*.[0-9]*.[0-9]*' -exec basename {} \; | sort -V | tail -1)
  fi

  if [ -n "$SEMCONV_VERSION" ]; then
    group "Updating semconv imports to ${SEMCONV_VERSION}"
    mapfile -t SEMCONV_FILES < <(
      find observability -type f -name '*.go' -exec \
        grep -lE 'go\.opentelemetry\.io/otel/semconv/v[0-9]+\.[0-9]+\.[0-9]+' {} + || true
    )
    SEMCONV_IMPORTS_CHANGED=false
    for file in "${SEMCONV_FILES[@]}"; do
      current_versions=$(grep -oE \
        'go\.opentelemetry\.io/otel/semconv/v[0-9]+\.[0-9]+\.[0-9]+' "$file" | \
        sed -E 's#.*(v[0-9]+\.[0-9]+\.[0-9]+)$#\1#' | sort -u)
      if [ "$current_versions" != "$SEMCONV_VERSION" ]; then
        group "  ${file}"
        sed -i -E "s|(go.opentelemetry.io/otel/semconv/)v[0-9]+\.[0-9]+\.[0-9]+|\1$SEMCONV_VERSION|g" "$file"
        SEMCONV_IMPORTS_CHANGED=true
      fi
    done

    [ "$SEMCONV_IMPORTS_CHANGED" = true ] && go_update_deps

  else
    group "Could not determine semconv version, skipping"
  fi
fi
