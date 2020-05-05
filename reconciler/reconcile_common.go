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

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
)

const failedGenerationBump = "NewObservedGenFailure"

// PreProcessReconcile contains logic to apply before reconciliation of a resource.
func PreProcessReconcile(ctx context.Context, new duckv1.KRShaped) {
	var condSet = apis.NewLivingConditionSet()
	newStatus := new.GetStatus()

	if newStatus.ObservedGeneration != new.GetObjectMeta().GetGeneration() {
		// Reset ready to unknown. The reconciler is expected to overwrite this.
		condSet.Manage(newStatus).MarkUnknown(
			apis.ConditionReady, failedGenerationBump, "unsucessfully observed a new generation")
	}
}

// PostProcessReconcile contains logic to apply after reconciliation of a resource.
func PostProcessReconcile(ctx context.Context, new duckv1.KRShaped) {
	logger := logging.FromContext(ctx)
	newStatus := new.GetStatus()

	// Bump observed generation to denote that we have processed this
	// generation regardless of success or failure.
	newStatus.ObservedGeneration = new.GetObjectMeta().GetGeneration()

	rc := newStatus.GetCondition(apis.ConditionReady)
	if rc.Reason == failedGenerationBump {
		logger.Warn("A reconconiler observed a new generation without updating the resource status")
	}
}
