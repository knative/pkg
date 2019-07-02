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

mkdir -p /mount/data/src/
chmod -R a+rw /mount/data

mkdir -p /mount/data/devstats_repos/knative
git clone https://github.com/knative/serving.git /mount/data/devstats_repos/knative/serving

cd /mount/data/src/
git clone https://github.com/knative/test-infra.git
cd test-infra/devstats
./scripts/copy_devstats_binaries.sh
./scripts/copy_grafana_files.sh
./scripts/generate_repo_groups.sh
cp -r $GOPATH/src/devstats/util_sql/ .
mkdir -p ./metrics
cp -r $GOPATH/src/devstats/metrics/shared ./metrics
cp -r $GOPATH/src/devstats/metrics/knative ./metrics
cp -r $GOPATH/src/devstats/hide .
cp -r $GOPATH/src/devstats/git .
cp -r $GOPATH/src/devstats/knative .
cp -r $GOPATH/src/devstats/shared .
cp $GOPATH/src/devstats/github_users.json .
cp $GOPATH/src/devstats/scripts/clean_affiliations.sql ./scripts
rm -rf /etc/gha2db && ln -sf /mount/data/src/test-infra/devstats /etc/gha2db
