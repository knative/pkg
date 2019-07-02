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

binaries="structure runq gha2db calc_metric gha2db_sync import_affs annotations tags webhook devstats get_repos merge_dbs replacer vars ghapi2db columns hide_data website_data sync_issues devstats_api_server gha2es sqlitedb"
for f in $binaries
do
  cp $GOPATH/bin/$f ./ || exit 1
done
echo "DevStats binaries copied."
