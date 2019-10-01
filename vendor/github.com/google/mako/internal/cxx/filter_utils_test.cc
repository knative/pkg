// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// see the license for the specific language governing permissions and
// limitations under the license.
#include "internal/cxx/filter_utils.h"

#include <algorithm>
#include <array>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/repeated_field.h"
#include "gtest/gtest.h"
#include "spec/proto/mako.pb.h"

using mako::BenchmarkInfo;
using mako::RunInfo;
using mako::Aggregate;
using mako::RunAggregate;
using mako::MetricAggregate;
using mako::KeyedValue;
using mako::LabeledRange;
using mako::Range;
using mako::SampleBatch;
using mako::SamplePoint;
using mako::DataFilter;

const char* kbenchmark_key = "benchmark_key";
const char* krun_key = "run_key";
const double krun_timestamp_ms = 1234567;
const double krun_benchmark_score = 45;
const double kerror_count = 1;

// Ignore regions
// First num is start, second is end.
const double kignore_region_1[] = {0, 1};
const double kignore_region_2[] = {24, 200};

// Aggregate values
const std::array<int, 3> kpercentile_milli_rank{{70000, 80000, 90000}};

// Metric 1 values
const char* kmetric_1_key = "m1";
const double kmetric_1_count = 2;
const double kmetric_1_min = 1;
const double kmetric_1_max = 10;
const double kmetric_1_mean = 5;
const double kmetric_1_median = 5.5;
const double kmetric_1_stddev = 1.5;
const double kmetric_1_mad = 3.5;
const std::array<double, 3> kmetric_1_percentiles{{7, 8, 9}};
// See HelperCreateMetric1Values() for actual values.
// See HelperCreateMetric1ValuesNotInIgnoreRange() as well.

// Metric 2 values
const char* kmetric_2_key = "m2";
const double kmetric_2_count = 3;
const double kmetric_2_min = 2;
const double kmetric_2_max = 20;
const double kmetric_2_mean = 10;
const double kmetric_2_median = 11;
const double kmetric_2_stddev = 3.5;
const double kmetric_2_mad = 4.5;
const std::array<double, 3> kmetric_2_percentiles{{70, 80, 90}};
// See HelperCreateMetric2Values() for actual values.
// See HelperCreateMetric2ValuesNotInIgnoreRange() as well.

// Metric 3 values
// (No aggregates calculated)
// Same input_value as metric 2.
const char* kmetric_3_key = "m3";
// See HelperCreateMetric3Values() for actual values.
// See HelperCreateMetric3ValuesNotInIgnoreRange() as well.

// Custom aggregates
const char* kcustom_aggregate_1_key = "ca1";
const double kcustom_aggregate_1_value = 23.1;
const char* kcustom_aggregate_2_key = "ca2";
const double kcustom_aggregate_2_value = 46.2;

std::vector<std::pair<double, double>> HelperCreateMetric1Values() {
  std::vector<std::pair<double, double>> tmp{
      std::make_pair(1, 2), std::make_pair(3, 4), std::make_pair(5, 6),
      std::make_pair(7, 8), std::make_pair(9, 10)};
  return tmp;
}

std::vector<std::pair<double, double>>
HelperCreateMetric1ValuesNotInIgnoreRange() {
  std::vector<std::pair<double, double>> tmp{
      std::make_pair(3, 4), std::make_pair(5, 6), std::make_pair(7, 8),
      std::make_pair(9, 10)};
  return tmp;
}

std::vector<std::pair<double, double>> HelperCreateMetric2Values() {
  std::vector<std::pair<double, double>> tmp{
      std::make_pair(7, 8),     std::make_pair(20, 40),
      std::make_pair(25, 30),   std::make_pair(100, 90),
      std::make_pair(200, 400), std::make_pair(201, 203)};
  return tmp;
}

std::vector<std::pair<double, double>>
HelperCreateMetric2ValuesNotInIgnoreRange() {
  std::vector<std::pair<double, double>> tmp{
      std::make_pair(7, 8), std::make_pair(20, 40), std::make_pair(201, 203)};
  return tmp;
}

