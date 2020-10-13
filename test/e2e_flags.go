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

// This file contains logic to encapsulate flags which are needed to specify
// what cluster, etc. to use for e2e tests.

package test

import (
	"bytes"
	"context"
	"flag"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/injection"
	"os"
	"sync"
	"text/template"
	"time"

	"knative.dev/pkg/test/logging"
)

var (
	// Flags holds the command line flags or defaults for settings in the user's environment.
	// See EnvironmentFlags for a list of supported fields.
	// Deprecated: use GetFlags()
	Flags = GetFlags()
)

// TODO: remove this when Flags is deleted.
func init() {
	// HACK HACK HACK
	injection.Default.RegisterClient(func(ctx context.Context, _ *rest.Config) context.Context {
		// This will happen after all init methods are called.
		Flags.Kubeconfig = injection.GetKubeConfigPath(ctx)
		return ctx
	})
}

// EnvironmentFlags define the flags that are needed to run the e2e tests.
type EnvironmentFlags struct {
	Cluster string // K8s cluster (defaults to cluster in kubeconfig)
	// Deprecated: Use injection.GetConfig(ctx)
	Kubeconfig           string        // Path to kubeconfig (defaults to ./kube/config)
	Namespace            string        // K8s namespace (blank by default, to be overwritten by test suite)
	IngressEndpoint      string        // Host to use for ingress endpoint
	ImageTemplate        string        // Template to build the image reference (defaults to {{.Repository}}/{{.Name}}:{{.Tag}})
	DockerRepo           string        // Docker repo (defaults to $KO_DOCKER_REPO)
	Tag                  string        // Tag for test images
	SpoofRequestInterval time.Duration // SpoofRequestInterval is the interval between requests in SpoofingClient
	SpoofRequestTimeout  time.Duration // SpoofRequestTimeout is the timeout for polling requests in SpoofingClient
}

var (
	flags *EnvironmentFlags
	fonce sync.Once
)

// Flags holds the command line flags or defaults for settings in the user's environment.
// See EnvironmentFlags for a list of supported fields.
func GetFlags() *EnvironmentFlags {
	fonce.Do(func() {
		flags = new(EnvironmentFlags)
		flag.StringVar(&flags.Cluster, "cluster", "",
			"Provide the cluster to test against. Defaults to the current cluster in kubeconfig.")

		flag.StringVar(&flags.Namespace, "namespace", "",
			"Provide the namespace you would like to use for these tests.")

		flag.StringVar(&flags.IngressEndpoint, "ingressendpoint", "", "Provide a static endpoint url to the ingress server used during tests.")

		flag.StringVar(&flags.ImageTemplate, "imagetemplate", "{{.Repository}}/{{.Name}}:{{.Tag}}",
			"Provide a template to generate the reference to an image from the test. Defaults to `{{.Repository}}/{{.Name}}:{{.Tag}}`.")

		flag.DurationVar(&flags.SpoofRequestInterval, "spoofinterval", 1*time.Second,
			"Provide an interval between requests for the SpoofingClient")

		flag.DurationVar(&flags.SpoofRequestTimeout, "spooftimeout", 5*time.Minute,
			"Provide a request timeout for the SpoofingClient")

		defaultRepo := os.Getenv("KO_DOCKER_REPO")
		flag.StringVar(&flags.DockerRepo, "dockerrepo", defaultRepo,
			"Provide the uri of the docker repo you have uploaded the test image to using `uploadtestimage.sh`. Defaults to $KO_DOCKER_REPO")

		flag.StringVar(&flags.Tag, "tag", "latest", "Provide the version tag for the test images.")
	})
	return flags
}

// TODO(coryrc): Remove once other repos are moved to call logging.InitializeLogger() directly
func SetupLoggingFlags() {
	logging.InitializeLogger()
}

// ImagePath is a helper function to transform an image name into an image reference that can be pulled.
func ImagePath(name string) string {
	tpl, err := template.New("image").Parse(GetFlags().ImageTemplate)
	if err != nil {
		panic("could not parse image template: " + err.Error())
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, struct {
		Repository string
		Name       string
		Tag        string
	}{
		Repository: GetFlags().DockerRepo,
		Name:       name,
		Tag:        GetFlags().Tag,
	}); err != nil {
		panic("could not apply the image template: " + err.Error())
	}
	return buf.String()
}
