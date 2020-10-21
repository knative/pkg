#!/usr/bin/env bash

# Copyright 2020 The Knative Authors
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

# This script runs specific integration tests which may be large and
# not play nicely with other

source $(dirname $0)/../vendor/knative.dev/test-infra/scripts/e2e-tests.sh

e2e_test_dirs=("metrics")
for module in ${e2e_test_dirs[@]}; do
    go_test_e2e -timeout=15m ./${module}
done