std::vector<std::pair<double, double>> HelperCreateMetric3Values() {
  std::vector<std::pair<double, double>> tmp{
      std::make_pair(7, 16),    std::make_pair(20, 80),
      std::make_pair(25, 60),   std::make_pair(100, 180),
      std::make_pair(200, 800), std::make_pair(201, 406)};
  return tmp;
}

std::vector<std::pair<double, double>>
HelperCreateMetric3ValuesNotInIgnoreRange() {
  std::vector<std::pair<double, double>> tmp{
      std::make_pair(7, 16), std::make_pair(20, 80), std::make_pair(201, 406)};
  return tmp;
}

class FilterUtilsTest : public ::testing::Test {
 protected:
  // Remove all of the functions below that you do not need.
  // See http://goto/gunitfaq#CtorVsSetUp for when to use SetUp/TearDown.

  FilterUtilsTest() {}

  ~FilterUtilsTest() override {}

  void SetUp() override {}

  void TearDown() override {
    for (auto p : sample_batches_) {
      delete p;
    }
  }

  std::vector<const SampleBatch*> sample_batches_;

  const std::vector<const SampleBatch*> HelperCreateSampleBatches() {
    // Add metric_1's data
    SampleBatch* metric_1 = new SampleBatch();
    metric_1->set_benchmark_key(kbenchmark_key);
    metric_1->set_run_key(krun_key);
    for (auto& a : HelperCreateMetric1Values()) {
      SamplePoint* point = metric_1->add_sample_point_list();
      point->set_input_value(a.first);
      KeyedValue* value = point->add_metric_value_list();
      value->set_value_key(kmetric_1_key);
      value->set_value(a.second);
    }
    sample_batches_.push_back(metric_1);

    // Add metric 2 and 3's data.
    SampleBatch* metric_2_3 = new SampleBatch();
    metric_2_3->set_benchmark_key(kbenchmark_key);
    metric_2_3->set_run_key(krun_key);
    auto metric_2_values = HelperCreateMetric2Values();
    auto metric_3_values = HelperCreateMetric3Values();
    CHECK_EQ(metric_2_values.size(), metric_3_values.size());
    for (auto metric_2_iter = metric_2_values.begin(),
              metric_3_iter = metric_3_values.begin();
         metric_2_iter != metric_2_values.end();
         metric_2_iter++, metric_3_iter++) {
      auto& metric_2_pair = *metric_2_iter;
      auto& metric_3_pair = *metric_3_iter;

      SamplePoint* point_2 = metric_2_3->add_sample_point_list();
      point_2->set_input_value(metric_2_pair.first);
      KeyedValue* value = point_2->add_metric_value_list();
      value->set_value_key(kmetric_2_key);
      value->set_value(metric_2_pair.second);

      SamplePoint* point_3 = metric_2_3->add_sample_point_list();
      point_3->set_input_value(metric_3_pair.first);
      value = point_3->add_metric_value_list();
      value->set_value_key(kmetric_3_key);
      value->set_value(metric_3_pair.second);
    }
    sample_batches_.push_back(metric_2_3);
    return sample_batches_;
  }
};

std::vector<std::pair<double, double>> PackInPair(double val) {
  std::vector<std::pair<double, double>> tmp;
  tmp.push_back(std::make_pair(krun_timestamp_ms, val));
  return tmp;
}

