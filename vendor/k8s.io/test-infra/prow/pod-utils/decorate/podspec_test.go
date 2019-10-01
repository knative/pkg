/*
Copyright 2018 The Kubernetes Authors.

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

package decorate

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	coreapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/diff"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/clonerefs"
	"k8s.io/test-infra/prow/entrypoint"
	"k8s.io/test-infra/prow/initupload"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/sidecar"
)

func cookieVolumeOnly(secret string) coreapi.Volume {
	v, _, _ := cookiefileVolume(secret)
	return v
}

func cookieMountOnly(secret string) coreapi.VolumeMount {
	_, vm, _ := cookiefileVolume(secret)
	return vm
}
func cookiePathOnly(secret string) string {
	_, _, vp := cookiefileVolume(secret)
	return vp
}

func TestCloneRefs(t *testing.T) {
	truth := true
	logMount := coreapi.VolumeMount{
		Name:      "log",
		MountPath: "/log-mount",
	}
	codeMount := coreapi.VolumeMount{
		Name:      "code",
		MountPath: "/code-mount",
	}
	envOrDie := func(opt clonerefs.Options) []coreapi.EnvVar {
		e, err := cloneEnv(opt)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	sshVolumeOnly := func(secret string) coreapi.Volume {
		v, _ := sshVolume(secret)
		return v
	}

	sshMountOnly := func(secret string) coreapi.VolumeMount {
		_, vm := sshVolume(secret)
		return vm
	}

	cases := []struct {
		name              string
		pj                prowapi.ProwJob
		codeMountOverride *coreapi.VolumeMount
		logMountOverride  *coreapi.VolumeMount
		expected          *coreapi.Container
		volumes           []coreapi.Volume
		err               bool
	}{
		{
			name: "empty returns nil",
		},
		{
			name: "nil refs and extrarefs returns nil",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					DecorationConfig: &prowapi.DecorationConfig{},
				},
			},
		},
		{
			name: "nil DecorationConfig returns nil",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					Refs: &prowapi.Refs{},
				},
			},
		},
		{
			name: "SkipCloning returns nil",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					Refs: &prowapi.Refs{},
					DecorationConfig: &prowapi.DecorationConfig{
						SkipCloning: &truth,
					},
				},
			},
		},
		{
			name: "reject empty code mount name",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					DecorationConfig: &prowapi.DecorationConfig{},
					Refs:             &prowapi.Refs{},
				},
			},
			codeMountOverride: &coreapi.VolumeMount{
				MountPath: "/whatever",
			},
			err: true,
		},
		{
			name: "reject empty code mountpath",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					DecorationConfig: &prowapi.DecorationConfig{},
					Refs:             &prowapi.Refs{},
				},
			},
			codeMountOverride: &coreapi.VolumeMount{
				Name: "wee",
			},
			err: true,
		},
		{
			name: "reject empty log mount name",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					DecorationConfig: &prowapi.DecorationConfig{},
					Refs:             &prowapi.Refs{},
				},
			},
			logMountOverride: &coreapi.VolumeMount{
				MountPath: "/whatever",
			},
			err: true,
		},
		{
			name: "reject empty log mountpath",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					DecorationConfig: &prowapi.DecorationConfig{},
					Refs:             &prowapi.Refs{},
				},
			},
			logMountOverride: &coreapi.VolumeMount{
				Name: "wee",
			},
			err: true,
		},
		{
			name: "create clonerefs container when refs are set",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					Refs: &prowapi.Refs{},
					DecorationConfig: &prowapi.DecorationConfig{
						UtilityImages: &prowapi.UtilityImages{},
					},
				},
			},
			expected: &coreapi.Container{
				Name:    cloneRefsName,
				Command: []string{cloneRefsCommand},
				Env: envOrDie(clonerefs.Options{
					GitRefs:      []prowapi.Refs{{}},
					GitUserEmail: clonerefs.DefaultGitUserEmail,
					GitUserName:  clonerefs.DefaultGitUserName,
					SrcRoot:      codeMount.MountPath,
					Log:          CloneLogPath(logMount),
				}),
				VolumeMounts: []coreapi.VolumeMount{logMount, codeMount},
			},
		},
		{
			name: "create clonerefs containers when extrarefs are set",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					ExtraRefs: []prowapi.Refs{{}},
					DecorationConfig: &prowapi.DecorationConfig{
						UtilityImages: &prowapi.UtilityImages{},
					},
				},
			},
			expected: &coreapi.Container{
				Name:    cloneRefsName,
				Command: []string{cloneRefsCommand},
				Env: envOrDie(clonerefs.Options{
					GitRefs:      []prowapi.Refs{{}},
					GitUserEmail: clonerefs.DefaultGitUserEmail,
					GitUserName:  clonerefs.DefaultGitUserName,
					SrcRoot:      codeMount.MountPath,
					Log:          CloneLogPath(logMount),
				}),
				VolumeMounts: []coreapi.VolumeMount{logMount, codeMount},
			},
		},
		{
			name: "append extrarefs after refs",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					Refs:      &prowapi.Refs{Org: "first"},
					ExtraRefs: []prowapi.Refs{{Org: "second"}, {Org: "third"}},
					DecorationConfig: &prowapi.DecorationConfig{
						UtilityImages: &prowapi.UtilityImages{},
					},
				},
			},
			expected: &coreapi.Container{
				Name:    cloneRefsName,
				Command: []string{cloneRefsCommand},
				Env: envOrDie(clonerefs.Options{
					GitRefs:      []prowapi.Refs{{Org: "first"}, {Org: "second"}, {Org: "third"}},
					GitUserEmail: clonerefs.DefaultGitUserEmail,
					GitUserName:  clonerefs.DefaultGitUserName,
					SrcRoot:      codeMount.MountPath,
					Log:          CloneLogPath(logMount),
				}),
				VolumeMounts: []coreapi.VolumeMount{logMount, codeMount},
			},
		},
		{
			name: "append ssh secrets when set",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					Refs: &prowapi.Refs{},
					DecorationConfig: &prowapi.DecorationConfig{
						UtilityImages: &prowapi.UtilityImages{},
						SSHKeySecrets: []string{"super", "secret"},
					},
				},
			},
			expected: &coreapi.Container{
				Name:    cloneRefsName,
				Command: []string{cloneRefsCommand},
				Env: envOrDie(clonerefs.Options{
					GitRefs:      []prowapi.Refs{{}},
					GitUserEmail: clonerefs.DefaultGitUserEmail,
					GitUserName:  clonerefs.DefaultGitUserName,
					KeyFiles:     []string{sshMountOnly("super").MountPath, sshMountOnly("secret").MountPath},
					SrcRoot:      codeMount.MountPath,
					Log:          CloneLogPath(logMount),
				}),
				VolumeMounts: []coreapi.VolumeMount{
					logMount,
					codeMount,
					sshMountOnly("super"),
					sshMountOnly("secret"),
				},
			},
			volumes: []coreapi.Volume{sshVolumeOnly("super"), sshVolumeOnly("secret")},
		},
		{
			name: "include ssh host fingerprints when set",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					ExtraRefs: []prowapi.Refs{{}},
					DecorationConfig: &prowapi.DecorationConfig{
						UtilityImages:       &prowapi.UtilityImages{},
						SSHHostFingerprints: []string{"thumb", "pinky"},
					},
				},
			},
			expected: &coreapi.Container{
				Name:    cloneRefsName,
				Command: []string{cloneRefsCommand},
				Env: envOrDie(clonerefs.Options{
					GitRefs:          []prowapi.Refs{{}},
					GitUserEmail:     clonerefs.DefaultGitUserEmail,
					GitUserName:      clonerefs.DefaultGitUserName,
					SrcRoot:          codeMount.MountPath,
					HostFingerprints: []string{"thumb", "pinky"},
					Log:              CloneLogPath(logMount),
				}),
				VolumeMounts: []coreapi.VolumeMount{logMount, codeMount},
			},
		},
		{
			name: "include cookiefile secrets when set",
			pj: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					ExtraRefs: []prowapi.Refs{{}},
					DecorationConfig: &prowapi.DecorationConfig{
						UtilityImages:    &prowapi.UtilityImages{},
						CookiefileSecret: "oatmeal",
					},
				},
			},
			expected: &coreapi.Container{
				Name:    cloneRefsName,
				Command: []string{cloneRefsCommand},
				Args:    []string{"--cookiefile=" + cookiePathOnly("oatmeal")},
				Env: envOrDie(clonerefs.Options{
					CookiePath:   cookiePathOnly("oatmeal"),
					GitRefs:      []prowapi.Refs{{}},
					GitUserEmail: clonerefs.DefaultGitUserEmail,
					GitUserName:  clonerefs.DefaultGitUserName,
					SrcRoot:      codeMount.MountPath,
					Log:          CloneLogPath(logMount),
				}),
				VolumeMounts: []coreapi.VolumeMount{logMount, codeMount, cookieMountOnly("oatmeal")},
			},
			volumes: []coreapi.Volume{cookieVolumeOnly("oatmeal")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lm := logMount
			if tc.logMountOverride != nil {
				lm = *tc.logMountOverride
			}
			cm := codeMount
			if tc.codeMountOverride != nil {
				cm = *tc.codeMountOverride
			}
			actual, refs, volumes, err := CloneRefs(tc.pj, cm, lm)
			switch {
			case err != nil:
				if !tc.err {
					t.Errorf("unexpected error: %v", err)
				}
			case tc.err:
				t.Error("failed to receive expected exception")
			case !equality.Semantic.DeepEqual(tc.expected, actual):
				t.Errorf("unexpected container:\n%s", diff.ObjectReflectDiff(tc.expected, actual))
			case !equality.Semantic.DeepEqual(tc.volumes, volumes):
				t.Errorf("unexpected volume:\n%s", diff.ObjectReflectDiff(tc.volumes, volumes))
			case actual != nil:
				var er []prowapi.Refs
				if tc.pj.Spec.Refs != nil {
					er = append(er, *tc.pj.Spec.Refs)
				}
				for _, r := range tc.pj.Spec.ExtraRefs {
					er = append(er, r)
				}
				if !equality.Semantic.DeepEqual(refs, er) {
					t.Errorf("unexpected refs:\n%s", diff.ObjectReflectDiff(er, refs))
				}
			}
		})
	}
}

func TestProwJobToPod(t *testing.T) {
	truth := true
	falseth := false
	var sshKeyMode int32 = 0400
	tests := []struct {
		podName string
		buildID string
		labels  map[string]string
		pjSpec  prowapi.ProwJobSpec

		expected *coreapi.Pod
	}{
		{
			podName: "pod",
			buildID: "blabla",
			labels:  map[string]string{"needstobe": "inherited"},
			pjSpec: prowapi.ProwJobSpec{
				Type:  prowapi.PresubmitJob,
				Job:   "job-name",
				Agent: prowapi.KubernetesAgent,
				Refs: &prowapi.Refs{
					Org:     "org-name",
					Repo:    "repo-name",
					BaseRef: "base-ref",
					BaseSHA: "base-sha",
					Pulls: []prowapi.Pull{{
						Number: 1,
						Author: "author-name",
						SHA:    "pull-sha",
					}},
				},
				PodSpec: &coreapi.PodSpec{
					Containers: []coreapi.Container{
						{
							Image: "tester",
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
							},
						},
					},
				},
			},

			expected: &coreapi.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
					Labels: map[string]string{
						kube.CreatedByProw:     "true",
						kube.ProwJobTypeLabel:  "presubmit",
						kube.ProwJobIDLabel:    "pod",
						"needstobe":            "inherited",
						kube.OrgLabel:          "org-name",
						kube.RepoLabel:         "repo-name",
						kube.PullLabel:         "1",
						kube.ProwJobAnnotation: "job-name",
					},
					Annotations: map[string]string{
						kube.ProwJobAnnotation: "job-name",
					},
				},
				Spec: coreapi.PodSpec{
					AutomountServiceAccountToken: &falseth,
					RestartPolicy:                "Never",
					Containers: []coreapi.Container{
						{
							Name:  "test",
							Image: "tester",
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
								{Name: "BUILD_ID", Value: "blabla"},
								{Name: "BUILD_NUMBER", Value: "blabla"},
								{Name: "JOB_NAME", Value: "job-name"},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}]}}`},
								{Name: "JOB_TYPE", Value: "presubmit"},
								{Name: "PROW_JOB_ID", Value: "pod"},
								{Name: "PULL_BASE_REF", Value: "base-ref"},
								{Name: "PULL_BASE_SHA", Value: "base-sha"},
								{Name: "PULL_NUMBER", Value: "1"},
								{Name: "PULL_PULL_SHA", Value: "pull-sha"},
								{Name: "PULL_REFS", Value: "base-ref:base-sha,1:pull-sha"},
								{Name: "REPO_NAME", Value: "repo-name"},
								{Name: "REPO_OWNER", Value: "org-name"},
							},
						},
					},
				},
			},
		},
		{
			podName: "pod",
			buildID: "blabla",
			labels:  map[string]string{"needstobe": "inherited"},
			pjSpec: prowapi.ProwJobSpec{
				Type: prowapi.PresubmitJob,
				Job:  "job-name",
				DecorationConfig: &prowapi.DecorationConfig{
					Timeout:     &prowapi.Duration{Duration: 120 * time.Minute},
					GracePeriod: &prowapi.Duration{Duration: 10 * time.Second},
					UtilityImages: &prowapi.UtilityImages{
						CloneRefs:  "clonerefs:tag",
						InitUpload: "initupload:tag",
						Entrypoint: "entrypoint:tag",
						Sidecar:    "sidecar:tag",
					},
					GCSConfiguration: &prowapi.GCSConfiguration{
						Bucket:       "my-bucket",
						PathStrategy: "legacy",
						DefaultOrg:   "kubernetes",
						DefaultRepo:  "kubernetes",
						MediaTypes:   map[string]string{"log": "text/plain"},
					},
					GCSCredentialsSecret: "secret-name",
					CookiefileSecret:     "yummy/.gitcookies",
				},
				Agent: prowapi.KubernetesAgent,
				Refs: &prowapi.Refs{
					Org:     "org-name",
					Repo:    "repo-name",
					BaseRef: "base-ref",
					BaseSHA: "base-sha",
					Pulls: []prowapi.Pull{{
						Number: 1,
						Author: "author-name",
						SHA:    "pull-sha",
					}},
					PathAlias: "somewhere/else",
				},
				ExtraRefs: []prowapi.Refs{},
				PodSpec: &coreapi.PodSpec{
					Containers: []coreapi.Container{
						{
							Image:   "tester",
							Command: []string{"/bin/thing"},
							Args:    []string{"some", "args"},
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
							},
						},
					},
				},
			},
			expected: &coreapi.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
					Labels: map[string]string{
						kube.CreatedByProw:     "true",
						kube.ProwJobTypeLabel:  "presubmit",
						kube.ProwJobIDLabel:    "pod",
						"needstobe":            "inherited",
						kube.OrgLabel:          "org-name",
						kube.RepoLabel:         "repo-name",
						kube.PullLabel:         "1",
						kube.ProwJobAnnotation: "job-name",
					},
					Annotations: map[string]string{
						kube.ProwJobAnnotation: "job-name",
					},
				},
				Spec: coreapi.PodSpec{
					AutomountServiceAccountToken: &falseth,
					RestartPolicy:                "Never",
					InitContainers: []coreapi.Container{
						{
							Name:    "clonerefs",
							Image:   "clonerefs:tag",
							Command: []string{"/clonerefs"},
							Args:    []string{"--cookiefile=" + cookiePathOnly("yummy/.gitcookies")},
							Env: []coreapi.EnvVar{
								{Name: "CLONEREFS_OPTIONS", Value: `{"src_root":"/home/prow/go","log":"/logs/clone.json","git_user_name":"ci-robot","git_user_email":"ci-robot@k8s.io","refs":[{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}],"cookie_path":"` + cookiePathOnly("yummy/.gitcookies") + `"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "code",
									MountPath: "/home/prow/go",
								},
								cookieMountOnly("yummy/.gitcookies"),
							},
						},
						{
							Name:    "initupload",
							Image:   "initupload:tag",
							Command: []string{"/initupload"},
							Env: []coreapi.EnvVar{
								{Name: "INITUPLOAD_OPTIONS", Value: `{"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","mediaTypes":{"log":"text/plain"},"gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false,"log":"/logs/clone.json"}`},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
						{
							Name:    "place-entrypoint",
							Image:   "entrypoint:tag",
							Command: []string{"/bin/cp"},
							Args: []string{
								"/entrypoint",
								"/tools/entrypoint",
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "tools",
									MountPath: "/tools",
								},
							},
						},
					},
					Containers: []coreapi.Container{
						{
							Name:       "test",
							Image:      "tester",
							Command:    []string{"/tools/entrypoint"},
							Args:       []string{},
							WorkingDir: "/home/prow/go/src/somewhere/else",
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
								{Name: "ARTIFACTS", Value: "/logs/artifacts"},
								{Name: "BUILD_ID", Value: "blabla"},
								{Name: "BUILD_NUMBER", Value: "blabla"},
								{Name: "GOPATH", Value: "/home/prow/go"},
								{Name: "JOB_NAME", Value: "job-name"},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
								{Name: "JOB_TYPE", Value: "presubmit"},
								{Name: "PROW_JOB_ID", Value: "pod"},
								{Name: "PULL_BASE_REF", Value: "base-ref"},
								{Name: "PULL_BASE_SHA", Value: "base-sha"},
								{Name: "PULL_NUMBER", Value: "1"},
								{Name: "PULL_PULL_SHA", Value: "pull-sha"},
								{Name: "PULL_REFS", Value: "base-ref:base-sha,1:pull-sha"},
								{Name: "REPO_NAME", Value: "repo-name"},
								{Name: "REPO_OWNER", Value: "org-name"},
								{Name: "ENTRYPOINT_OPTIONS", Value: `{"timeout":7200000000000,"grace_period":10000000000,"artifact_dir":"/logs/artifacts","args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "tools",
									MountPath: "/tools",
								},
								{
									Name:      "code",
									MountPath: "/home/prow/go",
								},
							},
						},
						{
							Name:    "sidecar",
							Image:   "sidecar:tag",
							Command: []string{"/sidecar"},
							Env: []coreapi.EnvVar{
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
								{Name: "SIDECAR_OPTIONS", Value: `{"gcs_options":{"items":["/logs/artifacts"],"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","mediaTypes":{"log":"text/plain"},"gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false},"entries":[{"args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
					},
					Volumes: []coreapi.Volume{
						{
							Name: "logs",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "tools",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "gcs-credentials",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName: "secret-name",
								},
							},
						},
						cookieVolumeOnly("yummy/.gitcookies"),
						{
							Name: "code",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
		{
			podName: "pod",
			buildID: "blabla",
			labels:  map[string]string{"needstobe": "inherited"},
			pjSpec: prowapi.ProwJobSpec{
				Type: prowapi.PresubmitJob,
				Job:  "job-name",
				DecorationConfig: &prowapi.DecorationConfig{
					Timeout:     &prowapi.Duration{Duration: 120 * time.Minute},
					GracePeriod: &prowapi.Duration{Duration: 10 * time.Second},
					UtilityImages: &prowapi.UtilityImages{
						CloneRefs:  "clonerefs:tag",
						InitUpload: "initupload:tag",
						Entrypoint: "entrypoint:tag",
						Sidecar:    "sidecar:tag",
					},
					GCSConfiguration: &prowapi.GCSConfiguration{
						Bucket:       "my-bucket",
						PathStrategy: "legacy",
						DefaultOrg:   "kubernetes",
						DefaultRepo:  "kubernetes",
					},
					GCSCredentialsSecret: "secret-name",
					CookiefileSecret:     "yummy",
				},
				Agent: prowapi.KubernetesAgent,
				Refs: &prowapi.Refs{
					Org:     "org-name",
					Repo:    "repo-name",
					BaseRef: "base-ref",
					BaseSHA: "base-sha",
					Pulls: []prowapi.Pull{{
						Number: 1,
						Author: "author-name",
						SHA:    "pull-sha",
					}},
					PathAlias: "somewhere/else",
				},
				ExtraRefs: []prowapi.Refs{},
				PodSpec: &coreapi.PodSpec{
					Containers: []coreapi.Container{
						{
							Image:   "tester",
							Command: []string{"/bin/thing"},
							Args:    []string{"some", "args"},
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
							},
						},
					},
				},
			},
			expected: &coreapi.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
					Labels: map[string]string{
						kube.CreatedByProw:     "true",
						kube.ProwJobTypeLabel:  "presubmit",
						kube.ProwJobIDLabel:    "pod",
						"needstobe":            "inherited",
						kube.OrgLabel:          "org-name",
						kube.RepoLabel:         "repo-name",
						kube.PullLabel:         "1",
						kube.ProwJobAnnotation: "job-name",
					},
					Annotations: map[string]string{
						kube.ProwJobAnnotation: "job-name",
					},
				},
				Spec: coreapi.PodSpec{
					AutomountServiceAccountToken: &falseth,
					RestartPolicy:                "Never",
					InitContainers: []coreapi.Container{
						{
							Name:    "clonerefs",
							Image:   "clonerefs:tag",
							Command: []string{"/clonerefs"},
							Args:    []string{"--cookiefile=" + cookiePathOnly("yummy")},
							Env: []coreapi.EnvVar{
								{Name: "CLONEREFS_OPTIONS", Value: `{"src_root":"/home/prow/go","log":"/logs/clone.json","git_user_name":"ci-robot","git_user_email":"ci-robot@k8s.io","refs":[{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}],"cookie_path":"` + cookiePathOnly("yummy") + `"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "code",
									MountPath: "/home/prow/go",
								},
								cookieMountOnly("yummy"),
							},
						},
						{
							Name:    "initupload",
							Image:   "initupload:tag",
							Command: []string{"/initupload"},
							Env: []coreapi.EnvVar{
								{Name: "INITUPLOAD_OPTIONS", Value: `{"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false,"log":"/logs/clone.json"}`},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
						{
							Name:    "place-entrypoint",
							Image:   "entrypoint:tag",
							Command: []string{"/bin/cp"},
							Args: []string{
								"/entrypoint",
								"/tools/entrypoint",
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "tools",
									MountPath: "/tools",
								},
							},
						},
					},
					Containers: []coreapi.Container{
						{
							Name:       "test",
							Image:      "tester",
							Command:    []string{"/tools/entrypoint"},
							Args:       []string{},
							WorkingDir: "/home/prow/go/src/somewhere/else",
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
								{Name: "ARTIFACTS", Value: "/logs/artifacts"},
								{Name: "BUILD_ID", Value: "blabla"},
								{Name: "BUILD_NUMBER", Value: "blabla"},
								{Name: "GOPATH", Value: "/home/prow/go"},
								{Name: "JOB_NAME", Value: "job-name"},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
								{Name: "JOB_TYPE", Value: "presubmit"},
								{Name: "PROW_JOB_ID", Value: "pod"},
								{Name: "PULL_BASE_REF", Value: "base-ref"},
								{Name: "PULL_BASE_SHA", Value: "base-sha"},
								{Name: "PULL_NUMBER", Value: "1"},
								{Name: "PULL_PULL_SHA", Value: "pull-sha"},
								{Name: "PULL_REFS", Value: "base-ref:base-sha,1:pull-sha"},
								{Name: "REPO_NAME", Value: "repo-name"},
								{Name: "REPO_OWNER", Value: "org-name"},
								{Name: "ENTRYPOINT_OPTIONS", Value: `{"timeout":7200000000000,"grace_period":10000000000,"artifact_dir":"/logs/artifacts","args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "tools",
									MountPath: "/tools",
								},
								{
									Name:      "code",
									MountPath: "/home/prow/go",
								},
							},
						},
						{
							Name:    "sidecar",
							Image:   "sidecar:tag",
							Command: []string{"/sidecar"},
							Env: []coreapi.EnvVar{
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
								{Name: "SIDECAR_OPTIONS", Value: `{"gcs_options":{"items":["/logs/artifacts"],"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false},"entries":[{"args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
					},
					Volumes: []coreapi.Volume{
						{
							Name: "logs",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "tools",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "gcs-credentials",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName: "secret-name",
								},
							},
						},
						cookieVolumeOnly("yummy"),
						{
							Name: "code",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
		{
			podName: "pod",
			buildID: "blabla",
			labels:  map[string]string{"needstobe": "inherited"},
			pjSpec: prowapi.ProwJobSpec{
				Type: prowapi.PresubmitJob,
				Job:  "job-name",
				DecorationConfig: &prowapi.DecorationConfig{
					Timeout:     &prowapi.Duration{Duration: 120 * time.Minute},
					GracePeriod: &prowapi.Duration{Duration: 10 * time.Second},
					UtilityImages: &prowapi.UtilityImages{
						CloneRefs:  "clonerefs:tag",
						InitUpload: "initupload:tag",
						Entrypoint: "entrypoint:tag",
						Sidecar:    "sidecar:tag",
					},
					GCSConfiguration: &prowapi.GCSConfiguration{
						Bucket:       "my-bucket",
						PathStrategy: "legacy",
						DefaultOrg:   "kubernetes",
						DefaultRepo:  "kubernetes",
					},
					GCSCredentialsSecret: "secret-name",
					SSHKeySecrets:        []string{"ssh-1", "ssh-2"},
					SSHHostFingerprints:  []string{"hello", "world"},
				},
				Agent: prowapi.KubernetesAgent,
				Refs: &prowapi.Refs{
					Org:     "org-name",
					Repo:    "repo-name",
					BaseRef: "base-ref",
					BaseSHA: "base-sha",
					Pulls: []prowapi.Pull{{
						Number: 1,
						Author: "author-name",
						SHA:    "pull-sha",
					}},
					PathAlias: "somewhere/else",
				},
				ExtraRefs: []prowapi.Refs{},
				PodSpec: &coreapi.PodSpec{
					Containers: []coreapi.Container{
						{
							Image:   "tester",
							Command: []string{"/bin/thing"},
							Args:    []string{"some", "args"},
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
							},
						},
					},
				},
			},
			expected: &coreapi.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
					Labels: map[string]string{
						kube.CreatedByProw:     "true",
						kube.ProwJobTypeLabel:  "presubmit",
						kube.ProwJobIDLabel:    "pod",
						"needstobe":            "inherited",
						kube.OrgLabel:          "org-name",
						kube.RepoLabel:         "repo-name",
						kube.PullLabel:         "1",
						kube.ProwJobAnnotation: "job-name",
					},
					Annotations: map[string]string{
						kube.ProwJobAnnotation: "job-name",
					},
				},
				Spec: coreapi.PodSpec{
					AutomountServiceAccountToken: &falseth,
					RestartPolicy:                "Never",
					InitContainers: []coreapi.Container{
						{
							Name:    "clonerefs",
							Image:   "clonerefs:tag",
							Command: []string{"/clonerefs"},
							Env: []coreapi.EnvVar{
								{Name: "CLONEREFS_OPTIONS", Value: `{"src_root":"/home/prow/go","log":"/logs/clone.json","git_user_name":"ci-robot","git_user_email":"ci-robot@k8s.io","refs":[{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}],"key_files":["/secrets/ssh/ssh-1","/secrets/ssh/ssh-2"],"host_fingerprints":["hello","world"]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "code",
									MountPath: "/home/prow/go",
								},
								{
									Name:      "ssh-keys-ssh-1",
									MountPath: "/secrets/ssh/ssh-1",
									ReadOnly:  true,
								},
								{
									Name:      "ssh-keys-ssh-2",
									MountPath: "/secrets/ssh/ssh-2",
									ReadOnly:  true,
								},
							},
						},
						{
							Name:    "initupload",
							Image:   "initupload:tag",
							Command: []string{"/initupload"},
							Env: []coreapi.EnvVar{
								{Name: "INITUPLOAD_OPTIONS", Value: `{"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false,"log":"/logs/clone.json"}`},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
						{
							Name:    "place-entrypoint",
							Image:   "entrypoint:tag",
							Command: []string{"/bin/cp"},
							Args: []string{
								"/entrypoint",
								"/tools/entrypoint",
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "tools",
									MountPath: "/tools",
								},
							},
						},
					},
					Containers: []coreapi.Container{
						{
							Name:       "test",
							Image:      "tester",
							Command:    []string{"/tools/entrypoint"},
							Args:       []string{},
							WorkingDir: "/home/prow/go/src/somewhere/else",
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
								{Name: "ARTIFACTS", Value: "/logs/artifacts"},
								{Name: "BUILD_ID", Value: "blabla"},
								{Name: "BUILD_NUMBER", Value: "blabla"},
								{Name: "GOPATH", Value: "/home/prow/go"},
								{Name: "JOB_NAME", Value: "job-name"},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
								{Name: "JOB_TYPE", Value: "presubmit"},
								{Name: "PROW_JOB_ID", Value: "pod"},
								{Name: "PULL_BASE_REF", Value: "base-ref"},
								{Name: "PULL_BASE_SHA", Value: "base-sha"},
								{Name: "PULL_NUMBER", Value: "1"},
								{Name: "PULL_PULL_SHA", Value: "pull-sha"},
								{Name: "PULL_REFS", Value: "base-ref:base-sha,1:pull-sha"},
								{Name: "REPO_NAME", Value: "repo-name"},
								{Name: "REPO_OWNER", Value: "org-name"},
								{Name: "ENTRYPOINT_OPTIONS", Value: `{"timeout":7200000000000,"grace_period":10000000000,"artifact_dir":"/logs/artifacts","args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "tools",
									MountPath: "/tools",
								},
								{
									Name:      "code",
									MountPath: "/home/prow/go",
								},
							},
						},
						{
							Name:    "sidecar",
							Image:   "sidecar:tag",
							Command: []string{"/sidecar"},
							Env: []coreapi.EnvVar{
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
								{Name: "SIDECAR_OPTIONS", Value: `{"gcs_options":{"items":["/logs/artifacts"],"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false},"entries":[{"args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
					},
					Volumes: []coreapi.Volume{
						{
							Name: "logs",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "tools",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "gcs-credentials",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName: "secret-name",
								},
							},
						},
						{
							Name: "ssh-keys-ssh-1",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName:  "ssh-1",
									DefaultMode: &sshKeyMode,
								},
							},
						},
						{
							Name: "ssh-keys-ssh-2",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName:  "ssh-2",
									DefaultMode: &sshKeyMode,
								},
							},
						},
						{
							Name: "code",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
		{
			podName: "pod",
			buildID: "blabla",
			labels:  map[string]string{"needstobe": "inherited"},
			pjSpec: prowapi.ProwJobSpec{
				Type: prowapi.PresubmitJob,
				Job:  "job-name",
				DecorationConfig: &prowapi.DecorationConfig{
					Timeout:     &prowapi.Duration{Duration: 120 * time.Minute},
					GracePeriod: &prowapi.Duration{Duration: 10 * time.Second},
					UtilityImages: &prowapi.UtilityImages{
						CloneRefs:  "clonerefs:tag",
						InitUpload: "initupload:tag",
						Entrypoint: "entrypoint:tag",
						Sidecar:    "sidecar:tag",
					},
					GCSConfiguration: &prowapi.GCSConfiguration{
						Bucket:       "my-bucket",
						PathStrategy: "legacy",
						DefaultOrg:   "kubernetes",
						DefaultRepo:  "kubernetes",
					},
					GCSCredentialsSecret: "secret-name",
					SSHKeySecrets:        []string{"ssh-1", "ssh-2"},
				},
				Agent: prowapi.KubernetesAgent,
				Refs: &prowapi.Refs{
					Org:     "org-name",
					Repo:    "repo-name",
					BaseRef: "base-ref",
					BaseSHA: "base-sha",
					Pulls: []prowapi.Pull{{
						Number: 1,
						Author: "author-name",
						SHA:    "pull-sha",
					}},
					PathAlias: "somewhere/else",
				},
				ExtraRefs: []prowapi.Refs{},
				PodSpec: &coreapi.PodSpec{
					Containers: []coreapi.Container{
						{
							Image:   "tester",
							Command: []string{"/bin/thing"},
							Args:    []string{"some", "args"},
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
							},
						},
					},
				},
			},
			expected: &coreapi.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
					Labels: map[string]string{
						kube.CreatedByProw:     "true",
						kube.ProwJobTypeLabel:  "presubmit",
						kube.ProwJobIDLabel:    "pod",
						"needstobe":            "inherited",
						kube.OrgLabel:          "org-name",
						kube.RepoLabel:         "repo-name",
						kube.PullLabel:         "1",
						kube.ProwJobAnnotation: "job-name",
					},
					Annotations: map[string]string{
						kube.ProwJobAnnotation: "job-name",
					},
				},
				Spec: coreapi.PodSpec{
					AutomountServiceAccountToken: &falseth,
					RestartPolicy:                "Never",
					InitContainers: []coreapi.Container{
						{
							Name:    "clonerefs",
							Image:   "clonerefs:tag",
							Command: []string{"/clonerefs"},
							Env: []coreapi.EnvVar{
								{Name: "CLONEREFS_OPTIONS", Value: `{"src_root":"/home/prow/go","log":"/logs/clone.json","git_user_name":"ci-robot","git_user_email":"ci-robot@k8s.io","refs":[{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}],"key_files":["/secrets/ssh/ssh-1","/secrets/ssh/ssh-2"]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "code",
									MountPath: "/home/prow/go",
								},
								{
									Name:      "ssh-keys-ssh-1",
									MountPath: "/secrets/ssh/ssh-1",
									ReadOnly:  true,
								},
								{
									Name:      "ssh-keys-ssh-2",
									MountPath: "/secrets/ssh/ssh-2",
									ReadOnly:  true,
								},
							},
						},
						{
							Name:    "initupload",
							Image:   "initupload:tag",
							Command: []string{"/initupload"},
							Env: []coreapi.EnvVar{
								{Name: "INITUPLOAD_OPTIONS", Value: `{"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false,"log":"/logs/clone.json"}`},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
						{
							Name:    "place-entrypoint",
							Image:   "entrypoint:tag",
							Command: []string{"/bin/cp"},
							Args: []string{
								"/entrypoint",
								"/tools/entrypoint",
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "tools",
									MountPath: "/tools",
								},
							},
						},
					},
					Containers: []coreapi.Container{
						{
							Name:       "test",
							Image:      "tester",
							Command:    []string{"/tools/entrypoint"},
							Args:       []string{},
							WorkingDir: "/home/prow/go/src/somewhere/else",
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
								{Name: "ARTIFACTS", Value: "/logs/artifacts"},
								{Name: "BUILD_ID", Value: "blabla"},
								{Name: "BUILD_NUMBER", Value: "blabla"},
								{Name: "GOPATH", Value: "/home/prow/go"},
								{Name: "JOB_NAME", Value: "job-name"},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
								{Name: "JOB_TYPE", Value: "presubmit"},
								{Name: "PROW_JOB_ID", Value: "pod"},
								{Name: "PULL_BASE_REF", Value: "base-ref"},
								{Name: "PULL_BASE_SHA", Value: "base-sha"},
								{Name: "PULL_NUMBER", Value: "1"},
								{Name: "PULL_PULL_SHA", Value: "pull-sha"},
								{Name: "PULL_REFS", Value: "base-ref:base-sha,1:pull-sha"},
								{Name: "REPO_NAME", Value: "repo-name"},
								{Name: "REPO_OWNER", Value: "org-name"},
								{Name: "ENTRYPOINT_OPTIONS", Value: `{"timeout":7200000000000,"grace_period":10000000000,"artifact_dir":"/logs/artifacts","args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "tools",
									MountPath: "/tools",
								},
								{
									Name:      "code",
									MountPath: "/home/prow/go",
								},
							},
						},
						{
							Name:    "sidecar",
							Image:   "sidecar:tag",
							Command: []string{"/sidecar"},
							Env: []coreapi.EnvVar{
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"}}`},
								{Name: "SIDECAR_OPTIONS", Value: `{"gcs_options":{"items":["/logs/artifacts"],"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false},"entries":[{"args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
					},
					Volumes: []coreapi.Volume{
						{
							Name: "logs",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "tools",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "gcs-credentials",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName: "secret-name",
								},
							},
						},
						{
							Name: "ssh-keys-ssh-1",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName:  "ssh-1",
									DefaultMode: &sshKeyMode,
								},
							},
						},
						{
							Name: "ssh-keys-ssh-2",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName:  "ssh-2",
									DefaultMode: &sshKeyMode,
								},
							},
						},
						{
							Name: "code",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
		{
			podName: "pod",
			buildID: "blabla",
			labels:  map[string]string{"needstobe": "inherited"},
			pjSpec: prowapi.ProwJobSpec{
				Type: prowapi.PeriodicJob,
				Job:  "job-name",
				DecorationConfig: &prowapi.DecorationConfig{
					Timeout:     &prowapi.Duration{Duration: 120 * time.Minute},
					GracePeriod: &prowapi.Duration{Duration: 10 * time.Second},
					UtilityImages: &prowapi.UtilityImages{
						CloneRefs:  "clonerefs:tag",
						InitUpload: "initupload:tag",
						Entrypoint: "entrypoint:tag",
						Sidecar:    "sidecar:tag",
					},
					GCSConfiguration: &prowapi.GCSConfiguration{
						Bucket:       "my-bucket",
						PathStrategy: "legacy",
						DefaultOrg:   "kubernetes",
						DefaultRepo:  "kubernetes",
					},
					GCSCredentialsSecret: "secret-name",
					SSHKeySecrets:        []string{"ssh-1", "ssh-2"},
				},
				Agent: prowapi.KubernetesAgent,
				PodSpec: &coreapi.PodSpec{
					Containers: []coreapi.Container{
						{
							Image:   "tester",
							Command: []string{"/bin/thing"},
							Args:    []string{"some", "args"},
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
							},
						},
					},
				},
			},
			expected: &coreapi.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
					Labels: map[string]string{
						kube.CreatedByProw:     "true",
						kube.ProwJobTypeLabel:  "periodic",
						kube.ProwJobIDLabel:    "pod",
						"needstobe":            "inherited",
						kube.ProwJobAnnotation: "job-name",
					},
					Annotations: map[string]string{
						kube.ProwJobAnnotation: "job-name",
					},
				},
				Spec: coreapi.PodSpec{
					AutomountServiceAccountToken: &falseth,
					RestartPolicy:                "Never",
					InitContainers: []coreapi.Container{
						{
							Name:    "initupload",
							Image:   "initupload:tag",
							Command: []string{"/initupload"},
							Env: []coreapi.EnvVar{
								{Name: "INITUPLOAD_OPTIONS", Value: `{"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false}`},
								{Name: "JOB_SPEC", Value: `{"type":"periodic","job":"job-name","buildid":"blabla","prowjobid":"pod"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								// don't mount log since we're not uploading a clone log
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
						{
							Name:    "place-entrypoint",
							Image:   "entrypoint:tag",
							Command: []string{"/bin/cp"},
							Args: []string{
								"/entrypoint",
								"/tools/entrypoint",
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "tools",
									MountPath: "/tools",
								},
							},
						},
					},
					Containers: []coreapi.Container{
						{
							Name:    "test",
							Image:   "tester",
							Command: []string{"/tools/entrypoint"},
							Args:    []string{},
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
								{Name: "ARTIFACTS", Value: "/logs/artifacts"},
								{Name: "BUILD_ID", Value: "blabla"},
								{Name: "BUILD_NUMBER", Value: "blabla"},
								{Name: "GOPATH", Value: "/home/prow/go"},
								{Name: "JOB_NAME", Value: "job-name"},
								{Name: "JOB_SPEC", Value: `{"type":"periodic","job":"job-name","buildid":"blabla","prowjobid":"pod"}`},
								{Name: "JOB_TYPE", Value: "periodic"},
								{Name: "PROW_JOB_ID", Value: "pod"},
								{Name: "ENTRYPOINT_OPTIONS", Value: `{"timeout":7200000000000,"grace_period":10000000000,"artifact_dir":"/logs/artifacts","args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "tools",
									MountPath: "/tools",
								},
							},
						},
						{
							Name:    "sidecar",
							Image:   "sidecar:tag",
							Command: []string{"/sidecar"},
							Env: []coreapi.EnvVar{
								{Name: "JOB_SPEC", Value: `{"type":"periodic","job":"job-name","buildid":"blabla","prowjobid":"pod"}`},
								{Name: "SIDECAR_OPTIONS", Value: `{"gcs_options":{"items":["/logs/artifacts"],"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false},"entries":[{"args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
					},
					Volumes: []coreapi.Volume{
						{
							Name: "logs",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "tools",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "gcs-credentials",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName: "secret-name",
								},
							},
						},
					},
				},
			},
		},
		{
			podName: "pod",
			buildID: "blabla",
			labels:  map[string]string{"needstobe": "inherited"},
			pjSpec: prowapi.ProwJobSpec{
				Type: prowapi.PresubmitJob,
				Job:  "job-name",
				DecorationConfig: &prowapi.DecorationConfig{
					Timeout:     &prowapi.Duration{Duration: 120 * time.Minute},
					GracePeriod: &prowapi.Duration{Duration: 10 * time.Second},
					UtilityImages: &prowapi.UtilityImages{
						CloneRefs:  "clonerefs:tag",
						InitUpload: "initupload:tag",
						Entrypoint: "entrypoint:tag",
						Sidecar:    "sidecar:tag",
					},
					GCSConfiguration: &prowapi.GCSConfiguration{
						Bucket:       "my-bucket",
						PathStrategy: "legacy",
						DefaultOrg:   "kubernetes",
						DefaultRepo:  "kubernetes",
					},
					GCSCredentialsSecret: "secret-name",
					SSHKeySecrets:        []string{"ssh-1", "ssh-2"},
					SkipCloning:          &truth,
				},
				Agent: prowapi.KubernetesAgent,
				Refs: &prowapi.Refs{
					Org:     "org-name",
					Repo:    "repo-name",
					BaseRef: "base-ref",
					BaseSHA: "base-sha",
					Pulls: []prowapi.Pull{{
						Number: 1,
						Author: "author-name",
						SHA:    "pull-sha",
					}},
					PathAlias: "somewhere/else",
				},
				ExtraRefs: []prowapi.Refs{
					{
						Org:  "extra-org",
						Repo: "extra-repo",
					},
				},
				PodSpec: &coreapi.PodSpec{
					Containers: []coreapi.Container{
						{
							Image:   "tester",
							Command: []string{"/bin/thing"},
							Args:    []string{"some", "args"},
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
							},
						},
					},
				},
			},
			expected: &coreapi.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
					Labels: map[string]string{
						kube.CreatedByProw:     "true",
						kube.ProwJobTypeLabel:  "presubmit",
						kube.ProwJobIDLabel:    "pod",
						"needstobe":            "inherited",
						kube.OrgLabel:          "org-name",
						kube.RepoLabel:         "repo-name",
						kube.PullLabel:         "1",
						kube.ProwJobAnnotation: "job-name",
					},
					Annotations: map[string]string{
						kube.ProwJobAnnotation: "job-name",
					},
				},
				Spec: coreapi.PodSpec{
					AutomountServiceAccountToken: &falseth,
					RestartPolicy:                "Never",
					InitContainers: []coreapi.Container{
						{
							Name:    "initupload",
							Image:   "initupload:tag",
							Command: []string{"/initupload"},
							Env: []coreapi.EnvVar{
								{Name: "INITUPLOAD_OPTIONS", Value: `{"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false}`},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"},"extra_refs":[{"org":"extra-org","repo":"extra-repo"}]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								// don't mount log since we're not uploading a clone log
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
						{
							Name:    "place-entrypoint",
							Image:   "entrypoint:tag",
							Command: []string{"/bin/cp"},
							Args: []string{
								"/entrypoint",
								"/tools/entrypoint",
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "tools",
									MountPath: "/tools",
								},
							},
						},
					},
					Containers: []coreapi.Container{
						{
							Name:    "test",
							Image:   "tester",
							Command: []string{"/tools/entrypoint"},
							Args:    []string{},
							Env: []coreapi.EnvVar{
								{Name: "MY_ENV", Value: "rocks"},
								{Name: "ARTIFACTS", Value: "/logs/artifacts"},
								{Name: "BUILD_ID", Value: "blabla"},
								{Name: "BUILD_NUMBER", Value: "blabla"},
								{Name: "GOPATH", Value: "/home/prow/go"},
								{Name: "JOB_NAME", Value: "job-name"},
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"},"extra_refs":[{"org":"extra-org","repo":"extra-repo"}]}`},
								{Name: "JOB_TYPE", Value: "presubmit"},
								{Name: "PROW_JOB_ID", Value: "pod"},
								{Name: "PULL_BASE_REF", Value: "base-ref"},
								{Name: "PULL_BASE_SHA", Value: "base-sha"},
								{Name: "PULL_NUMBER", Value: "1"},
								{Name: "PULL_PULL_SHA", Value: "pull-sha"},
								{Name: "PULL_REFS", Value: "base-ref:base-sha,1:pull-sha"},
								{Name: "REPO_NAME", Value: "repo-name"},
								{Name: "REPO_OWNER", Value: "org-name"},
								{Name: "ENTRYPOINT_OPTIONS", Value: `{"timeout":7200000000000,"grace_period":10000000000,"artifact_dir":"/logs/artifacts","args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "tools",
									MountPath: "/tools",
								},
							},
						},
						{
							Name:    "sidecar",
							Image:   "sidecar:tag",
							Command: []string{"/sidecar"},
							Env: []coreapi.EnvVar{
								{Name: "JOB_SPEC", Value: `{"type":"presubmit","job":"job-name","buildid":"blabla","prowjobid":"pod","refs":{"org":"org-name","repo":"repo-name","base_ref":"base-ref","base_sha":"base-sha","pulls":[{"number":1,"author":"author-name","sha":"pull-sha"}],"path_alias":"somewhere/else"},"extra_refs":[{"org":"extra-org","repo":"extra-repo"}]}`},
								{Name: "SIDECAR_OPTIONS", Value: `{"gcs_options":{"items":["/logs/artifacts"],"bucket":"my-bucket","path_strategy":"legacy","default_org":"kubernetes","default_repo":"kubernetes","gcs_credentials_file":"/secrets/gcs/service-account.json","dry_run":false},"entries":[{"args":["/bin/thing","some","args"],"process_log":"/logs/process-log.txt","marker_file":"/logs/marker-file.txt","metadata_file":"/logs/artifacts/metadata.json"}]}`},
							},
							VolumeMounts: []coreapi.VolumeMount{
								{
									Name:      "logs",
									MountPath: "/logs",
								},
								{
									Name:      "gcs-credentials",
									MountPath: "/secrets/gcs",
								},
							},
						},
					},
					Volumes: []coreapi.Volume{
						{
							Name: "logs",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "tools",
							VolumeSource: coreapi.VolumeSource{
								EmptyDir: &coreapi.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "gcs-credentials",
							VolumeSource: coreapi.VolumeSource{
								Secret: &coreapi.SecretVolumeSource{
									SecretName: "secret-name",
								},
							},
						},
					},
				},
			},
		},
	}

	findContainer := func(name string, pod coreapi.Pod) *coreapi.Container {
		for _, c := range pod.Spec.Containers {
			if c.Name == name {
				return &c
			}
		}
		return nil
	}
	findEnv := func(key string, container coreapi.Container) *string {
		for _, env := range container.Env {
			if env.Name == key {
				v := env.Value
				return &v
			}

		}
		return nil
	}

	type checker interface {
		ConfigVar() string
		LoadConfig(string) error
		Validate() error
	}

	checkEnv := func(pod coreapi.Pod, name string, opt checker) error {
		c := findContainer(name, pod)
		if c == nil {
			return nil
		}
		env := opt.ConfigVar()
		val := findEnv(env, *c)
		if val == nil {
			return fmt.Errorf("missing %s env var", env)
		}
		if err := opt.LoadConfig(*val); err != nil {
			return fmt.Errorf("load: %v", err)
		}
		if err := opt.Validate(); err != nil {
			return fmt.Errorf("validate: %v", err)
		}
		return nil
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			pj := prowapi.ProwJob{ObjectMeta: metav1.ObjectMeta{Name: test.podName, Labels: test.labels}, Spec: test.pjSpec}
			got, err := ProwJobToPod(pj, test.buildID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !equality.Semantic.DeepEqual(got, test.expected) {
				t.Errorf("unexpected pod diff:\n%s", diff.ObjectReflectDiff(test.expected, got))
			}
			if err := checkEnv(*got, "sidecar", sidecar.NewOptions()); err != nil {
				t.Errorf("bad sidecar env: %v", err)
			}
			if err := checkEnv(*got, "initupload", initupload.NewOptions()); err != nil {
				t.Errorf("bad clonerefs env: %v", err)
			}
			if err := checkEnv(*got, "clonerefs", &clonerefs.Options{}); err != nil {
				t.Errorf("bad clonerefs env: %v", err)
			}
			if test.pjSpec.DecorationConfig != nil { // all jobs get a test container
				// But only decorated jobs need valid entrypoint options
				if err := checkEnv(*got, "test", entrypoint.NewOptions()); err != nil {
					t.Errorf("bad test entrypoint: %v", err)
				}
			}
		})
	}
}
