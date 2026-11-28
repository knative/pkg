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

package controller

import (
	"context"
	"slices"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"knative.dev/pkg/system"
)

// assignPreferredHolders runs one placement pass. The prototype assumes that
// each Ready pod of the named Deployment participates in the same queue and
// uses its Pod UID as its elector identity.
func assignPreferredHolders(ctx context.Context, kc kubernetes.Interface, leaseName string) error {
	ns := system.Namespace()
	leases := kc.CoordinationV1().Leases(ns)
	seed, err := leases.Get(ctx, leaseName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	deployment, err := kc.AppsV1().Deployments(ns).Get(ctx, seed.Labels[deploymentLabel], metav1.GetOptions{})
	if err != nil {
		return err
	}
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return err
	}
	pods, err := kc.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return err
	}
	var candidates []string
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				candidates = append(candidates, string(pod.UID))
				break
			}
		}
	}
	slices.Sort(candidates)
	candidates = candidates[:min(len(candidates), int(ptr.Deref(deployment.Spec.Replicas, 1)))]
	if len(candidates) == 0 {
		return nil
	}
	buckets, err := leases.List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{deploymentLabel: deployment.Name}.String(),
	})
	if err != nil {
		return err
	}
	sort.Slice(buckets.Items, func(i, j int) bool { return buckets.Items[i].Name < buckets.Items[j].Name })

	// Round-robin keeps the fixture evenly distributed. Changes in membership
	// can move additional buckets; placement policy is independent of handoff.
	for i, lease := range buckets.Items {
		target := candidates[i%len(candidates)]
		if target == lease.Labels[preferredHolderLabel] {
			continue
		}
		// A renewal may race this update. Keep the latest holder and renew time.
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			current, err := leases.Get(ctx, lease.Name, metav1.GetOptions{})
			if err != nil || current.Labels[preferredHolderLabel] == target {
				return err
			}
			current.Labels[preferredHolderLabel] = target
			_, err = leases.Update(ctx, current, metav1.UpdateOptions{})
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}
