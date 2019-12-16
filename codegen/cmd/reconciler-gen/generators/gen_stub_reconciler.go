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

// genStubReconciler produces the stub reconciler.
type genStubReconciler struct {
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

func NewGenStubReconciler(sanitizedName, targetPackage string, kind, injectionClient, injectionInformer, lister, clientset string) generator.Generator {
	return &genStubReconciler{
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

func (g *genStubReconciler) Namers(c *generator.Context) namer.NameSystems {
	namers := NameSystems()
	namers["raw"] = namer.NewRawNamer(g.targetPackage, g.imports)
	return namers
}

func (g *genStubReconciler) Filter(c *generator.Context, t *types.Type) bool {
	return false
}

func (g *genStubReconciler) isOtherPackage(pkg string) bool {
	if pkg == g.targetPackage {
		return false
	}
	if strings.HasSuffix(pkg, "\""+g.targetPackage+"\"") {
		return false
	}
	return true
}

func (g *genStubReconciler) Imports(c *generator.Context) (imports []string) {
	importLines := []string{}
	for _, singleImport := range g.imports.ImportLines() {
		if g.isOtherPackage(singleImport) {
			importLines = append(importLines, singleImport)
		}
	}
	return importLines
}

func (g *genStubReconciler) Init(c *generator.Context, w io.Writer) error {
	klog.Infof("Generating stub reconciler for kind %v", g.kind)

	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	pkg := g.kind[:strings.LastIndex(g.kind, ".")]
	name := g.kind[strings.LastIndex(g.kind, ".")+1:]

	m := map[string]interface{}{
		"type": c.Universe.Type(types.Name{Package: pkg, Name: name}),
	}

	sw.Do(stubReconcilerFactory, m)

	return sw.Error()
}

func (g *genStubReconciler) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	return nil
}

var stubReconcilerFactory = `
// Reconciler implements controller.Reconciler for {{.type|singularKind}} resources.
type Reconciler struct {
	Core
}

// Check that our Reconciler implements Interface
var _ Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, o *{{.type|raw}}) error {
	if o.GetDeletionTimestamp() != nil {
		// Check for a DeletionTimestamp.  If present, elide the normal reconcile logic.
		// When a controller needs finalizer handling, it would go here.
		return nil
	}	
	o.Status.InitializeConditions()

	// TODO: add custom reconciliation logic here.

	o.Status.ObservedGeneration = o.Generation
	return nil
}

`
