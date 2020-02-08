# Knative Dependency Injection

This library supports the production of controller processes with minimal
boilerplate outside of the reconciler implementation.

## Building Controllers Leveraging Generated Reconciler

To adopt this model of controller construction, implementations should start
with the following controller constructor:

## Building Controllers

To adopt this model of controller construction, implementations should start
with the following controller constructor:

```go
import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)

	// TODO(you): Access informers

	c := &Reconciler{
		// TODO(you): Pass listers, clients, and other stuff.
	}
	impl := controller.NewImpl(c, logger, "NameOfController")

	// TODO(you): Set up event handlers.

	return impl
}
```

### Generated Reconcilers

A code generator is aviable for simple subset of reconciliation requirements. A
label above the API type will signal to the injection code generator to generate
a strongly typed reconciler. Use `+genreconciler` to generate the reconcilers.

```go
// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ExampleType struct {
	...
}
```

`+genreconciler` will produce a helper method to get a controller impl.

Update `NewController` as follows:

```go
"knative.dev/pkg/controller"
...
impl := controller.NewImpl(c, logger, "NameOfController")
```

becomes

```go
reconciler "knative.dev/<repo>/pkg/client/injection/reconciler/<clientgroup>/<version>/<resource>"
...
impl := reconciler.NewImpl(ctx, c)
```

