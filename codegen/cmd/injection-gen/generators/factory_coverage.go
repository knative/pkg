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
type factoryTestGenerator struct {
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

var _ generator.Generator = &factoryTestGenerator{}

func (g *factoryTestGenerator) Filter(c *generator.Context, t *types.Type) bool {
	if !g.filtered {
		g.filtered = true
		return true
	}
	return false
}

func (g *factoryTestGenerator) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

func (g *factoryTestGenerator) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	return
}

func (g *factoryTestGenerator) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
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

var injectionFactoryTest = `
func TestRegistration(t *testing.T) {
	ctx := context.Background()

	// Get before registration
	if empty := Get(ctx); empty != nil {
		t.Errorf("Unexpected informer factory: %v", empty)
	}

	// Check how many informer factories have registered.
	inffs := injection.Default.GetInformerFactories()
	if want, got := 1, len(inffs); want != got {
		t.Errorf("GetInformerFactories() = %d, wanted %d", want, got)
	}

	// Setup the informers.
	var infs []controller.Informer
	ctx, infs = injection.Default.SetupInformers(ctx, &rest.Config{})

	// We should see that a single informer was set up.
	if want, got := 0, len(infs); want != got {
		t.Errorf("SetupInformers() = %d, wanted %d", want, got)
	}

	// Get our informer from the context.
	if inf := Get(ctx); inf == nil {
		t.Error("Get() = nil, wanted non-nil")
	}
}
`
