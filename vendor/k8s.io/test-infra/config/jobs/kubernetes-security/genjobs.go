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

/*
genjobs automatically generates the security repo presubmits from the
kubernetes presubmits

NOTE: this makes a few assumptions
- $PWD/../../prow/config.yaml is where the config lives (unless you supply --config=)
- $PWD/.. is where the job configs live (unless you supply --jobs=)
- the output is job configs ($PWD/..) + /kubernetes-security/generated-security-jobs.yaml (unless you supply --output)
*/
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	flag "github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	coreapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
)

var configPath = flag.String("config", "", "path to prow/config.yaml, defaults to $PWD/../../prow/config.yaml")
var jobsPath = flag.String("jobs", "", "path to prowjobs, defaults to $PWD/../")
var outputPath = flag.String("output", "", "path to output the generated jobs to, defaults to $PWD/generated-security-jobs.yaml")

// remove merged presets from a podspec
func undoPreset(preset *config.Preset, labels map[string]string, pod *coreapi.PodSpec) {
	// skip presets that do not match the job labels
	for l, v := range preset.Labels {
		if v2, ok := labels[l]; !ok || v2 != v {
			return
		}
	}

	// collect up preset created keys
	removeEnvNames := sets.NewString()
	for _, e1 := range preset.Env {
		removeEnvNames.Insert(e1.Name)
	}
	removeVolumeNames := sets.NewString()
	for _, volume := range preset.Volumes {
		removeVolumeNames.Insert(volume.Name)
	}
	removeVolumeMountNames := sets.NewString()
	for _, volumeMount := range preset.VolumeMounts {
		removeVolumeMountNames.Insert(volumeMount.Name)
	}

	// remove volumes from spec
	filteredVolumes := []coreapi.Volume{}
	for _, volume := range pod.Volumes {
		if !removeVolumeNames.Has(volume.Name) {
			filteredVolumes = append(filteredVolumes, volume)
		}
	}
	pod.Volumes = filteredVolumes

	// remove env and volume mounts from containers
	for i := range pod.Containers {
		filteredEnv := []coreapi.EnvVar{}
		for _, env := range pod.Containers[i].Env {
			if !removeEnvNames.Has(env.Name) {
				filteredEnv = append(filteredEnv, env)
			}
		}
		pod.Containers[i].Env = filteredEnv

		filteredVolumeMounts := []coreapi.VolumeMount{}
		for _, mount := range pod.Containers[i].VolumeMounts {
			if !removeVolumeMountNames.Has(mount.Name) {
				filteredVolumeMounts = append(filteredVolumeMounts, mount)
			}
		}
		pod.Containers[i].VolumeMounts = filteredVolumeMounts
	}
}

// undo merged presets from loaded presubmit and its children
func undoPresubmitPresets(presets []config.Preset, presubmit *config.Presubmit) {
	if presubmit.Spec == nil {
		return
	}
	for _, preset := range presets {
		undoPreset(&preset, presubmit.Labels, presubmit.Spec)
	}
}

