# Window Deviation Analyzer (WDA)

## Overview

WDA is a trend deviation analyzer that looks for recent changes in an aggregate
value (e.g. mean, median, percentile, or custom aggregate). It is particularly
useful when performance data is noisy. Some key features of WDA include:

*   Computes a "delta" by splitting the aggregate values into two
    [windows](#windows), recent and historic, and subtracting the average
    historic value from the average recent value. Either mean or median can be
    used to compute the averages.
*   If the delta is not within a user-configured tolerance, the run being
    analyzed is declared a regression.

As a user you need to [configure](#configuration):

*   The query used for runs prior to the current run.
*   The filter(s) for aggregate values to be compared across runs. For example,
    the median for metric y1.
*   The size of the recent window
    ([described below](#recommendations-for-recent-window-size)).
*   An optional direction bias (bigger is better, smaller is better).
*   A tolerance via multiple parameters ([described below](#mean_vs_median)).

[TOC]

## Configuration

The WDA analyzer is configured using
[window_deviation.proto](../clients/proto/analyzers/window_deviation.proto).
The remainder of this page describes how to set the fields therein.

## Quick Start

Follow the advice in this section alone for a quick example of creating a WDA
that will work for most cases. If your case is uncommon or you desire more
knowledge and control, read the rest of this document.

1.  Choose the metrics you would like to analyze. These are normally the most
    important metrics that directly measure performance.

2.  Choose the aggregate type you would like to analyze. Median works best in
    most cases, but see the [Selecting aggregate data via filter](#data_filter)
    section below for details.

3.  Choose whether you will compare against the historic baseline mean or
    median. This is a second level aggregate on top of step 2. For example, you
    might want to compare the Median of Medians or the Mean of the Medians.
    Median works best in most cases, but see the
    [MeanToleranceParams vs. MedianToleranceParams](#mean_vs_median) section
    below for details.

4.  Choose the size of your recent and historic windows. These two windows will
    be compared, and the historic window is considered the baseline. A recent
    window size of 7 and historic window size of 100 usually works well and is
    resilient to outliers when comparing against baseline median. In this case,
    a regression will not be declared unless 4 out of the 7 recent runs look bad
    when compared to baseline. See the [Historic and Recent Windows](#windows)
    section below for more details.

5.  Choose your [tolerance](#tolerance) parameters. These values define the
    acceptable delta (relative percent change) between the historic and recent
    windows. When using median for baseline comparisons, the following values
    are often a good choice:

    *   `const_term: 0.00`: Warning: If your values are very small, you should
        set this to a minimum tolerance. See the ["const_term"](#const-term)
        section below.
    *   `median_coeff: 0.05`: looks for more than 5% change.
    *   `mad_coeff: 1.0`: allows for noise as seen in historic data.

6.  Choose a direction bias. This lets you ignore either increases or decreases
    in value. If you are measuring latency, you want to ignore decreases. If you
    are measuring throughput, you want to ignore increases.

7.  Consider applying [tags](CONCEPTS.md#tags)
    to “official” runs. The analyzer query can be configured to only retrieve
    runs with certain tags, so you can easily prevent random/experimental
    executions of a test from polluting the baseline history.

8.  Wait until about 20-50 runs have built up in history before enabling the
    analyzer. This provides a decent baseline for the analyzer.

Using the basic suggestions above, here is an example in Python for latency
regression detection of somewhat noisy results:

```python
analyzers = []
...
metrics = ['y1', 'y2']
metric_checks = []
tolerance = window_deviation_pb2.MedianToleranceParams(median_coeff=0.05,
                                                       mad_coeff=1.0)
for metric in metrics:
  metric_checks.append(window_deviation_pb2.ToleranceCheck(
      data_filter=mako_pb2.DataFilter(
          data_type=mako_pb2.DataFilter.METRIC_AGGREGATE_MEDIAN,
          value_key=metric),
      recent_window_size=7,
      median_tolerance_params_list=[tolerance],
      direction_bias=window_deviation_pb2.ToleranceCheck.IGNORE_DECREASE))
analyzers.append(window_deviation.Analyzer(
    window_deviation_pb2.WindowDeviationInput(
        run_info_query_list=[mako_pb2.RunInfoQuery(
            benchmark_key=<your benchmark key here>,
            tags=['env=baseline'],  # or whatever tag(s) you desire
            limit=106],  # sets historic size to 100
        tolerance_check_list=metric_checks)))
...
pg_main = load_main.LoadMain(...)
pg_main.SetAnalyzerFactory(
   passthrough_analyzer_factory.PassthroughAnalyzerFactory(analyzers))
```

## Historic and Recent Windows {#windows}

All run data is placed in one of two windows. The **historic window** is used as
the baseline for comparison. The **recent window** is the set of runs that
includes at least the current run being analyzed as well as a configurable count
of additional recent runs.

*   The size of the **recent window** is equal to the `recent_window_size` proto
    field.

*   The size of the **historic window** is derived indirectly from the
    `run_info_query_list` and `recent_window_size` fields.

The total number of runs supplied to the analyzer is equal to the count of the
query results plus one (the current run being analyzed). So, the historic window
size is:

`count(query_results) + 1 - recent_window_size`

The most convenient way to specify your queries is to provide a single query
that minimally sets the `RunInfoQuery.benchmark_key` and `RunInfoQuery.limit`
fields. You can create more complex queries you need them. We recommend that you
use [tags](/testing/performance/mako/g3doc/guide/tags.md) for these queries
to help isolate environments under analysis.

For example, with these configurations:

```proto
query: RunInfoQuery.benchmark_key: "123"
       RunInfoQuery.limit: 22
       RunInfoQuery.tags: "sometag=x"
recent_window_size: 3
```

The historic window size is 20 and the recent window size is 3. An illustration
of this case: <!-- TODO: turn this ascii art into a real image -->

```
|[<<<<<<<<<<<<<<<< Historic Window >>>>>>>>>>>>>>>>]  [ Recent Window ]
|
|               *8
|     *3  *5      *9      *12            *17
| *1          *7       *11         *15           *20
|   *2  *4          *10      *13      *16   *18
|           *6                                 *19
|                               *14                         *22
|                                                      *21      *23 (current)
|----------------------------------------------------------------------->
                             run timestamp
```

### Recommendations for historic window size

The size of the historic window should be balanced to take the points listed
below into consideration as well as your desire to be alerted:

*   The size of the historic window must be at least 3 (this default value is
    configurable via the `minimum_historical_window_size` field), or the
    analyzer may fail due to insufficient data, depending on the
    `data_filter.ignore_missing_data` values on the `ToleranceCheck` fields.
*   For noisy data, the window should be large enough to capture the typical
    range of values.
*   The larger the historic window, the longer it may take for WDA to adjust to
    a new trend and stop flagging it as a regression.
*   Each run can be up to 1 MB in size, so be sure that you have enough memory
    when choosing a large historic window.

### Recommendations for recent window size

The size of the recent window should be balanced to take the points listed below
into consideration as well as your desire to be alerted:

*   If the recent window is small (1-3), a small number of outlier aggregates
    may trigger a regression.
*   If the recent window is large, it may take longer for a regression to be
    identified.
*   If you choose the median-based tolerance (`MedianToleranceParams`), set the
    recent window size to `(2N + 1)`, where `N` is the maximum number of outlier
    aggregates you wish to tolerate in the recent window.
*   When running the test as part of a release process, you may desire to
    override your normal recent window to a size of one. This will let you spot
    a regression in a single test execution. However, if the test result has an
    outlier aggregate, you may need to execute the test multiple times to rule
    out a regression in the release.

## Detecting performance creep

If you want to catch performance creep (very slow performance changes), you need
to configure your queries and windows slightly differently.

For example, let’s say you want to see if performance has changed from 3 months
ago. You don’t want to have a historical window that covers the last 3 months as
that would mean you’re comparing the recent window to all the runs of the last 3
months. Instead you’ll want to use a typical historical window size, often 100,
in combination with a run info query that only retrieves runs from ~3 months
ago.

### Example

This can easily be configured in a [`RunInfoQuery`](../spec/proto/mako.proto)
using the `min_timestamp_ms` and `max_timestamp_ms` fields to define a time
range filter on runs fetched, or using the alternative `min_build_id` and
`max_build_id` fields with `run_order = BUILD_ID` to define a build ID range
filter on runs fetched. To achieve this you will need to use a pair of
`RunInfoQuery` instances, one to bring the historical data and one to bring the
recent data.

The first `RunInfoQuery` should be configured to a window a while back, using
either `min_timestamp_ms` and `max_timestamp_ms` or `min_build_id` and
`max_build_id`. The second `RunInfoQuery` should be configured with the `limit`
value to indicate how many recent samples you want. Then the
`recent_window_size` in the
[`ToleranceCheck`](../clients/proto/analyzers/window_deviation.proto)
should be the same as the `limit` value in the second `RunInfoQuery`, so all
recent data would be used as "recent".

Python example using a time-based window:

```python
wda_input = window_deviation_pb2.WindowDeviationInput(name='wda_creep')

# First query brings 3 days of data from 3 months ago
wda_run_info_query1 = mako_pb2.RunInfoQuery(
    benchmark_key=benchmark_key,
    min_timestamp_ms=now_ms-90_DAYS_MS,
    max_timestamp_ms=now_ms-87_DAYS_MS,
    tags=mako_tags)

# Second query brings 20 most recent samples, to be used for recent_window
recent_samples = 20
wda_run_info_query2 = mako_pb2.RunInfoQuery(
    benchmark_key=benchmark_key, limit=recent_samples, tags=mako_tags)

wda_input.run_info_query_list.extend(
        [wda_run_info_query1, wda_run_info_query2])
....
....

# ToleranceCheck.recent_window_size=RunInfoQuery2.limit
wda_tol_check = window_deviation_pb2.ToleranceCheck(
    data_filter=data_filter,
    recent_window_size=recent_samples,
    median_tolerance_params_list=[wda_tolerance_params])
```

## Multiple configurations

You may find that no single configuration does everything you want. For example
you might want to detect:

*   Very large changes immediately. You use a historic window size of 100 and a
    recent window size of 1.
*   Moderate changes within a few runs while being resistant to noise. You use a
    historic window size of 100 and a recent window size of 7.

These are just a few ideas with potential values and shouldn’t be seen as the
only options or golden configurations.

## Tolerance {#tolerance}

For each `ToleranceCheck` field and the selected aggregate values, WDA computes
the following values:

*   `historic_mean`: the mean of historic values.

*   `historic_median`: the median of historic values.

*   `historic_stddev`: the stddev of historic values.

*   `historic_mad`: the median absolute deviation of historic values.

*   `recent_mean`: the mean of recent values.

*   `recent_median`: the median of recent values.

For each `*ToleranceParams` field of a `ToleranceCheck`, WDA computes the
relevant delta and tolerance:

```
If MeanToleranceParams (with fields const_term, mean_coeff, stddev_coeff):
  delta = recent_mean - historic_mean
  tolerance = const_term +
              mean_coeff * abs(historic_mean) +
              stddev_coeff * historic_stddev

If MedianToleranceParams (with fields const_term, median_coeff, mad_coeff):
  delta = recent_median - historic_median
  tolerance = const_term +
              median_coeff * abs(historic_median) +
              mad_coeff * historic_mad
```

When checking for a regression, WDA compares the delta and tolerance:

```
if abs(delta) > tolerance:
   regression = true
```

Also see the [direction bias](#direction-bias) section below, which may
supersede the tolerance check.

The configuration parameters supplied to `*ToleranceParams` fields are described
in detail below with some recommendations.

### MeanToleranceParams vs. MedianToleranceParams {#mean_vs_median}

These are the two options for defining tolerance ([see above](#tolerance)), and
you should choose one based on your desired analysis results. This section
describes each option in detail.

**Recommendation Summary**

*   If you want to ignore relatively small outlier aggregates but wish to be
    notified of large outlier aggregates, choose `MeanToleranceParams`.

*   If you want to ignore all small and large outlier aggregates, choose
    `MedianToleranceParams`.

*   If your data is very noisy, choose `MedianToleranceParams`.

#### MeanToleranceParams

`MeanToleranceParams` defines a tolerance based on historic mean and historic
standard deviation. This tolerance is compared to the mean delta (`recent -
historic`). `MeanToleranceParams` is based on mean and noise about the mean
(stddev), so it may be susceptible to outlier aggregates that are large enough
to skew the historic and/or recent means.

#### MedianToleranceParams

`MedianToleranceParams` defines a tolerance based on historic median and
historic median absolute deviation (MAD, see [`mad_coeff`](#mad_coeff) below).
This tolerance is compared to the median delta (`recent - historic`).
`MedianToleranceParams` is based on median and noise about the median (MAD), so
it will ignore outlier aggregates, even when they are very large. Both median
and MAD are generally more resilient to noise.

## const_term

`MeanToleranceParams.const_term` and `MedianToleranceParams.const_term` act as
constant terms of the tolerance equations, while other terms may vary with
historic data.

When used alone, `const_term` provides a simple, fixed tolerance.

When used in conjunction with other terms, `const_term` acts as a minimum
tolerance.

Using just `const_term` may be useful when your data is relatively noise free
and the constraints are very tight. For example, if a latency operation normally
takes 100ms +/- 5ms, and there is rarely noise, you could set `const_term` to 10
and leave other fields unset.

Either on its own or with other terms, the `const_term` can be useful for
defining a tolerance for measurements that are typically of very small
magnitude. For example, if a metric normally records a latency of 10ms, a 50%
increase in this latency may not warrant a regression. Setting `const_term` to
5ms would prevent a regression from being flagged in this case, regardless of
what the other parameters are set to.

## mean_coeff

`MeanToleranceParams.mean_coeff` is a coefficient of the historic mean term.

This term is useful to set a tolerance that is based on a percent deviation from
historic mean. For example, setting this to 0.05 says that the recent mean
should not deviate by more than 5% of the historic mean.

Using just the `mean_coeff` term is possible when your data is relatively noise
free. Just set it to a percentage of change that would be acceptable.

If your data has noise, you should also add the [`stddev_coeff`](#stddev_coeff)
term.

## stddev_coeff

`MeanToleranceParams.stddev_coeff` is a coefficient of the historic standard
deviation term.

This term is useful to add some amount of tolerance that is related to the
historic noise about the mean.

This term should most often be used in addition to one or both other terms,
where it serves as a margin of error about the mean.

Using just `stddev_coeff` is not recommended unless your data is guaranteed to
have noise, otherwise the term may approach zero and become too sensitive to
minor changes.

In most cases, we recommend setting this coefficient to 2.0. Twice the standard
deviation is often used for defining the margin of error for a normal
distribution, because about 95% of historic values will fall in the
historic_mean +/- 2 * `historic_stddev` range (see this
[Wikipedia article](https://en.wikipedia.org/wiki/Standard_deviation) on
standard deviation). So, if most historic points are close to mean, 2.0 works
well.

However, if your data is not well categorized as a normal distribution (e.g.
bimodal or uniform distribution), you may want to experiment with values in the
range 1.0 to 2.0.

## median_coeff

`MedianToleranceParams.median_coeff` is a coefficient of the historic median
term.

This term is useful to set a tolerance that is based on a percent deviation from
historic median. For example, setting this to 0.05 says that the recent median
should not deviate by more than 5% of the historic median.

Using just the `median_coeff` term is possible when your data is relatively
noise free. Simply set `median_coeff` to a percentage of change that would be
acceptable.

If your data has noise, you should also add the [`mad_coeff`](#mad_coeff) term.

## mad_coeff

`MedianToleranceParams.mad_coeff` is a coefficient of the historic median
absolute deviation (MAD) term. If you are unfamiliar with MAD, it is similar to
stddev, in that it provides a measure of variability. However, it is based on
median values and is a more robust measure of variability.

This term is useful to add some amount of tolerance that is related to the
historic noise about the median.

This term should most often be used in addition to one or both other terms,
where it serves as a margin of error about the median.

Using just `mad_coeff` is not recommended unless your data is guaranteed to have
noise, otherwise the term may approach zero and become too sensitive to minor
changes.

Setting this coefficient to 1.0 is recommended in most cases. The definition of
MAD implies that half of all historic window values will be within median +/-
1.0 * MAD. If half of all recent window values also fall within normal noise
levels, the same should be true of the recent values. Therefore, 1.0 is
recommended.

If your recent window is very large, the median of recent values may not be very
affected by noise, so you may choose to leave `mad_coeff` unset (0.0).

## Selecting aggregate data via filter {#data_filter}

Each `ToleranceCheck` accepts a single `data_filter` field. This field selects
the data that will be analyzed. For example, you could select metric y1's mean,
y2's median, or y3's 99th percentile. In most cases, you will select either the
mean or median. All `DataFilter.DataType` options except `METRIC_SAMPLEPOINTS`
are valid.

A `data_filter` also has an `ignore_missing_data` subfield. If set to true (the
default), then errors resulting from too little data do not cause analysis
failures. Instead, they are logged and analysis is allowed to continue. This
allows a Window Deviation Analyzer to be added to a Mako benchmark from the
beginning, instead of having to either deal with the first few runs failing or
waiting until enough runs have been generated and then adding the analyzer
separately. The minimum amount of data required for a check to proceed is
configured via the `minimum_historical_window_size` field.

**Recommendation Summary**

*   If you want to be alerted by significant outlier sample points in a run, use
    mean.

*   If you want all outlier sample points in a run to be ignored, use median.

*   Set `ignore_missing_data = true`.

**Details**

A large quantity and/or large magnitude of outlier sample points within a single
run may affect the run's mean enough to indicate a regression. These same
outliers will have no impact on the run's median.

Your choice of mean or median for filter is not related to your choice of
`MeanToleranceParams` or `MedianToleranceParams` for tolerance. If you filter by
mean and choose `MedianToleranceParams`, WDA compares the recent and historic
median of many run means. However, you can also filter by median and choose
`MedianToleranceParams`, which compares the recent and historic median of many
run medians.

## Direction Bias

The `ToleranceCheck.direction_bias` field provides a setting for direction bias:

*   `IGNORE_INCREASE` means that bigger is better (e.g. throughput), and when
    the delta is positive, it is never a regression.
*   `IGNORE_DECREASE` means that smaller is better (e.g. latency), and when the
    delta is negative, it is never a regression.
*   `NO_BIAS` (default) means that any absolute delta above tolerance is a
    regression.

## ANDing and ORing Tolerance Checks

In most cases, providing one or more `ToleranceChecks` where each check has a
single `*ToleranceParams` field is effective. In this case every one of the
checks must pass analysis, or the run will be flagged as a regression. However,
you can use more complex logic statements. For example, `WindowDeviationInput`
accepts a list of `ToleranceChecks`. If any one of these checks indicate a
regression, an overall regression is flagged for the analyzer. i.e. if each
check returns true for no regression:

```
okay = *ToleranceChecks_0 AND *ToleranceChecks_1 AND ... *ToleranceChecks_n
```

`ToleranceCheck` accepts a list of `*ToleranceParams`. A regression is only
flagged for the single `ToleranceCheck` when every one of the `*ToleranceParams`
result in regression. i.e. if each params field returns true for no regression:

```
okay = *ToleranceParams_0 OR *ToleranceParams_1 OR ... *ToleranceParams_n
```

Using the behavior described above, you can form complex statements like:

```python
okay =
  (delta for y1 mean is within tolerance_y1_mean) AND
  (delta for y1 99th percentile is within tolerance_y1_99) AND
  ((delta for y2 mean is within tolerance_y2_mean) OR
   (delta for y2 median is within tolerance_y2_median))
```
