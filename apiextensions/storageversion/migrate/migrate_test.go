/*
Copyright 2026 The Knative Authors

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

package migrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParseResources(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []schema.GroupResource
		wantErr string
	}{
		{
			name: "no resources",
			want: []schema.GroupResource{},
		},
		{
			name: "group resources in argument order",
			args: []string{"services.serving.knative.dev", "brokers.eventing.knative.dev"},
			want: []schema.GroupResource{
				{Resource: "services", Group: "serving.knative.dev"},
				{Resource: "brokers", Group: "eventing.knative.dev"},
			},
		},
		{
			name: "core group resource",
			args: []string{"pods"},
			want: []schema.GroupResource{{Resource: "pods"}},
		},
		{
			name:    "empty argument",
			args:    []string{""},
			wantErr: "unable to parse group version: ",
		},
		{
			name:    "empty group and resource",
			args:    []string{"."},
			wantErr: "unable to parse group version: .",
		},
		{
			name:    "invalid argument after valid resource",
			args:    []string{"services.serving.knative.dev", ""},
			wantErr: "unable to parse group version: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseResources(tt.args)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseResources(%q) error = %v, want %q", tt.args, err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("parseResources(%q) error = %v", tt.args, err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("parseResources(%q) mismatch (-want +got):\n%s", tt.args, diff)
			}
		})
	}
}
