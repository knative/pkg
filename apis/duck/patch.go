/*
Copyright 2018 The Knative Authors

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

package duck

import (
	"encoding/json"
	"sort"

	jsonmergepatch "github.com/evanphx/json-patch"
	"github.com/mattbaird/jsonpatch"
)

func marshallBeforeAfter(before, after interface{}) ([]byte, []byte, error) {
	rawBefore, err := json.Marshal(before)
	if err != nil {
		return nil, nil, err
	}

	rawAfter, err := json.Marshal(after)
	if err != nil {
		return rawBefore, nil, err
	}

	return rawBefore, rawAfter, nil
}

func CreateMergePatch(before, after interface{}) ([]byte, error) {
	rawBefore, rawAfter, err := marshallBeforeAfter(before, after)
	if err != nil {
		return nil, err
	}
	return jsonmergepatch.CreateMergePatch(rawBefore, rawAfter)
}

func CreatePatch(before, after interface{}) (JSONPatch, error) {
	rawBefore, rawAfter, err := marshallBeforeAfter(before, after)
	if err != nil {
		return nil, err
	}
	patch, err := jsonpatch.CreatePatch(rawBefore, rawAfter)
	if err != nil {
		return nil, err
	}

	// Give the patch a deterministic ordering.
	sort.Slice(patch, func(i, j int) bool {
		lhs, rhs := patch[i], patch[j]
		if lhs.Operation != rhs.Operation {
			return lhs.Operation < rhs.Operation
		}
		return lhs.Path < rhs.Path
	})
	return patch, nil
}

type JSONPatch []jsonpatch.JsonPatchOperation

func (p JSONPatch) MarshalJSON() ([]byte, error) {
	return json.Marshal([]jsonpatch.JsonPatchOperation(p))
}