RunInfo HelperCreateRunInfo() {
  RunInfo run_info;
  run_info.set_benchmark_key(kbenchmark_key);
  run_info.set_run_key(krun_key);
  run_info.set_timestamp_ms(krun_timestamp_ms);

  Aggregate* aggregate = run_info.mutable_aggregate();

  aggregate->mutable_percentile_milli_rank_list()->CopyFrom(
      google::protobuf::RepeatedField<int>(kpercentile_milli_rank.begin(),
                                 kpercentile_milli_rank.end()));

  // Populate metric_1
  MetricAggregate* metric_1_aggregate = aggregate->add_metric_aggregate_list();
  metric_1_aggregate->set_metric_key(kmetric_1_key);
  metric_1_aggregate->set_count(kmetric_1_count);
  metric_1_aggregate->set_min(kmetric_1_min);
  metric_1_aggregate->set_max(kmetric_1_max);
  metric_1_aggregate->set_mean(kmetric_1_mean);
  metric_1_aggregate->set_median(kmetric_1_median);
  metric_1_aggregate->set_standard_deviation(kmetric_1_stddev);
  metric_1_aggregate->set_median_absolute_deviation(kmetric_1_mad);
  metric_1_aggregate->mutable_percentile_list()->CopyFrom(
      google::protobuf::RepeatedField<double>(kmetric_1_percentiles.begin(),
                                    kmetric_1_percentiles.end()));

  // Populate metric_2
  MetricAggregate* metric_2_aggregate = aggregate->add_metric_aggregate_list();
  metric_2_aggregate->set_metric_key(kmetric_2_key);
  metric_2_aggregate->set_count(kmetric_2_count);
  metric_2_aggregate->set_min(kmetric_2_min);
  metric_2_aggregate->set_max(kmetric_2_max);
  metric_2_aggregate->set_mean(kmetric_2_mean);
  metric_2_aggregate->set_median(kmetric_2_median);
  metric_2_aggregate->set_standard_deviation(kmetric_2_stddev);
  metric_2_aggregate->set_median_absolute_deviation(kmetric_2_mad);
  metric_2_aggregate->add_percentile_list(0);
  metric_2_aggregate->mutable_percentile_list()->CopyFrom(
      google::protobuf::RepeatedField<double>(kmetric_2_percentiles.begin(),
                                    kmetric_2_percentiles.end()));

  // Set Run Aggregates
  RunAggregate* run_aggregate = aggregate->mutable_run_aggregate();
  run_aggregate->set_error_sample_count(kerror_count);
  run_aggregate->set_benchmark_score(krun_benchmark_score);

  // Set custom aggregate 1
  KeyedValue* custom_aggregate_1 = run_aggregate->add_custom_aggregate_list();
  custom_aggregate_1->set_value_key(kcustom_aggregate_1_key);
  custom_aggregate_1->set_value(kcustom_aggregate_1_value);

  // Set custom aggregate 2
  KeyedValue* custom_aggregate_2 = run_aggregate->add_custom_aggregate_list();
  custom_aggregate_2->set_value_key(kcustom_aggregate_2_key);
  custom_aggregate_2->set_value(kcustom_aggregate_2_value);

  // Set ignore range 1
  LabeledRange* ignore_region_1 = run_info.add_ignore_range_list();
  ignore_region_1->set_label("ignore_range_1");
  Range* range_1 = ignore_region_1->mutable_range();
  range_1->set_start(kignore_region_1[0]);
  range_1->set_end(kignore_region_1[1]);

  // Set ignore range 2
  LabeledRange* ignore_region_2 = run_info.add_ignore_range_list();
  ignore_region_2->set_label("ignore_range_2");
  Range* range_2 = ignore_region_2->mutable_range();
  range_2->set_start(kignore_region_2[0]);
  range_2->set_end(kignore_region_2[1]);

  return run_info;
}

bool Success(std::string s) {
  VLOG(2) << s;
  return s.length() == 0;
}

