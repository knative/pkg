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
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

func TestCronJobValidation(t *testing.T) {
	tests := []struct {
		name string
		with func(context.Context) context.Context
		want *apis.FieldError
	}{{
		name: "no check",
		with: func(ctx context.Context) context.Context {
			return ctx
		},
		want: nil,
	}, {
		name: "no error",
		with: func(ctx context.Context) context.Context {
			return WithCronJobValidator(ctx, func(ctx context.Context, c *CronJob) *apis.FieldError {
				return nil
			})
		},
		want: nil,
	}, {
		name: "no busybox",
		with: func(ctx context.Context) context.Context {
			return WithCronJobValidator(ctx, func(ctx context.Context, c *CronJob) *apis.FieldError {
				for i, con := range c.Spec.JobTemplate.Spec.Template.Spec.InitContainers {
					if con.Image == "busybox" {
						return apis.ErrInvalidValue(con.Image, "image").ViaFieldIndex("spec.template.spec.initContainers", i)
					}
				}
				for i, con := range c.Spec.JobTemplate.Spec.Template.Spec.Containers {
					if con.Image == "busybox" {
						return apis.ErrInvalidValue(con.Image, "image").ViaFieldIndex("spec.template.spec.containers", i)
					}
				}
				return nil
			})
		},
		want: apis.ErrInvalidValue("busybox", "spec.template.spec.containers[0].image"),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
			ctx := test.with(context.Background())
			got := c.Validate(ctx)
			if test.want.Error() != got.Error() {
				t.Errorf("Validate() = %v, wanted %v", got, test.want)
			}
		})
	}
}
