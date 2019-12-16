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

// genController produces the controller.
type genController struct {
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

func NewGenController(sanitizedName, targetPackage string, kind, injectionClient, injectionInformer, lister, clientset string) generator.Generator {
	return &genController{
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

func (g *genController) Namers(c *generator.Context) namer.NameSystems {
	namers := NameSystems()
	namers["raw"] = namer.NewRawNamer(g.targetPackage, g.imports)
	return namers
}

func (g *genController) Filter(c *generator.Context, t *types.Type) bool {
	return false
}

func (g *genController) isOtherPackage(pkg string) bool {
	if pkg == g.targetPackage {
		return false
	}
	if strings.HasSuffix(pkg, "\""+g.targetPackage+"\"") {
		return false
	}
	return true
}

func (g *genController) Imports(c *generator.Context) (imports []string) {
	importLines := []string{}
	for _, singleImport := range g.imports.ImportLines() {
		if g.isOtherPackage(singleImport) {
			importLines = append(importLines, singleImport)
		}
	}
	return importLines
}

func (g *genController) Init(c *generator.Context, w io.Writer) error {
	klog.Infof("Generating controller for kind %v", g.kind)

	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	klog.V(5).Infof("processing kind %v", g.kind)

	pkg := g.kind[:strings.LastIndex(g.kind, ".")]
	name := g.kind[strings.LastIndex(g.kind, ".")+1:]

	m := map[string]interface{}{
		"type": c.Universe.Type(types.Name{Package: pkg, Name: name}),
		// Methods.
		"controllerImpl": c.Universe.Type(types.Name{Package: "knative.dev/pkg/controller", Name: "Impl"}),
		"loggingFromContext": c.Universe.Function(types.Name{
			Package: "knative.dev/pkg/logging",
			Name:    "FromContext",
		}),
		"clientGet": c.Universe.Function(types.Name{
			Package: g.injectionClient,
			Name:    "Get",
		}),
		"informerGet": c.Universe.Function(types.Name{
			Package: g.injectionInformer,
			Name:    "Get",
		}),
		"corev1EventSource": c.Universe.Function(types.Name{
			Package: "k8s.io/api/core/v1",
			Name:    "EventSource",
		}),
	}

	sw.Do(controllerFactory, m)

	return sw.Error()
}

func (g *genController) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	return nil
}

var controllerFactory = `

const (
	controllerAgentName = "{{.type|allLowercase}}-controller"
	finalizerName       = "{{.type|allLowercase}}"
)

func NewImpl(ctx context.Context, r *Reconciler) *{{.controllerImpl|raw}} {
	logger := {{.loggingFromContext|raw}}(ctx)

	impl := controller.NewImpl(r, logger, "{{.type|allLowercasePlural}}")

	injectionInformer := {{.informerGet|raw}}(ctx)

	r.Core = Core{
		Client:  {{.clientGet|raw}}(ctx),
		Lister:  injectionInformer.Lister(),
		Tracker: tracker.New(impl.EnqueueKey, controller.GetTrackerLease(ctx)),
		Recorder: record.NewBroadcaster().NewRecorder(
			scheme.Scheme, {{.corev1EventSource|raw}}{Component: controllerAgentName}),
		FinalizerName: finalizerName,
		Reconciler:    r,
	}

	logger.Info("Setting up core event handlers")
	injectionInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	return impl
}
`
