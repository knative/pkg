/*
Copyright 2016 The Kubernetes Authors.

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
	"github.com/knative/pkg/codegen/cmd/injection-gen/generators/util"
	"path"
	"path/filepath"
	"strings"

	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
	"k8s.io/klog"

	informergenargs "github.com/knative/pkg/codegen/cmd/injection-gen/args"
	clientgentypes "github.com/knative/pkg/codegen/cmd/injection-gen/types"
)

// NameSystems returns the name system used by the generators in this package.
func NameSystems() namer.NameSystems {
	pluralExceptions := map[string]string{
		"Endpoints": "Endpoints",
	}
	return namer.NameSystems{
		"public":             namer.NewPublicNamer(0),
		"private":            namer.NewPrivateNamer(0),
		"raw":                namer.NewRawNamer("", nil),
		"publicPlural":       namer.NewPublicPluralNamer(pluralExceptions),
		"allLowercasePlural": namer.NewAllLowercasePluralNamer(pluralExceptions),
		"lowercaseSingular":  &lowercaseSingularNamer{},
	}
}

// lowercaseSingularNamer implements Namer
type lowercaseSingularNamer struct{}

// Name returns t's name in all lowercase.
func (n *lowercaseSingularNamer) Name(t *types.Type) string {
	return strings.ToLower(t.Name.Name)
}

// DefaultNameSystem returns the default name system for ordering the types to be
// processed by the generators in this package.
func DefaultNameSystem() string {
	return "public"
}

// objectMetaForPackage returns the type of ObjectMeta used by package p.
func objectMetaForPackage(p *types.Package) (*types.Type, bool, error) {
	generatingForPackage := false
	for _, t := range p.Types {
		if !util.MustParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...)).GenerateClient {
			continue
		}
		generatingForPackage = true
		for _, member := range t.Members {
			klog.Info("got  ---> ", member)

			if member.Name == "ObjectMeta" {

				return member.Type, isInternal(member), nil
			}
		}
	}
	if generatingForPackage {
		return nil, false, fmt.Errorf("unable to find ObjectMeta for any types in package %s", p.Path)
	}
	return nil, false, nil
}

// isInternal returns true if the tags for a member do not contain a json tag
func isInternal(m types.Member) bool {
	return !strings.Contains(m.Tags, "json")
}

func packageForInternalInterfaces(base string) string {
	return filepath.Join(base, "internalinterfaces")
}

func vendorless(p string) string {
	if pos := strings.LastIndex(p, "/vendor/"); pos != -1 {
		return p[pos+len("/vendor/"):]
	}
	return p
}

// Packages makes the client package definition.
func Packages(context *generator.Context, arguments *args.GeneratorArgs) generator.Packages {

	klog.Info("HERE", arguments)

	boilerplate, err := arguments.LoadGoBoilerplate()
	if err != nil {
		klog.Fatalf("Failed loading boilerplate: %v", err)
	}

	customArgs, ok := arguments.CustomArgs.(*informergenargs.CustomArgs)
	if !ok {
		klog.Fatalf("Wrong CustomArgs type: %T", arguments.CustomArgs)
	}

	versionPackagePath := filepath.Join(arguments.OutputPackagePath)

	var packageList generator.Packages
	typesForGroupVersion := make(map[clientgentypes.GroupVersion][]*types.Type)

	groupVersions := make(map[string]clientgentypes.GroupVersions)
	groupGoNames := make(map[string]string)
	for _, inputDir := range arguments.InputDirs {
		p := context.Universe.Package(vendorless(inputDir))

		objectMeta, _, err := objectMetaForPackage(p) // TODO: ignoring internal.
		if err != nil {
			klog.Fatal(err)
		}
		if objectMeta == nil {
			// no types in this package had genclient
			continue
		}

		var gv clientgentypes.GroupVersion
		var targetGroupVersions map[string]clientgentypes.GroupVersions

		klog.Info("path  ---> ", p.Path)

		parts := strings.Split(p.Path, "/")
		gv.Group = clientgentypes.Group(parts[len(parts)-2])
		gv.Version = clientgentypes.Version(parts[len(parts)-1])
		targetGroupVersions = groupVersions

		groupPackageName := gv.Group.NonEmpty()
		gvPackage := path.Clean(p.Path)

		// If there's a comment of the form "// +groupName=somegroup" or
		// "// +groupName=somegroup.foo.bar.io", use the first field (somegroup) as the name of the
		// group when generating.
		if override := types.ExtractCommentTags("+", p.Comments)["groupName"]; override != nil {
			gv.Group = clientgentypes.Group(override[0])
		}

		// If there's a comment of the form "// +groupGoName=SomeUniqueShortName", use that as
		// the Go group identifier in CamelCase. It defaults
		groupGoNames[groupPackageName] = namer.IC(strings.Split(gv.Group.NonEmpty(), ".")[0])
		if override := types.ExtractCommentTags("+", p.Comments)["groupGoName"]; override != nil {
			groupGoNames[groupPackageName] = namer.IC(override[0])
		}

		var typesToGenerate []*types.Type
		for _, t := range p.Types {
			tags := util.MustParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...))
			if !tags.GenerateClient || tags.NoVerbs || !tags.HasVerb("list") || !tags.HasVerb("watch") {
				continue
			}

			typesToGenerate = append(typesToGenerate, t)

			klog.Info("typesToGenerate  ---> ", typesToGenerate)

			if _, ok := typesForGroupVersion[gv]; !ok {
				typesForGroupVersion[gv] = []*types.Type{}
			}
			typesForGroupVersion[gv] = append(typesForGroupVersion[gv], t)
		}
		if len(typesToGenerate) == 0 {
			continue
		}

		groupVersionsEntry, ok := targetGroupVersions[groupPackageName]
		if !ok {
			groupVersionsEntry = clientgentypes.GroupVersions{
				PackageName: groupPackageName,
				Group:       gv.Group,
			}
		}
		groupVersionsEntry.Versions = append(groupVersionsEntry.Versions, clientgentypes.PackageVersion{Version: gv.Version, Package: gvPackage})
		targetGroupVersions[groupPackageName] = groupVersionsEntry

		klog.Info("targetGroupVersions[groupPackageName]  ---> ", targetGroupVersions[groupPackageName])

		orderer := namer.Orderer{Namer: namer.NewPrivateNamer(0)}
		typesToGenerate = orderer.OrderTypes(typesToGenerate)

		packageList = append(packageList, versionInformerPackages(versionPackagePath, groupPackageName, gv, groupGoNames[groupPackageName], boilerplate, typesToGenerate, customArgs.VersionedClientSetPackage)...)
		packageList = append(packageList, versionClientsPackages(versionPackagePath, groupPackageName, gv, groupGoNames[groupPackageName], boilerplate, typesToGenerate, customArgs.VersionedClientSetPackage))
		packageList = append(packageList, versionFactoryPackages(versionPackagePath, groupPackageName, gv, groupGoNames[groupPackageName], boilerplate, typesToGenerate, customArgs.VersionedClientSetPackage))
	}

	return packageList
}

func versionInformerPackages(basePackage string, groupPkgName string, gv clientgentypes.GroupVersion, groupGoName string, boilerplate []byte, typesToGenerate []*types.Type, clientSetPackage string) []generator.Package {
	packagePath := filepath.Join(basePackage, "informers", groupPkgName, strings.ToLower(gv.Version.NonEmpty()))

	vers := []generator.Package(nil)

	for _, t := range typesToGenerate {

		packagePath := packagePath + "/" + strings.ToLower(t.Name.Name)

		t := t

		// Impl
		vers = append(vers, &generator.DefaultPackage{
			PackageName: strings.ToLower(t.Name.Name),
			PackagePath: packagePath,
			HeaderText:  boilerplate,
			GeneratorFunc: func(c *generator.Context) (generators []generator.Generator) {
				// Impl
				generators = append(generators, &injectionGenerator{
					DefaultGen: generator.DefaultGen{
						OptionalName: strings.ToLower(t.Name.Name),
					},
					outputPackage:               packagePath,
					groupPkgName:                groupPkgName,
					groupVersion:                gv,
					groupGoName:                 groupGoName,
					typeToGenerate:              t,
					imports:                     generator.NewImportTracker(),
					clientSetPackage:            clientSetPackage,
					internalInterfacesPackage:   packageForInternalInterfaces(basePackage),
					typedInformerPackage:        strings.Replace(clientSetPackage, "clientset/versioned", "informers/externalversions", 1) + fmt.Sprintf("/%s/%s", strings.ToLower(gv.Group.NonEmpty())[:strings.Index(gv.Group.NonEmpty(), ".")], gv.Version),
					groupInformerFactoryPackage: fmt.Sprintf("%s/informers/%s/factory", basePackage, strings.ToLower(groupGoName)),
				})

				// Test
				generators = append(generators, &injectionTestGenerator{
					DefaultGen: generator.DefaultGen{
						OptionalName: strings.ToLower(t.Name.Name) + "_test",
					},
					outputPackage:               packagePath,
					groupPkgName:                groupPkgName,
					groupVersion:                gv,
					groupGoName:                 groupGoName,
					typeToGenerate:              t,
					imports:                     generator.NewImportTracker(),
					clientSetPackage:            clientSetPackage,
					internalInterfacesPackage:   packageForInternalInterfaces(basePackage),
					typedInformerPackage:        strings.Replace(clientSetPackage, "clientset/versioned", "informers/externalversions", 1) + fmt.Sprintf("/%s/%s", strings.ToLower(gv.Group.NonEmpty())[:strings.Index(gv.Group.NonEmpty(), ".")], gv.Version),
					groupInformerFactoryPackage: fmt.Sprintf("%s/informers/%s/factory", basePackage, strings.ToLower(groupGoName)),
				})

				return generators
			},
			FilterFunc: func(c *generator.Context, t *types.Type) bool {
				tags := util.MustParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...))
				return tags.GenerateClient && tags.HasVerb("list") && tags.HasVerb("watch")
			},
		})
	}
	return vers
}

func versionFactoryPackages(basePackage string, groupPkgName string, gv clientgentypes.GroupVersion, groupGoName string, boilerplate []byte, typesToGenerate []*types.Type, clientSetPackage string) generator.Package {
	packagePath := filepath.Join(basePackage, "informers", groupPkgName, "factory")

	return &generator.DefaultPackage{
		PackageName: strings.ToLower(groupPkgName + "factory"),
		PackagePath: packagePath,
		HeaderText:  boilerplate,
		GeneratorFunc: func(c *generator.Context) (generators []generator.Generator) {
			// Impl
			generators = append(generators, &factoryGenerator{
				DefaultGen: generator.DefaultGen{
					OptionalName: groupPkgName + "factory",
				},
				outputPackage:                packagePath,
				cachingClientSetPackage:      fmt.Sprintf("%s/clients/%s", basePackage, groupPkgName),
				sharedInformerFactoryPackage: strings.Replace(clientSetPackage, "clientset/versioned", "informers/externalversions", 1),
				imports:                      generator.NewImportTracker(),
				clientSetPackage:             clientSetPackage,
				internalInterfacesPackage:    packageForInternalInterfaces(basePackage),
			})

			// Test
			generators = append(generators, &factoryTestGenerator{
				DefaultGen: generator.DefaultGen{
					OptionalName: groupPkgName + "factory_test",
				},
				outputPackage:                packagePath,
				cachingClientSetPackage:      fmt.Sprintf("%s/clients/%s", basePackage, groupPkgName),
				sharedInformerFactoryPackage: strings.Replace(clientSetPackage, "clientset/versioned", "informers/externalversions", 1),
				imports:                      generator.NewImportTracker(),
				clientSetPackage:             clientSetPackage,
				internalInterfacesPackage:    packageForInternalInterfaces(basePackage),
			})

			return generators
		},
		FilterFunc: func(c *generator.Context, t *types.Type) bool {
			tags := util.MustParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...))
			return tags.GenerateClient && tags.HasVerb("list") && tags.HasVerb("watch")
		},
	}
}

func versionClientsPackages(basePackage string, groupPkgName string, gv clientgentypes.GroupVersion, groupGoName string, boilerplate []byte, typesToGenerate []*types.Type, clientSetPackage string) generator.Package {
	packagePath := filepath.Join(basePackage, "clients", groupPkgName)

	// Impl
	return &generator.DefaultPackage{
		PackageName: strings.ToLower(groupPkgName + "client"),
		PackagePath: packagePath,
		HeaderText:  boilerplate,
		GeneratorFunc: func(c *generator.Context) (generators []generator.Generator) {
			// Impl
			generators = append(generators, &clientGenerator{
				DefaultGen: generator.DefaultGen{
					OptionalName: groupPkgName + "client",
				},
				outputPackage:             packagePath,
				imports:                   generator.NewImportTracker(),
				clientSetPackage:          clientSetPackage,
				internalInterfacesPackage: packageForInternalInterfaces(basePackage),
			})

			// Test
			generators = append(generators, &clientTestGenerator{
				DefaultGen: generator.DefaultGen{
					OptionalName: groupPkgName + "client_test",
				},
				outputPackage:             packagePath,
				imports:                   generator.NewImportTracker(),
				clientSetPackage:          clientSetPackage,
				internalInterfacesPackage: packageForInternalInterfaces(basePackage),
			})

			return generators
		},
		FilterFunc: func(c *generator.Context, t *types.Type) bool {
			tags := util.MustParseClientGenTags(append(t.SecondClosestCommentLines, t.CommentLines...))
			return tags.GenerateClient && tags.HasVerb("list") && tags.HasVerb("watch")
		},
	}
}
