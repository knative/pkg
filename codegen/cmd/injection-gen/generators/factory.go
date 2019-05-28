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
	"io"
	"path"

	clientgentypes "github.com/knative/pkg/codegen/cmd/injection-gen/types"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"

	"k8s.io/klog"
)

// factoryTestGenerator produces a file of listers for a given GroupVersion and
// type.
type factoryGenerator struct {
	generator.DefaultGen
	outputPackage                string
	imports                      namer.ImportTracker
	groupVersions                map[string]clientgentypes.GroupVersions
	gvGoNames                    map[string]string
	clientSetPackage             string
	cachingClientSetPackage      string
	sharedInformerFactoryPackage string
	internalInterfacesPackage    string
	filtered                     bool
}

var _ generator.Generator = &factoryGenerator{}

func (g *factoryGenerator) Filter(c *generator.Context, t *types.Type) bool {
	if !g.filtered {
		g.filtered = true
		return true
	}
	return false
}

func (g *factoryGenerator) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

func (g *factoryGenerator) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	return
}

func (g *factoryGenerator) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	klog.V(5).Infof("processing type %v", t)

	gvInterfaces := make(map[string]*types.Type)
	gvNewFuncs := make(map[string]*types.Type)
	for groupPkgName := range g.groupVersions {
		gvInterfaces[groupPkgName] = c.Universe.Type(types.Name{Package: path.Join(g.outputPackage, groupPkgName), Name: "Interface"})
		gvNewFuncs[groupPkgName] = c.Universe.Function(types.Name{Package: path.Join(g.outputPackage, groupPkgName), Name: "New"})
	}
	m := map[string]interface{}{
		"cacheSharedIndexInformer":          c.Universe.Type(cacheSharedIndexInformer),
		"groupVersions":                     g.groupVersions,
		"gvInterfaces":                      gvInterfaces,
		"gvNewFuncs":                        gvNewFuncs,
		"gvGoNames":                         g.gvGoNames,
		"interfacesNewInformerFunc":         c.Universe.Function(types.Name{Package: g.internalInterfacesPackage, Name: "NewInformerFunc"}),
		"interfacesTweakListOptionsFunc":    c.Universe.Type(types.Name{Package: g.internalInterfacesPackage, Name: "TweakListOptionsFunc"}),
		"informerFactoryInterface":          c.Universe.Type(types.Name{Package: g.internalInterfacesPackage, Name: "SharedInformerFactory"}),
		"reflectType":                       c.Universe.Type(reflectType),
		"runtimeObject":                     c.Universe.Type(runtimeObject),
		"schemaGroupVersionResource":        c.Universe.Type(schemaGroupVersionResource),
		"syncMutex":                         c.Universe.Type(syncMutex),
		"timeDuration":                      c.Universe.Type(timeDuration),
		"namespaceAll":                      c.Universe.Type(metav1NamespaceAll),
		"object":                            c.Universe.Type(metav1Object),
		"cachingClientGet":                  c.Universe.Type(types.Name{Package: g.cachingClientSetPackage, Name: "Get"}),
		"informersNewSharedInformerFactory": c.Universe.Function(types.Name{Package: g.sharedInformerFactoryPackage, Name: "NewSharedInformerFactory"}),
		"informersSharedInformerFactory":    c.Universe.Function(types.Name{Package: g.sharedInformerFactoryPackage, Name: "SharedInformerFactory"}),
		"clientSetNewForConfigOrDie":        c.Universe.Function(types.Name{Package: g.clientSetPackage, Name: "NewForConfigOrDie"}),
		"clientSetInterface":                c.Universe.Type(types.Name{Package: g.clientSetPackage, Name: "Interface"}),
		"injectionRegisterInformerFactory":  c.Universe.Function(types.Name{Package: "github.com/knative/pkg/injection", Name: "RegisterInformerFactory"}),
	}

	sw.Do(injectionFactory, m)

	return sw.Error()
}

var injectionFactory = `
func init() {
	{{.injectionRegisterInformerFactory|raw}}(withInformerFactory)
}

// key is used as the key for associating information with a context.Context.
type Key struct{}

func withInformerFactory(ctx context.Context, resyncPeriod {{.timeDuration|raw}}) context.Context {
	sc := {{.cachingClientGet|raw}}(ctx)
	return context.WithValue(ctx, Key{}, {{.informersNewSharedInformerFactory|raw}}(sc, resyncPeriod))
}

// Get extracts the InformerFactory from the context.
func Get(ctx context.Context) {{.informersSharedInformerFactory|raw}} {
	return ctx.Value(Key{}).({{.informersSharedInformerFactory|raw}})
}
`

/*


import (
	"context"
	"time"

cachingClientSetPackage   string
	sharedInformerFactoryPackage string

	informers "github.com/knative/serving/pkg/client/informers/externalversions"

	"github.com/knative/pkg/injection"
	"github.com/knative/serving/pkg/injection/clients/servingclient"
)

func init() {
	injection.RegisterInformerFactory(withServingInformerFactory)
}

// servingInformerFactoryKey is used as the key for associating information
// with a context.Context.
type servingInformerFactoryKey struct{}

func withServingInformerFactory(ctx context.Context, resyncPeriod time.Duration) context.Context {
	sc := servingclient.Get(ctx)
	return context.WithValue(ctx, servingInformerFactoryKey{},
		informers.NewSharedInformerFactory(sc, resyncPeriod))
}

// Get extracts the Serving InformerFactory from the context.
func Get(ctx context.Context) informers.SharedInformerFactory {
	return ctx.Value(servingInformerFactoryKey{}).(informers.SharedInformerFactory)
}
*/
