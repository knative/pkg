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

package ha

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"

	"knative.dev/pkg/test"
	"knative.dev/pkg/test/logging"
)

func countingRFind(wr rune, wc int) func(rune) bool {
	cnt := 0
	return func(r rune) bool {
		if r == wr {
			cnt++
		}
		return cnt == wc
	}
}

func extractDeployment(pod string) string {
	if x := strings.LastIndexFunc(pod, countingRFind('-', 2)); x != -1 {
		return pod[:x]
	}
	return ""
}

// GetLeaders collects all of the leader pods from the specified deployment.
func GetLeaders(t *testing.T, client *test.KubeClient, deploymentName, namespace string) (sets.String, error) {
	leases, err := client.Kube.CoordinationV1().Leases(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting leases for deployment %q: %w", deploymentName, err)
	}
	ret := sets.NewString()
	for _, lease := range leases.Items {
		if lease.Spec.HolderIdentity == nil {
			continue
		}
		pod := strings.SplitN(*lease.Spec.HolderIdentity, "_", 2)[0]

		// Deconstruct the pod name and look for the deployment.  This won't work for very long deployment names.
		if extractDeployment(pod) != deploymentName {
			continue
		}
		ret.Insert(pod)
	}
	return ret, nil
}

// WaitForNewLeaders waits until the collection of current leaders consists of "n" leaders
// which do not include the specified prior leaders.
func WaitForNewLeaders(t *testing.T, client *test.KubeClient, deploymentName, namespace string, previousLeaders sets.String, n int) (sets.String, error) {
	span := logging.GetEmitableSpan(context.Background(), "WaitForNewLeaders/"+deploymentName)
	defer span.End()

	var (
		leaders sets.String
		ierr    error
	)
	err := wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		leaders, ierr = GetLeaders(t, client, deploymentName, namespace)
		if ierr != nil {
			return false, ierr
		}
		if len(leaders) < n {
			t.Logf("WaitForNewLeaders[%s] not enough leaders, got: %d, want: %d", deploymentName, len(leaders), n)
			return false, nil
		}
		if isect := previousLeaders.Intersection(leaders); len(isect) > 0 {
			t.Logf("WaitForNewLeaders[%s] still see intersection: %v", deploymentName, isect)
			return false, nil
		}
		return true, nil
	})
	return leaders, err
}

// DEPRECATED WaitForNewLeader waits until the holder of the given lease is different from the previousLeader.
func WaitForNewLeader(client *test.KubeClient, lease, namespace, previousLeader string) (string, error) {
	span := logging.GetEmitableSpan(context.Background(), "WaitForNewLeader/"+lease)
	defer span.End()
	var leader string
	err := wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
		lease, err := client.Kube.CoordinationV1().Leases(namespace).Get(lease, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("error getting lease %s: %w", lease, err)
		}
		leader = strings.Split(*lease.Spec.HolderIdentity, "_")[0]
		return leader != previousLeader, nil
	})
	return leader, err
}