TEST_F(FilterUtilsTest, DataFilterMissingDataType) {
  DataFilter data_filter;
  data_filter.set_value_key(kmetric_3_key);
  std::vector<std::pair<double, double>> tmp;

  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, DataFilterMissingValueKey) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  std::vector<std::pair<double, double>> tmp;

  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, DataFilterMissingValueKeyError) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::ERROR_COUNT);
  std::vector<std::pair<double, double>> tmp;

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, DataFilterMissingValueKeyBenchmarkScore) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::BENCHMARK_SCORE);
  std::vector<std::pair<double, double>> tmp;

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, RunInfoMissingAggregate) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_AGGREGATE_MIN);
  data_filter.set_value_key(kmetric_1_key);
  std::vector<std::pair<double, double>> tmp;

  auto run_info = HelperCreateRunInfo();
  run_info.clear_aggregate();

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, RunInfoMissingBenchmarkScore) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::BENCHMARK_SCORE);
  data_filter.set_value_key(kmetric_1_key);
  std::vector<std::pair<double, double>> tmp;

  auto run_info = HelperCreateRunInfo();
  run_info.mutable_aggregate()
      ->mutable_run_aggregate()
      ->clear_benchmark_score();

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  run_info.mutable_aggregate()->mutable_run_aggregate()->set_benchmark_score(1);

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));
}

TEST_F(FilterUtilsTest, RunInfoMissingErrorCount) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::ERROR_COUNT);
  data_filter.set_value_key(kmetric_1_key);
  std::vector<std::pair<double, double>> tmp;

  auto run_info = HelperCreateRunInfo();
  run_info.mutable_aggregate()
      ->mutable_run_aggregate()
      ->clear_error_sample_count();

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  run_info.mutable_aggregate()->mutable_run_aggregate()->set_error_sample_count(
      1);

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));
}

TEST_F(FilterUtilsTest, RunInfoMissingRunAggregate) {
  // RunAggregate is a required proto field, so even setting the
  // 'ignore_missing_data' field on DataFilter still causes an error.
  DataFilter data_filter;
  data_filter.set_value_key(kcustom_aggregate_1_key);
  std::vector<std::pair<double, double>> tmp;
  auto run_info = HelperCreateRunInfo();

  // Clear run aggregate
  run_info.mutable_aggregate()->clear_run_aggregate();

  // Missing run aggregates are ignored if ignore_missing_data = true
  data_filter.set_ignore_missing_data(true);

  data_filter.set_data_type(mako::DataFilter::CUSTOM_AGGREGATE);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_data_type(mako::DataFilter::ERROR_COUNT);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_data_type(mako::DataFilter::BENCHMARK_SCORE);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  // Missing run aggregates will error if ignore_missing_data = false
  data_filter.set_ignore_missing_data(false);

  data_filter.set_data_type(mako::DataFilter::CUSTOM_AGGREGATE);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_data_type(mako::DataFilter::ERROR_COUNT);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_data_type(mako::DataFilter::BENCHMARK_SCORE);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  // But works with a good run_info.
  run_info = HelperCreateRunInfo();

  data_filter.set_data_type(mako::DataFilter::CUSTOM_AGGREGATE);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_data_type(mako::DataFilter::ERROR_COUNT);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_data_type(mako::DataFilter::BENCHMARK_SCORE);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));
}

TEST_F(FilterUtilsTest, MissingCustomAggregateIgnored) {
  // data_filter.ignore_missing_data will cause the error that the data is
  // missing to be ignored.
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::CUSTOM_AGGREGATE);
  data_filter.set_value_key("unknown_key");
  std::vector<std::pair<double, double>> tmp;

  // Request was successful, but results are empty
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
  ASSERT_TRUE(tmp.empty());

  tmp.clear();
  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));

  // A different key works.
  tmp.clear();
  data_filter.set_value_key(kcustom_aggregate_2_key);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
  ASSERT_FALSE(tmp.empty());
}

