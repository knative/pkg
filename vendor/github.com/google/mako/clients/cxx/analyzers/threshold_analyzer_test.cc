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
// See the License for the specific language governing permissions and
// limitations under the License.
#include "clients/cxx/analyzers/threshold_analyzer.h"

#include <vector>

#include "src/google/protobuf/text_format.h"
#include "gtest/gtest.h"
#include "clients/proto/analyzers/threshold_analyzer.pb.h"
#include "absl/strings/str_cat.h"
#include "spec/proto/mako.pb.h"
#include "testing/cxx/protocol-buffer-matchers.h"

using ::mako::EqualsProto;

namespace mako {
namespace threshold_analyzer {

class AnalyzerTest : public ::testing::Test {
 protected:
  AnalyzerTest() {}

  ~AnalyzerTest() override {}

  void SetUp() override {}

  void TearDown() override {}
};

using mako::AnalyzerInput;
using mako::AnalyzerOutput;
using mako::DataFilter;
using mako::BenchmarkInfo;
using mako::RunInfo;
using mako::RunBundle;
using mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput;
using mako::analyzers::threshold_analyzer::ThresholdAnalyzerOutput;
using mako::analyzers::threshold_analyzer::ThresholdConfig;

const char* kbenchmark_key = "BenchmarkKey";

const char* krun_key = "RunKey";
const double kbenchmark_score = 3.6;

const char* kmetric_1_key = "metric1";
const char* kmetric_1_label = "MetricOne";
const double kmetric_1_count = 15;
const double kmetric_1_min = 5;
const double kmetric_1_max = 10;
const double kmetric_1_mean = 7;
const double kmetric_1_median = 6;
const double kmetric_1_stddev = 3.5;
const double kmetric_1_mad = 4.5;

const char* kmetric_2_key = "metric2";
const char* kmetric_2_label = "MetricTwo";
const double kmetric_2_count = 16;
const double kmetric_2_min = 50;
const double kmetric_2_max = 100;
const double kmetric_2_mean = 60;
const double kmetric_2_median = 55;
const double kmetric_2_stddev = 7.7;
const double kmetric_2_mad = 10.5;

BenchmarkInfo HelperCreateBenchmarkInfo() {
  BenchmarkInfo benchmark_info;
  benchmark_info.set_benchmark_key(kbenchmark_key);

  // Add metric aggregates
  auto metric_1_aggregate = benchmark_info.add_metric_info_list();
  metric_1_aggregate->set_value_key(kmetric_1_key);
  metric_1_aggregate->set_label(kmetric_1_label);

  auto metric_2_aggregate = benchmark_info.add_metric_info_list();
  metric_2_aggregate->set_value_key(kmetric_2_key);
  metric_2_aggregate->set_label(kmetric_2_label);

  return benchmark_info;
}

RunInfo HelperCreateRunInfo() {
  RunInfo run_info;
  run_info.set_benchmark_key(kbenchmark_key);
  run_info.set_run_key(krun_key);

  auto aggregate = run_info.mutable_aggregate();
  aggregate->add_percentile_milli_rank_list(50000);

  // Set run aggregates
  auto run_aggregate = aggregate->mutable_run_aggregate();
  run_aggregate->set_benchmark_score(kbenchmark_score);

  // Set metric aggregates
  auto metric_1_aggregate = aggregate->add_metric_aggregate_list();
  metric_1_aggregate->set_metric_key(kmetric_1_key);
  metric_1_aggregate->set_count(kmetric_1_count);
  metric_1_aggregate->set_min(kmetric_1_min);
  metric_1_aggregate->set_max(kmetric_1_max);
  metric_1_aggregate->set_mean(kmetric_1_mean);
  metric_1_aggregate->set_median(kmetric_1_median);
  metric_1_aggregate->set_standard_deviation(kmetric_1_stddev);
  metric_1_aggregate->set_median_absolute_deviation(kmetric_1_mad);
  metric_1_aggregate->add_percentile_list(kmetric_1_median);

  auto metric_2_aggregate = aggregate->add_metric_aggregate_list();
  metric_2_aggregate->set_metric_key(kmetric_2_key);
  metric_2_aggregate->set_count(kmetric_2_count);
  metric_2_aggregate->set_min(kmetric_2_min);
  metric_2_aggregate->set_max(kmetric_2_max);
  metric_2_aggregate->set_mean(kmetric_2_mean);
  metric_2_aggregate->set_median(kmetric_2_median);
  metric_2_aggregate->set_standard_deviation(kmetric_2_stddev);
  metric_2_aggregate->set_median_absolute_deviation(kmetric_2_mad);
  metric_2_aggregate->add_percentile_list(kmetric_2_median);

  return run_info;
}

RunBundle HelperCreateRunBundle() {
  RunBundle run_bundle;
  *run_bundle.mutable_run_info() = HelperCreateRunInfo();
  *run_bundle.mutable_benchmark_info() = HelperCreateBenchmarkInfo();
  return run_bundle;
}

AnalyzerInput HelperCreateAnalyzerInput() {
  AnalyzerInput analyzer_input;
  *analyzer_input.mutable_run_to_be_analyzed() = HelperCreateRunBundle();
  return analyzer_input;
}

ThresholdConfig HelperCreateThresholdAnalyzerConfig(
    double min, double max, double outliers, const DataFilter& data_filter) {
  ThresholdConfig config;
  config.set_min(min);
  config.set_max(max);
  config.set_outlier_percent_max(outliers);
  *config.mutable_data_filter() = data_filter;
  return config;
}

ThresholdAnalyzerInput HelperCreateThresholdAnalyzerInput(
    double min, double max, double outliers, const DataFilter& data_filter) {
  ThresholdAnalyzerInput input;
  *input.add_configs() =
      HelperCreateThresholdAnalyzerConfig(min, max, outliers, data_filter);
  return input;
}

Analyzer HelperCreateThresholdAnalyzer(double min, double max,
                                                double outliers,
                                                const DataFilter& data_filter) {
  Analyzer analyzer = Analyzer(
      HelperCreateThresholdAnalyzerInput(min, max, outliers, data_filter));
  return analyzer;
}

bool SuccessfulStatus(const AnalyzerOutput& output) {
  return output.has_status() && output.status().has_code() &&
         output.status().code() == mako::Status_Code_SUCCESS;
}

void HelperAddSamplePoints(std::string value_key, const std::vector<double>& data,
                           RunBundle* run_bundle) {
  for (auto& d : data) {
    auto sample_batch = run_bundle->add_batch_list();
    auto sample_point = sample_batch->add_sample_point_list();
    sample_point->set_input_value(1);
    auto keyed_value = sample_point->add_metric_value_list();
    keyed_value->set_value_key(value_key);
    keyed_value->set_value(d);
  }
}

TEST_F(AnalyzerTest, AnalyzerInputMissingRunToBeAnalyzed) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_1_key);
  auto analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);

  // Clear run to be analyzed
  auto input = HelperCreateAnalyzerInput();
  input.clear_run_to_be_analyzed();

  AnalyzerOutput output;
  EXPECT_FALSE(analyzer.Analyze(input, &output));
  EXPECT_FALSE(SuccessfulStatus(output));
}

