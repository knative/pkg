#!/usr/bin/env bash

# Copyright 2019 The Knative Authors.
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

GENS="$1"
CLIENT_PKG="$2"
APIS_PKG="$3"
GROUPS_WITH_VERSIONS="$4"
shift 4

function codegen::join() { local IFS="$1"; shift; echo "$*"; }

# enumerate group versions
FQ_APIS=() # e.g. k8s.io/api/apps/v1
for GVs in ${GROUPS_WITH_VERSIONS}; do
  IFS=: read G Vs <<<"${GVs}"

  # enumerate versions
  for V in ${Vs//,/ }; do
    FQ_APIS+=(${APIS_PKG}/${G}/${V})
  done
done

if grep -qw "injection" <<<"${GENS}"; then
  if [[ -z "${OUTPUT_PKG:-}" ]]; then
    OUTPUT_PKG="${CLIENT_PKG}/injection"
  fi

  if [[ -z "${VERSIONED_CLIENTSET_PKG:-}" ]]; then
    VERSIONED_CLIENTSET_PKG="${CLIENT_PKG}/clientset/versioned"
  fi

  if [[ -z "${EXTERNAL_INFORMER_PKG:-}" ]]; then
    EXTERNAL_INFORMER_PKG="${CLIENT_PKG}/informers/externalversions"
  fi

  if [[ -z "${LISTERS_PKG:-}" ]]; then
    LISTERS_PKG="${CLIENT_PKG}/listers"
  fi

  echo "Generating injection for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}"

  # Clear old injection
  rm -rf ${OUTPUT_PKG}

  go run knative.dev/pkg/codegen/cmd/injection-gen \
    --input-dirs $(codegen::join , "${FQ_APIS[@]}") \
    --versioned-clientset-package ${VERSIONED_CLIENTSET_PKG} \
    --external-versions-informers-package ${EXTERNAL_INFORMER_PKG} \
    --listers-package ${LISTERS_PKG} \
    --output-package ${OUTPUT_PKG} \
    "$@"
fi