TEST_F(FilterUtilsTest, InvalidCustomAggregateKey) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::CUSTOM_AGGREGATE);
  data_filter.set_value_key("unknown_key");
  std::vector<std::pair<double, double>> tmp;

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));

  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));

  data_filter.set_value_key(kcustom_aggregate_2_key);

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, InvalidMetricAggregate) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_AGGREGATE_MIN);
  data_filter.set_value_key("NoSuchMetric");
  std::vector<std::pair<double, double>> tmp;

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));

  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, PercentileNoSuchMetric) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_AGGREGATE_PERCENTILE);

  data_filter.set_value_key("NoSuchMetric");
  data_filter.set_percentile_milli_rank(kpercentile_milli_rank[0]);
  std::vector<std::pair<double, double>> tmp;

  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));

  data_filter.set_ignore_missing_data(true);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));


  // But with a valid metric key, should work
  data_filter.set_value_key(kmetric_1_key);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, NoSuchPercentileMilliRank) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_AGGREGATE_PERCENTILE);
  data_filter.set_value_key(kmetric_1_key);

  // Not in kpercentile_milli_rank
  data_filter.set_percentile_milli_rank(1);
  std::vector<std::pair<double, double>> tmp;

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));

  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));

  // Set to a valid milli_rank
  data_filter.set_percentile_milli_rank(kpercentile_milli_rank[0]);

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, NoPercentilesForAggregateOrMetrics) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_AGGREGATE_PERCENTILE);

  data_filter.set_value_key(kmetric_1_key);
  data_filter.set_percentile_milli_rank(kpercentile_milli_rank[0]);
  std::vector<std::pair<double, double>> tmp;

  // Clear Metric 1's percentiles
  RunInfo run_info = HelperCreateRunInfo();
  for (auto& metric_aggregate :
       *run_info.mutable_aggregate()->mutable_metric_aggregate_list()) {
    if (metric_aggregate.metric_key() == kmetric_1_key) {
      metric_aggregate.clear_percentile_list();
    }
  }
  // Clear Aggregates percentiles
  run_info.mutable_aggregate()->clear_percentile_milli_rank_list();

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  data_filter.set_ignore_missing_data(false);
  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));
}

TEST_F(FilterUtilsTest, MetricMissingPercentile) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_AGGREGATE_PERCENTILE);

  data_filter.set_value_key(kmetric_1_key);
  data_filter.set_percentile_milli_rank(kpercentile_milli_rank[0]);
  std::vector<std::pair<double, double>> tmp;

  // Clear Metric 1's percentiles
  RunInfo run_info = HelperCreateRunInfo();
  for (auto& metric_aggregate :
       *run_info.mutable_aggregate()->mutable_metric_aggregate_list()) {
    if (metric_aggregate.metric_key() == kmetric_1_key) {
      metric_aggregate.clear_percentile_list();
    }
  }

  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));

  // But calling for metric 2 should be fine.
  data_filter.set_value_key(kmetric_2_key);

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(

      run_info, HelperCreateSampleBatches(), data_filter, false, &tmp)));
}

TEST_F(FilterUtilsTest, MetricMissingKey) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_AGGREGATE_PERCENTILE);
  data_filter.set_value_key(kmetric_1_key);
  std::vector<std::pair<double, double>> tmp;

  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));

  data_filter.set_percentile_milli_rank(kpercentile_milli_rank[0]);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &tmp)));
}

TEST_F(FilterUtilsTest, SamplePointMetricMissingValue) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_2_key);
  std::vector<std::pair<double, double>> results;

  std::vector<const SampleBatch*> clear_sample_batches;
  SampleBatch metric_2_batch;
  auto sample_point = metric_2_batch.add_sample_point_list();
  sample_point->set_input_value(1);
  auto keyed_value = sample_point->add_metric_value_list();
  keyed_value->set_value_key(kmetric_2_key);
  clear_sample_batches.push_back(&metric_2_batch);

  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), clear_sample_batches, data_filter, false,
      &results)));

  // Add value then should return success.
  keyed_value->set_value(2);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), clear_sample_batches, data_filter, false,
      &results)));
}

TEST_F(FilterUtilsTest, SamplePointMetricMissingValueKey) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_2_key);
  std::vector<std::pair<double, double>> results;

  std::vector<const SampleBatch*> clear_sample_batches;
  SampleBatch metric_2_batch;
  auto sample_point = metric_2_batch.add_sample_point_list();
  sample_point->set_input_value(1);
  auto keyed_value = sample_point->add_metric_value_list();
  keyed_value->set_value(2);
  clear_sample_batches.push_back(&metric_2_batch);

  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), clear_sample_batches, data_filter, false,
      &results)));

  // Add value key then should return success.
  keyed_value->set_value_key(kmetric_2_key);
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), clear_sample_batches, data_filter, false,
      &results)));
}

