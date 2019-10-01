# U-Test analyzer

You can use a two-sample U-Test to determine if two distributions have the same
central tendency within a given confidence level. Read the [Details](#details)
section for more about how U-Tests work.

The U-Test analyzer is appropriate when:

*   A/B testing.
*   Your samples contain more than 20 points.
*   Your sample points are independent (the occurrence of one does not affect
    the other).

[TOC]

## Configuration

The U-Test analyzer is configured using
[utest_analyzer.proto](../clients/proto/analyzers/utest_analyzer.proto).
The remainder of this page describes how to set the fields therein.

### Shift Value or Effect Size

When performing a U-Test, you will most likely want to ensure that detected
changes are no less than a certain magnitude. This is often referred to as the
Effect Size, but in Mako you can use `shift_value` or `relative_shift_value`
to roughly achieve the same goal.

If you want to detect both increases and decreases using `shift_value` you need
to configure two different U-Tests, one for increases and one for decreases, as
`shift_value` is not yet bidirectional (see b/79419586).

## Details

The Mann-Whitney U-Test is a nonparametric alternative to the common T-Test.
Unlike T-Test, it  can be used in cases where the sample distributions are not
normal (i.e. heavily skewed/contain outliers).
Instead of the mean, U-instead compares sums of ranks (see
[Central Tendency](#central-tendency) below).

Some suggested readings about U-Tests:

*   https://en.wikipedia.org/wiki/Mann%E2%80%93Whitney_U_test
*   http://www.itl.nist.gov/div898/handbook/prc/section3/prc35.htm

This implementation of the U-Test performs a
[continuity correction](https://en.wikipedia.org/wiki/Continuity_correction).

This analyzer compares two samples:

*   A (control)
*   B (variation or current).

Each sample is defined by one or more Mako runs. When many runs are combined
in a sample, the data is simply concatenated to form the sample, and order
within a sample is unimportant.

Multiple configurations can be supplied for the defined samples, and each
configuration results in an execution of U-Test analysis. Each configuration
selects A/B metric keys and U-Test parameters.

This enables A/B testing for many cases including the following:

*   A is run X. B is run X. Compare metric `y1` in A to `y2` in B.
*   A is run X. B is run Y. Compare metric `y1` in A to `y1` in B. Compare
    metric `y2` in A to `y2` in B.
*   A is the set of runs [X<sub>1</sub>, X<sub>2</sub>, ... X<sub>N</sub>]. B is
    the set of runs [Y<sub>1</sub>, Y<sub>2</sub>, ... Y<sub>N</sub>]. (X and Y
    runs may be interleaved A/B runs). Compare metric `y1` in A to `y1` in B.
    Compare metric `y2` in A to `y2` in B.

The `direction_bias` field can be set to `IGNORE_DECREASE` or `IGNORE_INCREASE`
if you only care about regressions in one direction.

### Limitations

*   Both sample sizes should be larger than 20. If either sample contains less
    than 20 points, a warning is recorded and the analysis continues. If either
    sample contains less than 3 points, the analysis is aborted.

## Statistical Caveats

### Representative Data

If your samples are not statistically
[independent](https://onlinelibrary.wiley.com/doi/full/10.1002/cem.2773) or they
are not
[representative](https://www.investopedia.com/terms/r/representative-sample.asp)
of the larger population, the U-Test may not give you meaningful results.
Performance variability or noise is often a reason why performance data may not
be independent or representative. It is important to critically consider whether
your samples meet these criteria. Be aware that performance data is often less
representative than you might expect and is unlikely to ever be truly
representative due to variability.

For example, say you have a new flag value that you want to test on an HTTP
server. You spin up your server with the old flag value and measure 1000
requests. You down it and spin it up with the new flag value and measure 1000
more requests. You take that data and run a U-Test with `IGNORE_DECREASE` and a
significance level of .05. The test fails and your reasonable takeaway is that
this result would happen no more than 5% of the time if the server performance
has not gotten worse.

While this might be true, it might also be possible that something else changed
(dropped network packets, busier host, etc). The more you can reduce these
confounding variables, the more representative and independent your samples will
be, and the more accurate your U-Test will be.

### Significance Levels and Sample Size

As you increase sample size you are able to detect changes of smaller magnitudes
and with more certainty (i.e. with a higher confidence level, lower significance
level). While this sounds attractive, you may detect tiny changes you don't care
about. If your data isn't truly representative, increasing the sample size may
lead to inaccurate U-Test results even with an extremely small significance
level.

The solution is not to reduce your sample size, but to instead to ignore
detected changes under a specified size. This is typically referred to as Effect
Size and in Mako, `shift_value` or `relative_shift_value` serve as a good proxy.

For more information on this topic, see
[Large Samples: Too Much of a Good Thing?](http://blog.minitab.com/blog/statistics-and-quality-data-analysis/large-samples-too-much-of-a-good-thing).

## Central Tendency

The central tendency of a sample or distribution can be thought of as its
typical value. The most common measures of central tendency are mean, median,
and mode. The T-Test uses the mean as the central tendency to compare, while the
U-Test uses the median. However, the U-Test does not strictly compare the
medians, instead it computes a *rank-sum* for both samples and compares those.

The rank-sum for each sample is calculated by first assigning ranks from `1` to
`n_1 + n_2`, where `n_i` is the size of sample `i`, to points in both samples
(`1` is assigned to the smallest value and `n_1 + n_2` to the largest) and then
computing the sum of the ranks of the points from that specific sample. This
process is illustrated below:

```
    Sample A  | rank |         Sample B  | rank |
       -1         1                3         3
        2         2                6         5
        5         4                7         6

   Sample A rank-sum: 1 + 2 + 4 = 7
   Sample B rank-sum: 3 + 5 + 6 = 14
```

These ranks-sums are then standardized and used to compute the *z-statistic*
which determines how different the central tendencies of the two samples
actually are.

Thus the variables [mentioned later](#example-configurations) in this document,
`a_sample_tendency` and `b_sample_tendency`, are not simply sample medians, but
they instead convey a more abstract idea of the sample's typical value in terms
of the U-Test. For example, `a_sample_tendency == b_sample_tendency` should not
be interpreted as `a_median == b_median`, but rather as the probability being
50% that a randomly selected point from sample A will be greater than a randomly
selected point from sample B.

The primary advantage of the U-Test over comparing medians is that U-Test can
take into account differences in the shape and spread of the two samples'
distributions. This advantage is discussed in more detail
[here](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC1120984/).

## Example Configurations

### Example 1

#### Goal

Compare metric Y1 in two sets of runs, making sure that sample B's central
tendency is at least as large as sample A's central tendency (within a 95%
confidence level).

#### Configuration

```proto
a_sample: {
  run_query_list: {
    run_key: "6237345368676776"
  }
  run_query_list: {
    run_key: "6345345349248329"
  }
  run_query_list: {
    run_key: "6235345345334543"
  }
}
b_sample: {
  run_query_list: {
    run_key: "3337345634534345"
  }
  run_query_list: {
    run_key: "3245346756757851"
  }
  include_current_run: true
}
config_list: {
  a_metric_key: "Y1"
  b_metric_key: "Y1"
  direction_bias: IGNORE_INCREASE
  significance_level: 0.05
}
```

### Example 2

#### Goal

Compare metric Y1 with metric Y2 from a single run, making sure that sample B's
central tendency is smaller than sample A's central tendency by 10 units (within
a 99% confidence level).

#### Configuration

```proto
a_sample: {
  include_current_run: true
}
b_sample: {
  include_current_run: true
}
config_list: {
  a_metric_key: "Y1"
  b_metric_key: "Y2"
  shift_value: -10
  direction_bias: IGNORE_DECREASE
  significance_level: 0.01
}
```
