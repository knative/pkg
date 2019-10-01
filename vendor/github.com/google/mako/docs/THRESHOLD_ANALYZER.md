# Threshold Analyzer

## Overview

The Threshold analyzer evaluates data that is within expected bounds. For
example, you might specify that 95% of requests to your SUT take between 1 and 2
seconds.

**Pros**

*   Good at preventing performance creep.
*   Doesn't require historic data.
*   Very simple to use and easy to understand.

**Cons**

*   Can be difficult to determine adequate thresholds.
*   Extreme thresholds could miss regressions.
*   SUT or SUT environment changes may require updates to the analyzer
    configuration.

## Configuration

The threshold analyzer is configured using
[threshold_analyzer.proto](../clients/proto/analyzers/threshold_analyzer.proto).

### Configuring historical context

When [triaging](ANALYZERS.md#analyzer-triage) a threshold analyzer
regression, it can be useful to compare the run that triggered the regression
detection to historical runs. Some historical runs, however, might represent
different modes, configurations, or environments that make them inappropriate
for comparison. As an example, a benchmark might be set up to use tags to
separate configurations like "10K requests" vs "1M requests" with tags such as
"request_size=10K" and "request_size=1M". The
[historical_context_tags](../clients/proto/analyzers/threshold_analyzer.proto?q=symbol:historical_context_tags)
field in the
[ThresholdAnalyzerInput](../clients/proto/analyzers/threshold_analyzer.proto?q=symbol:ThresholdAnalyzerInput)
proto message allows you to specify tags to filter on when graphing the
historical context.

## Implementation/Examples
* [C++](../examples/cxx_quickstore/example_test.cc)
* [Go](../examples/go_quickstore/example_test.go)
