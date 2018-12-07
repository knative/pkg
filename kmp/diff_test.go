/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kmp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestCompareKcmpDefault(t *testing.T) {
	a := resource.MustParse("50m")
	b := resource.MustParse("100m")

	want := "{resource.Quantity}:\n\t-: resource.Quantity{i: resource.int64Amount{value: 50, scale: resource.Scale(-3)}, s: \"50m\", Format: resource.Format(\"DecimalSI\")}\n\t+: resource.Quantity{i: resource.int64Amount{value: 100, scale: resource.Scale(-3)}, s: \"100m\", Format: resource.Format(\"DecimalSI\")}\n"

	if got, err := SafeDiff(a, b); err != nil {
		t.Fatalf("unexpected SafeDiff err: %v", err)
	} else if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SafeDiff (-want, +got): %v", diff)
	}
}

func TestRecovery(t *testing.T) {
	type foo struct {
		bar string
	}

	a := foo{"a"}
	b := foo{"b"}

	want := "recovered in kmp.SafeDiff: cannot handle unexported field: {kmp.foo}.bar\nconsider using AllowUnexported or cmpopts.IgnoreUnexported"

	if _, err := SafeDiff(a, b); err == nil {
		t.Fatalf("expected err, got nil")
	} else if diff := cmp.Diff(want, err.Error()); diff != "" {
		t.Errorf("SafeDiff (-want, +got): %v", diff)
	}
}