See
[Generated Reconciler Responsibilities](#generated-reconciler-responsibilities)
for more information.

## Implementing Reconcilers

Type `Reconciler` is expected to implement `Reconcile`:

```go
func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	...
}
```

### Generated Reconcilers

If generated reconcilers are used, Type `Reconciler` is expected to implement
`ReconcileKind`:

```go
func (r *Reconciler) ReconcileKind(ctx context.Context, o *samplesv1alpha1.AddressableService) reconciler.Event {
    ...
}
```

See
[Generated Reconciler Responsibilities](#generated-reconciler-responsibilities)
for more information.

## Consuming Informers

Knative controllers use "informers" to set up the various event hooks needed to
queue work, and pass the "listers" fed by the informers' caches to the nested
"Reconciler" for accessing objects.

Our controller constructor is passed a `context.Context` onto which we inject
any informers we access. The accessors for these informers are in little stub
libraries, which we have hand rolled for Kubernetes (more on how to generate
these below).

```go
import (
	// These are how you access a client or informer off of the "ctx" passed
	// to set up the controller.
	"knative.dev/pkg/client/injection/kube/client"
	svcinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/service"

	// Other imports ...
)

func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)

	// Access informers
	svcInformer := svcinformer.Get(ctx)

	c := &Reconciler{
		// Pass the lister and client to the Reconciler.
		Client:        kubeclient.Get(ctx),
		ServiceLister: svcInformer.Lister(),
	}
	impl := controller.NewImpl(c, logger, "NameOfController")

	// Set up event handlers.
	svcInformer.Informer().AddEventHandler(...)

	return impl
}

```

> How it works: by importing the accessor for a client or informer we link it
> and trigger the `init()` method for its package to run at startup. Each of
> these libraries registers themselves similar to our `init()` and controller
> processes can leverage this to setup and inject all of the registered things
> onto a context to pass to your `NewController()`.

## Testing Controllers

Similar to `injection.Default`, we also have `injection.Fake`. While linking the
normal accessors sets up the former, linking their fakes set up the latter.

```go
import (
	"testing"

	// Link the fakes for any informers our controller accesses.
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/service/fake"

	"k8s.io/client-go/rest"
	"knative.dev/pkg/injection"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestFoo(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	// Setup a context from all of the injected fakes.
	ctx, _ = injection.Fake.SetupInformers(ctx, &rest.Config{})
	cmw := configmap.NewStaticWatcher(...)
	ctrl := NewController(ctx, cmw)

	// Test the controller process.
}
```

The fake clients also support manually setting up contexts seeded with objects:

```go
import (
	"testing"

	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"

	"k8s.io/client-go/rest"
	"knative.dev/pkg/injection"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestFoo(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	objs := []runtime.Object{
		// Some list of initial objects in the client.
	}

	ctx, kubeClient := fakekubeclient.With(ctx, objs...)

	// The fake clients returned by our library are the actual fake type,
	// which enables us to access test-specific methods, e.g.
	kubeClient.AppendReactor(...)

	c := &Reconciler{
		Client: kubeClient,
	}

	// Test the reconciler...
}
```

## Starting controllers

All we do is import the controller packages and pass their constructors along
with a component name (single word) to our shared main. Then our shared main
method sets it all up and runs our controllers.

```go
package main

import (
	// The set of controllers this process will run.
	"github.com/knative/foo/pkg/reconciler/bar"
	"github.com/knative/baz/pkg/reconciler/blah"

	// This defines the shared main for injected controllers.
	"knative.dev/pkg/injection/sharedmain"
)

func main() {
	sharedmain.Main("componentname",
       bar.NewController,
       blah.NewController,
    )
}

```

## Generating Injection Stubs.

To make generating stubs simple, we have harnessed the Kubernetes
code-generation tooling to produce `injection-gen`. Similar to how you might
ordinarily run the other `foo-gen` processed:

```shell
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${REPO_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/knative/sample-controller/pkg/client github.com/knative/sample-controller/pkg/apis \
  "samples:v1alpha1" \
  --go-header-file ${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt
```

To run `injection-gen` you run the following (replacing the import path and api
group):

```shell

KNATIVE_CODEGEN_PKG=${KNATIVE_CODEGEN_PKG:-$(cd ${REPO_ROOT}; ls -d -1 ./vendor/knative.dev/pkg 2>/dev/null || echo ../pkg)}

${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh "injection" \
  github.com/knative/sample-controller/pkg/client github.com/knative/sample-controller/pkg/apis \
  "samples:v1alpha1" \
  --go-header-file ${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt

```

To ensure the appropriate tooling is vendored, add the following to
`Gopkg.toml`:

```toml
required = [
  "knative.dev/pkg/codegen/cmd/injection-gen",
]

# .. Constraints

# Keeps things like the generate-knative.sh script
[[prune.project]]
  name = "knative.dev/pkg"
  unused-packages = false
  non-go = false
```

## Generated Reconciler responsibility

The goal of generating the reconcilers is to provide the controller implementor
a strongly typed interface, and ensure correct reconciler behaviour around
status updates, Kubernetes event creation, and queue management.

We have already helped the queue management with libraries in this repo. But
there was a gap in support and standards around how status updates (and retries)
are performed, and when Kubernetes events are created for the resource.

The general flow with generated reconcilers looks like the following:

```
[k8s] -> [watches] -> [reconciler enqeueue] -> [Reconcile(key)] -> [ReconcileKind(resource)]
            ^-- you set up.                          ^-- generated       ^-- stubbed and you customize
```

The responsibility and consequences of using the generated
`ReconcileKind(resource)` method are as follows:

- In `NewController`, set up watches and reconciler enqueue requests as before.
- Implementing `ReconcileKind(ctx, resource)`
- Resulting changes from `Reconcile` calling `ReconcileKind(ctx, resource)`:
  - DO NOT edit the spec of `resource`, it will be ignored.
  - DO NOT edit the metadata of `resource`, it will be ignored.
  - If `resource.status` is changed, `Reconcile` will synchronize it back to the
    API Server.
    - Note: the watches setup for `resource.Kind` will see the update to status
      and cause another reconciliation.
- `ReconcileKind(ctx, resource)` returns a
  [`reconciler.Event`](../reconciler/events.go) results in:
- If `event` is an `error` (`reconciler.Event` extends `error` internally),
  `Reconciler` will produce a `Warning` kubernetes event with _reason_
  `InternalError` and the body of the error as the message.
  - Additionally, the `error` will be returned from `Reconciler` and `key` will
    requeue back into the reconciler key queue.
- If `event` is a `reconciler.Event`, `Reconciler` will log a typed and
  reasoned Kubernetes Event based on the contents of `event`.
  - `event` is not considered an error for requeue and nil is returned from
    `Reconciler`.
- If additional events are required to be produced, an implementation can pull a
  recorder from the context: `recorder := controller.GetEventRecorder(ctx)`.

Pending Features:

- Leverage `configStore` and specifically `ctx = r.configStore.ToContext(ctx)`
  inside `Reconcile`.
- Resulting changes from `Reconcile` calling `ReconcileKind(ctx, resource)`:
  - If resource.status.finalizers are updated, `Reconcile` will synchronize it
    back to the API Server.
  - If `resource.metadata.labels` or `.annotations` are updated, `Reconcile`
    will synchronize it back to the API Server.
