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
func PreProcessReconcile(ctx context.Context, resource duckv1.KRShaped) {
	newStatus := resource.GetStatus()

	// We may be reading a version of the object that was stored at an older version
	// and may not have had all of the assumed defaults specified.  This won't result
	// in this getting written back to the API Server, but lets downstream logic make
	// assumptions about defaulting.
	if d, ok := resource.(apis.Defaultable); ok {
		d.SetDefaults(ctx)
	}

	// Ensure conditions are initialized before we modify.
	condSet := resource.GetConditionSet()
	manager := condSet.Manage(newStatus)
	manager.InitializeConditions()

	if newStatus.ObservedGeneration != resource.GetGeneration() {
		// Reset Ready/Successful to unknown. The reconciler is expected to overwrite this.
		manager.MarkUnknown(condSet.GetTopLevelConditionType(), failedGenerationBump, "unsuccessfully observed a new generation")
	}
}

// PostProcessReconcile contains logic to apply after reconciliation of a resource.
func PostProcessReconcile(ctx context.Context, resource duckv1.KRShaped) {
	logger := logging.FromContext(ctx)
	newStatus := resource.GetStatus()
	mgr := resource.GetConditionSet().Manage(newStatus)

	// Bump observed generation to denote that we have processed this
	// generation regardless of success or failure.
	newStatus.ObservedGeneration = resource.GetGeneration()

	if rc := mgr.GetTopLevelCondition(); rc == nil {
		logger.Warn("A reconciliation included no top-level condition")
	} else if rc.Reason == failedGenerationBump {
		logger.Warn("A reconciler observed a new generation without updating the resource status")
	}
}
