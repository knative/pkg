/*
Copyright 2017 The Kubernetes Authors.

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

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"text/template"
	"time"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	prowjobv1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/pod-utils/decorate"
	"k8s.io/test-infra/prow/pod-utils/downwardapi"
)

func TestDefaultJobBase(t *testing.T) {
	bar := "bar"
	filled := JobBase{
		Agent:     "foo",
		Namespace: &bar,
		Cluster:   "build",
	}
	cases := []struct {
		name     string
		config   ProwConfig
		base     func(j *JobBase)
		expected func(j *JobBase)
	}{
		{
			name: "no changes when fields are already set",
		},
		{
			name: "empty agent results in kubernetes",
			base: func(j *JobBase) {
				j.Agent = ""
			},
			expected: func(j *JobBase) {
				j.Agent = string(prowapi.KubernetesAgent)
			},
		},
		{
			name: "nil namespace becomes PodNamespace",
			config: ProwConfig{
				PodNamespace:     "pod-namespace",
				ProwJobNamespace: "wrong",
			},
			base: func(j *JobBase) {
				j.Namespace = nil
			},
			expected: func(j *JobBase) {
				p := "pod-namespace"
				j.Namespace = &p
			},
		},
		{
			name: "empty namespace becomes PodNamespace",
			config: ProwConfig{
				PodNamespace:     "new-pod-namespace",
				ProwJobNamespace: "still-wrong",
			},
			base: func(j *JobBase) {
				var empty string
				j.Namespace = &empty
			},
			expected: func(j *JobBase) {
				p := "new-pod-namespace"
				j.Namespace = &p
			},
		},
		{
			name: "empty cluster becomes DefaultClusterAlias",
			base: func(j *JobBase) {
				j.Cluster = ""
			},
			expected: func(j *JobBase) {
				j.Cluster = kube.DefaultClusterAlias
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := filled
			if tc.base != nil {
				tc.base(&actual)
			}
			expected := actual
			if tc.expected != nil {
				tc.expected(&expected)
			}
			tc.config.defaultJobBase(&actual)
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("expected %#v\n!=\nactual %#v", expected, actual)
			}
		})
	}
}

func TestSpyglassConfig(t *testing.T) {
	testCases := []struct {
		name                 string
		spyglassConfig       string
		expectedViewers      map[string][]string
		expectedRegexMatches map[string][]string
		expectedSizeLimit    int64
		expectError          bool
	}{
		{
			name: "Default: build log, metadata, junit",
			spyglassConfig: `
deck:
  spyglass:
    size_limit: 500e+6
    viewers:
      "started.json|finished.json":
      - "metadata"
      "build-log.txt":
      - "buildlog"
      "artifacts/junit.*\\.xml":
      - "junit"
`,
			expectedViewers: map[string][]string{
				"started.json|finished.json": {"metadata"},
				"build-log.txt":              {"buildlog"},
				"artifacts/junit.*\\.xml":    {"junit"},
			},
			expectedRegexMatches: map[string][]string{
				"started.json|finished.json": {"started.json", "finished.json"},
				"build-log.txt":              {"build-log.txt"},
				"artifacts/junit.*\\.xml":    {"artifacts/junit01.xml", "artifacts/junit_runner.xml"},
			},
			expectedSizeLimit: 500e6,
			expectError:       false,
		},
		{
			name: "Backwards compatibility",
			spyglassConfig: `
deck:
  spyglass:
    size_limit: 500e+6
    viewers:
      "started.json|finished.json":
      - "metadata-viewer"
      "build-log.txt":
      - "build-log-viewer"
      "artifacts/junit.*\\.xml":
      - "junit-viewer"
`,
			expectedViewers: map[string][]string{
				"started.json|finished.json": {"metadata"},
				"build-log.txt":              {"buildlog"},
				"artifacts/junit.*\\.xml":    {"junit"},
			},
			expectedSizeLimit: 500e6,
			expectError:       false,
		},
		{
			name: "Invalid spyglass size limit",
			spyglassConfig: `
deck:
  spyglass:
    size_limit: -4
    viewers:
      "started.json|finished.json":
      - "metadata-viewer"
      "build-log.txt":
      - "build-log-viewer"
      "artifacts/junit.*\\.xml":
      - "junit-viewer"
`,
			expectError: true,
		},
		{
			name: "Invalid Spyglass regexp",
			spyglassConfig: `
deck:
  spyglass:
    size_limit: 5
    viewers:
      "started.json\|]finished.json":
      - "metadata-viewer"
`,
			expectError: true,
		},
	}
	for _, tc := range testCases {
		// save the config
		spyglassConfigDir, err := ioutil.TempDir("", "spyglassConfig")
		if err != nil {
			t.Fatalf("fail to make tempdir: %v", err)
		}
		defer os.RemoveAll(spyglassConfigDir)

		spyglassConfig := filepath.Join(spyglassConfigDir, "config.yaml")
		if err := ioutil.WriteFile(spyglassConfig, []byte(tc.spyglassConfig), 0666); err != nil {
			t.Fatalf("fail to write spyglass config: %v", err)
		}

		cfg, err := Load(spyglassConfig, "")
		if (err != nil) != tc.expectError {
			t.Fatalf("tc %s: expected error: %v, got: %v, error: %v", tc.name, tc.expectError, (err != nil), err)
		}

		if err != nil {
			continue
		}
		got := cfg.Deck.Spyglass.Viewers
		for re, viewNames := range got {
			expected, ok := tc.expectedViewers[re]
			if !ok {
				t.Errorf("With re %s, got %s, was not found in expected.", re, viewNames)
				continue
			}
			if !reflect.DeepEqual(expected, viewNames) {
				t.Errorf("With re %s, got %s, expected view name %s", re, viewNames, expected)
			}

		}
		for re, viewNames := range tc.expectedViewers {
			gotNames, ok := got[re]
			if !ok {
				t.Errorf("With re %s, expected %s, was not found in got.", re, viewNames)
				continue
			}
			if !reflect.DeepEqual(gotNames, viewNames) {
				t.Errorf("With re %s, got %s, expected view name %s", re, gotNames, viewNames)
			}
		}

		for expectedRegex, matches := range tc.expectedRegexMatches {
			compiledRegex, ok := cfg.Deck.Spyglass.RegexCache[expectedRegex]
			if !ok {
				t.Errorf("tc %s, regex %s was not found in the spyglass regex cache", tc.name, expectedRegex)
				continue
			}
			for _, match := range matches {
				if !compiledRegex.MatchString(match) {
					t.Errorf("tc %s expected compiled regex %s to match %s, did not match.", tc.name, expectedRegex, match)
				}
			}

		}
		if cfg.Deck.Spyglass.SizeLimit != tc.expectedSizeLimit {
			t.Errorf("%s expected SizeLimit %d, got %d", tc.name, tc.expectedSizeLimit, cfg.Deck.Spyglass.SizeLimit)
		}
	}

}

func TestDecorationRawYaml(t *testing.T) {
	var testCases = []struct {
		name        string
		expectError bool
		rawConfig   string
		expected    *prowapi.DecorationConfig
	}{
		{
			name:        "no default",
			expectError: true,
			rawConfig: `
periodics:
- name: kubernetes-defaulted-decoration
  interval: 1h
  decorate: true
  spec:
    containers:
    - image: golang:latest
      args:
      - "test"
      - "./..."`,
		},
		{
			name: "with default, no explicit decorate",
			rawConfig: `
plank:
  default_decoration_config:
    timeout: 2h
    grace_period: 15s
    utility_images:
      clonerefs: "clonerefs:default"
      initupload: "initupload:default"
      entrypoint: "entrypoint:default"
      sidecar: "sidecar:default"
    gcs_configuration:
      bucket: "default-bucket"
      path_strategy: "legacy"
      default_org: "kubernetes"
      default_repo: "kubernetes"
    gcs_credentials_secret: "default-service-account"

periodics:
- name: kubernetes-defaulted-decoration
  interval: 1h
  decorate: true
  spec:
    containers:
    - image: golang:latest
      args:
      - "test"
      - "./..."`,
			expected: &prowapi.DecorationConfig{
				Timeout:     &prowapi.Duration{Duration: 2 * time.Hour},
				GracePeriod: &prowapi.Duration{Duration: 15 * time.Second},
				UtilityImages: &prowapi.UtilityImages{
					CloneRefs:  "clonerefs:default",
					InitUpload: "initupload:default",
					Entrypoint: "entrypoint:default",
					Sidecar:    "sidecar:default",
				},
				GCSConfiguration: &prowapi.GCSConfiguration{
					Bucket:       "default-bucket",
					PathStrategy: prowapi.PathStrategyLegacy,
					DefaultOrg:   "kubernetes",
					DefaultRepo:  "kubernetes",
				},
				GCSCredentialsSecret: "default-service-account",
			},
		},
		{
			name: "with default, has explicit decorate",
			rawConfig: `
plank:
  default_decoration_config:
    timeout: 2h
    grace_period: 15s
    utility_images:
      clonerefs: "clonerefs:default"
      initupload: "initupload:default"
      entrypoint: "entrypoint:default"
      sidecar: "sidecar:default"
    gcs_configuration:
      bucket: "default-bucket"
      path_strategy: "legacy"
      default_org: "kubernetes"
      default_repo: "kubernetes"
    gcs_credentials_secret: "default-service-account"

periodics:
- name: kubernetes-defaulted-decoration
  interval: 1h
  decorate: true
  decoration_config:
    timeout: 1
    grace_period: 1
    utility_images:
      clonerefs: "clonerefs:explicit"
      initupload: "initupload:explicit"
      entrypoint: "entrypoint:explicit"
      sidecar: "sidecar:explicit"
    gcs_configuration:
      bucket: "explicit-bucket"
      path_strategy: "explicit"
    gcs_credentials_secret: "explicit-service-account"
  spec:
    containers:
    - image: golang:latest
      args:
      - "test"
      - "./..."`,
			expected: &prowapi.DecorationConfig{
				Timeout:     &prowapi.Duration{Duration: 1 * time.Nanosecond},
				GracePeriod: &prowapi.Duration{Duration: 1 * time.Nanosecond},
				UtilityImages: &prowapi.UtilityImages{
					CloneRefs:  "clonerefs:explicit",
					InitUpload: "initupload:explicit",
					Entrypoint: "entrypoint:explicit",
					Sidecar:    "sidecar:explicit",
				},
				GCSConfiguration: &prowapi.GCSConfiguration{
					Bucket:       "explicit-bucket",
					PathStrategy: prowapi.PathStrategyExplicit,
					DefaultOrg:   "kubernetes",
					DefaultRepo:  "kubernetes",
				},
				GCSCredentialsSecret: "explicit-service-account",
			},
		},
	}

	for _, tc := range testCases {
		// save the config
		prowConfigDir, err := ioutil.TempDir("", "prowConfig")
		if err != nil {
			t.Fatalf("fail to make tempdir: %v", err)
		}
		defer os.RemoveAll(prowConfigDir)

		prowConfig := filepath.Join(prowConfigDir, "config.yaml")
		if err := ioutil.WriteFile(prowConfig, []byte(tc.rawConfig), 0666); err != nil {
			t.Fatalf("fail to write prow config: %v", err)
		}

		cfg, err := Load(prowConfig, "")
		if tc.expectError && err == nil {
			t.Errorf("tc %s: Expect error, but got nil", tc.name)
		} else if !tc.expectError && err != nil {
			t.Errorf("tc %s: Expect no error, but got error %v", tc.name, err)
		}

		if tc.expected != nil {
			if len(cfg.Periodics) != 1 {
				t.Fatalf("tc %s: Expect to have one periodic job, got none", tc.name)
			}

			if !reflect.DeepEqual(cfg.Periodics[0].DecorationConfig, tc.expected) {
				t.Errorf("%s: expected defaulted config:\n%#v\n but got:\n%#v\n", tc.name, tc.expected, cfg.Periodics[0].DecorationConfig)
			}
		}
	}

}

func TestValidateAgent(t *testing.T) {
	b := string(prowjobv1.KnativeBuildAgent)
	jenk := string(prowjobv1.JenkinsAgent)
	k := string(prowjobv1.KubernetesAgent)
	ns := "default"
	base := JobBase{
		Agent:     k,
		Namespace: &ns,
		Spec:      &v1.PodSpec{},
		UtilityConfig: UtilityConfig{
			DecorationConfig: &prowapi.DecorationConfig{},
		},
	}

	cases := []struct {
		name string
		base func(j *JobBase)
		pass bool
	}{
		{
			name: "accept unknown agent",
			base: func(j *JobBase) {
				j.Agent = "random-agent"
			},
			pass: true,
		},
		{
			name: "spec requires kubernetes agent",
			base: func(j *JobBase) {
				j.Agent = b
			},
		},
		{
			name: "kubernetes agent requires spec",
			base: func(j *JobBase) {
				j.Spec = nil
			},
		},
		{
			name: "build_spec requires knative-build agent",
			base: func(j *JobBase) {
				j.DecorationConfig = nil
				j.Spec = nil

				j.BuildSpec = &buildv1alpha1.BuildSpec{}
			},
		},
		{
			name: "knative-build agent requires build_spec",
			base: func(j *JobBase) {
				j.DecorationConfig = nil
				j.Spec = nil

				j.Agent = b
			},
		},
		{
			name: "decoration requires kubernetes agent",
			base: func(j *JobBase) {
				j.Agent = b
				j.BuildSpec = &buildv1alpha1.BuildSpec{}
			},
		},
		{
			name: "non-nil namespace required",
			base: func(j *JobBase) {
				j.Namespace = nil
			},
		},
		{
			name: "filled namespace required",
			base: func(j *JobBase) {
				var s string
				j.Namespace = &s
			},
		},
		{
			name: "custom namespace requires knative-build agent",
			base: func(j *JobBase) {
				s := "custom-namespace"
				j.Namespace = &s
			},
		},
		{
			name: "accept kubernetes agent",
			pass: true,
		},
		{
			name: "accept kubernetes agent without decoration",
			base: func(j *JobBase) {
				j.DecorationConfig = nil
			},
			pass: true,
		},
		{
			name: "accept knative-build agent",
			base: func(j *JobBase) {
				j.Agent = b
				j.BuildSpec = &buildv1alpha1.BuildSpec{}
				ns := "custom-namespace"
				j.Namespace = &ns
				j.Spec = nil
				j.DecorationConfig = nil
			},
			pass: true,
		},
		{
			name: "accept jenkins agent",
			base: func(j *JobBase) {
				j.Agent = jenk
				j.Spec = nil
				j.DecorationConfig = nil
			},
			pass: true,
		},
		{
			name: "error_on_eviction requires kubernetes agent",
			base: func(j *JobBase) {
				j.Agent = b
				j.ErrorOnEviction = true
			},
		},
		{
			name: "error_on_eviction allowed for kubernetes agent",
			base: func(j *JobBase) {
				j.ErrorOnEviction = true
			},
			pass: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jb := base
			if tc.base != nil {
				tc.base(&jb)
			}
			switch err := validateAgent(jb, ns); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

func TestValidatePodSpec(t *testing.T) {
	periodEnv := sets.NewString(downwardapi.EnvForType(prowapi.PeriodicJob)...)
	postEnv := sets.NewString(downwardapi.EnvForType(prowapi.PostsubmitJob)...)
	preEnv := sets.NewString(downwardapi.EnvForType(prowapi.PresubmitJob)...)
	cases := []struct {
		name    string
		jobType prowapi.ProwJobType
		spec    func(s *v1.PodSpec)
		noSpec  bool
		pass    bool
	}{
		{
			name:   "allow nil spec",
			noSpec: true,
			pass:   true,
		},
		{
			name: "happy case",
			pass: true,
		},
		{
			name: "reject init containers",
			spec: func(s *v1.PodSpec) {
				s.InitContainers = []v1.Container{
					{},
				}
			},
		},
		{
			name: "reject 0 containers",
			spec: func(s *v1.PodSpec) {
				s.Containers = nil
			},
		},
		{
			name: "reject 2 containers",
			spec: func(s *v1.PodSpec) {
				s.Containers = append(s.Containers, v1.Container{})
			},
		},
		{
			name:    "reject reserved presubmit env",
			jobType: prowapi.PresubmitJob,
			spec: func(s *v1.PodSpec) {
				// find a presubmit value
				for n := range preEnv.Difference(postEnv).Difference(periodEnv) {

					s.Containers[0].Env = append(s.Containers[0].Env, v1.EnvVar{Name: n, Value: "whatever"})
				}
				if len(s.Containers[0].Env) == 0 {
					t.Fatal("empty env")
				}
			},
		},
		{
			name:    "reject reserved postsubmit env",
			jobType: prowapi.PostsubmitJob,
			spec: func(s *v1.PodSpec) {
				// find a postsubmit value
				for n := range postEnv.Difference(periodEnv) {

					s.Containers[0].Env = append(s.Containers[0].Env, v1.EnvVar{Name: n, Value: "whatever"})
				}
				if len(s.Containers[0].Env) == 0 {
					t.Fatal("empty env")
				}
			},
		},
		{
			name:    "reject reserved periodic env",
			jobType: prowapi.PeriodicJob,
			spec: func(s *v1.PodSpec) {
				// find a postsubmit value
				for n := range periodEnv {

					s.Containers[0].Env = append(s.Containers[0].Env, v1.EnvVar{Name: n, Value: "whatever"})
				}
				if len(s.Containers[0].Env) == 0 {
					t.Fatal("empty env")
				}
			},
		},
		{
			name: "reject reserved mount name",
			spec: func(s *v1.PodSpec) {
				s.Containers[0].VolumeMounts = append(s.Containers[0].VolumeMounts, v1.VolumeMount{
					Name:      decorate.VolumeMounts()[0],
					MountPath: "/whatever",
				})
			},
		},
		{
			name: "reject reserved mount path",
			spec: func(s *v1.PodSpec) {
				s.Containers[0].VolumeMounts = append(s.Containers[0].VolumeMounts, v1.VolumeMount{
					Name:      "fun",
					MountPath: decorate.VolumeMountPaths()[0],
				})
			},
		},
		{
			name: "reject conflicting mount paths (decorate in user)",
			spec: func(s *v1.PodSpec) {
				s.Containers[0].VolumeMounts = append(s.Containers[0].VolumeMounts, v1.VolumeMount{
					Name:      "foo",
					MountPath: filepath.Dir(decorate.VolumeMountPaths()[0]),
				})
			},
		},
		{
			name: "reject conflicting mount paths (user in decorate)",
			spec: func(s *v1.PodSpec) {
				s.Containers[0].VolumeMounts = append(s.Containers[0].VolumeMounts, v1.VolumeMount{
					Name:      "foo",
					MountPath: filepath.Join(decorate.VolumeMountPaths()[0], "extra"),
				})
			},
		},
		{
			name: "reject reserved volume",
			spec: func(s *v1.PodSpec) {
				s.Volumes = append(s.Volumes, v1.Volume{Name: decorate.VolumeMounts()[0]})
			},
		},
	}

	spec := v1.PodSpec{
		Containers: []v1.Container{
			{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jt := prowapi.PresubmitJob
			if tc.jobType != "" {
				jt = tc.jobType
			}
			current := spec.DeepCopy()
			if tc.noSpec {
				current = nil
			} else if tc.spec != nil {
				tc.spec(current)
			}
			switch err := validatePodSpec(jt, current); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

func TestValidatePipelineRunSpec(t *testing.T) {
	cases := []struct {
		name      string
		jobType   prowapi.ProwJobType
		spec      func(s *pipelinev1alpha1.PipelineRunSpec)
		extraRefs []prowapi.Refs
		noSpec    bool
		pass      bool
	}{
		{
			name:   "allow nil spec",
			noSpec: true,
			pass:   true,
		},
		{
			name: "happy case",
			pass: true,
		},
		{
			name:    "reject implicit ref for periodic",
			jobType: prowapi.PeriodicJob,
			spec: func(s *pipelinev1alpha1.PipelineRunSpec) {
				s.Resources = append(s.Resources, pipelinev1alpha1.PipelineResourceBinding{
					Name:        "git ref",
					ResourceRef: pipelinev1alpha1.PipelineResourceRef{Name: "PROW_IMPLICIT_GIT_REF"},
				})
			},
			pass: false,
		},
		{
			name:    "allow implicit ref for presubmit",
			jobType: prowapi.PresubmitJob,
			spec: func(s *pipelinev1alpha1.PipelineRunSpec) {
				s.Resources = append(s.Resources, pipelinev1alpha1.PipelineResourceBinding{
					Name:        "git ref",
					ResourceRef: pipelinev1alpha1.PipelineResourceRef{Name: "PROW_IMPLICIT_GIT_REF"},
				})
			},
			pass: true,
		},
		{
			name:    "allow implicit ref for postsubmit",
			jobType: prowapi.PostsubmitJob,
			spec: func(s *pipelinev1alpha1.PipelineRunSpec) {
				s.Resources = append(s.Resources, pipelinev1alpha1.PipelineResourceBinding{
					Name:        "git ref",
					ResourceRef: pipelinev1alpha1.PipelineResourceRef{Name: "PROW_IMPLICIT_GIT_REF"},
				})
			},
			pass: true,
		},
		{
			name: "reject extra refs usage with no extra refs",
			spec: func(s *pipelinev1alpha1.PipelineRunSpec) {
				s.Resources = append(s.Resources, pipelinev1alpha1.PipelineResourceBinding{
					Name:        "git ref",
					ResourceRef: pipelinev1alpha1.PipelineResourceRef{Name: "PROW_EXTRA_GIT_REF_0"},
				})
			},
			pass: false,
		},
		{
			name: "allow extra refs usage with extra refs",
			spec: func(s *pipelinev1alpha1.PipelineRunSpec) {
				s.Resources = append(s.Resources, pipelinev1alpha1.PipelineResourceBinding{
					Name:        "git ref",
					ResourceRef: pipelinev1alpha1.PipelineResourceRef{Name: "PROW_EXTRA_GIT_REF_0"},
				})
			},
			extraRefs: []prowapi.Refs{{Org: "o", Repo: "r"}},
			pass:      true,
		},
		{
			name: "reject wrong extra refs index usage",
			spec: func(s *pipelinev1alpha1.PipelineRunSpec) {
				s.Resources = append(s.Resources, pipelinev1alpha1.PipelineResourceBinding{
					Name:        "git ref",
					ResourceRef: pipelinev1alpha1.PipelineResourceRef{Name: "PROW_EXTRA_GIT_REF_1"},
				})
			},
			extraRefs: []prowapi.Refs{{Org: "o", Repo: "r"}},
			pass:      false,
		},
		{
			name:      "reject extra refs without usage",
			extraRefs: []prowapi.Refs{{Org: "o", Repo: "r"}},
			pass:      false,
		},
		{
			name: "allow unrelated resource refs",
			spec: func(s *pipelinev1alpha1.PipelineRunSpec) {
				s.Resources = append(s.Resources, pipelinev1alpha1.PipelineResourceBinding{
					Name:        "git ref",
					ResourceRef: pipelinev1alpha1.PipelineResourceRef{Name: "some-other-ref"},
				})
			},
			pass: true,
		},
		{
			name: "reject leading zeros when extra ref usage is otherwise valid",
			spec: func(s *pipelinev1alpha1.PipelineRunSpec) {
				s.Resources = append(s.Resources, pipelinev1alpha1.PipelineResourceBinding{
					Name:        "git ref",
					ResourceRef: pipelinev1alpha1.PipelineResourceRef{Name: "PROW_EXTRA_GIT_REF_000"},
				})
			},
			extraRefs: []prowapi.Refs{{Org: "o", Repo: "r"}},
			pass:      false,
		},
	}

	spec := pipelinev1alpha1.PipelineRunSpec{}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jt := prowapi.PresubmitJob
			if tc.jobType != "" {
				jt = tc.jobType
			}
			current := spec.DeepCopy()
			if tc.noSpec {
				current = nil
			} else if tc.spec != nil {
				tc.spec(current)
			}
			switch err := ValidatePipelineRunSpec(jt, tc.extraRefs, current); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

func TestValidateDecoration(t *testing.T) {
	defCfg := prowapi.DecorationConfig{
		UtilityImages: &prowjobv1.UtilityImages{
			CloneRefs:  "clone-me",
			InitUpload: "upload-me",
			Entrypoint: "enter-me",
			Sidecar:    "official-drink-of-the-org",
		},
		GCSCredentialsSecret: "upload-secret",
		GCSConfiguration: &prowjobv1.GCSConfiguration{
			PathStrategy: prowjobv1.PathStrategyExplicit,
			DefaultOrg:   "so-org",
			DefaultRepo:  "very-repo",
		},
	}
	cases := []struct {
		name      string
		container v1.Container
		config    *prowapi.DecorationConfig
		pass      bool
	}{
		{
			name: "allow no decoration",
			pass: true,
		},
		{
			name:   "happy case with cmd",
			config: &defCfg,
			container: v1.Container{
				Command: []string{"hello", "world"},
			},
			pass: true,
		},
		{
			name:   "happy case with args",
			config: &defCfg,
			container: v1.Container{
				Args: []string{"hello", "world"},
			},
			pass: true,
		},
		{
			name:   "reject invalid decoration config",
			config: &prowapi.DecorationConfig{},
			container: v1.Container{
				Command: []string{"hello", "world"},
			},
		},
		{
			name:   "reject container that has no cmd, no args",
			config: &defCfg,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			switch err := validateDecoration(tc.container, tc.config); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

func TestValidateLabels(t *testing.T) {
	cases := []struct {
		name   string
		labels map[string]string
		pass   bool
	}{
		{
			name: "happy case",
			pass: true,
		},
		{
			name: "reject reserved label",
			labels: map[string]string{
				decorate.Labels()[0]: "anything",
			},
		},
		{
			name: "reject bad label key",
			labels: map[string]string{
				"_underscore-prefix": "annoying",
			},
		},
		{
			name: "reject bad label value",
			labels: map[string]string{
				"whatever": "_private-is-rejected",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			switch err := validateLabels(tc.labels); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

func TestValidateJobBase(t *testing.T) {
	ka := string(prowjobv1.KubernetesAgent)
	ba := string(prowjobv1.KnativeBuildAgent)
	ja := string(prowjobv1.JenkinsAgent)
	goodSpec := v1.PodSpec{
		Containers: []v1.Container{
			{},
		},
	}
	ns := "target-namespace"
	cases := []struct {
		name string
		base JobBase
		pass bool
	}{
		{
			name: "valid kubernetes job",
			base: JobBase{
				Name:      "name",
				Agent:     ka,
				Spec:      &goodSpec,
				Namespace: &ns,
			},
			pass: true,
		},
		{
			name: "valid build job",
			base: JobBase{
				Name:      "name",
				Agent:     ba,
				BuildSpec: &buildv1alpha1.BuildSpec{},
				Namespace: &ns,
			},
			pass: true,
		},
		{
			name: "valid jenkins job",
			base: JobBase{
				Name:      "name",
				Agent:     ja,
				Namespace: &ns,
			},
			pass: true,
		},
		{
			name: "invalid concurrency",
			base: JobBase{
				Name:           "name",
				MaxConcurrency: -1,
				Agent:          ka,
				Spec:           &goodSpec,
				Namespace:      &ns,
			},
		},
		{
			name: "invalid agent",
			base: JobBase{
				Name:      "name",
				Agent:     ba,
				Spec:      &goodSpec, // want BuildSpec
				Namespace: &ns,
			},
		},
		{
			name: "invalid pod spec",
			base: JobBase{
				Name:      "name",
				Agent:     ka,
				Namespace: &ns,
				Spec:      &v1.PodSpec{}, // no containers
			},
		},
		{
			name: "invalid decoration",
			base: JobBase{
				Name:  "name",
				Agent: ka,
				Spec:  &goodSpec,
				UtilityConfig: UtilityConfig{
					DecorationConfig: &prowjobv1.DecorationConfig{}, // missing many fields
				},
				Namespace: &ns,
			},
		},
		{
			name: "invalid labels",
			base: JobBase{
				Name:  "name",
				Agent: ka,
				Spec:  &goodSpec,
				Labels: map[string]string{
					"_leading_underscore": "_rejected",
				},
				Namespace: &ns,
			},
		},
		{
			name: "invalid name",
			base: JobBase{
				Name:      "a/b",
				Agent:     ka,
				Spec:      &goodSpec,
				Namespace: &ns,
			},
			pass: false,
		},
		{
			name: "valid complex name",
			base: JobBase{
				Name:      "a-b.c",
				Agent:     ka,
				Spec:      &goodSpec,
				Namespace: &ns,
			},
			pass: true,
		},
		{
			name: "invalid rerun_permissions",
			base: JobBase{
				RerunAuthConfig: &prowapi.RerunAuthConfig{
					AllowAnyone: true,
					GitHubUsers: []string{"user"},
				},
			},
			pass: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			switch err := validateJobBase(tc.base, prowjobv1.PresubmitJob, ns); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

// integration test for fake config loading
func TestValidConfigLoading(t *testing.T) {
	var testCases = []struct {
		name               string
		prowConfig         string
		jobConfigs         []string
		expectError        bool
		expectPodNameSpace string
		expectEnv          map[string][]v1.EnvVar
	}{
		{
			name:       "one config",
			prowConfig: ``,
		},
		{
			name:       "reject invalid kubernetes periodic",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- interval: 10m
  agent: kubernetes
  build_spec:
  name: foo`,
			},
			expectError: true,
		},
		{
			name:       "reject invalid build periodic",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- interval: 10m
  agent: knative-build
  spec:
  name: foo`,
			},
			expectError: true,
		},
		{
			name:       "one periodic",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: foo
  spec:
    containers:
    - image: alpine`,
			},
		},
		{
			name:       "one periodic no agent, should default",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- interval: 10m
  name: foo
  spec:
    containers:
    - image: alpine`,
			},
		},
		{
			name:       "two periodics",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: foo
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: bar
  spec:
    containers:
    - image: alpine`,
			},
		},
		{
			name:       "duplicated periodics",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: foo
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: foo
  spec:
    containers:
    - image: alpine`,
			},
			expectError: true,
		},
		{
			name:       "one presubmit no context should default",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: kubernetes
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
		},
		{
			name:       "one presubmit no agent should default",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - context: bar
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
		},
		{
			name:       "one presubmit, ok",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: kubernetes
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
			},
		},
		{
			name:       "two presubmits",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: kubernetes
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
				`
presubmits:
  foo/baz:
  - agent: kubernetes
    name: presubmit-baz
    context: baz
    spec:
      containers:
      - image: alpine`,
			},
		},
		{
			name:       "dup presubmits, one file",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: kubernetes
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine
  - agent: kubernetes
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: true,
		},
		{
			name:       "dup presubmits, two files",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: kubernetes
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
				`
presubmits:
  foo/bar:
  - agent: kubernetes
    context: bar
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: true,
		},
		{
			name:       "dup presubmits not the same branch, two files",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: kubernetes
    name: presubmit-bar
    context: bar
    branches:
    - master
    spec:
      containers:
      - image: alpine`,
				`
presubmits:
  foo/bar:
  - agent: kubernetes
    context: bar
    branches:
    - other
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: false,
		},
		{
			name: "dup presubmits main file",
			prowConfig: `
presubmits:
  foo/bar:
  - agent: kubernetes
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine
  - agent: kubernetes
    context: bar
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			expectError: true,
		},
		{
			name: "dup presubmits main file not on the same branch",
			prowConfig: `
presubmits:
  foo/bar:
  - agent: kubernetes
    name: presubmit-bar
    context: bar
    branches:
    - other
    spec:
      containers:
      - image: alpine
  - agent: kubernetes
    context: bar
    branches:
    - master
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			expectError: false,
		},

		{
			name:       "one postsubmit, ok",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - agent: kubernetes
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
		},
		{
			name:       "one postsubmit no agent, should default",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
		},
		{
			name:       "two postsubmits",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - agent: kubernetes
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
				`
postsubmits:
  foo/baz:
  - agent: kubernetes
    name: postsubmit-baz
    spec:
      containers:
      - image: alpine`,
			},
		},
		{
			name:       "dup postsubmits, one file",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - agent: kubernetes
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine
  - agent: kubernetes
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: true,
		},
		{
			name:       "dup postsubmits, two files",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - agent: kubernetes
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
				`
postsubmits:
  foo/bar:
  - agent: kubernetes
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: true,
		},
		{
			name: "test valid presets in main config",
			prowConfig: `
presets:
- labels:
    preset-baz: "true"
  env:
  - name: baz
    value: fejtaverse`,
			jobConfigs: []string{
				`periodics:
- interval: 10m
  agent: kubernetes
  name: foo
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: bar
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
			},
			expectEnv: map[string][]v1.EnvVar{
				"foo": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
				"bar": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
			},
		},
		{
			name:       "test valid presets in job configs",
			prowConfig: ``,
			jobConfigs: []string{
				`
presets:
- labels:
    preset-baz: "true"
  env:
  - name: baz
    value: fejtaverse
periodics:
- interval: 10m
  agent: kubernetes
  name: foo
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: bar
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
			},
			expectEnv: map[string][]v1.EnvVar{
				"foo": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
				"bar": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
			},
		},
		{
			name: "test valid presets in both main & job configs",
			prowConfig: `
presets:
- labels:
    preset-baz: "true"
  env:
  - name: baz
    value: fejtaverse`,
			jobConfigs: []string{
				`
presets:
- labels:
    preset-k8s: "true"
  env:
  - name: k8s
    value: kubernetes
periodics:
- interval: 10m
  agent: kubernetes
  name: foo
  labels:
    preset-baz: "true"
    preset-k8s: "true"
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: bar
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
			},
			expectEnv: map[string][]v1.EnvVar{
				"foo": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
					{
						Name:  "k8s",
						Value: "kubernetes",
					},
				},
				"bar": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
			},
		},
		{
			name:       "decorated periodic missing `command`",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- interval: 10m
  agent: kubernetes
  name: foo
  decorate: true
  spec:
    containers:
    - image: alpine`,
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		// save the config
		prowConfigDir, err := ioutil.TempDir("", "prowConfig")
		if err != nil {
			t.Fatalf("fail to make tempdir: %v", err)
		}
		defer os.RemoveAll(prowConfigDir)

		prowConfig := filepath.Join(prowConfigDir, "config.yaml")
		if err := ioutil.WriteFile(prowConfig, []byte(tc.prowConfig), 0666); err != nil {
			t.Fatalf("fail to write prow config: %v", err)
		}

		jobConfig := ""
		if len(tc.jobConfigs) > 0 {
			jobConfigDir, err := ioutil.TempDir("", "jobConfig")
			if err != nil {
				t.Fatalf("fail to make tempdir: %v", err)
			}
			defer os.RemoveAll(jobConfigDir)

			// cover both job config as a file & a dir
			if len(tc.jobConfigs) == 1 {
				// a single file
				jobConfig = filepath.Join(jobConfigDir, "config.yaml")
				if err := ioutil.WriteFile(jobConfig, []byte(tc.jobConfigs[0]), 0666); err != nil {
					t.Fatalf("fail to write job config: %v", err)
				}
			} else {
				// a dir
				jobConfig = jobConfigDir
				for idx, config := range tc.jobConfigs {
					subConfig := filepath.Join(jobConfigDir, fmt.Sprintf("config_%d.yaml", idx))
					if err := ioutil.WriteFile(subConfig, []byte(config), 0666); err != nil {
						t.Fatalf("fail to write job config: %v", err)
					}
				}
			}
		}

		cfg, err := Load(prowConfig, jobConfig)
		if tc.expectError && err == nil {
			t.Errorf("tc %s: Expect error, but got nil", tc.name)
		} else if !tc.expectError && err != nil {
			t.Errorf("tc %s: Expect no error, but got error %v", tc.name, err)
		}

		if err == nil {
			if tc.expectPodNameSpace == "" {
				tc.expectPodNameSpace = "default"
			}

			if cfg.PodNamespace != tc.expectPodNameSpace {
				t.Errorf("tc %s: Expect PodNamespace %s, but got %v", tc.name, tc.expectPodNameSpace, cfg.PodNamespace)
			}

			if len(tc.expectEnv) > 0 {
				for _, j := range cfg.AllPresubmits(nil) {
					if envs, ok := tc.expectEnv[j.Name]; ok {
						if !reflect.DeepEqual(envs, j.Spec.Containers[0].Env) {
							t.Errorf("tc %s: expect env %v for job %s, got %+v", tc.name, envs, j.Name, j.Spec.Containers[0].Env)
						}
					}
				}

				for _, j := range cfg.AllPostsubmits(nil) {
					if envs, ok := tc.expectEnv[j.Name]; ok {
						if !reflect.DeepEqual(envs, j.Spec.Containers[0].Env) {
							t.Errorf("tc %s: expect env %v for job %s, got %+v", tc.name, envs, j.Name, j.Spec.Containers[0].Env)
						}
					}
				}

				for _, j := range cfg.AllPeriodics() {
					if envs, ok := tc.expectEnv[j.Name]; ok {
						if !reflect.DeepEqual(envs, j.Spec.Containers[0].Env) {
							t.Errorf("tc %s: expect env %v for job %s, got %+v", tc.name, envs, j.Name, j.Spec.Containers[0].Env)
						}
					}
				}
			}
		}
	}
}

func TestBrancher_Intersects(t *testing.T) {
	testCases := []struct {
		name   string
		a, b   Brancher
		result bool
	}{
		{
			name: "TwodifferentBranches",
			a: Brancher{
				Branches: []string{"a"},
			},
			b: Brancher{
				Branches: []string{"b"},
			},
		},
		{
			name: "Opposite",
			a: Brancher{
				SkipBranches: []string{"b"},
			},
			b: Brancher{
				Branches: []string{"b"},
			},
		},
		{
			name:   "BothRunOnAllBranches",
			a:      Brancher{},
			b:      Brancher{},
			result: true,
		},
		{
			name: "RunsOnAllBranchesAndSpecified",
			a:    Brancher{},
			b: Brancher{
				Branches: []string{"b"},
			},
			result: true,
		},
		{
			name: "SkipBranchesAndSet",
			a: Brancher{
				SkipBranches: []string{"a", "b", "c"},
			},
			b: Brancher{
				Branches: []string{"a"},
			},
		},
		{
			name: "SkipBranchesAndSet",
			a: Brancher{
				Branches: []string{"c"},
			},
			b: Brancher{
				Branches: []string{"a"},
			},
		},
		{
			name: "BothSkipBranches",
			a: Brancher{
				SkipBranches: []string{"a", "b", "c"},
			},
			b: Brancher{
				SkipBranches: []string{"d", "e", "f"},
			},
			result: true,
		},
		{
			name: "BothSkipCommonBranches",
			a: Brancher{
				SkipBranches: []string{"a", "b", "c"},
			},
			b: Brancher{
				SkipBranches: []string{"b", "e", "f"},
			},
			result: true,
		},
		{
			name: "NoIntersectionBecauseRegexSkip",
			a: Brancher{
				SkipBranches: []string{`release-\d+\.\d+`},
			},
			b: Brancher{
				Branches: []string{`release-1.14`, `release-1.13`},
			},
			result: false,
		},
		{
			name: "IntersectionDespiteRegexSkip",
			a: Brancher{
				SkipBranches: []string{`release-\d+\.\d+`},
			},
			b: Brancher{
				Branches: []string{`release-1.14`, `master`},
			},
			result: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(st *testing.T) {
			a, err := setBrancherRegexes(tc.a)
			if err != nil {
				st.Fatalf("Failed to set brancher A regexes: %v", err)
			}
			b, err := setBrancherRegexes(tc.b)
			if err != nil {
				st.Fatalf("Failed to set brancher B regexes: %v", err)
			}
			r1 := a.Intersects(b)
			r2 := b.Intersects(a)
			for _, result := range []bool{r1, r2} {
				if result != tc.result {
					st.Errorf("Expected %v got %v", tc.result, result)
				}
			}
		})
	}
}

// Integration test for fake secrets loading in a secret agent.
// Checking also if the agent changes the secret's values as expected.
func TestSecretAgentLoading(t *testing.T) {
	tempTokenValue := "121f3cb3e7f70feeb35f9204f5a988d7292c7ba1"
	changedTokenValue := "121f3cb3e7f70feeb35f9204f5a988d7292c7ba0"

	// Creating a temporary directory.
	secretDir, err := ioutil.TempDir("", "secretDir")
	if err != nil {
		t.Fatalf("fail to create a temporary directory: %v", err)
	}
	defer os.RemoveAll(secretDir)

	// Create the first temporary secret.
	firstTempSecret := filepath.Join(secretDir, "firstTempSecret")
	if err := ioutil.WriteFile(firstTempSecret, []byte(tempTokenValue), 0666); err != nil {
		t.Fatalf("fail to write secret: %v", err)
	}

	// Create the second temporary secret.
	secondTempSecret := filepath.Join(secretDir, "secondTempSecret")
	if err := ioutil.WriteFile(secondTempSecret, []byte(tempTokenValue), 0666); err != nil {
		t.Fatalf("fail to write secret: %v", err)
	}

	tempSecrets := []string{firstTempSecret, secondTempSecret}
	// Starting the agent and add the two temporary secrets.
	secretAgent := &secret.Agent{}
	if err := secretAgent.Start(tempSecrets); err != nil {
		t.Fatalf("Error starting secrets agent. %v", err)
	}

	// Check if the values are as expected.
	for _, tempSecret := range tempSecrets {
		tempSecretValue := secretAgent.GetSecret(tempSecret)
		if string(tempSecretValue) != tempTokenValue {
			t.Fatalf("In secret %s it was expected %s but found %s",
				tempSecret, tempTokenValue, tempSecretValue)
		}
	}

	// Change the values of the files.
	if err := ioutil.WriteFile(firstTempSecret, []byte(changedTokenValue), 0666); err != nil {
		t.Fatalf("fail to write secret: %v", err)
	}
	if err := ioutil.WriteFile(secondTempSecret, []byte(changedTokenValue), 0666); err != nil {
		t.Fatalf("fail to write secret: %v", err)
	}

	retries := 10
	var errors []string

	// Check if the values changed as expected.
	for _, tempSecret := range tempSecrets {
		// Reset counter
		counter := 0
		for counter <= retries {
			tempSecretValue := secretAgent.GetSecret(tempSecret)
			if string(tempSecretValue) != changedTokenValue {
				if counter == retries {
					errors = append(errors, fmt.Sprintf("In secret %s it was expected %s but found %s\n",
						tempSecret, changedTokenValue, tempSecretValue))
				} else {
					// Secret agent needs some time to update the values. So wait and retry.
					time.Sleep(400 * time.Millisecond)
				}
			} else {
				break
			}
			counter++
		}
	}

	if len(errors) > 0 {
		t.Fatal(errors)
	}

}

func TestValidGitHubReportType(t *testing.T) {
	var testCases = []struct {
		name        string
		prowConfig  string
		expectError bool
		expectTypes []prowapi.ProwJobType
	}{
		{
			name:        "empty config should default to report for both presubmit and postsubmit",
			prowConfig:  ``,
			expectTypes: []prowapi.ProwJobType{prowapi.PresubmitJob, prowapi.PostsubmitJob},
		},
		{
			name: "reject unsupported job types",
			prowConfig: `
github_reporter:
  job_types_to_report:
  - presubmit
  - batch
`,
			expectError: true,
		},
		{
			name: "accept valid job types",
			prowConfig: `
github_reporter:
  job_types_to_report:
  - presubmit
  - postsubmit
`,
			expectTypes: []prowapi.ProwJobType{prowapi.PresubmitJob, prowapi.PostsubmitJob},
		},
	}

	for _, tc := range testCases {
		// save the config
		prowConfigDir, err := ioutil.TempDir("", "prowConfig")
		if err != nil {
			t.Fatalf("fail to make tempdir: %v", err)
		}
		defer os.RemoveAll(prowConfigDir)

		prowConfig := filepath.Join(prowConfigDir, "config.yaml")
		if err := ioutil.WriteFile(prowConfig, []byte(tc.prowConfig), 0666); err != nil {
			t.Fatalf("fail to write prow config: %v", err)
		}

		cfg, err := Load(prowConfig, "")
		if tc.expectError && err == nil {
			t.Errorf("tc %s: Expect error, but got nil", tc.name)
		} else if !tc.expectError && err != nil {
			t.Errorf("tc %s: Expect no error, but got error %v", tc.name, err)
		}

		if err == nil {
			if !reflect.DeepEqual(cfg.GitHubReporter.JobTypesToReport, tc.expectTypes) {
				t.Errorf("tc %s: expected %#v\n!=\nactual %#v", tc.name, tc.expectTypes, cfg.GitHubReporter.JobTypesToReport)
			}
		}
	}
}

func TestValidRerunAuthConfig(t *testing.T) {
	var testCases = []struct {
		name        string
		prowConfig  string
		expectError bool
	}{
		{
			name: "valid rerun auth config",
			prowConfig: `
deck:
  rerun_auth_config:
    allow_anyone: false
    github_users:
    - someperson
    - someotherperson
`,
			expectError: false,
		},
		{
			name: "allow anyone and whitelist specified",
			prowConfig: `
deck:
  rerun_auth_config:
    allow_anyone: true
    github_users:
    - someperson
    - anotherperson
`,
			expectError: true,
		},
		{
			name: "empty config",
			prowConfig: `
deck:
  rerun_auth_config:
`,
			expectError: false,
		},
		{
			name: "allow anyone with empty whitelist",
			prowConfig: `
deck:
  rerun_auth_config:
    allow_anyone: true
    github_users:
`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		// save the config
		prowConfigDir, err := ioutil.TempDir("", "prowConfig")
		if err != nil {
			t.Fatalf("fail to make tempdir: %v", err)
		}
		defer os.RemoveAll(prowConfigDir)

		prowConfig := filepath.Join(prowConfigDir, "config.yaml")
		if err := ioutil.WriteFile(prowConfig, []byte(tc.prowConfig), 0666); err != nil {
			t.Fatalf("fail to write prow config: %v", err)
		}

		_, err = Load(prowConfig, "")
		if tc.expectError && err == nil {
			t.Errorf("tc %s: Expect error, but got nil", tc.name)
		} else if !tc.expectError && err != nil {
			t.Errorf("tc %s: Expect no error, but got error %v", tc.name, err)
		}
	}
}

func TestMergeCommitTemplateLoading(t *testing.T) {
	var testCases = []struct {
		name        string
		prowConfig  string
		expectError bool
		expect      map[string]TideMergeCommitTemplate
	}{
		{
			name: "no template",
			prowConfig: `
tide:
  merge_commit_template:
`,
			expect: nil,
		},
		{
			name: "empty template",
			prowConfig: `
tide:
  merge_commit_template:
    kubernetes/ingress:
`,
			expect: map[string]TideMergeCommitTemplate{
				"kubernetes/ingress": {},
			},
		},
		{
			name: "two proper templates",
			prowConfig: `
tide:
  merge_commit_template:
    kubernetes/ingress:
      title: "{{ .Title }}"
      body: "{{ .Body }}"
`,
			expect: map[string]TideMergeCommitTemplate{
				"kubernetes/ingress": {
					TitleTemplate: "{{ .Title }}",
					BodyTemplate:  "{{ .Body }}",
					Title:         template.Must(template.New("CommitTitle").Parse("{{ .Title }}")),
					Body:          template.Must(template.New("CommitBody").Parse("{{ .Body }}")),
				},
			},
		},
		{
			name: "only title template",
			prowConfig: `
tide:
  merge_commit_template:
    kubernetes/ingress:
      title: "{{ .Title }}"
`,
			expect: map[string]TideMergeCommitTemplate{
				"kubernetes/ingress": {
					TitleTemplate: "{{ .Title }}",
					BodyTemplate:  "",
					Title:         template.Must(template.New("CommitTitle").Parse("{{ .Title }}")),
					Body:          nil,
				},
			},
		},
		{
			name: "only body template",
			prowConfig: `
tide:
  merge_commit_template:
    kubernetes/ingress:
      body: "{{ .Body }}"
`,
			expect: map[string]TideMergeCommitTemplate{
				"kubernetes/ingress": {
					TitleTemplate: "",
					BodyTemplate:  "{{ .Body }}",
					Title:         nil,
					Body:          template.Must(template.New("CommitBody").Parse("{{ .Body }}")),
				},
			},
		},
		{
			name: "malformed title template",
			prowConfig: `
tide:
  merge_commit_template:
    kubernetes/ingress:
      title: "{{ .Title"
`,
			expectError: true,
		},
		{
			name: "malformed body template",
			prowConfig: `
tide:
  merge_commit_template:
    kubernetes/ingress:
      body: "{{ .Body"
`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		// save the config
		prowConfigDir, err := ioutil.TempDir("", "prowConfig")
		if err != nil {
			t.Fatalf("fail to make tempdir: %v", err)
		}
		defer os.RemoveAll(prowConfigDir)

		prowConfig := filepath.Join(prowConfigDir, "config.yaml")
		if err := ioutil.WriteFile(prowConfig, []byte(tc.prowConfig), 0666); err != nil {
			t.Fatalf("fail to write prow config: %v", err)
		}

		cfg, err := Load(prowConfig, "")
		if tc.expectError && err == nil {
			t.Errorf("tc %s: Expect error, but got nil", tc.name)
		} else if !tc.expectError && err != nil {
			t.Errorf("tc %s: Expect no error, but got error %v", tc.name, err)
		}

		if err == nil {
			if !reflect.DeepEqual(cfg.Tide.MergeTemplate, tc.expect) {
				t.Errorf("tc %s: expected %#v\n!=\nactual %#v", tc.name, tc.expect, cfg.Tide.MergeTemplate)
			}
		}
	}
}

func TestPlankJobURLPrefix(t *testing.T) {
	testCases := []struct {
		name                 string
		plank                Plank
		refs                 *prowapi.Refs
		expectedJobURLPrefix string
	}{
		{
			name:                 "Nil refs returns default JobURLPrefix",
			plank:                Plank{JobURLPrefixConfig: map[string]string{"*": "https://my-prow"}},
			expectedJobURLPrefix: "https://my-prow",
		},
		{
			name: "No matching refs returns default JobURLPrefx",
			plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":      "https://my-prow",
					"my-org": "https://my-alternate-prow",
				},
			},
			refs:                 &prowapi.Refs{Org: "my-default-org", Repo: "my-default-repo"},
			expectedJobURLPrefix: "https://my-prow",
		},
		{
			name: "Matching repo returns JobURLPrefix from repo",
			plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":                        "https://my-prow",
					"my-alternate-org":         "https://my-third-prow",
					"my-alternate-org/my-repo": "https://my-alternate-prow",
				},
			},
			refs:                 &prowapi.Refs{Org: "my-alternate-org", Repo: "my-repo"},
			expectedJobURLPrefix: "https://my-alternate-prow",
		},
		{
			name: "Matching org and not matching repo returns JobURLPrefix from org",
			plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":                        "https://my-prow",
					"my-alternate-org":         "https://my-third-prow",
					"my-alternate-org/my-repo": "https://my-alternate-prow",
				},
			},
			refs:                 &prowapi.Refs{Org: "my-alternate-org", Repo: "my-second-repo"},
			expectedJobURLPrefix: "https://my-third-prow",
		},
		{
			name: "Matching org without url returns default JobURLPrefix",
			plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":                        "https://my-prow",
					"my-alternate-org/my-repo": "https://my-alternate-prow",
				},
			},
			refs:                 &prowapi.Refs{Org: "my-alternate-org", Repo: "my-second-repo"},
			expectedJobURLPrefix: "https://my-prow",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if prefix := tc.plank.GetJobURLPrefix(tc.refs); prefix != tc.expectedJobURLPrefix {
				t.Errorf("expected JobURLPrefix to be %q but was %q", tc.expectedJobURLPrefix, prefix)
			}
		})
	}
}

func TestValidateComponentConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Config
		errExpected bool
	}{
		{
			name: `JobURLPrefix and JobURLPrefixConfig["*"].URL set, err`,
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefix: "https://my-default-prow",
				JobURLPrefixConfig: map[string]string{
					"*": "https://my-alternate-prow",
				},
			}}},
			errExpected: true,
		},
		{
			name: `JobURLPrefix and JobURLPrefixConfig["*"].URL unset, no err`,
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefix: "https://my-default-prow",
				JobURLPrefixConfig: map[string]string{
					"my-other-org": "https://my-alternate-prow",
				},
			}}},
			errExpected: false,
		},
		{
			name: "Valid default URL, no err",
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefixConfig: map[string]string{"*": "https://my-prow"}}}},
			errExpected: false,
		},
		{
			name: "Invalid default URL, err",
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefixConfig: map[string]string{"*": "https:// my-prow"}}}},
			errExpected: true,
		},
		{
			name: "Org config, valid URLs, no err",
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":      "https://my-prow",
					"my-org": "https://my-alternate-prow",
				},
			}}},
			errExpected: false,
		},
		{
			name: "Org override, invalid default jobURLPrefix URL, err",
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":      "https:// my-prow",
					"my-org": "https://my-alternate-prow",
				},
			}}},
			errExpected: true,
		},
		{
			name: "Org override, invalid org URL, err",
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":      "https://my-prow",
					"my-org": "https:// my-alternate-prow",
				},
			}}},
			errExpected: true,
		},
		{
			name: "Org override, invalid URLs, err",
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":      "https:// my-prow",
					"my-org": "https:// my-alternate-prow",
				},
			}}},
			errExpected: true,
		},
		{
			name: "Repo override, valid URLs, no err",
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":              "https://my-prow",
					"my-org":         "https://my-alternate-prow",
					"my-org/my-repo": "https://my-third-prow",
				}}}},
			errExpected: false,
		},
		{
			name: "Repo override, invalid repo URL, err",
			config: &Config{ProwConfig: ProwConfig{Plank: Plank{
				JobURLPrefixConfig: map[string]string{
					"*":              "https://my-prow",
					"my-org":         "https://my-alternate-prow",
					"my-org/my-repo": "https:// my-third-prow",
				}}}},
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if hasErr := tc.config.validateComponentConfig() != nil; hasErr != tc.errExpected {
				t.Errorf("expected err: %t but was %t", tc.errExpected, hasErr)
			}
		})
	}
}

func TestSlackReporterValidation(t *testing.T) {
	testCases := []struct {
		name            string
		channel         string
		reportTemplate  string
		successExpected bool
	}{
		{
			name:            "Valid config - no error",
			channel:         "my-channel",
			successExpected: true,
		},
		{
			name: "No channel - error",
		},
		{
			name:           "Invalid template - error",
			channel:        "my-channel",
			reportTemplate: "{{ if .Spec.Name}}",
		},
		{
			name:           "Template accessed invalid property - error",
			channel:        "my-channel",
			reportTemplate: "{{ .Undef}}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &SlackReporter{
				Channel:        tc.channel,
				ReportTemplate: tc.reportTemplate,
			}

			if err := cfg.DefaultAndValidate(); (err == nil) != tc.successExpected {
				t.Errorf("Expected success=%t but got err=%v", tc.successExpected, err)
			}
		})
	}
}

func TestValidateTriggering(t *testing.T) {
	testCases := []struct {
		name        string
		presubmit   Presubmit
		errExpected bool
	}{
		{
			name: "Trigger set, rerun command unset, err",
			presubmit: Presubmit{
				Trigger: "my-trigger",
				Reporter: Reporter{
					Context: "my-context",
				},
			},
			errExpected: true,
		},
		{
			name: "Triger unset, rerun command set, err",
			presubmit: Presubmit{
				RerunCommand: "my-rerun-command",
				Reporter: Reporter{
					Context: "my-context",
				},
			},
			errExpected: true,
		},
		{
			name: "Both trigger and rerun command set, no err",
			presubmit: Presubmit{
				Trigger:      "my-trigger",
				RerunCommand: "my-rerun-command",
				Reporter: Reporter{
					Context: "my-context",
				},
			},
			errExpected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTriggering(tc.presubmit)
			if err != nil != tc.errExpected {
				t.Errorf("Expected err: %t but got err %v", tc.errExpected, err)
			}
		})
	}
}
