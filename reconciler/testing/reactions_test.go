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

package testing

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"

	"testing"
)

var imc = schema.GroupVersionResource{
	Group:    "messaging.knative.dev",
	Version:  "v1",
	Resource: "inmemorychannels",
}

var revision = schema.GroupVersionResource{
	Group:    "serving.knative.dev",
	Version:  "v1",
	Resource: "revisions",
}

func TestInduceFailure(t *testing.T) {
	tests := []struct {
		name string
		// These set up the InduceFailure function
		verb     string
		resource string
		// This is the resource fed to the function
		testSource schema.GroupVersionResource
		// This is the subresource fed to the function
		testSubresource string
		wantHandled     bool
	}{{
		name:        "resource",
		verb:        "patch",
		testSource:  imc,
		resource:    "inmemorychannels",
		wantHandled: true,
	}, {
		name:        "resource, wrong verb",
		verb:        "create",
		testSource:  imc,
		resource:    "inmemorychannels",
		wantHandled: false,
	}, {
		name:        "wrong resource",
		verb:        "patch",
		testSource:  revision,
		resource:    "inmemorychannels",
		wantHandled: false,
	}, {
		name:            "resource and subresource",
		verb:            "patch",
		testSource:      imc,
		resource:        "inmemorychannels/status",
		testSubresource: "status",
		wantHandled:     true,
	}, {
		name:            "resource and subresource, wrong verb",
		verb:            "get",
		testSource:      imc,
		resource:        "inmemorychannels/status",
		testSubresource: "status",
		wantHandled:     false,
	}, {
		name:            "resource and subresource, subresource does not match",
		verb:            "patch",
		testSource:      imc,
		resource:        "inmemorychannels/status",
		testSubresource: "finalizers",
		wantHandled:     false,
	}}
	for _, tc := range tests {
		var f clientgotesting.ReactionFunc

		f = InduceFailure(tc.verb, tc.resource)
		var patchAction clientgotesting.PatchActionImpl
		if tc.testSubresource != "" {
			patchAction = clientgotesting.NewPatchSubresourceAction(tc.testSource, "testns", "test", types.JSONPatchType, []byte{}, tc.testSubresource)
		} else {
			patchAction = clientgotesting.NewPatchAction(tc.testSource, "testns", "test", types.JSONPatchType, []byte{})
		}
		if handled, _, _ := f(patchAction); handled != tc.wantHandled {
			t.Errorf("%q failed wanted %v got %v", tc.name, tc.wantHandled, handled)
		}
	}
}
