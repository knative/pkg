# ocdiscover

This tool runs all known resource discovery mechanisms in the repository and
returns the result encoded in the well-known environment variables.

## Installation

Install the `ocdiscover` tool using:

```bash
go get contrib.go.opencensus.io/resource/cmd/ocdiscover
```

## Usage

To run an arbitrary command with resource information provided via the
OpenCensus environment varibales use:

```bash
env $(ocdiscovery) <command> [ <arg> ... ]
```

