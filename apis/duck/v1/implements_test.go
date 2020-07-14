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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"

	"knative.dev/pkg/apis/duck"
	"knative.dev/pkg/apis/duck/ducktypes"
)

func TestTypesImplements(t *testing.T) {
	testCases := []struct {
		instance interface{}
		iface    ducktypes.Implementable
	}{
		{instance: &AddressableType{}, iface: &Addressable{}},
		{instance: &KResource{}, iface: &Conditions{}},
	}
	for _, tc := range testCases {
		if err := duck.VerifyType(tc.instance, tc.iface); err != nil {
			t.Error(err)
		}
	}
}

func TestImplementsPodSpecable(t *testing.T) {
	instances := []interface{}{
		&WithPod{},
		&appsv1.ReplicaSet{},
		&appsv1.Deployment{},
		&appsv1.StatefulSet{},
		&appsv1.DaemonSet{},
		&batchv1.Job{},
	}
	for _, instance := range instances {
		if err := duck.VerifyType(instance, &PodSpecable{}); err != nil {
			t.Error(err)
		}
	}
}
