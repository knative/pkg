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

package targeter

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
	"knative.dev/pkg/test/helpers"
)

func TestSwitchClientConfigLocal(t *testing.T) {
	name := helpers.ObjectNameForTest(t)
	bundle := []byte("pem")

	got := SwitchClientConfig(&admissionregistrationv1.WebhookClientConfig{
		Service: &admissionregistrationv1.ServiceReference{
			Namespace: system.Namespace(),
			Name:      name,
			Path:      ptr.String("/local/path/here"),
		},
		CABundle: bundle,
	})

	want := &apixv1.WebhookClientConfig{
		Service: &apixv1.ServiceReference{
			Namespace: system.Namespace(),
			Name:      name,
			Path:      ptr.String("/local/path/here"),
		},
		CABundle: bundle,
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("SwitchClientConfig(-got, +want) = %s", diff)
	}
}

func TestSwitchClientConfigURL(t *testing.T) {
	got := SwitchClientConfig(&admissionregistrationv1.WebhookClientConfig{
		URL: ptr.String("https://google.com"),
	})

	want := &apixv1.WebhookClientConfig{
		URL: ptr.String("https://google.com"),
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("SwitchClientConfig(-got, +want) = %s", diff)
	}
}
