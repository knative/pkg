/*
Copyright 2019 The Knative Authors.

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

package args

import (
	"fmt"
	"path"

	"github.com/spf13/pflag"
	codegenutil "k8s.io/code-generator/pkg/util"
	"k8s.io/gengo/args"
)

// CustomArgs is used by the gengo framework to pass args specific to this generator.
type CustomArgs struct {
	VersionedClientSetPackage string
}

// NewDefaults returns default arguments for the generator.
func NewDefaults() (*args.GeneratorArgs, *CustomArgs) {
	genericArgs := args.Default().WithoutDefaultFlagParsing()
	customArgs := &CustomArgs{}
	genericArgs.CustomArgs = customArgs

	if pkg := codegenutil.CurrentPackage(); len(pkg) != 0 {
		genericArgs.OutputPackagePath = path.Join(pkg, "pkg/client/injection/informers")
		customArgs.VersionedClientSetPackage = path.Join(pkg, "pkg/client/injection/clientset/versioned")
	}

	return genericArgs, customArgs
}

// AddFlags add the generator flags to the flag set.
func (ca *CustomArgs) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&ca.VersionedClientSetPackage, "versioned-clientset-package", ca.VersionedClientSetPackage, "the full package name for the versioned injection clientset to use")
}

// Validate checks the given arguments.
func Validate(genericArgs *args.GeneratorArgs) error {
	customArgs := genericArgs.CustomArgs.(*CustomArgs)

	if len(genericArgs.OutputPackagePath) == 0 {
		return fmt.Errorf("output package cannot be empty")
	}
	if len(customArgs.VersionedClientSetPackage) == 0 {
		return fmt.Errorf("versioned clientset package cannot be empty")
	}

	return nil
}