TEST_F(FilterUtilsTest, IgnoreRangeMissingRange) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_1_key);
  std::vector<std::pair<double, double>> results;

  auto run_info = HelperCreateRunInfo();

  for (auto& ignore_range : *run_info.mutable_ignore_range_list()) {
    ignore_range.clear_range();
  }

  ASSERT_FALSE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &results)));
}

TEST_F(FilterUtilsTest, NoIgnoreRanges) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_1_key);
  std::vector<std::pair<double, double>> results;

  auto run_info = HelperCreateRunInfo();
  run_info.clear_ignore_range_list();

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, HelperCreateSampleBatches(), data_filter, false, &results)));

  // Ignore range should have been ignored, all points should exist.
  ASSERT_EQ(HelperCreateMetric1Values(), results);
}

TEST_F(FilterUtilsTest, Metric1SamplePoints) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_1_key);
  std::vector<std::pair<double, double>> results;

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &results)));

  ASSERT_EQ(HelperCreateMetric1ValuesNotInIgnoreRange(), results);
}

TEST_F(FilterUtilsTest, Metric2SamplePoints) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_2_key);
  std::vector<std::pair<double, double>> results;

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &results)));
  ASSERT_EQ(HelperCreateMetric2ValuesNotInIgnoreRange(), results);
}

TEST_F(FilterUtilsTest, Metric3SamplePoints) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_3_key);
  std::vector<std::pair<double, double>> results;

  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
      &results)));
  ASSERT_EQ(HelperCreateMetric3ValuesNotInIgnoreRange(), results);
}

