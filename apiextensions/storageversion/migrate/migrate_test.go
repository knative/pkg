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
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestMainFlagSet(t *testing.T) {
	t.Setenv("KUBECONFIG", "/tmp/example-kubeconfig")
	t.Setenv("KUBE_API_QPS", "12.5")
	t.Setenv("KUBE_API_BURST", "25")

	globalFlags := func() []string {
		var names []string
		flag.CommandLine.VisitAll(func(f *flag.Flag) {
			names = append(names, f.Name)
		})
		return names
	}
	before := globalFlags()

	// Independent callers can register their own flags without using global state.
	for i := 0; i < 2; i++ {
		flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
		var output bytes.Buffer
		flags.SetOutput(&output)
		custom := flags.Bool("custom", false, "caller-provided flag")

		Main(flags, []string{"-custom", "-help"})

		if !*custom {
			t.Error("Main did not parse the caller-provided arguments")
		}
		for name, want := range map[string]string{
			"kubeconfig":     "/tmp/example-kubeconfig",
			"kube-api-qps":   "12.5",
			"kube-api-burst": "25",
		} {
			f := flags.Lookup(name)
			if f == nil || f.Value.String() != want {
				t.Errorf("flag %q = %v, want %q", name, f, want)
			}
		}
		if !strings.Contains(output.String(), "Usage of migrate:") {
			t.Errorf("Main did not write help to the supplied flag set: %s", output.String())
		}
	}

	if diff := cmp.Diff(before, globalFlags()); diff != "" {
		t.Errorf("Main changed the global flag set (-before +after):\n%s", diff)
	}
}

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
