/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"testing"

	"knative.dev/pkg/apis"
)

func TestGetTopLevelCondition(t *testing.T) {
	resource := KResource{}

	condSet := apis.NewLivingConditionSet("Foo")
	mgr := condSet.Manage(resource.GetStatus())
	mgr.InitializeConditions()

	if resource.GetTopLevelConditionType() != apis.ConditionReady {
		t.Error("Expected Ready as happy condition for living condition set type")
	}

	condSet = apis.NewBatchConditionSet("Foo")
	mgr = condSet.Manage(resource.GetStatus())
	mgr.InitializeConditions()

	if resource.GetTopLevelConditionType() != apis.ConditionSucceeded {
		t.Error("Expected Succeeded as happy condition for living condition set type")
	}
}
