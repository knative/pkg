/*
Copyright 2019 The Knative Authors

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

package defaulting

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"knative.dev/pkg/apis"
)

// setUserInfoAnnotations sets creator and updater annotations on a resource.
func setUserInfoAnnotations(ctx context.Context, resource apis.HasSpec, groupName string) {
	if ui := apis.GetUserInfo(ctx); ui != nil {
		objectMetaAccessor, ok := resource.(metav1.ObjectMetaAccessor)
		if !ok {
			return
		}

		annotations := objectMetaAccessor.GetObjectMeta().GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
			objectMetaAccessor.GetObjectMeta().SetAnnotations(annotations)
		}

		if apis.IsInUpdate(ctx) {
			old := apis.GetBaseline(ctx).(apis.HasSpec)
			if equality.Semantic.DeepEqual(old.GetUntypedSpec(), resource.GetUntypedSpec()) {
				return
			}
			annotations[groupName+apis.UpdaterAnnotationSuffix] = ui.Username
		} else {
			annotations[groupName+apis.CreatorAnnotationSuffix] = ui.Username
			annotations[groupName+apis.UpdaterAnnotationSuffix] = ui.Username
		}
	}
}

// setUserInfoAnnotationsUnstructured sets creator and updater annotations on a resource.
func setUserInfoAnnotationsUnstructured(ctx context.Context, after *unstructured.Unstructured, before *unstructured.Unstructured, req *admissionv1.AdmissionRequest) {
	if v, ok := after.Object["metadata"]; ok {
		if metadata, ok := v.(map[string]interface{}); ok {
			if v, ok := metadata["annotations"]; ok {
				if annotations, ok := v.(map[string]interface{}); ok {
					if apis.IsInUpdate(ctx) {

						if equality.Semantic.DeepEqual(before.UnstructuredContent(), after.UnstructuredContent()) {
							return
						}

						annotations[req.Resource.Group+apis.UpdaterAnnotationSuffix] = req.UserInfo.Username
						return
					}

					annotations[req.Resource.Group+apis.CreatorAnnotationSuffix] = req.UserInfo.Username
					annotations[req.Resource.Group+apis.UpdaterAnnotationSuffix] = req.UserInfo.Username
				}
			}

		}
	}
}
