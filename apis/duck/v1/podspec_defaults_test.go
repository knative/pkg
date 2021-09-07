/*
Copyright 2021 The Knative Authors

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
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
)

func TestPodSpecDefaulting(t *testing.T) {
	p := WithPod{
		Spec: WithPodSpec{
			Template: PodSpecable{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "blah",
						Image: "busybox",
					}},
				},
			},
		},
	}

	tests := []struct {
		name string
		with func(context.Context) context.Context
		want *WithPod
	}{{
		name: "no check",
		with: func(ctx context.Context) context.Context {
			return ctx
		},
		want: p.DeepCopy(),
	}, {
		name: "no change",
		with: func(ctx context.Context) context.Context {
			return WithPodSpecDefaulter(ctx, func(ctx context.Context, wp *WithPod) {
			})
		},
		want: p.DeepCopy(),
	}, {
		name: "no busybox",
		with: func(ctx context.Context) context.Context {
			return WithPodSpecDefaulter(ctx, func(ctx context.Context, wp *WithPod) {
				for i, c := range wp.Spec.Template.Spec.InitContainers {
					if !strings.Contains(c.Image, "@") {
						wp.Spec.Template.Spec.InitContainers[i].Image = c.Image + "@sha256:deadbeef"
					}
				}
				for i, c := range wp.Spec.Template.Spec.Containers {
					if !strings.Contains(c.Image, "@") {
						wp.Spec.Template.Spec.Containers[i].Image = c.Image + "@sha256:deadbeef"
					}
				}
			})
		},
		want: &WithPod{
			Spec: WithPodSpec{
				Template: PodSpecable{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "blah",
							Image: "busybox@sha256:deadbeef",
						}},
					},
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := test.with(context.Background())
			got := p.DeepCopy()
			got.SetDefaults(ctx)
			if !cmp.Equal(test.want, got) {
				t.Errorf("SetDefaults (-want, +got) = %s", cmp.Diff(test.want, got))
			}
		})
	}
}
