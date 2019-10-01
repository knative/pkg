# Building Mako

Only building on Linux and MacOS is supported at this time.

## Prerequisites

#### Bazel

See Installing Bazel (https://docs.bazel.build/versions/master/install.html) for
instructions for installing Bazel on your system. Version 0.28.1 is known to
work. Version 0.23 is known to not work, so please upgrade Bazel if you are on
0.23 or earlier.

#### Git

See Installing Git
(https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).

## Cloning the repository
```bash
$ git clone https://github.com/google/mako
$ cd mako
```

## Building and running tests
```bash
$ bazel test ...
```

## Building with Mako as a dependency using Bazel (C++ or Go)

Import Mako as a dependency in your WORKSPACE file:

```
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "mako",
    remote = "https://github.com/google/mako.git",
    tag = "v0.0.1",
)
```

Then, to depend on a Mako Bazel target from your own Bazel targets:

for C++:
```
cc_library(
    name = "use_mako",
    srcs = ["use_mako.cc"],
    hdrs = ["use_mako.h"],
    deps = [
        "@mako//helpers/cxx/quickstore",
    ],
)
```

for Go:
```
go_library(
    name = "usemako",
    srcs = ["usemako.go"],
    importpath = "github.com/your/project/usemako"
    deps = [
        "@mako//helpers/go/quickstore",
    ],
)
```

## Building with Mako as a dependency using `go build/test`

You should depend on http://github.com/google/mako in the same way you depend on
other third-party libraries. This might involve vendoring, using a tool like
[dep](https://github.com/golang/dep), or using
[Go Modules](https://github.com/golang/go/wiki/Modules). We recommend using
Go Modules.

Regardless of the tool you use, when you import Mako from your `.go` files it
will look roughly like this:
```go
import (
	"github.com/google/mako/quickstore/go/quickstore"
	qpb "github.com/google/mako/quickstore/quickstore_proto"
)
```

Note when using `go build/test` that the Go client doesn't stand alone, it needs
to connect to a running Mako microservice. Learn more at
[CONCEPTS.md](CONCEPTS.md#microservice).

See the [GUIDE.md](GUIDE.md) for a step-by-step guide to writing and running a Mako
Quickstore test.

## Building the command-line tool

```bash
$ bazel build internal/tools/cli:mako
```

The binary will be found in `bazel-bin/internal/tools/cli/*/mako` (the wildcard
will differ based on your platform, check the Bazel output). You can copy this
binary into a more convenient location (e.g. somewhere on your $PATH).
Alternatively, run directly from Bazel:

```bash
$ bazel run internal/tools/cli:mako help
```

## Building the Quickstore microservice
```bash
$ bazel build internal/quickstore_microservice:quickstore_microservice_mako
```

The binary will be found in
`bazel-bin/internal/quickstore_microservice/quickstore_microservice_mako`. You
can copy this binary into a more convenient location (e.g. somewhere on your
$PATH). Alternatively, run directly from Bazel:

```bash
$ bazel run internal/quickstore_microservice:quickstore_microservice_mako
```

### Microservice Docker image

The Quickstore microservice can also be built into a Docker image:

> **WARNING**:  Docker does not run natively in OSX, so building the image from
> OSX will require cross-compiling for Linux. We have not yet determined how to
> configure Bazel accordingly, so for now we recommend only building the
> microservice in Linux.

```bash
$ bazel build internal/quickstore_microservice:quickstore_microservice_mako_image.tar
```

This image can then be loaded into a Docker client for running:

```bash
$ docker load -i bazel-bin/internal/quickstore_microservice/quickstore_microservice_mako_image.tar
```

The container can then be run:

```bash
$ docker run <docker arguments> bazel/internal/quickstore_microservice:quickstore_microservice_mako_image
```

Read more at
https://github.com/bazelbuild/rules_docker#using-with-docker-locally.

Note, when running a microservice that has been packaged as a Docker image,
you must pass the right set of flags so that the Docker container has access
to your credentials. Read more at
[AUTHENTICATION.md](AUTHENTICATION.md#authenticating-from-a-docker-container).
