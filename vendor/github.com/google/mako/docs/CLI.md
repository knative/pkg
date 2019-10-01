# Mako CLI

The Mako command-line interface (CLI) tool can be used to gain programmatic
access to the same Mako data that’s available on the dashboard. This tool is
used by benchmark owners to manage their benchmarks and runs.

Before you can use the Mako CLI, you’ll have to
[set up authentication](AUTHENTICATION.md).

To learn how to build the Mako command-line tool, please read
[BUILDING.MD](BUILDING.md#building-the-command-line-tool).

Once the command-line binary is built, use its built-in help to get a list of
subcommands:
```bash
mako help
```

As you might expect, each subcommand has its own help:
```bash
mako help display_benchmark
```

## Common operation: Creating a Benchmark

We’re going to use the `create_benchmark` subcommand. To see the help for this
subcommand:
```bash
mako help create_benchmark
```

Let’s leave the path blank so that we can create the benchmark from a template.
Execute:
```bash
mako create_benchmark
```

This will bring the template up in your shell’s default editor. Replace the
template text with your own information. To read about benchmarks and their role
in Mako, read [CONCEPTS.md](CONCEPTS.md#benchmarks).

Now save and quit your editor. Assuming there are no syntax errors or other
issues with your data, the `create_benchmark` subcommand should complete
successfully and report the https://mako.dev URL where you can find your
benchmark.

If you prefer to store a version of your benchmark, the `create_benchmark`
subcommand works with files too.

First use the `display_benchmark` subcommand to get the benchmark and write it
to file:
```bash
BENCHMARK=5251279936815104
BENCHMARK_PATH=~/myproject/my_benchmark.config
mako display_benchmark --benchmark_key=${BENCHMARK} > ${BENCHMARK_PATH}
```

Then you can save that config (e.g. in source control, along with your
performance test code). When you make changes to the file, write them back to
Mako:
```bash
mako update_benchmark ${BENCHMARK_PATH}
```

## Common operation: Listing Your Benchmarks

The `list_benchmarks` subcommand makes it easy to list the benchmark keys for
the benchmarks you own.

For example, to get all the benchmarks for your project, execute:

```bash
PROJECT=MakoExample
mako list_benchmarks --project_name=${PROJECT}
```

To get the benchmarks that you are an owner of, execute:

```bash
mako list_benchmarks --owner=youremail@example.com
```

## Common operation: Listing Runs With Filters

The `list_runs` subcommand lists run keys that match a filter.

For example, to get all the run keys for a benchmark, execute:
```bash
BENCHMARK=5675196467904512
mako list_runs --benchmark_key=${BENCHMARK}
```

To get only runs matching a set of tags, use the `--tag_list` flag:
```bash
TAGS=num_samplers=1,env=exclusive
mako list_runs --benchmark_key=${BENCHMARK} -tag_list=${TAGS}
```

Learn more about tags at [CONCEPTS.md](CONCEPTS.md#tags).

Use the subcommand help to learn about the other attributes you can filter by.
