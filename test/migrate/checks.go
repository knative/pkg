package migrate

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

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckStoredVersions verifies that all status.storedVersions from Knative CRDs are listed in the spec
// with storage: true. It means the CRDs have been migrated and previous/unused API versions
// can be safely removed from the spec.
func CheckStoredVersions(ctx context.Context, apiextensions *apiextensionsv1.ApiextensionsV1Client) error {
	crdClient := apiextensions.CustomResourceDefinitions()

	crdList, err := crdClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to fetch crd list %w", err)
	}

	var (
		failed bool
		errMsg strings.Builder
	)
	for _, crd := range crdList.Items {
		if strings.Contains(crd.Name, "knative.dev") {
			for _, stored := range crd.Status.StoredVersions {
				for _, v := range crd.Spec.Versions {
					if stored == v.Name && !v.Storage {
						failed = true
						fmt.Fprintf(&errMsg, "\"%s\" is invalid: spec.versions.storage must be true for \"%s\" or "+
							"version %s must be removed from status.storageVersions\n", crd.Name, v.Name, v.Name)
					}
				}
			}
		}
	}

	if failed {
		return errors.New(errMsg.String())
	}

	return nil
}
