/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconciler

import (
	"context"

	"github.com/pkg/errors"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
)

const failedGenerationBump = "NewObservedGenFailure"

// PreProcessReconcile contains logic to apply before reconciliation of a resource.
func PreProcessReconcile(ctx context.Context, resource duckv1.KRShaped) {
	condSet, err := getDefaultConditionSet(resource)
	if err != nil {
		return
	}

	newStatus := resource.GetStatus()
	if newStatus.ObservedGeneration != resource.GetObjectMeta().GetGeneration() {
		// Reset ready to unknown. The reconciler is expected to overwrite this.
		manager := condSet.Manage(newStatus)
		manager.MarkUnknown(
			manager.GetTopLevelCondition().Type, failedGenerationBump, "unsucessfully observed a new generation")
	}
}

// PostProcessReconcile contains logic to apply after reconciliation of a resource.
func PostProcessReconcile(ctx context.Context, resource duckv1.KRShaped) {
	logger := logging.FromContext(ctx)
	newStatus := resource.GetStatus()

	// Bump observed generation to denote that we have processed this
	// generation regardless of success or failure.
	newStatus.ObservedGeneration = resource.GetObjectMeta().GetGeneration()

	rc := newStatus.GetCondition(resource.GetTopLevelConditionType())
	if rc.Reason == failedGenerationBump {
		logger.Warn("A reconconiler observed a new generation without updating the resource status")
	}
}

func getDefaultConditionSet(resource duckv1.KRShaped) (apis.ConditionSet, error) {
	switch resource.GetTopLevelConditionType() {
	case apis.ConditionReady:
		return apis.NewLivingConditionSet(), nil
	case apis.ConditionSucceeded:
		return apis.NewBatchConditionSet(), nil
	default:
		return apis.ConditionSet{}, errors.Errorf("No ConditionSet found for type %s", resource.GetTypeMeta().Kind)
	}
}
