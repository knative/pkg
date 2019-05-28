/*
Copyright 2019 The Kubernetes Authors.

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
	clientgentypes "github.com/knative/pkg/codegen/cmd/injection-gen/types"
	"io"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
	"path"

	"k8s.io/klog"
)

// factoryTestGenerator produces a file of listers for a given GroupVersion and
// type.
type clientGenerator struct {
	generator.DefaultGen
	outputPackage             string
	imports                   namer.ImportTracker
	groupVersions             map[string]clientgentypes.GroupVersions
	gvGoNames                 map[string]string
	clientSetPackage          string
	internalInterfacesPackage string
	filtered                  bool
}

var _ generator.Generator = &clientGenerator{}

func (g *clientGenerator) Filter(c *generator.Context, t *types.Type) bool {
	if !g.filtered {
		g.filtered = true
		return true
	}
	return false
}

func (g *clientGenerator) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

func (g *clientGenerator) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	return
}

func (g *clientGenerator) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	klog.V(5).Infof("processing type %v", t)

	gvInterfaces := make(map[string]*types.Type)
	gvNewFuncs := make(map[string]*types.Type)
	for groupPkgName := range g.groupVersions {
		gvInterfaces[groupPkgName] = c.Universe.Type(types.Name{Package: path.Join(g.outputPackage, groupPkgName), Name: "Interface"})
		gvNewFuncs[groupPkgName] = c.Universe.Function(types.Name{Package: path.Join(g.outputPackage, groupPkgName), Name: "New"})
	}
	m := map[string]interface{}{
		"cacheSharedIndexInformer":       c.Universe.Type(cacheSharedIndexInformer),
		"groupVersions":                  g.groupVersions,
		"gvInterfaces":                   gvInterfaces,
		"gvNewFuncs":                     gvNewFuncs,
		"gvGoNames":                      g.gvGoNames,
		"interfacesNewInformerFunc":      c.Universe.Function(types.Name{Package: g.internalInterfacesPackage, Name: "NewInformerFunc"}),
		"interfacesTweakListOptionsFunc": c.Universe.Type(types.Name{Package: g.internalInterfacesPackage, Name: "TweakListOptionsFunc"}),
		"informerFactoryInterface":       c.Universe.Type(types.Name{Package: g.internalInterfacesPackage, Name: "SharedInformerFactory"}),
		"reflectType":                    c.Universe.Type(reflectType),
		"runtimeObject":                  c.Universe.Type(runtimeObject),
		"schemaGroupVersionResource":     c.Universe.Type(schemaGroupVersionResource),
		"syncMutex":                      c.Universe.Type(syncMutex),
		"timeDuration":                   c.Universe.Type(timeDuration),
		"namespaceAll":                   c.Universe.Type(metav1NamespaceAll),
		"object":                         c.Universe.Type(metav1Object),
		"clientSetNewForConfigOrDie":     c.Universe.Function(types.Name{Package: g.clientSetPackage, Name: "NewForConfigOrDie"}),
		"clientSetInterface":             c.Universe.Type(types.Name{Package: g.clientSetPackage, Name: "Interface"}),
		"injectionRegisterClient":        c.Universe.Function(types.Name{Package: "github.com/knative/pkg/injection", Name: "RegisterClient"}),
		"restConfig":                     c.Universe.Type(types.Name{Package: "k8s.io/client-go/rest", Name: "Config"}),
	}

	sw.Do(injectionClient, m)

	return sw.Error()
}

var injectionClient = `
func init() {
	{{.injectionRegisterClient|raw}}(withClient)
}

// key is used as the key for associating information with a context.Context.
type Key struct{}

func withClient(ctx context.Context, cfg *{{.restConfig|raw}}) context.Context {
	return context.WithValue(ctx, Key{}, {{.clientSetNewForConfigOrDie|raw}}(cfg))
}

// Get extracts the {{.name}} client from the context.
func Get(ctx context.Context) {{.clientSetInterface|raw}} {
	return ctx.Value(Key{}).({{.clientSetInterface|raw}})
}
`