TEST_F(AnalyzerTest, NoSuchMetricInRunInfo) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter.set_value_key("missing metric");

  AnalyzerOutput output;
  auto analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);
  EXPECT_TRUE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output))
      << output.status().fail_message();
  EXPECT_TRUE(SuccessfulStatus(output)) << output.status().fail_message();

  // Correct metric_name then expect success.
  data_filter.set_value_key(kmetric_1_key);
  analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);
  EXPECT_TRUE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output))
      << output.status().fail_message();
  EXPECT_TRUE(SuccessfulStatus(output)) << output.status().fail_message();
}

TEST_F(AnalyzerTest, TypeAndNameSetDuringFailure) {
  // Make sure in failure type and name are still set.
  std::string my_name = "MyName";
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter.set_value_key("missing metric");
  data_filter.set_ignore_missing_data(false);

  AnalyzerOutput output;
  auto analyzer_input =
      HelperCreateThresholdAnalyzerInput(0, 0, 0, data_filter);
  analyzer_input.set_name(my_name);
  Analyzer analyzer = Analyzer(analyzer_input);
  EXPECT_FALSE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output))
      << output.status().fail_message();
  EXPECT_FALSE(SuccessfulStatus(output)) << output.status().fail_message();
  EXPECT_EQ("Threshold", output.analyzer_type());
  EXPECT_EQ(my_name, output.analyzer_name());

  // Correct metric_name then expect success.
  data_filter.set_value_key(kmetric_1_key);
  analyzer_input = HelperCreateThresholdAnalyzerInput(0, 0, 0, data_filter);
  analyzer_input.set_name(my_name);
  analyzer = Analyzer(analyzer_input);
  EXPECT_TRUE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output))
      << output.status().fail_message();
  EXPECT_TRUE(SuccessfulStatus(output)) << output.status().fail_message();
  EXPECT_EQ("Threshold", output.analyzer_type());
  EXPECT_EQ(my_name, output.analyzer_name());
}

TEST_F(AnalyzerTest, AnalyzerInputMissingRunInfo) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_1_key);
  auto analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);

  // Clear run_to_be_analyzed.run_info
  auto input = HelperCreateAnalyzerInput();
  input.mutable_run_to_be_analyzed()->clear_run_info();

  AnalyzerOutput output;
  EXPECT_FALSE(analyzer.Analyze(input, &output));
  EXPECT_FALSE(SuccessfulStatus(output));
}

