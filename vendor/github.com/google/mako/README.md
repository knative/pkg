# Mako - A performance testing library and service

![Mako](./docs/images/mako_vertical_round_hdpi.png)

Mako is a set of tools for pre-production performance testing. It offers:

* data storage
* charting
* statistical aggregation
* automated regression analysis
* workflows for triaging potential regressions

Mako is narrowly focused on meeting the performance testing needs of Google open
source projects. At this time, Mako is not intended to be used by projects
outside this limited scope.

# Using Mako

## Performance tests that use Mako
Mako client libraries are designed to be called from a client project’s
performance test code. See the client project’s documentation for how to run
their performance tests.

Since Mako performance tests upload data to https://mako.dev, they require
access rights to run. Please see [ACCESS.md](docs/ACCESS.md).

## Mako Dashboard
Results from performance tests that use Mako are visible to the world. Visit
https://mako.dev to browse the results. Find dashboard documentation at
https://mako.dev/help.

## Mako Command-line Tool
The Mako command-line tool can be used to gain programmatic access to the same
Mako data that’s available on the dashboard. This tool is used by benchmark
owners to manage their benchmarks and runs.

To learn about how to use the Mako command-line tool, please read
[CLI.md](docs/CLI.md).

## Writing a new performance test using Mako
If you’re writing a new performance test that will use a Mako client to store
results in https://mako.dev and to guard against performance regressions, please
read [GUIDE.md](docs/GUIDE.md).

There are example performance tests in the [`examples/`](./examples)
folder.

## Accessing Mako data with the storage libaries
The most common programmatic use of Mako is to create a new run from performance
test data, and for that use case we recommend
[Quickstore](docs/CONCEPTS.md#quickstore). But if you need different kinds of
access to the data (e.g. updating or deleting existing data), you’ll want to use
the C++ or Go [storage library](docs/STORAGE_LIBRARY.md).
