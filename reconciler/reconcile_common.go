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

	"k8s.io/apimachinery/pkg/api/equality"
	"knative.dev/pkg/apis"
	v1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/logging"
)

var condSet = apis.NewLivingConditionSet()

// PostProcessReconcile contains logic to apply after reconciliation of a resource.
func PostProcessReconcile(ctx context.Context, old v1.KRShaped, new v1.KRShaped, reconcileEvent Event) {
	logger := logging.FromContext(ctx)
	newStatus := new.GetStatus()

	// Bump observed generation to denote that we have processed this
	// generation regardless of success or failure.
	newStatus.ObservedGeneration = new.GetObjectMeta().Generation

	if newStatus.ObservedGeneration != old.GetStatus().ObservedGeneration && reconcileEvent != nil {
		oldRc := old.GetStatus().GetCondition(apis.ConditionReady)
		rc := newStatus.GetCondition(apis.ConditionReady)
		// if a new generation is observed and reconciliation reported an error event
		// the reconciler should change the ready state. By default we will set unknown.
		if equality.Semantic.DeepEqual(oldRc, rc) {
			logger.Warn("A reconconiler observed a new generation without updating the resource status")
			condSet.Manage(newStatus).MarkUnknown(apis.ConditionReady, "", "unsucessfully observed a new generation")
		}
	}
}