TEST_F(AnalyzerTest, AnalyzeAggregateWithUnknownMetricKey) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_AGGREGATE_MIN);
  data_filter.set_value_key("UnknownKey");
  auto analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);

  // Clear run_to_be_analyzed.run_info
  auto input = HelperCreateAnalyzerInput();

  AnalyzerOutput output;
  EXPECT_TRUE(analyzer.Analyze(input, &output))
      << output.status().fail_message();
  EXPECT_TRUE(SuccessfulStatus(output)) << output.status().fail_message();
}

TEST_F(AnalyzerTest,
       AnalyzeAggregateWithUnknownMetricKeyAndNoIgnoreMissingData) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_AGGREGATE_MIN);
  data_filter.set_value_key("UnknownKey");
  data_filter.set_ignore_missing_data(false);
  auto analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);

  // Clear run_to_be_analyzed.run_info
  auto input = HelperCreateAnalyzerInput();

  AnalyzerOutput output;
  EXPECT_FALSE(analyzer.Analyze(input, &output))
      << output.status().fail_message();
  EXPECT_FALSE(SuccessfulStatus(output)) << output.status().fail_message();
}

TEST_F(AnalyzerTest,
       AnalyzeSamplePointsWithUnknownMetricKey) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
  data_filter.set_value_key("UnknownKey");

  auto analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);

  // Clear run_to_be_analyzed.run_info
  auto input = HelperCreateAnalyzerInput();

  AnalyzerOutput output;
  EXPECT_TRUE(analyzer.Analyze(input, &output))
      << output.status().fail_message();
  EXPECT_TRUE(SuccessfulStatus(output)) << output.status().fail_message();
}

TEST_F(AnalyzerTest,
       AnalyzeSamplePointsWithUnknownMetricKeyAndNoIgnoreMissingData) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
  data_filter.set_value_key("UnknownKey");
  data_filter.set_ignore_missing_data(false);

  auto analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);

  // Clear run_to_be_analyzed.run_info
  auto input = HelperCreateAnalyzerInput();

  AnalyzerOutput output;
  EXPECT_FALSE(analyzer.Analyze(input, &output))
      << output.status().fail_message();
  EXPECT_FALSE(SuccessfulStatus(output)) << output.status().fail_message();
}

TEST_F(AnalyzerTest, ConstructHistoricQueryUnset) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_1_key);
  auto analyzer = HelperCreateThresholdAnalyzer(0, 0, 0, data_filter);
  mako::AnalyzerHistoricQueryInput input;
  mako::AnalyzerHistoricQueryOutput output;

  EXPECT_TRUE(analyzer.ConstructHistoricQuery(input, &output));
  EXPECT_FALSE(output.has_get_batches());
  EXPECT_EQ(0, output.run_info_query_list_size());
  EXPECT_TRUE(output.has_status());
  EXPECT_EQ(mako::Status_Code_SUCCESS, output.status().code());
}

