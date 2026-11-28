/*
Copyright 2026 The Knative Authors

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

package ha

import (
	"errors"
	"strconv"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	fakekube "k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// EnableLeaseResourceVersions makes Lease creates and updates on a fake client
// enforce the resourceVersion check used by competing leader electors.
func EnableLeaseResourceVersions(kc *fakekube.Clientset) {
	var version int
	stamp := func(lease *coordinationv1.Lease) {
		version++
		lease.ResourceVersion = strconv.Itoa(version)
	}
	kc.PrependReactor("create", "leases", func(action ktesting.Action) (bool, runtime.Object, error) {
		stamp(action.(ktesting.CreateAction).GetObject().(*coordinationv1.Lease))
		return false, nil, nil
	})
	kc.PrependReactor("update", "leases", func(action ktesting.Action) (bool, runtime.Object, error) {
		lease := action.(ktesting.UpdateAction).GetObject().(*coordinationv1.Lease)
		current, err := kc.Tracker().Get(action.GetResource(), action.GetNamespace(), lease.Name)
		if err != nil {
			return true, nil, err
		}
		if lease.ResourceVersion != current.(*coordinationv1.Lease).ResourceVersion {
			return true, nil, apierrors.NewConflict(action.GetResource().GroupResource(), lease.Name, errors.New("stale resourceVersion"))
		}
		stamp(lease)
		return false, nil, nil
	})
}
