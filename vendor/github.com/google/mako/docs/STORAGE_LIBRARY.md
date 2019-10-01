# Mako Storage Library

The Mako storage library can be used to directly interact with Mako storage
objects.

> **WARNING**: If you just want to add performance test data to Mako, you should
> use [Quickstore](CONCEPTS.md#quickstore) instead.

Authentication is required to use the Mako storage library. Read more at
[AUTHENTICATION.md](AUTHENTICATION.md).

The following objects can be manipulated using the Mako storage library:
* BenchmarkInfo - corresponding to a Mako [benchmark](CONCEPTS.md#benchmarks)
* RunInfo - corresponding to a Mako [run(CONCEPTS.md#runs)]
* SampleBatch - a bundle of sample point data from a run. Each run has up to 6
  of these.

> **NOTE** The definitions of each of those types can be found in
> [mako.proto](../spec/proto/mako.proto), along with per-type documentation in
> the comments.

The following operations can be performed on objects of the above types:
* Query - fetch specific objects by key, or fetch all objects matching a filter.
* Count - count objects matching a filter. This performs much better than a
  Query.
* Create
* Update
* Delete

> **NOTE**: Comprehensive documentation for each method is available in the
> source files for the interface types.
> * For the C++ storage library
>   [mako_client.h](../clients/cxx/storage/mako_client.h), see the per-method
>   documentation in [storage.h](../spec/cxx/storage.h).
> * For the Go storage library [mako.go](../clients/go/storage/mako.go), see the
>   per-method documentation in [storage.go](../spec/go/storage.go).

## Language and build system support

The Mako storage library is available in C++ and Go. Go with Bazel is supported
now, and Go with `go build/test` will be supported in the near future.

## Storage limitations

See [mako.proto](../spec/proto/mako.proto) for definitions of the types
mentioned below, as well as in-file documentation.

> **NOTE** Quickstore performs automatic downsampling of data to respect these
> server limits. If you're using the storage library directly, you'll have to
> do this yourself.

* A maximum of 100,000 runs per benchmark will retain sample data. If this
  maximum is reached, the system will prune the oldest runs periodically in
  order to maintain this limit for each benchmark. Pruned runs will have all
  SampleBatches associated with the run deleted, but the RunInfo entity will
  still be present (aggregates will be available, but individual data points
  will be missing). Also see the benchmark's BenchmarkInfo.retained_run_tag
  field to prevent special runs from getting deleted.
* A maximum of 1000 RunInfo creations per hour for a single benchmark. This
  limit is enforced via server failure response.
* A maximum of 20 RunInfo.tags values per run. This limit is enforced via server
  failure response.
* A maximum of 1 MB for a single serialized BenchmarkInfo, RunInfo, or
  SampleBatch. This limit is enforced via server failure response. The standard
  Downsampler handles the size of SampleBatch data.
* A maximum of 6 SampleBatchs per run . The limit is enforced via server failure
  response. The standard Downsampler handles this.
* A maximum of 50,000 metric values per run. This is the total count of
  SamplePoint.metric_value_list fields populated for all SampleBatchs written
  for a single run. The standard Downsampler handles this. The limit is enforced
  via server failure response.
* A maximum of 5,000 SampleErrors per run . The limit is enforced via server
  failure response. The standard Downsampler handles this.
* A maximum of 1000 Metrics (actually ValueInfos) per benchmark. This limit is
  enforced via server failure response.
* A maximum of 800 KB for SampleAnnotations per run. The limit is enforced in
  the standard Downsampler by dropping annotations that exceed the limit.

## Cursors and Limits

Cursors are used in Query calls to allow large queries to be broken up into
manageable chunks. When making Query calls, you should check for a cursor in
query responses to know if there's more data to fetch.

Limits are used when you want to manually control the amount of data you Query
or the number of objects you Delete.

Read details about using cursors and limits in
[mako.proto](../spec/proto/mako.proto) -- search for the term "Querying with and
without limits and cursors".