// convert a kubernetes/kubernetes job to a kubernetes-security/kubernetes job
// dropLabels should be a set of "k: v" strings
// xref: prow/config/config_test.go replace(...)
// it will return the same job mutated, or nil if the job should be removed
func convertJobToSecurityJob(j *config.Presubmit, dropLabels sets.String, defaultDecoration *prowapi.DecorationConfig, podNamespace string) *config.Presubmit {
	// if a GKE job, disable it
	if strings.Contains(j.Name, "gke") {
		return nil
	}

	// filter out the unwanted labels
	if len(j.Labels) > 0 {
		filteredLabels := make(map[string]string)
		for k, v := range j.Labels {
			if !dropLabels.Has(fmt.Sprintf("%s: %s", k, v)) {
				filteredLabels[k] = v
			}
		}
		j.Labels = filteredLabels
	}

	originalName := j.Name

	// fix name and triggers for all jobs
	j.Name = strings.Replace(originalName, "pull-kubernetes", "pull-security-kubernetes", -1)
	j.RerunCommand = strings.Replace(j.RerunCommand, "pull-kubernetes", "pull-security-kubernetes", -1)
	j.Trigger = strings.Replace(j.Trigger, "pull-kubernetes", "pull-security-kubernetes", -1)
	j.Context = strings.Replace(j.Context, "pull-kubernetes", "pull-security-kubernetes", -1)
	if j.Namespace != nil && *j.Namespace == podNamespace {
		j.Namespace = nil
	}

	// Drop annotations to avoid confusing other tools
	j.Annotations = nil

	// handle k8s job args, volumes etc
	if j.Agent == "kubernetes" {
		j.Cluster = "security"
		container := &j.Spec.Containers[0]
		// check for args that need hijacking
		endsWithScenarioArgs := false
		needGCSFlag := false
		needGCSSharedFlag := false
		needStagingFlag := false
		isGCPe2e := false
		for i, arg := range container.Args {
			if arg == "--" {
				endsWithScenarioArgs = true

				// handle --repo substitution for main repo
			} else if arg == "--repo=k8s.io/kubernetes" || strings.HasPrefix(arg, "--repo=k8s.io/kubernetes=") || arg == "--repo=k8s.io/$(REPO_NAME)" || strings.HasPrefix(arg, "--repo=k8s.io/$(REPO_NAME)=") {
				container.Args[i] = strings.Replace(arg, "k8s.io/", "github.com/kubernetes-security/", 1)

				// handle upload bucket
			} else if strings.HasPrefix(arg, "--upload=") {
				container.Args[i] = "--upload=gs://kubernetes-security-prow/pr-logs"
				// check if we need to change staging artifact location for bazel-build and e2es
			} else if strings.HasPrefix(arg, "--release") {
				needGCSFlag = true
				needGCSSharedFlag = true
			} else if strings.HasPrefix(arg, "--stage") {
				needStagingFlag = true
			} else if strings.HasPrefix(arg, "--use-shared-build") {
				needGCSSharedFlag = true
			}
		}
		// NOTE: this needs to be before the bare -- and then bootstrap args so we prepend it
		container.Args = append([]string{"--ssh=/etc/ssh-security/ssh-security"}, container.Args...)

		// check for scenario specific tweaks
		// NOTE: jobs are remapped to their original name in bootstrap to de-dupe config

		scenario := ""
		for _, arg := range container.Args {
			if strings.HasPrefix(arg, "--scenario=") {
				scenario = strings.TrimPrefix(arg, "--scenario=")
			}
		}
		// check if we need to change staging artifact location for bazel-build and e2es
		if scenario == "kubernetes_bazel" {
			for _, arg := range container.Args {
				if strings.HasPrefix(arg, "--release") {
					needGCSFlag = true
					needGCSSharedFlag = true
					break
				}
			}
		}

		if scenario == "kubernetes_e2e" {
			for _, arg := range container.Args {
				if strings.Contains(arg, "gcp") {
					isGCPe2e = true
				}
				if strings.HasPrefix(arg, "--stage") {
					needStagingFlag = true
				} else if strings.HasPrefix(arg, "--use-shared-build") {
					needGCSSharedFlag = true
				}
			}
		}

		// NOTE: these needs to be at the end and after a -- if there is none (it's a scenario arg)
		if !endsWithScenarioArgs && (needGCSFlag || needGCSSharedFlag || needStagingFlag) {
			container.Args = append(container.Args, "--")
		}
		if needGCSFlag {
			container.Args = append(container.Args, "--gcs=gs://kubernetes-security-prow/ci/"+j.Name)
		}
		if needGCSSharedFlag {
			container.Args = append(container.Args, "--gcs-shared=gs://kubernetes-security-prow/bazel")
		}
		if needStagingFlag {
			container.Args = append(container.Args, "--stage=gs://kubernetes-security-prow/ci/"+j.Name)
		}
		// GCP e2e use a fixed project for security testing
		if isGCPe2e {
			container.Args = append(container.Args, "--gcp-project=k8s-jkns-pr-gce-etcd3")
		}

		// add ssh key volume / mount
		container.VolumeMounts = append(
			container.VolumeMounts,
			coreapi.VolumeMount{
				Name:      "ssh-security",
				MountPath: "/etc/ssh-security",
			},
		)
		defaultMode := int32(0400)
		j.Spec.Volumes = append(
			j.Spec.Volumes,
			coreapi.Volume{
				Name: "ssh-security",
				VolumeSource: coreapi.VolumeSource{
					Secret: &coreapi.SecretVolumeSource{
						SecretName:  "ssh-security",
						DefaultMode: &defaultMode,
					},
				},
			},
		)

		// ensure decorated jobs upload to the private bucket
		if j.Decorate {
			j.DecorationConfig.GCSConfiguration.Bucket = "kubernetes-security-prow"
		}

		// de-dupe default utility images config so verify checks pass
		if j.DecorationConfig != nil && reflect.DeepEqual(j.DecorationConfig.UtilityImages, defaultDecoration.UtilityImages) {
			j.DecorationConfig.UtilityImages = nil
		}
	}
	return j
}

