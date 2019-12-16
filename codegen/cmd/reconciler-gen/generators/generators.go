/*
Copyright 2019 The Knative Authors

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

package generators

import (
	"fmt"
	"path/filepath"
	"strings"

	codegennamer "k8s.io/code-generator/pkg/namer"
	"k8s.io/gengo/examples/set-gen/sets"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
	"knative.dev/pkg/codegen/cmd/reconciler-gen/args"

	"k8s.io/klog"
)

// This is the comment tag that carries parameters for reconciler generation.
const (
	tagEnabledName           = "genreconciler"
	kindTagName              = tagEnabledName + ":kind"
	stubsTagName             = tagEnabledName + ":stubs"
	injectionClientTagName   = tagEnabledName + ":injectionClient"
	injectionInformerTagName = tagEnabledName + ":injectionInformer"
	listerTagName            = tagEnabledName + ":lister"
	clientsetTagName         = tagEnabledName + ":clientset"
)

// enabledTagValue holds parameters from a tagName tag.
type tagValue struct {
	stubs             bool
	kind              string
	injectionClient   string
	injectionInformer string
	lister            string
	clientset         string
}

func extractTag(comments []string) *tagValue {
	tags := types.ExtractCommentTags("+", comments)
	if tags[tagEnabledName] == nil {
		return nil
	}

	// If there are multiple values, abort.
	if len(tags[tagEnabledName]) > 1 {
		klog.Fatalf("Found %d %s tags: %q", len(tags[tagEnabledName]), tagEnabledName, tags[tagEnabledName])
	}

	// If we got here we are returning something.
	tag := &tagValue{}

	if v := tags[kindTagName]; v != nil {
		tag.kind = v[0]
	}

	if v := tags[stubsTagName]; v != nil {
		tag.stubs = true
	}

	if v := tags[injectionClientTagName]; v != nil {
		tag.injectionClient = v[0]
	}

	if v := tags[injectionInformerTagName]; v != nil {
		tag.injectionInformer = v[0]
	}

	if v := tags[listerTagName]; v != nil {
		tag.lister = v[0]
	}

	if v := tags[clientsetTagName]; v != nil {
		tag.clientset = v[0]
	}

	return tag
}

type lowercaseNamer struct{}

func (r *lowercaseNamer) Name(t *types.Type) string {
	return strings.ToLower(t.Name.Name)
}

type versionedClientsetNamer struct {
	public *ExceptionNamer
}

func (r *versionedClientsetNamer) Name(t *types.Type) string {
	// Turns type into a GroupVersion type string based on package.
	parts := strings.Split(t.Name.Package, "/")
	group := parts[len(parts)-2]
	version := parts[len(parts)-1]

	g := r.public.Name(&types.Type{Name: types.Name{Name: group, Package: t.Name.Package}})
	v := r.public.Name(&types.Type{Name: types.Name{Name: version, Package: t.Name.Package}})

	return g + v
}

// NameSystems returns the name system used by the generators in this package.
func NameSystems() namer.NameSystems {
	pluralExceptions := map[string]string{
		"Endpoints": "Endpoints",
	}
	lowercasePluralNamer := namer.NewAllLowercasePluralNamer(pluralExceptions)

	publicNamer := &ExceptionNamer{
		Exceptions: map[string]string{},
		KeyFunc: func(t *types.Type) string {
			return t.Name.Package + "." + t.Name.Name
		},
		Delegate: namer.NewPublicNamer(0),
	}
	privateNamer := &ExceptionNamer{
		Exceptions: map[string]string{},
		KeyFunc: func(t *types.Type) string {
			return t.Name.Package + "." + t.Name.Name
		},
		Delegate: namer.NewPrivateNamer(0),
	}
	publicPluralNamer := &ExceptionNamer{
		Exceptions: map[string]string{},
		KeyFunc: func(t *types.Type) string {
			return t.Name.Package + "." + t.Name.Name
		},
		Delegate: namer.NewPublicPluralNamer(pluralExceptions),
	}
	privatePluralNamer := &ExceptionNamer{
		Exceptions: map[string]string{},
		KeyFunc: func(t *types.Type) string {
			return t.Name.Package + "." + t.Name.Name
		},
		Delegate: namer.NewPrivatePluralNamer(pluralExceptions),
	}

	return namer.NameSystems{
		"singularKind":       namer.NewPublicNamer(0),
		"publicPlural":       publicPluralNamer,
		"privatePlural":      privatePluralNamer,
		"public":             publicNamer,
		"private":            privateNamer,
		"allLowercase":       &lowercaseNamer{},
		"allLowercasePlural": lowercasePluralNamer,
		"versionedClientset": &versionedClientsetNamer{public: publicNamer},
		"apiGroup":           codegennamer.NewTagOverrideNamer("publicPlural", publicPluralNamer),
	}
}

// ExceptionNamer allows you specify exceptional cases with exact names.  This allows you to have control
// for handling various conflicts, like group and resource names for instance.
type ExceptionNamer struct {
	Exceptions map[string]string
	KeyFunc    func(*types.Type) string

	Delegate namer.Namer
}

// Name provides the requested name for a type.
func (n *ExceptionNamer) Name(t *types.Type) string {
	key := n.KeyFunc(t)
	if exception, ok := n.Exceptions[key]; ok {
		return exception
	}
	return n.Delegate.Name(t)
}

// DefaultNameSystem returns the default name system for ordering the types to be
// processed by the generators in this package.
func DefaultNameSystem() string {
	return "public"
}

func Packages(context *generator.Context, arguments *args.GeneratorArgs) generator.Packages {
	boilerplate, err := arguments.LoadGoBoilerplate()
	if err != nil {
		klog.Fatalf("Failed loading boilerplate: %v", err)
	}

	inputs := sets.NewString(context.Inputs...)
	packages := generator.Packages{}
	header := append([]byte(fmt.Sprintf("// +build !%s\n\n", arguments.GeneratedBuildTag)), boilerplate...)

	editHeader, err := arguments.LoadEditGoBoilerplate()
	if err != nil {
		klog.Fatalf("Failed loading edit boilerplate: %v", err)
	}

	for i := range inputs {
		klog.V(5).Infof("Considering pkg %q", i)
		pkg := context.Universe[i]
		if pkg == nil {
			// If the input had no Go files, for example.
			continue
		}

		ptag := extractTag(pkg.Comments)
		if ptag != nil {
			klog.V(5).Infof("  tag: %+v", ptag)
		} else {
			klog.V(5).Infof("  no tag")
			continue
		}

		packages = append(packages,
			&generator.DefaultPackage{
				PackageName: strings.Split(filepath.Base(pkg.Path), ".")[0],
				PackagePath: pkg.Path,
				HeaderText:  header,
				GeneratorFunc: func(c *generator.Context) (generators []generator.Generator) {
					return []generator.Generator{
						NewGenController(arguments.OutputFileBaseName+".controller", pkg.Path, ptag.kind, ptag.injectionClient, ptag.injectionInformer, ptag.lister, ptag.clientset),
						NewGenReconciler(arguments.OutputFileBaseName+".reconciler", pkg.Path, ptag.kind, ptag.injectionClient, ptag.injectionInformer, ptag.lister, ptag.clientset),
					}
				},
				FilterFunc: func(c *generator.Context, t *types.Type) bool {
					return false
				},
			})

		if ptag.stubs {

			name := ptag.kind[strings.LastIndex(ptag.kind, ".")+1:]

			packages = append(packages,
				&generator.DefaultPackage{
					PackageName: strings.Split(filepath.Base(pkg.Path), ".")[0],
					PackagePath: pkg.Path,
					HeaderText:  editHeader,
					GeneratorFunc: func(c *generator.Context) (generators []generator.Generator) {
						return []generator.Generator{
							NewGenStubController("controller", pkg.Path, ptag.kind, ptag.injectionClient, ptag.injectionInformer, ptag.lister, ptag.clientset),
							NewGenStubReconciler(strings.ToLower(name), pkg.Path, ptag.kind, ptag.injectionClient, ptag.injectionInformer, ptag.lister, ptag.clientset),
						}
					},
					FilterFunc: func(c *generator.Context, t *types.Type) bool {
						return false
					},
				},
			)
		}

	}
	return packages
}
