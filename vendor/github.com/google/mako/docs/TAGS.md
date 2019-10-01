# Tags

## Overview

Tags are a mechanism to identify and group runs. Tags are applied to runs and
can be used to filter query results on Mako dashboards, analyzers, etc.

Tags can be used to identify any number of features and run characteristics,
but here are some of the most common.

*   Prod/staging/dev
*   Pass/fail analysis
*   Experiments
*   SUT configuration

## Filtering by tags

When filtering by tags, all tags are ANDed together. A query supplying tags
returns all runs where the query tags are a subset of the run tags. Each query
tag must be an exact, case-sensitive match of one of the run tags, and they do
not support regular expressions.

## Setting tags

You can set tags in one of three ways:

*   From a Mako Quickstore test, set the `tags`, `analysis_pass`, and/or
    `analysis_fail` fields on QuickstoreInput
    ([quickstore.proto](../proto/quickstore/quickstore.proto)).
*   Using the storage library, set the `RunInfo.tags` field when updating the
    run.
*   Use the [Mako Command Line Tool](CLI.md) to edit tags of
    existing runs.

## Tag best practices

1.  **Label Official Runs**: Always label official data/runs as such. Mako
    does not support negative tag matching (e.g. !dev) so it is important to
    explicitly tag official runs.
2.  **Key Value Tags**: Mako does not have any explicit support for key
    value tags today, but users still find it convenient and helpful to use
    key value style tags such as "env=prod", "env=test", etc. We recommend
    using "=" as your key value separator as it is easy to understand and we
    may support it in the future.
3.  **When to use Aux Data instead of tags**: If you don't plan to ever filter
    your runs by a piece of information then it is preferable to store it as
    [`RunInfo.aux_data`](../spec/proto/mako.proto) or a hyperlink in
    `RunInfo.hyperlink_list`.

### When to use tags vs creating new benchmarks

As a team's usage of Mako grows they often measure many related things and
wonder when they should create new benchmarks vs just using tags. For example,
let's say the Bigtable team wants to measure latency when reading and writing
data.

Here is a reasonable set of benchmarks and tags they might use.

Benchmark: single_row_read_latency

*  Tags for run 1: row_size_bytes=1024, bloomfilter=mode_a, cell=pc, env=staging
*  Tags for run 2: row_size_bytes=4096, cell=jd, env=test

Benchmark: table_scan_latency

*  Tags for run 1: num_rows=500, experiment=fast_scan, cell=vn, env=dev

Benchmark: single_row_write_latency

*  Tags for run 1: row_size_bytes=1024, cell=pc, env=staging

In theory, the team could have just made one benchmark called latency and had
tags like "mode=single_row_read" and "mode=table_scan". While technically
possible this results in benchmarks that are hard to use and can become a
dumping ground for all data and metrics.

Instead we recommend that dissimilar data, or data with vastly different
metrics are written to distinct benchmarks. In the example above, there is a
unique benchmark for each Bigtable testing scenario and then tags to
distinguish various configurations. These guidelines are fairly ambiguous and
different teams have found different balances. That said if you find yourself
with thousands of benchmarks or a single benchmark that contains many unrelated
metrics, you've probably gone too far in one direction.

## Adding tags to old runs

If you need to add tags to historical runs there are a number of programmatic
ways to do so. The recommended approach is to use the [Mako Storage API]
(#STORAGE_LIBRARY.md) and write a small amount of code.

## Tag limits

-   There is a limit of 20 tags per run (see limits at
    [STORAGE_LIBRARY.md](STORAGE_LIBRARY.md#storage-limitation)).
-   These are the only legal characters:
    *   Letters (A-Z, a-z)
    *   Numbers (0-9)
    *   Hyphen (-)
    *   Underscore (_) (cannot be first character)
    *   Equals sign (=)
    *   Period (.)
    *   Colon (:)