// these are unnecessary, and make the config larger so we strip them out
func yamlBytesStripNulls(yamlBytes []byte) []byte {
	nullRE := regexp.MustCompile("(?m)[\n]+^[^\n]+: null$")
	return nullRE.ReplaceAll(yamlBytes, []byte{})
}

func yamlBytesToEntry(yamlBytes []byte, indent int) []byte {
	var buff bytes.Buffer
	// spaces of length indent
	prefix := bytes.Repeat([]byte{32}, indent)
	// `- ` before the first field of a yaml entry
	prefix[len(prefix)-2] = byte(45)
	buff.Write(prefix)
	// put back space
	prefix[len(prefix)-2] = byte(32)
	for i, b := range yamlBytes {
		buff.WriteByte(b)
		// indent after newline, except the last one
		if b == byte(10) && i+1 != len(yamlBytes) {
			buff.Write(prefix)
		}
	}
	return buff.Bytes()
}

func copyFile(srcPath, destPath string) error {
	// fallback to copying the file instead
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(destPath, os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}
	dst.Sync()
	dst.Close()
	src.Close()
	return nil
}

func main() {
	flag.Parse()
	// default to $PWD/prow/config.yaml
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get $PWD: %v", err)
	}
	if *configPath == "" {
		*configPath = pwd + "/../../prow/config.yaml"
	}
	if *jobsPath == "" {
		*jobsPath = pwd + "/../"
	}
	if *outputPath == "" {
		*outputPath = pwd + "/generated-security-jobs.yaml"
	}
	// read in current prow config
	parsed, err := config.Load(*configPath, *jobsPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	// create temp file to write updated config
	f, err := ioutil.TempFile(filepath.Dir(*configPath), "temp")
	if err != nil {
		log.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	// write the header
	io.WriteString(f, "# Autogenerated by genjobs.go, do NOT edit!\n")
	io.WriteString(f, "# see genjobs.go, which you can run with hack/update-config.sh\n")
	io.WriteString(f, "presubmits:\n  kubernetes-security/kubernetes:\n")

	// this is the set of preset labels we want to remove
	// we remove the bazel remote cache because we do not deploy one to this build cluster
	dropLabels := sets.NewString("preset-bazel-remote-cache-enabled: true")

	// convert each kubernetes/kubernetes presubmit to a
	// kubernetes-security/kubernetes presubmit and write to the file
	for i := range parsed.Presubmits["kubernetes/kubernetes"] {
		job := &parsed.Presubmits["kubernetes/kubernetes"][i]
		// undo merged presets, this needs to occur first!
		undoPresubmitPresets(parsed.Presets, job)
		// now convert the job
		job = convertJobToSecurityJob(job, dropLabels, parsed.Plank.DefaultDecorationConfig, parsed.PodNamespace)
		if job == nil {
			continue
		}
		jobBytes, err := yaml.Marshal(job)
		if err != nil {
			log.Fatalf("Failed to marshal job: %v", err)
		}
		// write, properly indented, and stripped of `foo: null`
		jobBytes = yamlBytesStripNulls(jobBytes)
		f.Write(yamlBytesToEntry(jobBytes, 4))
	}
	f.Sync()

	// move file to replace original
	f.Close()
	err = os.Rename(f.Name(), *outputPath)
	if err != nil {
		// fallback to copying the file instead
		err = copyFile(f.Name(), *outputPath)
		if err != nil {
			log.Fatalf("Failed to replace config with updated version: %v", err)
		}
	}
}
