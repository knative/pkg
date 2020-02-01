/*
Copyright 2020 The Kubernetes Authors.

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

	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"
	"k8s.io/klog"
)

// fakeClientGenerator produces a file of listers for a given GroupVersion and
// type.
type reconcilerReconcilerStubGenerator struct {
	generator.DefaultGen
	outputPackage string
	imports       namer.ImportTracker
	filtered      bool

	reconcilerPkg       string
	clientPkg           string
	clientInjectionPkg  string
	informerPackagePath string
}

var _ generator.Generator = (*reconcilerReconcilerStubGenerator)(nil)

func (g *reconcilerReconcilerStubGenerator) Filter(c *generator.Context, t *types.Type) bool {
	// We generate a single client, so return true once.
	if !g.filtered {
		g.filtered = true
		return true
	}
	return false
}

func (g *reconcilerReconcilerStubGenerator) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

func (g *reconcilerReconcilerStubGenerator) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	return
}

func (g *reconcilerReconcilerStubGenerator) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	klog.V(5).Infof("processing type %v", t)

	m := map[string]interface{}{
		"type":            t,
		"controllerImpl":  c.Universe.Type(types.Name{Package: "knative.dev/pkg/controller", Name: "Impl"}),
		"reconcilerEvent": c.Universe.Type(types.Name{Package: "knative.dev/pkg/reconciler", Name: "Event"}),
		"loggingFromContext": c.Universe.Function(types.Name{
			Package: "knative.dev/pkg/logging",
			Name:    "FromContext",
		}),
		"corev1EventSource": c.Universe.Function(types.Name{
			Package: "k8s.io/api/core/v1",
			Name:    "EventSource",
		}),
		"clientGet": c.Universe.Function(types.Name{
			Package: g.clientPkg,
			Name:    "Get",
		}),
		"informerGet": c.Universe.Function(types.Name{
			Package: g.informerPackagePath,
			Name:    "Get",
		}),
		"reconcilerCore": c.Universe.Type(types.Name{
			Package: g.reconcilerPkg,
			Name:    "Core",
		}),
	}

	sw.Do(reconcilerReconcilerStub, m)

	return sw.Error()
}

var reconcilerReconcilerStub = `
// Reconciler implements controller.Reconciler for {{.type|public}} resources.
type Reconciler struct {
	{{.reconcilerCore|raw}}
}

// Check that our Reconciler implements Interface
var _ Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, o *{{.type|raw}}) {{.reconcilerEvent|raw}} {
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