TEST_F(AnalyzerTest, ConfigMissingDataFilter) {
  ThresholdAnalyzerInput threshold_input;
  auto config = threshold_input.add_configs();
  config->set_max(100);
  config->set_min(0);
  config->set_outlier_percent_max(0);

  auto analyzer = Analyzer(threshold_input);

  AnalyzerOutput output;
  EXPECT_FALSE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_FALSE(SuccessfulStatus(output));

  // Add DataFilter and should see success.
  auto data_filter = config->mutable_data_filter();
  data_filter->set_data_type(
      mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter->set_value_key(kmetric_1_key);
  analyzer = Analyzer(threshold_input);

  EXPECT_TRUE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_TRUE(SuccessfulStatus(output));
  EXPECT_FALSE(output.regression());
}

TEST_F(AnalyzerTest, Metric2MaxFindRegression) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter.set_value_key(kmetric_2_key);
  AnalyzerOutput output;

  auto analyzer_finds_regression =
      HelperCreateThresholdAnalyzer(98, 99, 0, data_filter);
  EXPECT_TRUE(
      analyzer_finds_regression.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_TRUE(SuccessfulStatus(output));
  EXPECT_TRUE(output.regression());
  // We should always generate an output, regardless of whether there was a
  // regression.
  EXPECT_LT(0, output.output().length());
}

TEST_F(AnalyzerTest, Metric2MaxNoRegression) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter.set_value_key(kmetric_2_key);
  AnalyzerOutput output;

  auto analyzer_finds_regression =
      HelperCreateThresholdAnalyzer(98, 101, 0, data_filter);
  EXPECT_TRUE(
      analyzer_finds_regression.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_TRUE(SuccessfulStatus(output));
  EXPECT_FALSE(output.regression());
  // We should always generate an output, regardless of whether there was a
  // regression.
  EXPECT_LT(0, output.output().length());
}

TEST_F(AnalyzerTest, AllDataFiltersTest) {
  const std::vector<mako::DataFilter::DataType> data_types{
      mako::DataFilter_DataType_METRIC_AGGREGATE_COUNT,
      mako::DataFilter_DataType_METRIC_AGGREGATE_MIN,
      mako::DataFilter_DataType_METRIC_AGGREGATE_MAX,
      mako::DataFilter_DataType_METRIC_AGGREGATE_MEAN,
      mako::DataFilter_DataType_METRIC_AGGREGATE_MEDIAN,
      mako::DataFilter_DataType_METRIC_AGGREGATE_STDDEV,
      mako::DataFilter_DataType_METRIC_AGGREGATE_MAD,
      mako::DataFilter_DataType_METRIC_AGGREGATE_PERCENTILE};
  for (const auto& data_type : data_types) {
    SCOPED_TRACE(absl::StrCat("data type: ", data_type));
    DataFilter data_filter;
    data_filter.set_data_type(data_type);
    data_filter.set_value_key(kmetric_2_key);
    if (data_type ==
        mako::DataFilter_DataType_METRIC_AGGREGATE_PERCENTILE) {
      data_filter.set_percentile_milli_rank(50000);
    }
    AnalyzerOutput output;

    auto analyzer_finds_regression =
        HelperCreateThresholdAnalyzer(0, 101, 0, data_filter);
    EXPECT_TRUE(analyzer_finds_regression.Analyze(HelperCreateAnalyzerInput(),
                                                  &output));
    EXPECT_TRUE(SuccessfulStatus(output));
    EXPECT_FALSE(output.regression());
    // We should always generate an output, regardless of whether there was a
    // regression.
    EXPECT_LT(0, output.output().length());
  }
}

TEST_F(AnalyzerTest, SamplePointTests) {
  struct SamplePointTest {
    std::string name;
    double min, max, max_outliers;
    std::vector<double> data;
    bool expect_regression;
  };

  std::vector<SamplePointTest> tests{
      {
       {"All data within range", 0, 10, 0, {1, 10, 5}, false},
       {"All data outside range", 5, 10, 0, {1, 2, 3, 4, 5}, true},
       {"Limit 50%%, Actual 75%% outside", 5, 10, 50, {2, 3, 4, 6}, true},
       {"Limit 50%%, Actual 25%% outside", 3, 10, 50, {2, 3, 4, 6}, false},
       {"Limit 50%%, Actual 50%% outside", 3, 10, 50, {2, 3, 4, 11}, false},
       {"All data at upper limit", 3, 10, 50, {10, 10, 10, 10}, false},
       {"All data at lower limit", 10, 100, 50, {10, 10, 10, 10}, false},
      },
  };

  for (const auto& test : tests) {
    DataFilter data_filter;
    data_filter.set_data_type(
        mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
    data_filter.set_value_key(kmetric_1_key);
    AnalyzerOutput output;

    AnalyzerInput input;
    *input.mutable_run_to_be_analyzed() = HelperCreateRunBundle();
    HelperAddSamplePoints(kmetric_1_key, test.data,
                          input.mutable_run_to_be_analyzed());

    auto analyzer = HelperCreateThresholdAnalyzer(
        test.min, test.max, test.max_outliers, data_filter);
    EXPECT_TRUE(analyzer.Analyze(input, &output))
        << output.status().fail_message() << " " << test.name;
    EXPECT_TRUE(SuccessfulStatus(output)) << output.status().fail_message()
                                          << " " << test.name;
    EXPECT_EQ(test.expect_regression, output.regression()) << output.output()
                                                           << test.name;
    EXPECT_EQ("Threshold", output.analyzer_type());
    // No name was set in input
    EXPECT_FALSE(output.has_analyzer_name());

    // Only parse if return status was successful
    if (SuccessfulStatus(output)) {
      ThresholdAnalyzerOutput analyzer_output;
      ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(output.output(),
                                                      &analyzer_output));
      for (const auto& config_output : analyzer_output.config_results()) {
        // Not setting any names so should always be false
        EXPECT_FALSE(config_output.config().has_config_name());
      }
    }
  }
}

TEST_F(AnalyzerTest, ParseAnalyzerOutput) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(kmetric_1_key);
  AnalyzerOutput output;

  AnalyzerInput input;
  *input.mutable_run_to_be_analyzed() = HelperCreateRunBundle();
  HelperAddSamplePoints(kmetric_1_key, {1, 10, 100, 1000, 1500, 2000},
                        input.mutable_run_to_be_analyzed());

  ThresholdAnalyzerInput analyzer_input;
  auto config = HelperCreateThresholdAnalyzerConfig(5, 200, 0, data_filter);
  config.set_config_name("TestingMetric1");
  *analyzer_input.add_configs() = config;
  Analyzer analyzer = Analyzer(analyzer_input);

  ASSERT_TRUE(analyzer.Analyze(input, &output))
      << output.status().fail_message();
  ASSERT_FALSE(output.output().empty());

  ThresholdAnalyzerOutput analyzer_output;
  ASSERT_TRUE(
      google::protobuf::TextFormat::ParseFromString(output.output(), &analyzer_output));
  ASSERT_EQ(1, analyzer_output.config_results_size());
  EXPECT_EQ("TestingMetric1",
            analyzer_output.config_results(0).config().config_name());
  EXPECT_NEAR(50, analyzer_output.config_results(0).percent_above_max(), 0.01);
  EXPECT_NEAR(16.66, analyzer_output.config_results(0).percent_below_min(),
              0.01);
  // When analyzing sample points and multiple points are outside of threshold,
  // this field is not set. It is only set when a single point is outside of
  // the threshold (primarily for analyzing aggregates).
  EXPECT_FALSE(analyzer_output.config_results(0).has_value_outside_threshold());
  EXPECT_TRUE(analyzer_output.config_results(0).regression());
}

