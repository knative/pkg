# Mako Quickstore Guide

This guide will walk you through using the Mako Quickstore client to store
performance test data in Mako and perform regression detection.

If you haven’t yet read [CONCEPTS.md](./CONCEPTS.md), please do before
proceeding.

> **NOTE**: Mako performance tests write data to the Mako service,
https://mako.dev. Running example performance tests, or authoring your own, will
require special access to the service. Please read ACCESS.md before proceeding.

## Preparing your performance test data

Before you can use Mako, you must have performance test data which you want to
store in https://mako.dev and which you’ll use to look for performance
regressions. This performance data can come from anywhere -- microbenchmarks,
load tests, etc.

The performance test data Mako understands can be classified into three
categories: **Sample Point Data**, **Custom Aggregates**, and **Run Metadata**.

#### Sample Point Data

Repeated measurements of a set of properties over the course of a performance
test. Each Sample Point consists of:
* An input value for the x-axis, usually a timestamp. This is usually the time
  associated with the sample point.
* A set of metrics. Each metric is a key-value pair representing a measured
  property.

Here’s an example of some sample point data, sorted by timestamp, consisting of
three metrics:

Input Value             | Write Latency (ms) | Read Latency (ms) | Instantaneous CPU Load
----------------------- | ------------------ | ----------------- | -------------
2011-09-22 17:47:08.128 | 258                |                   | 0.13
2011-09-22 17:47:08.386 |                    | 737               | 0.19
2011-09-22 17:47:09.123 | 1256               |                   | 0.28
2011-09-22 17:51:10.379 |                    | 455               | 0.34
2011-09-22 17:51:10.834 | 279                |                   | 0.38
2011-09-22 17:52:11.133 |                    | 383               | 0.51

