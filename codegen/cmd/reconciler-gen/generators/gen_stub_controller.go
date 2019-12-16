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
	"io"
	"strings"

	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
	"k8s.io/klog"
)

// genController produces a stub controller.
type genStubController struct {
	generator.DefaultGen
	targetPackage string

	kind              string
	injectionClient   string
	injectionInformer string
	lister            string
	clientset         string

	imports      namer.ImportTracker
	typesForInit []*types.Type
}

func NewGenStubController(sanitizedName, targetPackage string, kind, injectionClient, injectionInformer, lister, clientset string) generator.Generator {
	return &genStubController{
		DefaultGen: generator.DefaultGen{
			OptionalName: sanitizedName,
		},
		targetPackage:     targetPackage,
		kind:              kind,
		injectionClient:   injectionClient,
		injectionInformer: injectionInformer,
		lister:            lister,
		clientset:         clientset,
		imports:           generator.NewImportTracker(),
		typesForInit:      make([]*types.Type, 0),
	}
}

func (g *genStubController) Namers(c *generator.Context) namer.NameSystems {
	namers := NameSystems()
	namers["raw"] = namer.NewRawNamer(g.targetPackage, g.imports)
	return namers
}

func (g *genStubController) Filter(c *generator.Context, t *types.Type) bool {
	return false
}

func (g *genStubController) isOtherPackage(pkg string) bool {
	if pkg == g.targetPackage {
		return false
	}
	if strings.HasSuffix(pkg, "\""+g.targetPackage+"\"") {
		return false
	}
	return true
}

func (g *genStubController) Imports(c *generator.Context) (imports []string) {
	importLines := []string{}
	for _, singleImport := range g.imports.ImportLines() {
		if g.isOtherPackage(singleImport) {
			importLines = append(importLines, singleImport)
		}
	}
	return importLines
}

func (g *genStubController) Init(c *generator.Context, w io.Writer) error {
	klog.Infof("Generating sub controller for kind %v", g.kind)

	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	m := map[string]interface{}{}

	sw.Do(stubControllerFactory, m)

	return sw.Error()
}

func (g *genStubController) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	return nil
}

var stubControllerFactory = `
// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {

	r := &Reconciler{}
	impl := NewImpl(ctx, r)

	return impl
}
`