TEST_F(AnalyzerTest, ValueOutsideThreshold) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter.set_value_key(kmetric_1_key);
  AnalyzerOutput output;

  AnalyzerInput input;
  *input.mutable_run_to_be_analyzed() = HelperCreateRunBundle();

  ThresholdAnalyzerInput analyzer_input;
  auto config = HelperCreateThresholdAnalyzerConfig(0, (kmetric_1_max - 1), 0,
                                                    data_filter);
  config.set_config_name("TestingMetric1");
  *analyzer_input.add_configs() = config;
  Analyzer analyzer = Analyzer(analyzer_input);

  ASSERT_TRUE(analyzer.Analyze(input, &output))
      << output.status().fail_message();
  ASSERT_FALSE(output.output().empty());

  ThresholdAnalyzerOutput analyzer_output;
  ASSERT_TRUE(
      google::protobuf::TextFormat::ParseFromString(output.output(), &analyzer_output));
  ASSERT_EQ(1, analyzer_output.config_results_size());
  EXPECT_EQ("TestingMetric1",
            analyzer_output.config_results(0).config().config_name());
  EXPECT_NEAR(100.0, analyzer_output.config_results(0).percent_above_max(),
              0.01);
  EXPECT_NEAR(0.0, analyzer_output.config_results(0).percent_below_min(), 0.01);

  // When analyzing sample points and multiple points are outside of threshold,
  // this field is not set. It is only set when a single point is outside of
  // the threshold (primarily for analyzing aggregates).
  EXPECT_NEAR(kmetric_1_max,
              analyzer_output.config_results(0).value_outside_threshold(),
              0.01);
}

