/*
Copyright 2022 The Knative Authors

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

package filtering

import (
	"context"
	"fmt"
	"log"
	"os"

	"k8s.io/apimachinery/pkg/labels"
	filteredFactory "knative.dev/pkg/client/injection/kube/informers/factory/filtered"
)

const (
	// Label Key/Value pair to be used by Knative controllers implementing filtering
	KnativeUsedbyKey   = "knative.dev/used-by"
	KnativeUsedByValue = "true"
	// Env var to pass the label selectors via a comma separated list eg. k1=v1,k2=v2
	InformerLabelSelectorsFilterEnv = "INFORMER_LABEL_SELECTORS_FILTER"
)

// InformersFilterByLabeladds checks if user has set label selectors
// to be used by filtering informers. Then passes the selectors to the current context so
// that filtering informer factories can pick up the labels.
func InformersFilterByLabel(ctx context.Context) context.Context {
	if labelsStr := os.Getenv(InformerLabelSelectorsFilterEnv); labelsStr != "" {
		// Validate labels
		labelsSet, err := labels.ConvertSelectorToLabelsMap(labelsStr)
		if err != nil {
			log.Fatal("Error converting label selector string: ", err)
			return ctx
		}
		labelList := make([]string, 0, len(labelsSet))
		for k, v := range labelsSet {
			labelList = append(labelList, fmt.Sprintf("%s=%s", k, v))
		}
		ctx = filteredFactory.WithSelectors(ctx, labelList...)
	} else {
		// Empty selector meant for initializing the factory in paths where filtered factory is imported
		// eg. sharedmain but no labels are set via the env var. This matches everything.
		ctx = filteredFactory.WithSelectors(ctx, "")
	}
	return ctx
}

// AddKnativeUsedByLabels adds a predefined label selector to a labels map.
// It is meant to be used with resource labels maps eg. secret labels.
func AddKnativeUsedByLabels(objectLabels map[string]string) map[string]string {
	if objectLabels == nil {
		objectLabels = map[string]string{}
	}
	if labelsStr := os.Getenv(InformerLabelSelectorsFilterEnv); labelsStr != "" {
		labelsSet, err := labels.ConvertSelectorToLabelsMap(labelsStr)
		if err != nil {
			log.Fatal("Error converting label selector string: ", err)
		}
		if val, ok := labelsSet[KnativeUsedbyKey]; ok && val == KnativeUsedByValue {
			objectLabels[KnativeUsedbyKey] = KnativeUsedByValue
		}
	}
	return objectLabels
}
