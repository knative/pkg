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

// genReconciler produces the reconciler.
type genReconciler struct {
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

func NewGenReconciler(sanitizedName, targetPackage string, kind, injectionClient, injectionInformer, lister, clientset string) generator.Generator {
	return &genReconciler{
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

func (g *genReconciler) Namers(c *generator.Context) namer.NameSystems {
	namers := NameSystems()
	namers["raw"] = namer.NewRawNamer(g.targetPackage, g.imports)
	return namers
}

func (g *genReconciler) Filter(c *generator.Context, t *types.Type) bool {
	return false
}

func (g *genReconciler) isOtherPackage(pkg string) bool {
	if pkg == g.targetPackage {
		return false
	}
	if strings.HasSuffix(pkg, "\""+g.targetPackage+"\"") {
		return false
	}
	return true
}

func (g *genReconciler) Imports(c *generator.Context) (imports []string) {
	importLines := []string{}
	for _, singleImport := range g.imports.ImportLines() {
		if g.isOtherPackage(singleImport) {
			importLines = append(importLines, singleImport)
		}
	}
	return importLines
}

func (g *genReconciler) Init(c *generator.Context, w io.Writer) error {
	klog.Infof("Generating reconciler for kind %v", g.kind)

	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	pkg := g.kind[:strings.LastIndex(g.kind, ".")]
	name := g.kind[strings.LastIndex(g.kind, ".")+1:]

	m := map[string]interface{}{
		"type": c.Universe.Type(types.Name{Package: pkg, Name: name}),
		// Deps
		"clientsetInterface":   c.Universe.Type(types.Name{Name: "Interface", Package: g.clientset}),
		"resourceLister":       c.Universe.Type(types.Name{Name: name + "Lister", Package: g.lister}),
		"trackerInterface":     c.Universe.Type(types.Name{Name: "Interface", Package: "knative.dev/pkg/tracker"}),
		"controllerReconciler": c.Universe.Type(types.Name{Name: "Reconciler", Package: "knative.dev/pkg/controller"}),
		// K8s types
		"recordEventRecorder": c.Universe.Type(types.Name{Name: "EventRecorder", Package: "k8s.io/injectionClient-go/tools/record"}),
		// methods
		"loggingFromContext": c.Universe.Function(types.Name{
			Package: "knative.dev/pkg/logging",
			Name:    "FromContext",
		}),
		"cacheSplitMetaNamespaceKey": c.Universe.Function(types.Name{
			Package: "k8s.io/injectionClient-go/tools/cache",
			Name:    "SplitMetaNamespaceKey",
		}),
		"retryRetryOnConflict": c.Universe.Function(types.Name{
			Package: "k8s.io/injectionClient-go/util/retry",
			Name:    "RetryOnConflict",
		}),
	}

	sw.Do(reconcilerInterfaceFactory, m)
	sw.Do(reconcilerImplFactory, m)
	sw.Do(reconcilerStatusFactory, m)
	sw.Do(reconcilerFinalizerFactory, m)

	return sw.Error()
}

func (g *genReconciler) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	return nil
}

var reconcilerInterfaceFactory = `
// Interface defines the strongly typed interfaces to be implemented by a
// controller reconciling {{.type|raw}}.
type Interface interface {
	// ReconcileKind implements custom logic to reconcile {{.type|raw}}. Any changes
	// to the objects .Status or .Finalizers will be propagated to the stored
	// object. It is recommended that implementors do not call any update calls
	// for the Kind inside of ReconcileKind, it is the responsibility of the core
	// controller to propagate those properties.
	ReconcileKind(ctx context.Context, o *{{.type|raw}}) error
}

// Reconciler implements controller.Reconciler for {{.type|raw}} resources.
type Core struct {
	// Client is used to write back status updates.
	Client {{.clientsetInterface|raw}}

	// Listers index properties about resources
	Lister {{.resourceLister|raw}}

	// Tracker builds an index of what resources are watching other
	// resources so that we can immediately react to changes to changes in
	// tracked resources.
	Tracker {{.trackerInterface|raw}}

	// Recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder {{.recordEventRecorder|raw}}

	// Reconciler is the implementation of the business logic of the resource.
	Reconciler Interface

	// FinalizerName is the name of the finalizer to use when finalizing the
	// resource.
	FinalizerName string
}

// Check that our Core implements controller.Reconciler
var _ controller.Reconciler = (*Core)(nil)

`

var reconcilerImplFactory = `
// Reconcile implements controller.Reconciler
func (r *Core) Reconcile(ctx context.Context, key string) error {
	logger := {{.loggingFromContext|raw}}(ctx)

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := {{.cacheSplitMetaNamespaceKey|raw}}(key)
	if err != nil {
		logger.Errorf("invalid resource key: %s", key)
		return nil
	}

    // TODO: this is needed for serving.
 	// If our controller has configuration state, we'd "freeze" it and
	// attach the frozen configuration to the context.
	//    ctx = r.configStore.ToContext(ctx)


	// Get the resource with this namespace/name.
	original, err := r.Lister.{{.type|apiGroup}}(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		// The resource may no longer exist, in which case we stop processing.
		logger.Errorf("resource %q no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}
	// Don't modify the informers copy.
	resource := original.DeepCopy()

	// Reconcile this copy of the resource and then write back any status
	// updates regardless of whether the reconciliation errored out.
	reconcileErr := r.Reconciler.ReconcileKind(ctx, resource)

	// Synchronize the finalizers.
	if equality.Semantic.DeepEqual(original.Finalizers, resource.Finalizers) {
		// If we didn't change finalizers then don't call updateFinalizers.
	} else if _, updated, fErr := r.updateFinalizers(ctx, resource); fErr != nil {
		logger.Warnw("Failed to update finalizers", zap.Error(fErr))
		r.Recorder.Eventf(resource, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to update finalizers for %q: %v", resource.Name, fErr)
		return fErr
	} else if updated {
		// There was a difference and updateFinalizers said it updated and did not return an error.
		r.Recorder.Eventf(resource, corev1.EventTypeNormal, "Updated", "Updated %q finalizers", resource.GetName())
	}

	// Synchronize the status.
	if equality.Semantic.DeepEqual(original.Status, resource.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the injectionInformer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if err = r.updateStatus(original, resource); err != nil {
		logger.Warnw("Failed to update resource status", zap.Error(err))
		r.Recorder.Eventf(resource, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to update status for %q: %v", resource.Name, err)
		return err
	}

	// Report the reconciler error, if any.
	if reconcileErr != nil {
		r.Recorder.Event(resource, corev1.EventTypeWarning, "InternalError", reconcileErr.Error())
	}
	return reconcileErr
}
`

var reconcilerStatusFactory = `
func (r *Core) updateStatus(existing *{{.type|raw}}, desired *{{.type|raw}}) error {
	existing = existing.DeepCopy()
	return RetryUpdateConflicts(func(attempts int) (err error) {
		// The first iteration tries to use the injectionInformer's state, subsequent attempts fetch the latest state via API.
		if attempts > 0 {
			existing, err = r.Client.{{.type|versionedClientset}}().{{.type|apiGroup}}(desired.Namespace).Get(desired.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}

		// If there's nothing to update, just return.
		if reflect.DeepEqual(existing.Status, desired.Status) {
			return nil
		}

		existing.Status = desired.Status
		_, err = r.Client.{{.type|versionedClientset}}().{{.type|apiGroup}}(existing.Namespace).UpdateStatus(existing)
		return err
	})
}


// TODO: move this to knative.dev/pkg/reconciler
// RetryUpdateConflicts retries the inner function if it returns conflict errors.
// This can be used to retry status updates without constantly reenqueuing keys.
func RetryUpdateConflicts(updater func(int) error) error {
	attempts := 0
	return {{.retryRetryOnConflict|raw}}(retry.DefaultRetry, func() error {
		err := updater(attempts)
		attempts++
		return err
	})
}
`

var reconcilerFinalizerFactory = `
// Update the Finalizers of the resource.
func (r *Core) updateFinalizers(ctx context.Context, desired *{{.type|raw}}) (*{{.type|raw}}, bool, error) {
	actual, err := r.Lister.{{.type|apiGroup}}(desired.Namespace).Get(desired.Name)
	if err != nil {
		return nil, false, err
	}

	// Don't modify the informers copy.
	existing := actual.DeepCopy()

	var finalizers []string

	// If there's nothing to update, just return.
	existingFinalizers := sets.NewString(existing.Finalizers...)
	desiredFinalizers := sets.NewString(desired.Finalizers...)

	if desiredFinalizers.Has(r.FinalizerName) {
		if existingFinalizers.Has(r.FinalizerName) {
			// Nothing to do.
			return desired, false, nil
		}
		// Add the finalizer.
		finalizers = append(existing.Finalizers, r.FinalizerName)
	} else {
		if !existingFinalizers.Has(r.FinalizerName) {
			// Nothing to do.
			return desired, false, nil
		}
		// Remove the finalizer.
		existingFinalizers.Delete(r.FinalizerName)
		finalizers = existingFinalizers.List()
	}

	mergePatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers":      finalizers,
			"resourceVersion": existing.ResourceVersion,
		},
	}

	patch, err := json.Marshal(mergePatch)
	if err != nil {
		return desired, false, err
	}

	update, err := r.Client.{{.type|versionedClientset}}().{{.type|apiGroup}}(desired.Namespace).Patch(existing.Name, types.MergePatchType, patch)
	return update, true, err
}

func (r *Core) setFinalizer(a *{{.type|raw}}) {
	finalizers := sets.NewString(a.Finalizers...)
	finalizers.Insert(r.FinalizerName)
	a.Finalizers = finalizers.List()
}

func (r *Core) unsetFinalizer(a *{{.type|raw}}) {
	finalizers := sets.NewString(a.Finalizers...)
	finalizers.Delete(r.FinalizerName)
	a.Finalizers = finalizers.List()
}
`