TEST_F(AnalyzerTest, OnlyMinThresholdNoRegression) {
  ThresholdAnalyzerInput threshold_input;
  auto config = threshold_input.add_configs();
  config->set_min(0);
  config->set_outlier_percent_max(0);

  auto data_filter = config->mutable_data_filter();
  data_filter->set_data_type(
      mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter->set_value_key(kmetric_1_key);
  auto analyzer = Analyzer(threshold_input);

  AnalyzerOutput output;
  EXPECT_TRUE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_TRUE(SuccessfulStatus(output));
  EXPECT_FALSE(output.regression());
}

TEST_F(AnalyzerTest, OnlyMaxThresholdNoRegression) {
  ThresholdAnalyzerInput threshold_input;
  auto config = threshold_input.add_configs();
  config->set_max(100);
  config->set_outlier_percent_max(0);

  auto data_filter = config->mutable_data_filter();
  data_filter->set_data_type(
      mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter->set_value_key(kmetric_1_key);
  auto analyzer = Analyzer(threshold_input);

  AnalyzerOutput output;
  EXPECT_TRUE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_TRUE(SuccessfulStatus(output));
  EXPECT_FALSE(output.regression());
}

TEST_F(AnalyzerTest, OnlyMinThresholdWithRegression) {
  ThresholdAnalyzerInput threshold_input;
  auto config = threshold_input.add_configs();
  config->set_min(100);
  config->set_outlier_percent_max(0);

  auto data_filter = config->mutable_data_filter();
  data_filter->set_data_type(
      mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter->set_value_key(kmetric_1_key);
  auto analyzer = Analyzer(threshold_input);

  AnalyzerOutput output;
  EXPECT_TRUE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_TRUE(SuccessfulStatus(output));
  EXPECT_TRUE(output.regression());
}

TEST_F(AnalyzerTest, OnlyMaxThresholdWithRegression) {
  ThresholdAnalyzerInput threshold_input;
  auto config = threshold_input.add_configs();
  config->set_max(2);
  config->set_outlier_percent_max(0);

  auto data_filter = config->mutable_data_filter();
  data_filter->set_data_type(
      mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter->set_value_key(kmetric_1_key);
  auto analyzer = Analyzer(threshold_input);

  AnalyzerOutput output;
  EXPECT_TRUE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_TRUE(SuccessfulStatus(output));
  EXPECT_TRUE(output.regression());
}

TEST_F(AnalyzerTest, NoMinOrMaxFails) {
  ThresholdAnalyzerInput threshold_input;
  auto config = threshold_input.add_configs();

  auto data_filter = config->mutable_data_filter();
  data_filter->set_data_type(
      mako::DataFilter_DataType_METRIC_AGGREGATE_MAX);
  data_filter->set_value_key(kmetric_1_key);
  auto analyzer = Analyzer(threshold_input);

  AnalyzerOutput output;
  EXPECT_FALSE(analyzer.Analyze(HelperCreateAnalyzerInput(), &output));
  EXPECT_FALSE(SuccessfulStatus(output));
}

TEST(ThresholdAnalyzerTest, QueryTest) {
  struct Case {
    std::string name;
    std::string ctor_in;
    std::string func_in;
    std::string want_func_out;
  };
  std::vector<Case> cases = {
      {
          "no_cross_run_config",
          // ThresholdAnalyzerInput with no cross_run_config message
          "name: \"ta\"",
          // AnalyzerHistoricQueryInput
          "benchmark_info: < "
          "  benchmark_key: \"b1\" "
          "> "
          "run_info: < "
          "  benchmark_key: \"b1\" "
          "  run_key: \"r1\" "
          "> ",
          // AnalyzerHistoricQueryOutput
          "status: < "
          "  code: SUCCESS "
          "> ",
      },
      {
          "cross_run_config_no_batches",
          // ThresholdAnalyzerInput with a cross_run_config message and no
          // ThresholdConfig using a data filter with METRIC_SAMPLEPOINTS
          "name: \"ta\""
          "cross_run_config: < "
          "  run_info_query_list: < "
          "    benchmark_key: \"b1\" "
          "    max_timestamp_ms: 10 "
          "    limit: 5 "
          "  > "
          "  run_info_query_list: < "
          "    benchmark_key: \"b2\" "
          "    max_timestamp_ms: 10 "
          "    limit: 5 "
          "  > "
          "> "
          "configs: < "
          "  data_filter: < "
          "    data_type: METRIC_AGGREGATE_MEAN "
          "    value_key: \"y1\" "
          "  > "
          "> ",
          // AnalyzerHistoricQueryInput
          "benchmark_info: < "
          "  benchmark_key: \"b1\" "
          "> "
          "run_info: < "
          "  benchmark_key: \"b1\" "
          "  run_key: \"r1\" "
          "> ",
          // AnalyzerHistoricQueryOutput
          "status: < "
          "  code: SUCCESS "
          "> "
          "get_batches: false "
          "run_info_query_list: < "
          "  benchmark_key: \"b1\" "
          "  max_timestamp_ms: 10 "
          "  limit: 5 "
          "> "
          "run_info_query_list: < "
          "  benchmark_key: \"b2\" "
          "  max_timestamp_ms: 10 "
          "  limit: 5 "
          "> ",
      },
      {
          "cross_run_config_with_batches",
          // ThresholdAnalyzerInput with a cross_run_config message and no
          // ThresholdConfig using a data filter with METRIC_SAMPLEPOINTS
          "name: \"ta\""
          "cross_run_config: < "
          "  run_info_query_list: < "
          "    benchmark_key: \"b1\" "
          "    max_timestamp_ms: 10 "
          "    limit: 5 "
          "  > "
          "  run_info_query_list: < "
          "    benchmark_key: \"b2\" "
          "    max_timestamp_ms: 10 "
          "    limit: 5 "
          "  > "
          "> "
          "configs: < "
          "  data_filter: < "
          "    data_type: METRIC_AGGREGATE_MEAN "
          "    value_key: \"y1\" "
          "  > "
          "> "
          "configs: < "
          "  data_filter: < "
          "    data_type: METRIC_SAMPLEPOINTS "
          "    value_key: \"y2\" "
          "  > "
          "> ",
          // AnalyzerHistoricQueryInput
          "benchmark_info: < "
          "  benchmark_key: \"b1\" "
          "> "
          "run_info: < "
          "  benchmark_key: \"b1\" "
          "  run_key: \"r1\" "
          "> ",
          // AnalyzerHistoricQueryOutput
          "status: < "
          "  code: SUCCESS "
          "> "
          "get_batches: true "
          "run_info_query_list: < "
          "  benchmark_key: \"b1\" "
          "  max_timestamp_ms: 10 "
          "  limit: 5 "
          "> "
          "run_info_query_list: < "
          "  benchmark_key: \"b2\" "
          "  max_timestamp_ms: 10 "
          "  limit: 5 "
          "> ",
      },
  };
  for (const Case& c : cases) {
    SCOPED_TRACE(c.name);
    LOG(INFO) << "Case: " << c.name;
    ThresholdAnalyzerInput ctor_in;
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(c.ctor_in, &ctor_in));
    AnalyzerHistoricQueryInput func_in;
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(c.func_in, &func_in));
    AnalyzerHistoricQueryOutput want_func_out;
    ASSERT_TRUE(
        google::protobuf::TextFormat::ParseFromString(c.want_func_out, &want_func_out));
    threshold_analyzer::Analyzer ta(ctor_in);
    AnalyzerHistoricQueryOutput got_func_out;
    bool success = ta.ConstructHistoricQuery(func_in, &got_func_out);
    LOG(INFO) << "Got:\n" << got_func_out.DebugString();
    EXPECT_TRUE(success);
    EXPECT_EQ(got_func_out.DebugString(), want_func_out.DebugString());
  }
}

TEST(WindownDeviationTest, AnalyzeCrossRunConfigTest) {
  struct Case {
    std::string name;
    std::string ctor_in;
    std::string func_in;
    bool want_regression;
    Status_Code want_status;
    // strip out config_results.config since it is just a copy of the ctor_in
    std::string threshold_analyzer_output;
  };
  std::vector<Case> cases = {
      {
          "no-regress-current-run",
          // ThresholdAnalyzerInput
          "name: \"ta\""
          "cross_run_config: < "
          "  run_info_query_list: < > "
          "> "
          "configs: < "
          "  min: 5 "
          "  data_filter: < "
          "    data_type: BENCHMARK_SCORE "
          "  > "
          "> ",
          // AnalyzerInput
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"r4\" "
          "    timestamp_ms: 4 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 50 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_status
          Status_Code_SUCCESS,
          // ThresholdAnalyzerOutput
          "min_timestamp_ms: 4 "
          "max_timestamp_ms: 4 "
          "config_results: < "
          "  percent_above_max: 0 "
          "  percent_below_min: 0 "
          "  value_outside_threshold: 50 "
          "  metric_label: \"benchmark_score\""
          "  regression: false "
          "> ",
      },
      {
          "regress-current-run-no-regress-history",
          // ThresholdAnalyzerInput
          "name: \"ta\""
          "cross_run_config: < "
          "  run_info_query_list: < > "
          "> "
          "configs: < "
          "  min: 5 "
          "  data_filter: < "
          "    data_type: BENCHMARK_SCORE "
          "  > "
          "> ",
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 6"
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h2\" "
          "    timestamp_ms: 2 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 6 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h3\" "
          "    timestamp_ms: 3 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 4 "
          "      > "
          "    > "
          "  > "
          "> "
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"r4\" "
          "    timestamp_ms: 4 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 4 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_status
          Status_Code_SUCCESS,
          // ThresholdAnalyzerOutput
          "min_timestamp_ms: 1 "
          "max_timestamp_ms: 4 "
          "config_results: < "
          "  percent_above_max: 0 "
          "  percent_below_min: 0 "
          "  value_outside_threshold: 5 "
          "  metric_label: \"benchmark_score\""
          "  regression: false "
          "  cross_run_config_exercised: true "
          "> ",
      },
      {
          "regress-current-run-and-regress-history",
          // ThresholdAnalyzerInput
          "name: \"ta\""
          "cross_run_config: < "
          "  run_info_query_list: < > "
          "> "
          "configs: < "
          "  min: 5 "
          "  data_filter: < "
          "    data_type: BENCHMARK_SCORE "
          "  > "
          "> ",
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 6"
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h2\" "
          "    timestamp_ms: 2 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 5 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h3\" "
          "    timestamp_ms: 3 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 4 "
          "      > "
          "    > "
          "  > "
          "> "
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"r4\" "
          "    timestamp_ms: 4 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 4 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          true,
          // want_status
          Status_Code_SUCCESS,
          // ThresholdAnalyzerOutput
          "min_timestamp_ms: 1 "
          "max_timestamp_ms: 4 "
          "config_results: < "
          "  percent_above_max: 0 "
          "  percent_below_min: 100 "
          "  value_outside_threshold: 4.5 "
          "  metric_label: \"benchmark_score\""
          "  regression: true "
          "  cross_run_config_exercised: true "
          "> ",
      },
      {
          "regress-current-run-skip-min_run_count",
          // ThresholdAnalyzerInput
          "name: \"ta\""
          "cross_run_config: < "
          "  min_run_count: 3 "
          "  run_info_query_list: < > "
          "> "
          "configs: < "
          "  min: 5 "
          "  data_filter: < "
          "    data_type: BENCHMARK_SCORE "
          "  > "
          "> ",
          // AnalyzerInput

          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h3\" "
          "    timestamp_ms: 3 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 4 "
          "      > "
          "    > "
          "  > "
          "> "
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"r4\" "
          "    timestamp_ms: 4 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 4 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_status
          Status_Code_SUCCESS,
          // ThresholdAnalyzerOutput
          "min_timestamp_ms: 3 "
          "max_timestamp_ms: 4 "
          "config_results: < "
          "  percent_above_max: 0 "
          "  percent_below_min: 100 "
          "  value_outside_threshold: 4 "
          "  metric_label: \"benchmark_score\""
          "  regression: false "
          "  cross_run_config_exercised: true "
          "> ",
      },
      {
          "regress-metric-samplepoints",
          // ThresholdAnalyzerInput
          "name: \"ta\""
          "cross_run_config: < "
          "  run_info_query_list: < > "
          "> "
          "configs: < "
          "  min: 5 "
          "  outlier_percent_max: 35 "
          "  data_filter: < "
          "    data_type: METRIC_SAMPLEPOINTS "
          "    value_key: \"y1\" "
          "  > "
          "> ",
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "  > "
          "  batch_list: < "
          "    run_key: \"h1\" "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h2\" "
          "    timestamp_ms: 2 "
          "  > "
          "  batch_list: < "
          "    run_key: \"h2\" "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h3\" "
          "    timestamp_ms: 3 "
          "  > "
          "  batch_list: < "
          "    run_key: \"h3\" "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 4 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "  > "
          "> "
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"r4\" "
          "    timestamp_ms: 4 "
          "  > "
          "  batch_list: < "
          "    run_key: \"r4\" "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 4 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 4 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_status
          Status_Code_SUCCESS,
          // ThresholdAnalyzerOutput
          "min_timestamp_ms: 1 "
          "max_timestamp_ms: 4 "
          "config_results: < "
          "  percent_above_max: 0 "
          "  percent_below_min: 30 "
          "  metric_label: \"y1\""
          "  regression: false "
          "  cross_run_config_exercised: true "
          "> ",
      },
      {
          "regress-metric-samplepoints-regression",
          // ThresholdAnalyzerInput
          "name: \"ta\""
          "cross_run_config: < "
          "  run_info_query_list: < > "
          "> "
          "configs: < "
          "  min: 5 "
          "  outlier_percent_max: 35 "
          "  data_filter: < "
          "    data_type: METRIC_SAMPLEPOINTS "
          "    value_key: \"y1\" "
          "  > "
          "> ",
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "  > "
          "  batch_list: < "
          "    run_key: \"h1\" "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 4 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h2\" "
          "    timestamp_ms: 2 "
          "  > "
          "  batch_list: < "
          "    run_key: \"h2\" "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h3\" "
          "    timestamp_ms: 3 "
          "  > "
          "  batch_list: < "
          "    run_key: \"h3\" "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 4 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 6 "
          "      > "
          "    > "
          "  > "
          "> "
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"r4\" "
          "    timestamp_ms: 4 "
          "  > "
          "  batch_list: < "
          "    run_key: \"r4\" "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 4 "
          "      > "
          "    > "
          "    sample_point_list: < "
          "      input_value: 1 "
          "      metric_value_list: < "
          "        value_key: \"y1\" "
          "        value: 4 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          true,
          // want_status
          Status_Code_SUCCESS,
          // ThresholdAnalyzerOutput
          "min_timestamp_ms: 1 "
          "max_timestamp_ms: 4 "
          "config_results: < "
          "  percent_above_max: 0 "
          "  percent_below_min: 40 "
          "  metric_label: \"y1\""
          "  regression: true "
          "  cross_run_config_exercised: true "
          "> ",
      },
  };
  for (const Case& c : cases) {
    SCOPED_TRACE(c.name);
    LOG(INFO) << "Case: " << c.name;
    ThresholdAnalyzerInput ctor_in;
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(c.ctor_in, &ctor_in));
    AnalyzerInput func_in;
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(c.func_in, &func_in));
    threshold_analyzer::Analyzer ta(ctor_in);
    AnalyzerOutput got;
    bool success = ta.Analyze(func_in, &got);
    LOG(INFO) << "Got:\n" << got.DebugString();
    ThresholdAnalyzerOutput ta_output;
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(got.output(), &ta_output));
    EXPECT_EQ(got.regression(), c.want_regression);
    EXPECT_EQ(got.status().code(), c.want_status);
    EXPECT_EQ(got.analyzer_type(), ta.analyzer_type());
    EXPECT_EQ(got.run_key(), func_in.run_to_be_analyzed().run_info().run_key());
    ThresholdAnalyzerInput parsed_input_config;
    ASSERT_TRUE(google::protobuf::TextFormat::ParseFromString(got.input_config(),
                                                    &parsed_input_config));
    // Clear the configs, we don't want to validate them explicitly in this
    // test.
    for (auto& config_result : *ta_output.mutable_config_results()) {
      config_result.clear_config();
    }
    EXPECT_THAT(ta_output, EqualsProto(c.threshold_analyzer_output));
    EXPECT_THAT(parsed_input_config, EqualsProto(ctor_in));
    if (c.want_status == Status_Code_SUCCESS) {
      EXPECT_TRUE(success);
    } else {
      EXPECT_FALSE(success);
      continue;
    }
  }
}  // NOLINT(readability/fn_size)

}  // namespace threshold_analyzer
}  // namespace mako
