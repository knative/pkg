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

CLUSTER            ?= prow
PROJECT            ?= knative-tests
ZONE               ?= us-central1-f
JOB_NAMESPACE      ?= test-pods
KNATIVE_CONFIG     ?= config_knative.yaml
SKIP_CONFIG_BACKUP ?=

PROW_DIR         ?= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
PROW_PLUGINS     ?= $(PROW_DIR)/plugins.yaml
PROW_CONFIG      ?= $(PROW_DIR)/config.yaml
PROW_DEPLOYS     ?= $(PROW_DIR)/deployments
PROW_CONFIG_GCS  ?= gs://knative-prow/configs

TESTGRID_CONFIG  ?= $(PROW_DIR)/../testgrid/config.yaml

BOSKOS_DIR       ?= $(PROW_DIR)/boskos
BOSKOS_DEPLOYS   ?= $(BOSKOS_DIR)/deployments
BOSKOS_RESOURCES ?= $(BOSKOS_DIR)/resources.yaml

SET_CONTEXT   := gcloud container clusters get-credentials "$(CLUSTER)" --project="$(PROJECT)" --zone="$(ZONE)"
UNSET_CONTEXT := kubectl config unset current-context

get-cluster-credentials:
	$(SET_CONTEXT)

unset-cluster-credentials:
	$(UNSET_CONTEXT)

config:
	go run *_config.go --prow-config-output=$(PROW_CONFIG) --testgrid-config-output=$(TESTGRID_CONFIG) $(KNATIVE_CONFIG)

# Update all prow job configs
update-config:
	$(SET_CONTEXT)
ifndef SKIP_CONFIG_BACKUP
	$(eval OLD_YAML_CONFIG := $(shell mktemp))
	$(eval NEW_YAML_CONFIG := $(shell mktemp))
	$(eval GCS_DEST := $(PROW_CONFIG_GCS)/config-$(shell date '+%Y_%m_%d_%H:%M:%S').yaml)
	@kubectl get configmap config -o jsonpath="{.data['config\.yaml']}" 2>/dev/null > "${OLD_YAML_CONFIG}"
	@gsutil cp "${OLD_YAML_CONFIG}" "${GCS_DEST}" > /dev/null
	@sed -e "s/((COMMIT_HASH_TOKEN))/$(shell git rev-parse HEAD)/" $(PROW_CONFIG) > "${NEW_YAML_CONFIG}"
else
        $(eval NEW_YAML_CONFIG := $(PROW_CONFIG))
endif
	kubectl create configmap config --from-file=config.yaml=${NEW_YAML_CONFIG} --dry-run -o yaml | kubectl replace configmap config -f -
	$(UNSET_CONTEXT)
ifndef SKIP_CONFIG_BACKUP
	diff "${OLD_YAML_CONFIG}" "${NEW_YAML_CONFIG}" --color=auto || true
	@echo "Inspect uploaded config file at: ${NEW_YAML_CONFIG}"
	@echo "Old config file saved at: ${GCS_DEST}"
endif

# Update plugins
update-plugins:
	$(SET_CONTEXT)
	kubectl create configmap plugins --from-file=plugins.yaml=$(PROW_PLUGINS) --dry-run -o yaml | kubectl replace configmap plugins -f -
	$(UNSET_CONTEXT)

# Update all deployments of boskos
update-all-boskos-deployments:
	$(SET_CONTEXT)
	@for f in $(shell ls $(BOSKOS_DEPLOYS)/*.yaml); do kubectl apply -f $${f}; done
	$(UNSET_CONTEXT)

# Update single deployment of boskos, expect passing in ${NAME} like `make update-single-boskos-deployment NAME=janitor`
update-single-boskos-deployment:
	$(SET_CONTEXT)
	kubectl apply -f $(BOSKOS_DEPLOYS)/$(NAME).yaml
	$(UNSET_CONTEXT)

update-boskos-resource:
	$(SET_CONTEXT)
	kubectl create configmap resources --from-file=config=$(BOSKOS_RESOURCES) --dry-run -o yaml | kubectl --namespace="$(JOB_NAMESPACE)" replace configmap resources -f -
	$(UNSET_CONTEXT)

# Update all deployment of cluster
update-all-cluster-deployments:
	$(SET_CONTEXT)
	@for f in $(shell ls $(PROW_DEPLOYS)/*.yaml); do kubectl apply -f $${f}; done
	$(UNSET_CONTEXT)

# Update single deployment of cluster, expect passing in ${NAME} like `make update-single-cluster-deployment NAME=crier_deployment`
update-single-cluster-deployment:
	$(SET_CONTEXT)
	kubectl apply -f $(PROW_DEPLOYS)/$(NAME).yaml
	$(UNSET_CONTEXT)

# Update all resources on Prow cluster except boskos resource
update-full: update-all-cluster-deployments update-all-boskos-deployments update-plugins update-config

test:
	@echo "*** Checking config generator for prow and testgrid"
	$(eval TMP_YAML_PROW := $(shell mktemp))
	$(eval TMP_YAML_TESTGRID := $(shell mktemp))
	go run *_config.go --prow-config-output="$(TMP_YAML_PROW)" --testgrid-config-output="$(TMP_YAML_TESTGRID)" $(KNATIVE_CONFIG)
	diff $(PROW_CONFIG) $(TMP_YAML_PROW)
	diff $(TESTGRID_CONFIG) $(TMP_YAML_TESTGRID)
	@echo "*** Checking configs validity"
	bazel run @k8s//prow/cmd/checkconfig -- --plugin-config=$(PROW_PLUGINS) --config-path=$(PROW_CONFIG)
	bazel run @k8s//testgrid/cmd/configurator -- \
		--yaml=$(TESTGRID_CONFIG) \
		--validate-config-file