When using Quickstore, sample point data is added to Mako using the
`AddSamplePoint` method (see
[Using the Mako Quickstore client](#using-the-mako-quickstore-client)).

#### Custom Aggregate Data

A set of key-value aggregates. Each key-value pair represents a single
measurement for the entire run. Here’s an example of some custom aggregate data:

Aggregate Metric          | Value
------------------------- | ------------------
Average Throughput (KB/s) | 4312.84
Branch miss percentage    | 1.048
Page faults               | 42383

When using Quickstore, sample point data is added to Mako using the
`AddCustomAggregate` method (see
[Using the Mako Quickstore client](#using-the-mako-quickstore-client)).

#### Run Metadata

Mako runs can have a diverse set of metadata associated with them, including:
* timestamp
* duration
* tags (used for filtering runs in queries and charts)
* description

When using Quickstore, this data is populated via the `QuickstoreInput` object
passed to the Quickstore client
constructors ([C++](../helpers/cxx/quickstore/quickstore.go),
[Go](../helpers/cxx/quickstore/quickstore.h)). See the comments on 
`QuickstoreInput` in
[quickstore.proto](../proto/quickstore/quickstore.proto) for the full
set of supported run metadata information.

Now that you understand the kinds of data that Mako can work with, think about
how your data can be made to fit into Mako’s concepts. By the end of this guide,
we’ll be using a Mako Quickstore client to upload your data to https://mako.dev.

#### Setting up authentication

The Mako command-line tool and the Mako clients communicate https://mako.dev
using the Application Default Credentials (ADC) strategy. You can find full
documentation of this strategy at http://cloud/docs/authentication/production.

To learn how to establish credentials to authenticate to https://mako.dev, read
[AUTHENTICATION.md](AUTHENTICATION.md).

#### Preparing your benchmark

Now that you have an idea of the kinds of data you’ll be storing in Mako and
you’ve set up authentication, it’s time to create your benchmark in
https://mako.dev. Before proceeding, see [ACCESS.md](./ACCESS.md) to learn how to
get access to create benchmarks.

To create the benchmark you’ll use the Mako command-line tool. Visit
[BUILDING.md](BUILDING.md#building-the-command-line-tool) to learn how to build
the command-line tool.

```bash
$ alias mako=<your mako directory>/bazel-bin/internal/tools/cli/mako
```

You can see all the CLI commands using the ‘help’ subcommand:
```bash
$ mako help
```

We’re going to use the `create_benchmark` subcommand. To see the help for this
subcommand:
```bash
$ mako help create_benchmark
```

Let’s leave the path blank so that we can create the benchmark from a template.
Execute:
```bash
$ mako create_benchmark
```

This will bring the template up in your shell’s default editor. For the example
data in the
[Preparing your performance test data](#preparing-your-performance-test-data)
section above, we might fill out the template as follows. You should replace the
configuration with a description of your own data.

```
benchmark_name: "Example Benchmark"

project_name: "Mako Example Project"

owner_list: "yourusername@yourdomain.com"
owner_list: "anotheruser@yourdomain.com"

input_value_info: <
  value_key: "t"
  label: "time"
  type: TIMESTAMP
>

# value_key: should be short and should not change. Tests will write points with this key.
# label: human-readable label to show on charts. Can can changed.
metric_info_list: <
  value_key: "w"
  label: "WriteLatency_ms"
>
metric_info_list: <
  value_key: "r"
  label: "ReadLatency_ms"
>
metric_info_list: <
  value_key: "c"
  label: "CPULoad"
>
custom_aggregation_info_list: <
  value_key: "tp"
  label: "Throughput"
>
custom_aggregation_info_list: <
  value_key: "bmp"
  label: "BranchMissPercent"
>
custom_aggregation_info_list: <
  value_key: "pf"
  label: "PageFaults"
>
```

Notice how we configured the run with ways of representing both the sample point
information (the `metric_info_list` items) and the custom aggregate information
(the `custom_aggregation_info_list` items).

Now save and quit your editor. Assuming there are no syntax errors or other
issues with your data, the `create_benchmark` subcommand should complete
successfully and report a benchmark key. Find the benchmark on https://mako.dev
by copying that benchmark key and visiting 'https://mako.dev/b/BBBBBBB',
replacing *BBBBBBB* with the benchmark.

The sections below will walk you through setting up code that writes actual
performance test results to this benchmark.

#### Depending on Mako

When using Mako to store performance results in https://mako.dev and to perform
regression detection, you must import the Mako Quickstore library into your own
code. How you go about that depends on your build system. Mako supports two
build systems: Bazel for C++ and Golang, and `go build` for Golang.

##### Bazel

If you use Bazel (for either C++ or Go), you can import Mako as a dependency in
your WORKSPACE file. Find directions at [BUILDING.md](BUILDING.md#bazel), and to
learn more about Bazel, visit http://bazel.build.

##### go build/test

If you are using Go and you build with `go build` or `go test`, we recommend
using [Go Modules](https://github.com/golang/go/wiki/Modules) to depend on Mako.
If you’re using an alternate dependency management system like
[dep](https://github.com/golang/dep), follow that tool’s typical procedure for
importing/vendoring a new dependency.

To import Mako using Go Modules, simply add Mako imports to your `.go` code as
needed:

```bash
cat <<EOF > mako_test.go
package main

import (
  "context"
	"fmt"
	"testing"
	"github.com/google/mako/helpers/go/quickstore"
	qpb "github.com/google/mako/proto/quickstore/quickstore_go_proto"
)

func TestPerformance(t *testing.T) {
	// This is just a stub for now to get the import working, we’ll fill it out later.
	_, _, _ := quickstore.NewAtAddress(context.Background(), "localhost:9813",&qpb.QuickstoreInput{})
	fmt.Println("Imported Quickstore")
}
EOF
```

Initialize your module:

```bash
$ go mod init example.com/your/mako/test
```

Then build and run, and Go should take care of importing the Mako module:

```bash
$ go test
```

Note that you need to ensure the
[Quickstore microservice](#quickstore-microservice) is running before starting
your test that uses Quickstore.

#### Quickstore microservice

The Go Quickstore client library, when building with `go build/test`, does not
stand alone -- it requires a running Quickstore microservice. When using Bazel,
the Quickstore client library is completely self-contained, so C++ and Go Bazel
users can ignore this section. Read more about the need for the microservice
in [CONCEPTS.md](./CONCEPTS.md#microservice).

The microservice is a C++ binary that is built with Bazel. For building
directions, see
[Building the Quickstore microservice](./BUILDING.md#building-the-quicksotre-microservice).

Once the microservice is built, run it with the `addr` flag at which it should
listen for client connections:

```bash
$ MAKO_PORT=9347  # could be any port
$ bazel run internal/quickstore_microservice:quickstore_microservice_mako -- --addr="localhost:${MAKO_PORT}"
```

Note that this command will fail if you haven’t set up authentication as
described in [AUTHENTICATION.md](AUTHENTICATION.md).

You will need to arrange for the microservice to be built by your test (or
pulled from a prebuilt location) and started, so that it listens for client
connections, whenever you run a Go Mako Quickstore test.

#### Quickstore microservice as a Docker image

You can alternatively build the Quickstore microservice into a Docker container.
Skip this step if you are happy with the microservice as a binary.

To build the microservice into a Docker image that can be loaded locally or
pushed to a repository:

> **WARNING**:  Docker does not run natively in OSX, so building the image from
> OSX will require cross-compiling for Linux. We have not yet determined how to
> configure Bazel accordingly, so for now we recommend only building the
> microservice in Linux.

```bash
$ bazel build internal/quickstore_microservice:quickstore_microservice_mako_image.tar
```

Documentation about the Docker Bazel rules can be found at
https://github.com/bazelbuild/rules_docker#using-with-docker-locally.

To load the tar file output by the above command as a local Docker container:

```bash
$ docker load -i bazel-bin/internal/quickstore_microservice/quickstore_microservice_mako_image.tar
```

The image is loaded and ready to run. Inside the container, the microservice
will listen on the `9813` port for incoming connections. We can map that to
a specific external port with the `-p` flag: `-p ${MAKO_PORT}:9813`.

Also, the Docker container's environment is going to need access to your
credentials for authentication. Read about making credentials available to the
Docker container in
[AUTHENTICATION.md](AUTHENTICATION.md#authenticating-from-a-docker-container).

The full `docker run` command will look something like:

```bash
$ MAKO_PORT=9347  # could be any port
$ docker run --rm -v ~/.config/gcloud/application_default_credentials.json:/root/adc.json -e "GOOGLE_APPLICATION_CREDENTIALS=/root/adc.json" -p ${MAKO_PORT}:9813 bazel/internal/quickstore_microservice:quickstore_microservice_mako_image
```

As mentioned above, you will need to arrange for the microservice image to be
built by your test (or pulled from a prebuilt location) and started, so that it
listens for client connections, whenever you run a Go Mako Quickstore test.

#### Using the Mako Quickstore client

Now that you’ve prepared your performance test data, established authentication,
prepared your benchmark, are pulling in Mako as a dependency, and (if you are
using Go) have started the microservice, you’re ready to write some data to
https://mako.dev using Quickstore.

The typical structure of a Quickstore run is:
1. Collect some performance data. Quickstore doesn’t care where it comes from,
   just that you can represent it in the forms described above in
   [Preparing your performance test data](#preparing-your-performance-test-data).
2. Configure a `QuickstoreInput`
   ([quickstore.proto](../proto/quickstore/quickstore.proto)) object
   with your run metadata described above (#run-metadata).
3. Also in the `QuickstoreInput` object, configure your run analyzers. If you’re
   just getting started, **skip this step** until you’ve got a test that runs
   and records results to https://mako.dev. Once you’re uploading data for long
   enough to get a sense of the performance characteristics of your system under
   test, then consider adding analyzers in order to automate regression
   detection. To read more about analyzers, visit
   [ANALYZERS.md](ANALYZERS.md).
4. Instantiate a Mako Quickstore client, passing the constructor the
   `QuickstoreInput` object.
5. Call the `AddSamplePoint` method repeatedly, feeding it your sample point
   data.
6. Call the `AddCustomAggregate` method repeatedly, feeding it your custom
   aggregate data.
7. Call the `Store` method to process the data and upload it to https://mako.dev.

The examples in [`examples/`](../examples/) illustrate these steps.

#### Add Regression Detection

As mentioned in step 3 above, consider skipping adding Mako analyzers for
regression detection until you’ve been automated your test and have been
collecting data for a while. Once you feel you understand the *status quo* of
your system’s performance, then you should strongly consider integrating
analyzers.

In
[`examples/go_quickstore/example_test.go`](../examples/go_quickstore/example_test.go)
we configure a threshold analyzer. This is the simplest analyzer conceptually
and the easiest to configure. Most tests should start with a threshold analyzer
and expand from there.

Note that in
[`examples/go_quickstore/example_test.go`](../examples/go_quickstore/example_test.go)
we fail the test when Quickstore reports an analyzer failure. This allows us to
treat the performance test like a correctness/functional test regarding how
failures are reported.
