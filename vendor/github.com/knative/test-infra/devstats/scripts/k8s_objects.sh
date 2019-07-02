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

function random_string {
  < /dev/urandom tr -dc A-Za-z0-9 | head -c14
}

kubectl create -f Kubernetes/env-variables.yaml
kubectl create secret generic grafana-pass --from-literal=grafana_admin_password="$(random_string)"
kubectl create secret generic postgres-pass --from-literal=pg_postgres_password="$(random_string)"
kubectl create secret generic gha-admin-pass --from-literal=pg_gha_admin_password="$(random_string)"
kubectl create -f Kubernetes/volumes.yaml
kubectl create -f Kubernetes/postgres-service.yaml
kubectl create -f Kubernetes/cli-home.yaml
