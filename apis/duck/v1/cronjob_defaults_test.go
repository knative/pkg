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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestCronJobDefaulting(t *testing.T) {
	c := CronJob{
		Spec: batchv1.CronJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "blah",
								Image: "busybox",
							}},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name string
		with func(context.Context) context.Context
		want *CronJob
	}{{
		name: "no check",
		with: func(ctx context.Context) context.Context {
			return ctx
		},
		want: c.DeepCopy(),
	}, {
		name: "no change",
		with: func(ctx context.Context) context.Context {
			return WithCronJobDefaulter(ctx, func(ctx context.Context, c *CronJob) {
			})
		},
		want: c.DeepCopy(),
	}, {
		name: "no busybox",
		with: func(ctx context.Context) context.Context {
			return WithCronJobDefaulter(ctx, func(ctx context.Context, c *CronJob) {
				for i, con := range c.Spec.JobTemplate.Spec.Template.Spec.InitContainers {
					if !strings.Contains(con.Image, "@") {
						c.Spec.JobTemplate.Spec.Template.Spec.InitContainers[i].Image = con.Image + "@sha256:deadbeef"
					}
				}
				for i, con := range c.Spec.JobTemplate.Spec.Template.Spec.Containers {
					if !strings.Contains(con.Image, "@") {
						c.Spec.JobTemplate.Spec.Template.Spec.Containers[i].Image = con.Image + "@sha256:deadbeef"
					}
				}
			})
		},
		want: &CronJob{
			Spec: batchv1.CronJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:  "blah",
									Image: "busybox@sha256:deadbeef",
								}},
							},
						},
					},
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := test.with(context.Background())
			got := c.DeepCopy()
			got.SetDefaults(ctx)
			if !cmp.Equal(test.want, got) {
				t.Errorf("SetDefaults (-want, +got) = %s", cmp.Diff(test.want, got))
			}
		})
	}
}