TEST_F(FilterUtilsTest, Aggregates) {
  struct AggregateTest {
    std::string test_name;
    mako::DataFilter_DataType data_type;
    std::string metric_name;
    double expected_value;
    int pmr;
  };

  const std::vector<AggregateTest> tests{
      {
          // Run Aggregates
          {"BenchmarkScore", mako::DataFilter::BENCHMARK_SCORE,
           kmetric_1_key, krun_benchmark_score},
          {"CustomAggregate1", mako::DataFilter::CUSTOM_AGGREGATE,
           kcustom_aggregate_1_key, kcustom_aggregate_1_value},
          {"CustomAggregate2", mako::DataFilter::CUSTOM_AGGREGATE,
           kcustom_aggregate_2_key, kcustom_aggregate_2_value},
          {"ErrorCount", mako::DataFilter::ERROR_COUNT, kmetric_1_key,
           kerror_count},

          // Metric 1 Aggregates
          {"Count1", mako::DataFilter::METRIC_AGGREGATE_COUNT,
           kmetric_1_key, kmetric_1_count},
          {"Min1", mako::DataFilter::METRIC_AGGREGATE_MIN, kmetric_1_key,
           kmetric_1_min},
          {"Max1", mako::DataFilter::METRIC_AGGREGATE_MAX, kmetric_1_key,
           kmetric_1_max},
          {"Mean1", mako::DataFilter::METRIC_AGGREGATE_MEAN, kmetric_1_key,
           kmetric_1_mean},
          {"Median1", mako::DataFilter::METRIC_AGGREGATE_MEDIAN,
           kmetric_1_key, kmetric_1_median},
          {"Stddev1", mako::DataFilter::METRIC_AGGREGATE_STDDEV,
           kmetric_1_key, kmetric_1_stddev},
          {"Mad1", mako::DataFilter::METRIC_AGGREGATE_MAD, kmetric_1_key,
           kmetric_1_mad},
          {"Percentile1_0", mako::DataFilter::METRIC_AGGREGATE_PERCENTILE,
           kmetric_1_key, kmetric_1_percentiles[0], kpercentile_milli_rank[0]},
          {"Percentile1_1", mako::DataFilter::METRIC_AGGREGATE_PERCENTILE,
           kmetric_1_key, kmetric_1_percentiles[1], kpercentile_milli_rank[1]},
          {"Percentile1_2", mako::DataFilter::METRIC_AGGREGATE_PERCENTILE,
           kmetric_1_key, kmetric_1_percentiles[2], kpercentile_milli_rank[2]},

          // Metric 2 Aggregates
          {"Count2", mako::DataFilter::METRIC_AGGREGATE_COUNT,
           kmetric_2_key, kmetric_2_count},
          {"Min2", mako::DataFilter::METRIC_AGGREGATE_MIN, kmetric_2_key,
           kmetric_2_min},
          {"Max2", mako::DataFilter::METRIC_AGGREGATE_MAX, kmetric_2_key,
           kmetric_2_max},
          {"Mean2", mako::DataFilter::METRIC_AGGREGATE_MEAN, kmetric_2_key,
           kmetric_2_mean},
          {"Median2", mako::DataFilter::METRIC_AGGREGATE_MEDIAN,
           kmetric_2_key, kmetric_2_median},
          {"Stddev2", mako::DataFilter::METRIC_AGGREGATE_STDDEV,
           kmetric_2_key, kmetric_2_stddev},
          {"Mad2", mako::DataFilter::METRIC_AGGREGATE_MAD, kmetric_2_key,
           kmetric_2_mad},
          {"Percentile2_0", mako::DataFilter::METRIC_AGGREGATE_PERCENTILE,
           kmetric_2_key, kmetric_2_percentiles[0], kpercentile_milli_rank[0]},
          {"Percentile2_1", mako::DataFilter::METRIC_AGGREGATE_PERCENTILE,
           kmetric_2_key, kmetric_2_percentiles[1], kpercentile_milli_rank[1]},
          {"Percentile2_2", mako::DataFilter::METRIC_AGGREGATE_PERCENTILE,
           kmetric_2_key, kmetric_2_percentiles[2], kpercentile_milli_rank[2]},
      },
  };

  for (const auto& test : tests) {
    DataFilter data_filter;
    data_filter.set_data_type(test.data_type);
    data_filter.set_value_key(test.metric_name);
    data_filter.set_percentile_milli_rank(test.pmr);
    std::vector<std::pair<double, double>> results;

    EXPECT_TRUE(Success(mako::internal::ApplyFilter(
        HelperCreateRunInfo(), HelperCreateSampleBatches(), data_filter, false,
        &results)))
        << test.test_name;
    EXPECT_EQ(PackInPair(test.expected_value), results) << test.test_name;
  }
}

TEST_F(FilterUtilsTest, SortedResults) {
  std::string metric_name = "my_metric_name";
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(metric_name);
  std::vector<std::pair<double, double>> results;

  std::vector<std::pair<double, double>> metric_values{
      std::make_pair(10, 2), std::make_pair(1000, 4), std::make_pair(1, 9),
      std::make_pair(4, 8),  std::make_pair(2, 1),
  };

  // Clear all ignore regions.
  auto run_info = HelperCreateRunInfo();
  run_info.clear_ignore_range_list();

  // Add data to SampleBatch
  std::vector<const SampleBatch*> sample_batches;
  SampleBatch batch;
  batch.set_benchmark_key(kbenchmark_key);
  batch.set_run_key(krun_key);
  for (auto& a : metric_values) {
    SamplePoint* point = batch.add_sample_point_list();
    point->set_input_value(a.first);
    KeyedValue* value = point->add_metric_value_list();
    value->set_value_key(metric_name);
    value->set_value(a.second);
  }
  sample_batches.push_back(&batch);

  // Sort the metrics
  std::sort(metric_values.begin(), metric_values.end());

  // Sort = False, so should NOT be equal.
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, sample_batches, data_filter, false, &results)));
  ASSERT_NE(metric_values, results);

  // Sort = True, so should be equal.
  results.clear();
  ASSERT_TRUE(Success(mako::internal::ApplyFilter(
      run_info, sample_batches, data_filter, true, &results)));
  ASSERT_EQ(metric_values, results);
}
