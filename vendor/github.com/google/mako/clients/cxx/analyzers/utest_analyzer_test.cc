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
#include "clients/cxx/analyzers/utest_analyzer.h"

#include <numeric>
#include <random>
#include <sstream>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/text_format.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/proto/analyzers/utest_analyzer.pb.h"
#include "spec/proto/mako.pb.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/string_view.h"

namespace mako {
namespace utest_analyzer {

using mako::AnalyzerInput;
using mako::AnalyzerOutput;
using mako::BenchmarkInfo;
using mako::RunBundle;
using mako::RunInfo;
using mako::utest_analyzer::UTestAnalyzerInput;
using mako::utest_analyzer::UTestAnalyzerOutput;
using mako::utest_analyzer::UTestConfig;
using mako::utest_analyzer::UTestConfig_DirectionBias_IGNORE_DECREASE;
using mako::utest_analyzer::UTestConfig_DirectionBias_IGNORE_INCREASE;
using mako::utest_analyzer::UTestConfig_DirectionBias_NO_BIAS;

constexpr char kSampleAKey[] = "a";
constexpr char kSampleBKey[] = "b";

const double kEpsilon = 0.005;

const char* kBenchmarkKey = "BenchmarkKey_1";
const double kBenchmarkScore = 3.6;

const char* kAnalyzerName = "Anlyzer_1";
const char* kConfigName1 = "MyUniqueConfigName_1";
const char* kConfigName2 = "MyUniqueConfigName_2";

const char* kRunKey1 = "RunKey_1";
const char* kMetricKey1 = "MetricKey_1";
const char* kRunKey2 = "RunKey_2";
const char* kMetricKey2 = "MetricKey_2";

const int kDistSize = 80;
const double kCenter = 0;
const double kDistStdDev = 0.6;
const double kDistSkewShift = 0.6;
const double kDistShift = 2;

std::vector<double> HelperCreateRandomSkewedDist(double mean, double mode,
                                                 double stddev, int size);

class AnalyzerTest : public ::testing::Test {
 protected:
  AnalyzerTest() {}

  ~AnalyzerTest() override {}

  void SetUp() override {}

  void TearDown() override {}

  void HelperCheckMultipleSampleCombinations(
      bool slight, bool very, const UTestConfig::DirectionBias& bias,
      double sig_level);

 public:
  const std::vector<double> kCenterDist =
      HelperCreateRandomSkewedDist(kCenter, kCenter, kDistStdDev, kDistSize);
  const std::vector<double> kLeftSkewedDist = HelperCreateRandomSkewedDist(
      kCenter - kDistSkewShift, kCenter, kDistStdDev, kDistSize);
  const std::vector<double> kRightSkewedDist = HelperCreateRandomSkewedDist(
      kCenter + kDistSkewShift, kCenter, kDistStdDev, kDistSize);
  const std::vector<double> kCenterDistShiftedLeft =
      HelperCreateRandomSkewedDist(kCenter - kDistShift, kCenter - kDistShift,
                                   kDistStdDev, kDistSize);
};

std::vector<double> HelperCreateRandomSkewedDist(double mean, double mode,
                                                 double stddev, int size) {
  std::minstd_rand0 generator(200600613);
  std::normal_distribution<double> base_dist(mean, stddev);
  std::normal_distribution<double> mode_dist(mode, stddev * 0.4);

  std::vector<double> dist;
  dist.reserve(size);

  for (int i = 0; i < size * 0.3; i++) {
    dist.push_back(base_dist(generator));
  }

  for (int i = 0; i < size * 0.70; i++) {
    dist.push_back(mode_dist(generator));
  }

  return dist;
}

BenchmarkInfo HelperCreateBenchmarkInfo() {
  BenchmarkInfo benchmark_info;
  benchmark_info.set_benchmark_key(kBenchmarkKey);
  return benchmark_info;
}

RunInfo HelperCreateRunInfo(std::string run_key) {
  RunInfo run_info;
  run_info.set_benchmark_key(kBenchmarkKey);
  run_info.set_run_key(run_key);

  auto aggregate = run_info.mutable_aggregate();

  // Set run aggregates
  auto run_aggregate = aggregate->mutable_run_aggregate();
  run_aggregate->set_benchmark_score(kBenchmarkScore);

  return run_info;
}

RunInfoQuery HelperCreateRunInfoQuery(std::string run_key) {
  RunInfoQuery run_query;
  run_query.set_run_key(run_key);
  return run_query;
}

RunBundle HelperCreateRunBundle(std::string run_key = kRunKey1) {
  RunBundle run_bundle;
  *run_bundle.mutable_run_info() = HelperCreateRunInfo(run_key);
  *run_bundle.mutable_benchmark_info() = HelperCreateBenchmarkInfo();
  return run_bundle;
}

AnalyzerInput HelperCreateAnalyzerInput() {
  AnalyzerInput analyzer_input;
  *analyzer_input.mutable_run_to_be_analyzed() = HelperCreateRunBundle();
  return analyzer_input;
}

UTestSample HelperCreateUTestSample(const std::vector<std::string> run_key_list,
                                    bool include_curr_run) {
  UTestSample sample;
  for (const std::string& run_key : run_key_list) {
    RunInfoQuery* query = sample.add_run_query_list();
    query->set_run_key(run_key);
  }
  sample.set_include_current_run(include_curr_run);
  return sample;
}

UTestConfig HelperCreateUTestAnalyzerConfig(
    const std::string& a_metric_key, const std::string& b_metric_key,
    const UTestConfig::DirectionBias& dir_bias, double sig_level) {
  UTestConfig config;
  config.set_a_metric_key(a_metric_key);
  config.set_b_metric_key(b_metric_key);
  config.set_direction_bias(dir_bias);
  config.set_significance_level(sig_level);
  return config;
}

UTestConfig HelperCreateUTestAnalyzerConfigWithShiftValue(
    const std::string& a_metric_key, const std::string& b_metric_key, double shift_value,
    const UTestConfig::DirectionBias& dir_bias, double sig_level) {
  UTestConfig config = HelperCreateUTestAnalyzerConfig(
      a_metric_key, b_metric_key, dir_bias, sig_level);
  config.set_shift_value(shift_value);
  return config;
}

UTestConfig HelperCreateUTestAnalyzerConfigWithRelativeShiftValue(
    const std::string& a_metric_key, const std::string& b_metric_key,
    double relative_shift_value, const UTestConfig::DirectionBias& dir_bias,
    double sig_level) {
  UTestConfig config = HelperCreateUTestAnalyzerConfig(
      a_metric_key, b_metric_key, dir_bias, sig_level);
  config.set_relative_shift_value(relative_shift_value);
  return config;
}

UTestAnalyzerInput HelperCreateUTestAnalyzerInput(
    const UTestSample& samp_a, const UTestSample& samp_b, std::string a_metric_key,
    std::string b_metric_key, double shift_value,
    const UTestConfig::DirectionBias& dir_bias, double sig_level) {
  UTestAnalyzerInput input;
  *input.mutable_a_sample() = samp_a;
  *input.mutable_b_sample() = samp_b;
  *input.add_config_list() = HelperCreateUTestAnalyzerConfigWithShiftValue(
      a_metric_key, b_metric_key, shift_value, dir_bias, sig_level);

  return input;
}

Analyzer HelperCreateUTestAnalyzer(const UTestSample& samp_a,
                                   const UTestSample& samp_b,
                                   std::string a_metric_key, std::string b_metric_key,
                                   double shift_value,
                                   const UTestConfig::DirectionBias& dir_bias,
                                   double sig_level) {
  return Analyzer(HelperCreateUTestAnalyzerInput(samp_a, samp_b, a_metric_key,
                                                 b_metric_key, shift_value,
                                                 dir_bias, sig_level));
}

void HelperAddSamplePoints(std::string value_key, const std::vector<double>& data,
                           RunBundle* run_bundle) {
  auto sample_batch = run_bundle->add_batch_list();
  for (auto& d : data) {
    auto sample_point = sample_batch->add_sample_point_list();
    sample_point->set_input_value(1);
    auto keyed_value = sample_point->add_metric_value_list();
    keyed_value->set_value_key(value_key);
    keyed_value->set_value(d);
  }
}

bool HelperCompareTwoSamples(const std::vector<double>& samp_1,
                             const std::vector<double>& samp_2,
                             double shift_value,
                             const UTestConfig::DirectionBias& dir_bias,
                             double sig_level, AnalyzerOutput* output) {
  UTestSample temp1 = HelperCreateUTestSample({kRunKey1}, true);
  UTestSample temp2 = HelperCreateUTestSample({kRunKey2}, false);
  auto analyzer = HelperCreateUTestAnalyzer(
      temp1, temp2, kMetricKey1, kMetricKey2, shift_value, dir_bias, sig_level);

  auto input = HelperCreateAnalyzerInput();

  HelperAddSamplePoints(kMetricKey1, samp_1,
                        input.mutable_run_to_be_analyzed());

  RunBundle* historical_run = (*input.mutable_historical_run_map())[kSampleBKey]
                                  .add_historical_run_list();
  HelperAddSamplePoints(kMetricKey2, samp_2, historical_run);
  *historical_run->mutable_run_info() = HelperCreateRunInfo(kRunKey2);

  return analyzer.Analyze(input, output);
}

bool SuccessfulStatus(const AnalyzerOutput& output) {
  return output.has_status() && output.status().has_code() &&
         output.status().code() == mako::Status_Code_SUCCESS;
}

bool InvalidProto(Analyzer* analyzer) {
  auto input = HelperCreateAnalyzerInput();
  AnalyzerOutput output;
  return !analyzer->Analyze(input, &output) && !SuccessfulStatus(output);
}

UTestAnalyzerOutput ExtractUTestAnalyzerOutput(const AnalyzerOutput& output) {
  UTestAnalyzerOutput a_out;
  google::protobuf::TextFormat::ParseFromString(output.output(), &a_out);
  return a_out;
}

TEST_F(AnalyzerTest, AnalyzerInputSampleIncomplete) {
  // UTestSample empty
  UTestSample temp;
  auto analyzer = HelperCreateUTestAnalyzer(
      temp, temp, "A", "B", 0, UTestConfig_DirectionBias_IGNORE_INCREASE, 0.01);

  EXPECT_TRUE(InvalidProto(&analyzer));
}

TEST_F(AnalyzerTest, AnalyzerInputInvalidConfigSigLevel) {
  LOG(INFO) << "No significance level specified";
  std::vector<std::string> run_key_list;
  UTestSample temp = HelperCreateUTestSample(run_key_list, true);

  auto input = HelperCreateUTestAnalyzerInput(
      temp, temp, "A", "B", 0, UTestConfig_DirectionBias_IGNORE_INCREASE, 0.03);

  // Clear significance level in config
  input.mutable_config_list(0)->clear_significance_level();

  auto analyzer_no_sig_level = Analyzer(input);

  EXPECT_TRUE(InvalidProto(&analyzer_no_sig_level));

  LOG(INFO) << "Significance level invalid (sig_level = 1.0)";
  auto analyzer_sig_level_one = HelperCreateUTestAnalyzer(
      temp, temp, "A", "B", 0, UTestConfig_DirectionBias_IGNORE_INCREASE, 1.0);

  EXPECT_TRUE(InvalidProto(&analyzer_sig_level_one));
}

TEST_F(AnalyzerTest, AnalyzerInputInvalidMetricString) {
  LOG(INFO) << "Sample A metric std::string is empty";
  std::vector<std::string> run_key_list;
  UTestSample temp = HelperCreateUTestSample(run_key_list, true);
  auto analyzer_empty_str_a = HelperCreateUTestAnalyzer(
      temp, temp, "", "B", 0, UTestConfig_DirectionBias_IGNORE_INCREASE, 0.03);
  EXPECT_TRUE(InvalidProto(&analyzer_empty_str_a));

  LOG(INFO) << "Sample B metric std::string is not present";
  auto input = HelperCreateUTestAnalyzerInput(
      temp, temp, "A", "B", 0, UTestConfig_DirectionBias_IGNORE_INCREASE, 0.03);
  input.mutable_config_list(0)->clear_b_metric_key();

  auto analyzer_b_metric_not_present = Analyzer(input);
  EXPECT_TRUE(InvalidProto(&analyzer_b_metric_not_present));
}

TEST_F(AnalyzerTest, AnalyzerInputInvalidBothMetricStringAndDataFilter) {
  LOG(INFO) << "Sample A metric std::string is empty";

  mako::DataFilter data_filter;
  data_filter.set_value_key("y1");
  data_filter.set_data_type(mako::DataFilter::METRIC_AGGREGATE_MEDIAN);

  std::vector<std::string> run_key_list;
  UTestSample temp = HelperCreateUTestSample(run_key_list, true);

  LOG(INFO) << "Sample B metric std::string is not present";
  auto input_a = HelperCreateUTestAnalyzerInput(
      temp, temp, "A", "B", 0, UTestConfig_DirectionBias_IGNORE_INCREASE, 0.03);
  *input_a.mutable_config_list(0)->mutable_a_data_filter() = data_filter;

  auto analyzer_a_metric_not_present = Analyzer(input_a);
  EXPECT_TRUE(InvalidProto(&analyzer_a_metric_not_present));

  LOG(INFO) << "Sample B metric std::string is not present";
  auto input_b = HelperCreateUTestAnalyzerInput(
      temp, temp, "A", "B", 0, UTestConfig_DirectionBias_IGNORE_INCREASE, 0.03);
  *input_b.mutable_config_list(0)->mutable_b_data_filter() = data_filter;

  auto analyzer_b_metric_not_present = Analyzer(input_b);
  EXPECT_TRUE(InvalidProto(&analyzer_b_metric_not_present));
}

TEST_F(AnalyzerTest, AnalyzerInputMissingRunToBeAnalyzedOrRunInfo) {
  UTestSample temp1 = HelperCreateUTestSample({kRunKey1}, true);
  UTestSample temp2 = HelperCreateUTestSample({kRunKey2}, false);
  auto analyzer =
      HelperCreateUTestAnalyzer(temp1, temp2, kMetricKey1, kMetricKey2, 0,
                                UTestConfig_DirectionBias_NO_BIAS, 0.01);

  LOG(INFO) << "Passing in input without run_to_be_analyzed";
  // Clear run to be analyzed
  auto input = HelperCreateAnalyzerInput();
  input.clear_run_to_be_analyzed();

  AnalyzerOutput output_no_run_to_be_analyzed;
  EXPECT_FALSE(analyzer.Analyze(input, &output_no_run_to_be_analyzed));
  EXPECT_FALSE(SuccessfulStatus(output_no_run_to_be_analyzed));

  LOG(INFO) << "Passing in input without run_to_be_analyzed's run_info";
  // Clear run info from run bundle
  input = HelperCreateAnalyzerInput();
  input.mutable_run_to_be_analyzed()->clear_run_info();

  AnalyzerOutput output_no_run_info_in_bundle;
  EXPECT_FALSE(analyzer.Analyze(input, &output_no_run_info_in_bundle));
  EXPECT_FALSE(SuccessfulStatus(output_no_run_info_in_bundle));
}

TEST_F(AnalyzerTest, AnalyzerInputNoSampleSpecified) {
  LOG(INFO) << "No B sample specified.";
  UTestAnalyzerInput input;
  *input.mutable_a_sample() = HelperCreateUTestSample({kRunKey1}, true);
  *input.add_config_list() = HelperCreateUTestAnalyzerConfigWithShiftValue(
      kMetricKey1, kMetricKey2, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05);

  auto analyzer_no_b_sample_specified = Analyzer(input);

  EXPECT_TRUE(InvalidProto(&analyzer_no_b_sample_specified));

  LOG(INFO) << "No A sample specified.";
  input.clear_a_sample();
  *input.mutable_b_sample() = HelperCreateUTestSample({kRunKey1}, true);

  auto analyzer_no_a_sample_specified = Analyzer(input);

  EXPECT_TRUE(InvalidProto(&analyzer_no_a_sample_specified));
}

TEST_F(AnalyzerTest, AnalyzerInputNoConfigsSpecified) {
  UTestAnalyzerInput input;
  *input.mutable_a_sample() = HelperCreateUTestSample({kRunKey1}, true);
  *input.mutable_b_sample() = HelperCreateUTestSample({kRunKey1}, true);

  auto analyzer = Analyzer(input);

  EXPECT_TRUE(InvalidProto(&analyzer));
}

TEST_F(AnalyzerTest,
       AnalyzerInputSpecifiedBothShiftValueAndRelativeShiftValue) {
  UTestAnalyzerInput input;
  *input.mutable_a_sample() = HelperCreateUTestSample({kRunKey1}, true);
  *input.mutable_b_sample() = HelperCreateUTestSample({kRunKey1}, true);
  mako::utest_analyzer::UTestConfig* config =
      input.mutable_config_list()->Add();

  config->set_shift_value(0.0);
  config->set_relative_shift_value(0.0);

  auto analyzer = Analyzer(input);

  EXPECT_TRUE(InvalidProto(&analyzer));
}

TEST_F(AnalyzerTest, AnalyzerInputIncludesCurrentRun) {
  UTestSample temp1 = HelperCreateUTestSample({"r1"}, false);
  UTestSample temp2 = HelperCreateUTestSample({"r2"}, false);
  // Note 0.999 significance level - samples should be found to be the exact
  // same
  auto analyzer = HelperCreateUTestAnalyzer(
      temp1, temp2, "k1", "k1", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);

  auto input = HelperCreateAnalyzerInput();

  // Current run - should only be found in temp2
  HelperAddSamplePoints("k1", kCenterDist, input.mutable_run_to_be_analyzed());

  // R1 run - should only be found in temp1
  auto& run_map = *input.mutable_historical_run_map();
  RunBundle* historical_run = run_map[kSampleAKey].add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r1");
  HelperAddSamplePoints("k1", kLeftSkewedDist, historical_run);

  // R2 run - should only be found in temp2
  historical_run = run_map[kSampleBKey].add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r2");
  HelperAddSamplePoints("k1", kLeftSkewedDist, historical_run);

  LOG(INFO) << "Neither sample sets include_current_run";
  AnalyzerOutput output;
  EXPECT_TRUE(analyzer.Analyze(input, &output));

  LOG(INFO) << "Sample A sets include_current_run";
  temp1 = HelperCreateUTestSample({"r1"}, true);
  analyzer = HelperCreateUTestAnalyzer(
      temp1, temp2, "k1", "k1", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);
  EXPECT_TRUE(analyzer.Analyze(input, &output));

  LOG(INFO) << "Both samples set include_current_run";
  temp2 = HelperCreateUTestSample({"r2"}, true);
  analyzer = HelperCreateUTestAnalyzer(
      temp1, temp2, "k1", "k1", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);
  EXPECT_TRUE(analyzer.Analyze(input, &output));

  LOG(INFO) << "Sample B sets include_current_run";
  temp1 = HelperCreateUTestSample({"r1"}, false);
  analyzer = HelperCreateUTestAnalyzer(
      temp1, temp2, "k1", "k1", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);
  EXPECT_TRUE(analyzer.Analyze(input, &output));
}

TEST_F(AnalyzerTest, AnalyzerInputSampleTooLittleDataPoints) {
  LOG(INFO) << "Sample A only has two points.";
  AnalyzerOutput output_sample_a_small;
  bool result_sample_a_small = HelperCompareTwoSamples(
      {3, 4}, {1, 5, 3, 4}, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
      &output_sample_a_small);
  EXPECT_FALSE(result_sample_a_small);
  EXPECT_FALSE(SuccessfulStatus(output_sample_a_small));

  LOG(INFO) << "Sample B only has two points.";
  AnalyzerOutput output_sample_b_small;
  bool result_sample_b_small = HelperCompareTwoSamples(
      {4, 6, 7, 3}, {1, 4}, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
      &output_sample_b_small);
  EXPECT_FALSE(result_sample_b_small);
  EXPECT_FALSE(SuccessfulStatus(output_sample_b_small));

  LOG(INFO) << "No datapoints in either sample.";
  AnalyzerOutput output_no_data_points;
  bool result_no_data_points =
      HelperCompareTwoSamples({}, {}, 0, UTestConfig_DirectionBias_NO_BIAS,
                              0.05, &output_no_data_points);
  EXPECT_FALSE(result_no_data_points);
  EXPECT_FALSE(SuccessfulStatus(output_no_data_points));
}

TEST_F(AnalyzerTest, AnalyzerRunInfoQueryForEveryRunKey) {
  UTestSample temp1 = HelperCreateUTestSample({}, true);
  UTestSample temp2 = HelperCreateUTestSample({}, false);

  // Add 100 RunInfoQueries
  for (int i = 0; i < 100; i += 2) {
    *temp1.add_run_query_list() =
        HelperCreateRunInfoQuery(absl::StrCat("run_key", i));
    *temp2.add_run_query_list() =
        HelperCreateRunInfoQuery(absl::StrCat("run_key", i + 1));
  }
  auto analyzer =
      HelperCreateUTestAnalyzer(temp1, temp2, kMetricKey1, kMetricKey2, 0,
                                UTestConfig_DirectionBias_NO_BIAS, 0.05);
  AnalyzerHistoricQueryInput input;
  AnalyzerHistoricQueryOutput output;
  EXPECT_TRUE(analyzer.ConstructHistoricQuery(input, &output));
  EXPECT_EQ(output.status().code(), Status_Code_SUCCESS);
  EXPECT_TRUE(output.get_batches());

  // Verify that map and list were both populated.
  EXPECT_EQ(output.run_info_query_map_size(), 2);
  EXPECT_EQ(
      output.run_info_query_map().at(kSampleAKey).run_info_query_list_size(),
      50);
  EXPECT_EQ(
      output.run_info_query_map().at(kSampleBKey).run_info_query_list_size(),
      50);

  EXPECT_EQ(output.run_info_query_list_size(), 100);

  // Verify all 100 RunInfoQueries are found in request
  bool run_key_found[100] = {};
  for (const std::string& sample_key : {kSampleAKey, kSampleBKey}) {
    for (const auto& query :
         output.run_info_query_map().at(sample_key).run_info_query_list()) {
      EXPECT_EQ(query.run_key().find("run_key"), 0);
      run_key_found[std::stoi(query.run_key().substr(7))] = true;
    }
  }
  for (int i = 0; i < 100; i++) {
    EXPECT_TRUE(run_key_found[i]);
  }
}

TEST_F(AnalyzerTest, AnalyzerInputQueryWithNoRunKey) {
  UTestSample temp1 = HelperCreateUTestSample({}, false);
  UTestSample temp2 = HelperCreateUTestSample({}, false);

  RunInfoQuery* q = temp1.add_run_query_list();
  q->set_limit(250);
  *(q->mutable_cursor()) = "A";
  q = temp2.add_run_query_list();
  q->set_limit(150);
  *(q->mutable_cursor()) = "B";

  auto analyzer =
      HelperCreateUTestAnalyzer(temp1, temp2, kMetricKey1, kMetricKey2, 0,
                                UTestConfig_DirectionBias_NO_BIAS, 0.999);

  AnalyzerHistoricQueryInput input;
  AnalyzerHistoricQueryOutput output;

  *input.mutable_benchmark_info()->mutable_benchmark_key() = kBenchmarkKey;

  EXPECT_TRUE(analyzer.ConstructHistoricQuery(input, &output));
  EXPECT_EQ(output.status().code(), Status_Code_SUCCESS);
  EXPECT_TRUE(output.get_batches());

  auto& a_sample_queries = output.run_info_query_map().at(kSampleAKey);
  auto& b_sample_queries = output.run_info_query_map().at(kSampleBKey);

  EXPECT_EQ(a_sample_queries.run_info_query_list_size(), 1);
  EXPECT_EQ(b_sample_queries.run_info_query_list_size(), 1);

  EXPECT_EQ(a_sample_queries.run_info_query_list().at(0).benchmark_key(),
            kBenchmarkKey);
  EXPECT_EQ(a_sample_queries.run_info_query_list().at(0).limit(), 250);
  EXPECT_EQ(a_sample_queries.run_info_query_list().at(0).cursor(), "A");

  EXPECT_EQ(b_sample_queries.run_info_query_list().at(0).benchmark_key(),
            kBenchmarkKey);
  EXPECT_EQ(b_sample_queries.run_info_query_list().at(0).limit(), 150);
  EXPECT_EQ(b_sample_queries.run_info_query_list().at(0).cursor(), "B");
}

TEST_F(AnalyzerTest, AnalyzerExtractsCorrectData) {
  UTestSample temp1 = HelperCreateUTestSample({"r1", "r3"}, false);
  UTestSample temp2 = HelperCreateUTestSample({"r2"}, true);
  // Note 0.999 significance level - samples should be found to be the exact
  // same
  auto analyzer_key1 = HelperCreateUTestAnalyzer(
      temp1, temp2, "k1", "k1", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);
  auto analyzer_key2 = HelperCreateUTestAnalyzer(
      temp1, temp2, "k2", "k2", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);

  auto input = HelperCreateAnalyzerInput();

  // Current run - should only be found in temp2
  HelperAddSamplePoints("k1", kCenterDist, input.mutable_run_to_be_analyzed());
  HelperAddSamplePoints("k2", kCenterDist, input.mutable_run_to_be_analyzed());

  // R1 run - should only be found in temp1
  auto& run_map = *input.mutable_historical_run_map();
  RunBundle* historical_run = run_map[kSampleAKey].add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r1");
  HelperAddSamplePoints("k1", kLeftSkewedDist, historical_run);
  HelperAddSamplePoints("k2", kLeftSkewedDist, historical_run);

  // R2 run - should only be found in temp2
  historical_run = run_map[kSampleBKey].add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r2");
  HelperAddSamplePoints("k1", kLeftSkewedDist, historical_run);
  HelperAddSamplePoints("k2", kLeftSkewedDist, historical_run);

  // R3 run - should only be found in temp1
  historical_run = run_map[kSampleAKey].add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r3");
  HelperAddSamplePoints("k1", kCenterDist, historical_run);
  HelperAddSamplePoints("k2", kCenterDist, historical_run);
  // K3 key should be ignored
  HelperAddSamplePoints("k3", kCenterDist, historical_run);

  // Samples should be found to be the exact same (no regression with 0.999
  // sig_level)
  AnalyzerOutput output_key1;
  EXPECT_TRUE(analyzer_key1.Analyze(input, &output_key1));
  EXPECT_FALSE(output_key1.regression());
  EXPECT_TRUE(SuccessfulStatus(output_key1));

  AnalyzerOutput output_key2;
  EXPECT_TRUE(analyzer_key2.Analyze(input, &output_key2));
  EXPECT_FALSE(output_key2.regression());
  EXPECT_TRUE(SuccessfulStatus(output_key2));
}

// Test is similar to AnalyzerExtractsCorrectData but populates the
// historical_run_list instead of historical_run_map with the run bundles.
TEST_F(AnalyzerTest, AnalyzerExtractsCorrectDataWithList) {
  UTestSample temp1 = HelperCreateUTestSample({"r1", "r3"}, false);
  UTestSample temp2 = HelperCreateUTestSample({"r2"}, true);

  auto analyzer1 = HelperCreateUTestAnalyzer(
      temp1, temp2, "k1", "k1", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);
  auto analyzer2 = HelperCreateUTestAnalyzer(
      temp1, temp2, "k2", "k2", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);

  auto input = HelperCreateAnalyzerInput();

  HelperAddSamplePoints("k1", kCenterDist, input.mutable_run_to_be_analyzed());
  HelperAddSamplePoints("k2", kCenterDist, input.mutable_run_to_be_analyzed());

  RunBundle* historical_run = input.add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r1");
  HelperAddSamplePoints("k1", kLeftSkewedDist, historical_run);
  HelperAddSamplePoints("k2", kLeftSkewedDist, historical_run);

  historical_run = input.add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r2");
  HelperAddSamplePoints("k1", kLeftSkewedDist, historical_run);
  HelperAddSamplePoints("k2", kLeftSkewedDist, historical_run);

  historical_run = input.add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r3");
  HelperAddSamplePoints("k1", kCenterDist, historical_run);
  HelperAddSamplePoints("k2", kCenterDist, historical_run);

  AnalyzerOutput output1;
  EXPECT_TRUE(analyzer1.Analyze(input, &output1));
  EXPECT_FALSE(output1.regression());
  EXPECT_TRUE(SuccessfulStatus(output1));

  AnalyzerOutput output2;
  EXPECT_TRUE(analyzer2.Analyze(input, &output2));
  EXPECT_FALSE(output2.regression());
  EXPECT_TRUE(SuccessfulStatus(output2));
}

TEST_F(AnalyzerTest, AnalyzerExtractsCorrectDataWithListButOneNoRunKey) {

  UTestSample temp1 = HelperCreateUTestSample({"r1", "r3"}, false);
  UTestSample temp2 = HelperCreateUTestSample({"r2"}, true);

  auto analyzer = HelperCreateUTestAnalyzer(
      temp1, temp2, "k1", "k2", 0, UTestConfig_DirectionBias_NO_BIAS, 0.999);

  auto input = HelperCreateAnalyzerInput();

  HelperAddSamplePoints("k1", kCenterDist, input.mutable_run_to_be_analyzed());
  HelperAddSamplePoints("k2", kCenterDist, input.mutable_run_to_be_analyzed());

  RunBundle* historical_run = input.add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r1");
  HelperAddSamplePoints("k1", kLeftSkewedDist, historical_run);
  HelperAddSamplePoints("k2", kLeftSkewedDist, historical_run);

  historical_run = input.add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r2");
  HelperAddSamplePoints("k1", kLeftSkewedDist, historical_run);
  HelperAddSamplePoints("k2", kLeftSkewedDist, historical_run);

  historical_run = input.add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r3");
  HelperAddSamplePoints("k1", kCenterDist, historical_run);
  HelperAddSamplePoints("k2", kCenterDist, historical_run);

  // This run will generate a warning because run_key = r4 does not exist in the
  // UTestSamples.
  historical_run = input.add_historical_run_list();
  *historical_run->mutable_run_info() = HelperCreateRunInfo("r4");
  HelperAddSamplePoints("k1", kCenterDist, historical_run);
  HelperAddSamplePoints("k2", kCenterDist, historical_run);

  AnalyzerOutput output;
  EXPECT_TRUE(analyzer.Analyze(input, &output));
  EXPECT_FALSE(output.regression());
  EXPECT_TRUE(SuccessfulStatus(output));
}

TEST_F(AnalyzerTest, AnalyzerDataFilters) {
  UTestSample temp1 = HelperCreateUTestSample({"r1", "r2", "r3"}, false);
  UTestSample temp2 = HelperCreateUTestSample({"r4", "r5"}, true);

  // Note 0.999 significance level - samples should be found to be the exact
  // same
  UTestConfig config_1;
  mako::DataFilter data_filter_k1;
  data_filter_k1.set_value_key("k1");
  data_filter_k1.set_data_type(
      mako::DataFilter::METRIC_AGGREGATE_PERCENTILE);
  data_filter_k1.set_percentile_milli_rank(99000);
  *config_1.mutable_a_data_filter() = data_filter_k1;
  *config_1.mutable_b_data_filter() = data_filter_k1;
  config_1.set_direction_bias(UTestConfig_DirectionBias_NO_BIAS);
  config_1.set_significance_level(0.999);
  UTestAnalyzerInput input_1;
  *input_1.mutable_a_sample() = temp1;
  *input_1.mutable_b_sample() = temp2;
  *input_1.add_config_list() = config_1;
  Analyzer analyzer_1(input_1);

  UTestConfig config_2;
  mako::DataFilter data_filter_k2;
  data_filter_k2.set_value_key("k2");
  data_filter_k2.set_data_type(mako::DataFilter::METRIC_AGGREGATE_MEDIAN);
  *config_2.mutable_a_data_filter() = data_filter_k2;
  *config_2.mutable_b_data_filter() = data_filter_k2;
  config_2.set_direction_bias(UTestConfig_DirectionBias_NO_BIAS);
  config_2.set_significance_level(0.999);
  UTestAnalyzerInput input_2;
  *input_2.mutable_a_sample() = temp1;
  *input_2.mutable_b_sample() = temp2;
  *input_2.add_config_list() = config_2;
  Analyzer analyzer_2(input_2);

  auto input = HelperCreateAnalyzerInput();

  // Current run - should only be found in temp2
  {
    auto* agg = input.mutable_run_to_be_analyzed()
                    ->mutable_run_info()
                    ->mutable_aggregate();
    agg->add_percentile_milli_rank_list(99000);
    auto* m_agg_k1 = agg->add_metric_aggregate_list();
    m_agg_k1->set_metric_key("k1");
    m_agg_k1->add_percentile_list(20);
    auto* m_agg_k2 = agg->add_metric_aggregate_list();
    m_agg_k2->set_metric_key("k2");
    m_agg_k2->set_median(10);
  }

  for (auto& key :
       {std::make_pair("r1", kSampleAKey), std::make_pair("r2", kSampleAKey),
        std::make_pair("r3", kSampleAKey), std::make_pair("r4", kSampleBKey),
        std::make_pair("r5", kSampleBKey)}) {
    RunBundle* historical_run =
        (*input.mutable_historical_run_map())[key.second]
            .add_historical_run_list();
    *historical_run->mutable_run_info() = HelperCreateRunInfo(key.first);
    auto* agg = historical_run->mutable_run_info()->mutable_aggregate();
    agg->add_percentile_milli_rank_list(99000);
    auto* m_agg_k1 = agg->add_metric_aggregate_list();
    m_agg_k1->set_metric_key("k1");
    m_agg_k1->add_percentile_list(20);
    auto* m_agg_k2 = agg->add_metric_aggregate_list();
    m_agg_k2->set_metric_key("k2");
    m_agg_k2->set_median(10);
    // K3 should be ignored
    auto* m_agg_k3 = agg->add_metric_aggregate_list();
    m_agg_k3->set_metric_key("k3");
    m_agg_k3->set_median(100);
    m_agg_k3->add_percentile_list(200);
  }

  // Samples should be found to be the exact same (no regression with 0.999
  // sig_level)
  AnalyzerOutput output_key1;
  EXPECT_TRUE(analyzer_1.Analyze(input, &output_key1));
  EXPECT_FALSE(output_key1.regression());
  EXPECT_TRUE(SuccessfulStatus(output_key1));

  AnalyzerOutput output_key2;
  EXPECT_TRUE(analyzer_2.Analyze(input, &output_key2));
  EXPECT_FALSE(output_key2.regression());
  EXPECT_TRUE(SuccessfulStatus(output_key2));
}

TEST_F(AnalyzerTest, AnalyzerWithAnalyzerName) {
  UTestSample temp1 = HelperCreateUTestSample({kRunKey1}, true);
  UTestSample temp2 = HelperCreateUTestSample({kRunKey2}, false);

  AnalyzerInput input = HelperCreateAnalyzerInput();
  HelperAddSamplePoints(kMetricKey1, {1, 2, 3},
                        input.mutable_run_to_be_analyzed());
  RunBundle* historical_run = (*input.mutable_historical_run_map())[kSampleBKey]
                                  .add_historical_run_list();
  HelperAddSamplePoints(kMetricKey2, {1, 1, 1}, historical_run);
  *historical_run->mutable_run_info() = HelperCreateRunInfo(kRunKey2);

  LOG(INFO) << "Checking if analyzer name passed by the user is present in the "
               "output (passing output):";
  auto config =
      HelperCreateUTestAnalyzerInput(temp1, temp2, kMetricKey1, kMetricKey2, 0,
                                     UTestConfig_DirectionBias_NO_BIAS, 0.001);
  config.set_name(kAnalyzerName);

  AnalyzerOutput output_passing;
  auto analyzer = Analyzer(config);

  EXPECT_TRUE(analyzer.Analyze(input, &output_passing));
  EXPECT_EQ(kAnalyzerName, analyzer.analyzer_name());
  EXPECT_EQ(kAnalyzerName, output_passing.analyzer_name());
  EXPECT_TRUE(SuccessfulStatus(output_passing));

  LOG(INFO) << "Checking if analyzer name passed by the user is present in the "
               "output (regression output):";
  config =
      HelperCreateUTestAnalyzerInput(temp1, temp2, kMetricKey1, kMetricKey2, 0,
                                     UTestConfig_DirectionBias_NO_BIAS, 0.9);
  config.set_name(kAnalyzerName);

  AnalyzerOutput output_regression;
  analyzer = Analyzer(config);

  EXPECT_TRUE(analyzer.Analyze(input, &output_regression));
  EXPECT_EQ(kAnalyzerName, analyzer.analyzer_name());
  EXPECT_EQ(kAnalyzerName, output_regression.analyzer_name());
  EXPECT_TRUE(SuccessfulStatus(output_regression));

  LOG(INFO) << "Checking if analyzer name passed by the user is present in the "
               "error output (error output):";
  input.clear_run_to_be_analyzed();

  AnalyzerOutput output_error;
  EXPECT_FALSE(analyzer.Analyze(input, &output_error));
  EXPECT_EQ(kAnalyzerName, analyzer.analyzer_name());
  EXPECT_EQ(kAnalyzerName, output_error.analyzer_name());
  EXPECT_FALSE(SuccessfulStatus(output_error));
}

TEST_F(AnalyzerTest, AnalyzerWithConfigNames) {
  UTestAnalyzerInput config;
  *config.mutable_a_sample() = HelperCreateUTestSample({kRunKey1}, true);
  *config.mutable_b_sample() = HelperCreateUTestSample({kRunKey2}, true);

  // Run two configs both with names
  UTestConfig config_1 = HelperCreateUTestAnalyzerConfigWithShiftValue(
      kMetricKey1, kMetricKey2, -2, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_1.set_config_name(kConfigName1);
  *config.add_config_list() = config_1;

  UTestConfig config_2 = HelperCreateUTestAnalyzerConfigWithShiftValue(
      kMetricKey1, kMetricKey2, -3, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_2.set_config_name(kConfigName2);
  *config.add_config_list() = config_2;

  AnalyzerInput input = HelperCreateAnalyzerInput();
  HelperAddSamplePoints(kMetricKey1, {1, 2, 2, 3},
                        input.mutable_run_to_be_analyzed());
  RunBundle* historical_run = (*input.mutable_historical_run_map())[kSampleBKey]
                                  .add_historical_run_list();
  HelperAddSamplePoints(kMetricKey2, {1, 2, 2, 3}, historical_run);
  *historical_run->mutable_run_info() = HelperCreateRunInfo(kRunKey2);

  AnalyzerOutput output;
  auto analyzer = Analyzer(config);

  // Both names should be found in the AnalyzerOutput output field
  EXPECT_TRUE(analyzer.Analyze(input, &output));
  EXPECT_TRUE(output.output().find(kConfigName1) != std::string::npos);
  EXPECT_TRUE(output.output().find(kConfigName2) != std::string::npos);
  EXPECT_TRUE(SuccessfulStatus(output));
}

TEST_F(AnalyzerTest, AnalyzerWithConfigNamesAndRelativeShiftValue) {
  UTestAnalyzerInput config;
  *config.mutable_a_sample() = HelperCreateUTestSample({kRunKey1}, true);
  *config.mutable_b_sample() = HelperCreateUTestSample({kRunKey2}, true);

  // Run two configs both with names
  UTestConfig config_1 = HelperCreateUTestAnalyzerConfigWithRelativeShiftValue(
      kMetricKey1, kMetricKey2, -1.0, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_1.set_config_name(kConfigName1);
  *config.add_config_list() = config_1;

  UTestConfig config_2 = HelperCreateUTestAnalyzerConfigWithRelativeShiftValue(
      kMetricKey1, kMetricKey2, -1.5, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_2.set_config_name(kConfigName2);
  *config.add_config_list() = config_2;

  AnalyzerInput input = HelperCreateAnalyzerInput();
  HelperAddSamplePoints(kMetricKey1, {1, 2, 2, 3},
                        input.mutable_run_to_be_analyzed());
  RunBundle* historical_run = (*input.mutable_historical_run_map())[kSampleBKey]
                                  .add_historical_run_list();
  HelperAddSamplePoints(kMetricKey2, {1, 2, 2, 3}, historical_run);
  *historical_run->mutable_run_info() = HelperCreateRunInfo(kRunKey2);

  AnalyzerOutput output;
  auto analyzer = Analyzer(config);

  // Both names should be found in the AnalyzerOutput output field
  EXPECT_TRUE(analyzer.Analyze(input, &output));
  EXPECT_TRUE(output.output().find(kConfigName1) != std::string::npos);
  EXPECT_TRUE(output.output().find(kConfigName2) != std::string::npos);
  EXPECT_TRUE(SuccessfulStatus(output));
}

TEST_F(AnalyzerTest, AnalyzerConfigOnlyInSummaryIfRegression) {
  UTestAnalyzerInput config;
  *config.mutable_a_sample() = HelperCreateUTestSample({kRunKey1}, true);
  *config.mutable_b_sample() = HelperCreateUTestSample({kRunKey2}, true);

  // Should not be in output (no regression)
  UTestConfig config_1 = HelperCreateUTestAnalyzerConfigWithShiftValue(
      kMetricKey1, kMetricKey2, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_1.set_config_name(kConfigName1);
  *config.add_config_list() = config_1;

  // Should be in output (regression)
  UTestConfig config_2 = HelperCreateUTestAnalyzerConfigWithShiftValue(
      kMetricKey1, kMetricKey2, -3, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_2.set_config_name(kConfigName2);
  *config.add_config_list() = config_2;

  AnalyzerInput input = HelperCreateAnalyzerInput();

  HelperAddSamplePoints(kMetricKey1, {1, 2, 2, 3},
                        input.mutable_run_to_be_analyzed());
  RunBundle* historical_run = (*input.mutable_historical_run_map())[kSampleBKey]
                                  .add_historical_run_list();
  HelperAddSamplePoints(kMetricKey2, {1, 2, 2, 3}, historical_run);
  *historical_run->mutable_run_info() = HelperCreateRunInfo(kRunKey2);

  AnalyzerOutput output;
  auto analyzer = Analyzer(config);

  EXPECT_TRUE(analyzer.Analyze(input, &output));
  // Make sure summary is correct
  UTestAnalyzerOutput a_out;
  google::protobuf::TextFormat::ParseFromString(output.output(), &a_out);
  EXPECT_EQ(a_out.summary(), absl::StrCat("1 failed: ", kConfigName2));
  // Make sure both are in result list
  EXPECT_TRUE(output.output().find(kConfigName1) != std::string::npos);
  EXPECT_TRUE(output.output().find(kConfigName2) != std::string::npos);
  EXPECT_TRUE(SuccessfulStatus(output));
}

TEST_F(AnalyzerTest,
       AnalyzerConfigOnlyInSummaryIfRegressionRelativeShiftValue) {
  UTestAnalyzerInput config;
  *config.mutable_a_sample() = HelperCreateUTestSample({kRunKey1}, true);
  *config.mutable_b_sample() = HelperCreateUTestSample({kRunKey2}, true);

  // Should not be in output (no regression)
  UTestConfig config_1 = HelperCreateUTestAnalyzerConfigWithRelativeShiftValue(
      kMetricKey1, kMetricKey2, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_1.set_config_name(kConfigName1);
  *config.add_config_list() = config_1;

  // Should be in output (regression)
  UTestConfig config_2 = HelperCreateUTestAnalyzerConfigWithRelativeShiftValue(
      kMetricKey1, kMetricKey2, -1.5, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_2.set_config_name(kConfigName2);
  *config.add_config_list() = config_2;

  AnalyzerInput input = HelperCreateAnalyzerInput();

  HelperAddSamplePoints(kMetricKey1, {1, 2, 2, 3},
                        input.mutable_run_to_be_analyzed());
  RunBundle* historical_run = (*input.mutable_historical_run_map())[kSampleBKey]
                                  .add_historical_run_list();
  HelperAddSamplePoints(kMetricKey2, {1, 2, 2, 3}, historical_run);
  *historical_run->mutable_run_info() = HelperCreateRunInfo(kRunKey2);

  AnalyzerOutput output;
  auto analyzer = Analyzer(config);

  EXPECT_TRUE(analyzer.Analyze(input, &output));
  // Make sure summary is correct
  UTestAnalyzerOutput a_out;
  google::protobuf::TextFormat::ParseFromString(output.output(), &a_out);
  EXPECT_EQ(a_out.summary(), absl::StrCat("1 failed: ", kConfigName2));
  // Make sure both are in result list
  EXPECT_TRUE(output.output().find(kConfigName1) != std::string::npos);
  EXPECT_TRUE(output.output().find(kConfigName2) != std::string::npos);
  EXPECT_TRUE(SuccessfulStatus(output));
}

TEST_F(AnalyzerTest, AnalyzerOkaySummaryIfNoOverallRegression) {
  UTestAnalyzerInput config;
  *config.mutable_a_sample() = HelperCreateUTestSample({kRunKey1}, true);
  *config.mutable_b_sample() = HelperCreateUTestSample({kRunKey2}, true);

  // Both configs should report no regression
  UTestConfig config_1 = HelperCreateUTestAnalyzerConfigWithShiftValue(
      kMetricKey1, kMetricKey2, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_1.set_config_name(kConfigName1);
  *config.add_config_list() = config_1;

  UTestConfig config_2 = HelperCreateUTestAnalyzerConfigWithShiftValue(
      kMetricKey1, kMetricKey2, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05);
  config_2.set_config_name(kConfigName2);
  *config.add_config_list() = config_2;

  AnalyzerInput input = HelperCreateAnalyzerInput();
  HelperAddSamplePoints(kMetricKey1, {1, 2, 2, 3},
                        input.mutable_run_to_be_analyzed());
  RunBundle* historical_run = (*input.mutable_historical_run_map())[kSampleBKey]
                                  .add_historical_run_list();
  HelperAddSamplePoints(kMetricKey2, {1, 2, 2, 3}, historical_run);
  *historical_run->mutable_run_info() = HelperCreateRunInfo(kRunKey2);

  AnalyzerOutput output;
  auto analyzer = Analyzer(config);

  EXPECT_TRUE(analyzer.Analyze(input, &output));

  // Output summary should be "okay"
  UTestAnalyzerOutput a_out;
  google::protobuf::TextFormat::ParseFromString(output.output(), &a_out);
  EXPECT_EQ(a_out.summary(), "okay");

  EXPECT_TRUE(SuccessfulStatus(output));
}

void AnalyzerTest::HelperCheckMultipleSampleCombinations(
    bool slight, bool very, const UTestConfig::DirectionBias& bias,
    double sig_level) {
  LOG(INFO) << "Comparing same two samples:";
  AnalyzerOutput output_same_dist;
  bool result = HelperCompareTwoSamples(kCenterDist, kCenterDist, 0, bias,
                                        sig_level, &output_same_dist);
  EXPECT_TRUE(result);
  EXPECT_FALSE(output_same_dist.regression());
  EXPECT_TRUE(SuccessfulStatus(output_same_dist));

  LOG(INFO)
      << "Comparing two slightly different samples (center vs right skewed):";
  AnalyzerOutput output_center_vs_slight_right;
  result = HelperCompareTwoSamples(kCenterDist, kRightSkewedDist, 0, bias,
                                   sig_level, &output_center_vs_slight_right);
  EXPECT_TRUE(result);
  EXPECT_EQ(output_center_vs_slight_right.regression(),
            slight && bias != UTestConfig_DirectionBias_IGNORE_INCREASE);
  EXPECT_TRUE(SuccessfulStatus(output_center_vs_slight_right));

  LOG(INFO)
      << "Comparing two slightly different samples (right skewed vs center):";
  AnalyzerOutput output_slight_right_vs_center;
  result = HelperCompareTwoSamples(kRightSkewedDist, kCenterDist, 0, bias,
                                   sig_level, &output_slight_right_vs_center);
  EXPECT_TRUE(result);
  EXPECT_EQ(output_slight_right_vs_center.regression(),
            slight && bias != UTestConfig_DirectionBias_IGNORE_DECREASE);
  EXPECT_TRUE(SuccessfulStatus(output_slight_right_vs_center));

  LOG(INFO) << "Comparing two very different samples (right vs left):";
  AnalyzerOutput output_slight_right_vs_slight_left;
  result =
      HelperCompareTwoSamples(kRightSkewedDist, kLeftSkewedDist, 0, bias,
                              sig_level, &output_slight_right_vs_slight_left);
  EXPECT_TRUE(result);
  EXPECT_EQ(output_slight_right_vs_slight_left.regression(),
            very && bias != UTestConfig_DirectionBias_IGNORE_DECREASE);
  EXPECT_TRUE(SuccessfulStatus(output_slight_right_vs_slight_left));
}

TEST_F(AnalyzerTest, AnalyzerNoBias10PSigLevel) {
  HelperCheckMultipleSampleCombinations(true, true,
                                        UTestConfig_DirectionBias_NO_BIAS, 0.1);
}

TEST_F(AnalyzerTest, AnalyzerNoBias05PSigLevel) {
  HelperCheckMultipleSampleCombinations(
      false, true, UTestConfig_DirectionBias_NO_BIAS, 0.05);
}
TEST_F(AnalyzerTest, AnalyzerIgnDec05PSigLevel) {
  HelperCheckMultipleSampleCombinations(
      true, true, UTestConfig_DirectionBias_IGNORE_DECREASE, 0.05);
}

TEST_F(AnalyzerTest, AnalyzerIgnDec025PSigLevel) {
  HelperCheckMultipleSampleCombinations(
      false, true, UTestConfig_DirectionBias_IGNORE_DECREASE, 0.025);
}

TEST_F(AnalyzerTest, AnalyzerIgnInc05PSigLevel) {
  HelperCheckMultipleSampleCombinations(
      true, true, UTestConfig_DirectionBias_IGNORE_INCREASE, 0.05);
}

TEST_F(AnalyzerTest, AnalyzerIgnInc025PSigLevel) {
  HelperCheckMultipleSampleCombinations(
      false, true, UTestConfig_DirectionBias_IGNORE_DECREASE, 0.025);
}

TEST_F(AnalyzerTest, AnalyzerRunWithShiftValue) {
  LOG(INFO) << "Comparing A = kCenterDistShiftedLeft and B = kCenterDist with "
               "shift value of kDistShift at 0.05 sig level:";
  AnalyzerOutput output_shift_no_bias_05;
  bool result = HelperCompareTwoSamples(
      kCenterDistShiftedLeft, kCenterDist, kDistShift,
      UTestConfig_DirectionBias_NO_BIAS, 0.05, &output_shift_no_bias_05);
  EXPECT_TRUE(result);
  EXPECT_FALSE(output_shift_no_bias_05.regression());
  EXPECT_TRUE(SuccessfulStatus(output_shift_no_bias_05));

  LOG(INFO) << "Comparing A = kCenterDistShiftedLeft and B = kCenterDist with "
               "shift value of kDistShift + 1 at 0.05 sig level, ignoring "
               "increases:";
  AnalyzerOutput output_shift_left_ign_inc;
  result = HelperCompareTwoSamples(kCenterDistShiftedLeft, kCenterDist,
                                   kDistShift + 1,
                                   UTestConfig_DirectionBias_IGNORE_INCREASE,
                                   0.05, &output_shift_left_ign_inc);
  EXPECT_TRUE(result);
  EXPECT_TRUE(output_shift_left_ign_inc.regression());
  EXPECT_TRUE(SuccessfulStatus(output_shift_left_ign_inc));

  LOG(INFO) << "Comparing A = kCenterDistShiftedLeft and B = kCenterDist with "
               "shift value of kDistFullShift + 1 at 0.05 sig level, ignoring "
               "decreases:";
  AnalyzerOutput output_shift_left_ign_dec;
  result = HelperCompareTwoSamples(kCenterDistShiftedLeft, kCenterDist,
                                   kDistShift + 1,
                                   UTestConfig_DirectionBias_IGNORE_DECREASE,
                                   0.05, &output_shift_left_ign_dec);
  EXPECT_TRUE(result);
  EXPECT_FALSE(output_shift_left_ign_dec.regression());
  EXPECT_TRUE(SuccessfulStatus(output_shift_left_ign_dec));
}

// Confirm z-statistic is computed correct
TEST_F(AnalyzerTest, AnalyzerConfirmUTestValues) {
  // Source data and exected z values from:
  // http://www.itl.nist.gov/div898/handbook/prc/section3/prc35.htm. The
  // values differ slightly because our algorithm does a continuity correction.
  // Expected p-values confirmed using scipy.stats.mannwhitneyu.
  const std::vector<double> kA1 = {0.55, 0.67, 0.43, 0.51, 0.48, 0.60,
                                   0.71, 0.53, 0.44, 0.65, 0.75};
  const std::vector<double> kA2 = {0.49, 0.68, 0.59, 0.72, 0.67, 0.75,
                                   0.65, 0.77, 0.62, 0.48, 0.59};
  // Randomly generated
  const std::vector<double> kA3 = {0.242, 0.287, 0.919, 0.853, 0.031, 0.706,
                                   0.452, 0.184, 0.870, 0.105, 0.299, 0.911,
                                   0.974, 0.850, 0.679, 0.062, 0.580, 0.120,
                                   0.120, 0.699, 0.624, 0.971, 0.984};
  const std::vector<double> kA4 = {0.768, 0.042, 0.168, 0.071, 0.027, 0.694,
                                   0.313, 0.252, 0.510, 0.221, 0.139, 0.284,
                                   0.440, 0.055, 0.855, 0.048, 0.601, 0.015,
                                   0.848, 0.397, 0.806, 0.247, 0.585};
  const std::vector<double> kA5 = {0.58,  -0.19, 0.56, -0.52, 0.67,
                                   -0.53, -0.49, 0.50, 0.92,  -0.26};
  const std::vector<double> kA6 = {-0.90, 0.88,  0.19,  -0.95, 0.47,
                                   0.58,  -0.03, 0.02,  -0.88, -0.61,
                                   -0.54, 0.68,  -0.77, -0.59, 0.15};
  // Example 1
  AnalyzerOutput output_a1_a2;
  HelperCompareTwoSamples(kA1, kA2, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output_a1_a2);
  UTestAnalyzerOutput utest_output_a1_a2 =
      ExtractUTestAnalyzerOutput(output_a1_a2);
  EXPECT_NEAR(1.31516, utest_output_a1_a2.config_result_list(0).z_statistic(),
              kEpsilon);
  EXPECT_NEAR(0.18846, utest_output_a1_a2.config_result_list(0).p_value(),
              kEpsilon);

  // Example 2
  AnalyzerOutput output_a3_a4;
  HelperCompareTwoSamples(kA3, kA4, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output_a3_a4);
  UTestAnalyzerOutput utest_output_a3_a4 =
      ExtractUTestAnalyzerOutput(output_a3_a4);
  EXPECT_NEAR(1.933348, utest_output_a3_a4.config_result_list(0).z_statistic(),
              kEpsilon);
  EXPECT_NEAR(0.053193, utest_output_a3_a4.config_result_list(0).p_value(),
              kEpsilon);

  // Example 3
  AnalyzerOutput output_a5_a6;
  HelperCompareTwoSamples(kA5, kA6, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output_a5_a6);
  UTestAnalyzerOutput utest_output_a5_a6 =
      ExtractUTestAnalyzerOutput(output_a5_a6);
  EXPECT_NEAR(1.3315, utest_output_a5_a6.config_result_list(0).z_statistic(),
              kEpsilon);
  EXPECT_NEAR(0.183013, utest_output_a5_a6.config_result_list(0).p_value(),
              kEpsilon);
}

TEST_F(AnalyzerTest, AnalyzerConfirmMedianValues) {
  // Randomly generated
  const std::vector<double> kA1 = {0.242, 0.287, 0.919, 0.853, 0.031, 0.706,
                                   0.452, 0.184, 0.870, 0.105, 0.299, 0.911,
                                   0.974, 0.850, 0.679, 0.062, 0.580, 0.120,
                                   0.120, 0.699, 0.624, 0.971, 0.984};
  const std::vector<double> kA2 = {0.768, 0.042, 0.168, 0.071, 0.027, 0.694,
                                   0.313, 0.252, 0.510, 0.221, 0.139, 0.284,
                                   0.440, 0.055, 0.855, 0.048, 0.601, 0.015,
                                   0.848, 0.397, 0.806, 0.247, 0.585};
  // Example 1
  AnalyzerOutput output1;
  HelperCompareTwoSamples(kA1, kA2, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output1);
  UTestAnalyzerOutput utest_output1 = ExtractUTestAnalyzerOutput(output1);
  EXPECT_NEAR(0.624, utest_output1.config_result_list(0).a_median(), kEpsilon);
  EXPECT_NEAR(0.284, utest_output1.config_result_list(0).b_median(), kEpsilon);

  // Example 2 (shift value)
  AnalyzerOutput output2;
  HelperCompareTwoSamples(kA1, kA2, 0.5, UTestConfig_DirectionBias_NO_BIAS,
                          0.05, &output2);
  UTestAnalyzerOutput utest_output2 = ExtractUTestAnalyzerOutput(output2);
  EXPECT_NEAR(0.624, utest_output2.config_result_list(0).a_median(), kEpsilon);
  EXPECT_NEAR(0.284, utest_output2.config_result_list(0).b_median(), kEpsilon);
}

TEST_F(AnalyzerTest, OneSided) {
  // This data comes from an actual user run which failed, even though B was
  // clearly smaller than A in all meaningful ways, and the user had set
  // IGNORE_DECREASING. See b/65302424.
  // Expected p-values confirmed using scipy.stats.mannwhitneyu.
  const std::vector<double> kA = {
      1.1034431234,  1.12211614847, 1.18589551002, 1.19859471917, 1.23877490312,
      1.31152456999, 1.4197044,     1.698906973,   1.76665723324, 1.92213051021,
      1.9902786985,  2.12063488364, 2.15956696868, 2.23855977505, 2.42619714886,
      2.51893886924, 2.58130697906, 2.6352327615,  2.90352138877, 2.94615848362,
      3.07369292527, 3.17751922458, 3.32529897988, 3.42142041773, 3.4795974344,
      3.55182073265, 3.71947975457, 3.87962091714, 3.91922153533, 4.12443432212,
      4.26995337009, 4.3807907626,  4.4104019627,  4.53639254719, 4.61659295112,
      4.80219464749, 4.96991574764, 5.11554452032, 5.15356784314, 5.28474779427,
      5.3822818324,  5.52246502787, 5.64011096209, 5.75500487536, 5.86093883216,
      6.0643415451,  6.21578039974, 6.38548867404, 6.66857817024, 6.9181356281,
      7.06029772013, 7.30595852435, 7.36335226148, 7.42161250859, 7.47860267013,
      7.63155966997, 7.82112625986, 8.00082240254, 8.17129514366, 8.21194901317,
      8.36191572249, 8.51036950201, 8.69446501881, 8.72536450624, 8.85854060948,
      9.0289484188,  9.31188673526, 9.46344145387, 9.50233242661, 9.57060558349,
      9.5962542519,  9.85621999204, 9.88450731337, 10.1860731542, 10.2900645435,
      10.4198040664, 10.5075150207, 10.7035273686, 10.7790218517, 10.8069107756,
      10.977338396,  11.0772356987, 11.4100953713, 11.7269480675, 11.8573470637,
      11.9254324809, 12.3440414816, 12.4078710377, 12.5613404661, 12.6054806188,
      12.6962161511, 12.8966978416, 13.0075703934, 13.0931808576, 13.1265275851,
      13.2654111236, 13.3466059342, 13.3880501017, 13.4884280935, 13.522758238,
      13.5952050164, 13.661462456,  13.6708573624, 13.7395897806, 13.7554136515,
      13.7786606103, 13.7858560383, 13.7953725383, 13.9247541428, 13.9996009693,
      14.0976889357, 14.1526360214, 14.2139641047, 14.1251540333, 14.1778874323,
      14.2121961266, 14.2280718535, 14.2360620648, 14.2826978192, 14.2882377133,
      14.3941307366, 14.4138443694, 14.460473828,  14.4857928157, 14.5022699982,
      14.5378043205, 14.5810222253, 14.6081864908, 14.6376632974, 14.6499325261,
      14.6716828719, 14.6850160286, 14.7526901439, 14.8107172027, 14.8247536942,
      14.8335792124, 14.8647520766, 14.8756508976, 14.9221191704, 14.9439390004,
      15.0281029642, 15.0860564113, 15.0194998085, 15.0300460309, 15.0797459409,
      15.0951109603, 15.1333904341, 15.1726790145, 15.1818518713, 15.1970233396,
      15.24860017,   15.2548580766, 15.3125065118, 15.3299764395, 15.3894923255,
      15.403090395,  15.415219143,  15.4766345471, 15.4820264131, 15.4928435087,
      15.5675683096, 15.5844712183, 15.6168427765, 15.6540893316, 15.6671876609,
      15.7169687003, 15.7515331432, 15.8031629324, 15.8222852945, 15.8301553801,
      15.8405016884, 15.8626597151, 16.0488057509, 15.9551296756, 16.0112998113,
      16.0229881033, 16.0425851718, 16.0962626636, 16.1260330305, 16.1668585762,
      16.1995536759, 16.2135182023, 16.2530468479, 16.2794092968, 16.2813149467,
      16.3165787458, 16.3360278606, 16.3606255576, 16.3871203884, 16.4111702815,
      16.4175295755, 16.4446501061, 16.5028212517, 16.5156627148, 16.5235482007,
      16.5672884509, 16.6222358868, 16.62686079,   16.653631486,  16.6944745556,
      16.7306577265, 16.7798237354, 16.86321567,   16.8180696815, 16.860664092,
      16.8622859493, 16.8639056832, 16.8740221635, 16.8866441697, 16.8895823732,
      16.8930908591, 16.9006015956, 16.909144856,  16.9165731743, 16.9240273908,
      16.9292977899, 16.9345212132, 16.9456611425, 16.9496054798, 16.9559111595,
      16.9593747556, 16.9920529574, 16.9928968474, 16.997160323,  16.9978038073,
      17.0064728558, 17.013693139,  17.0170809925, 17.0308941826, 17.055451639,
      17.1303829104, 17.1656138524, 17.1478537917, 17.067614466,  17.0693847388,
      17.0721944049, 17.0747030526, 17.0844788328, 17.0994713083, 17.1097938493,
      17.1124458984, 17.1218608692, 17.129214935,  17.1355258003, 17.138836816,
      17.1461543739, 17.1574616209, 17.1626503915, 17.1668172926, 17.1665525287,
      17.1741153672, 17.175533019,  17.1755449548, 17.1756235138, 17.1758653298,
      17.1761068925, 17.1760978848, 17.1761236936, 17.1761240363, 17.1760813445,
      17.2087262124, 17.2436949313, 17.2967325971, 17.1758706346, 17.1758760735,
      17.1758972779, 17.1758905128, 17.1761454344, 17.1761600897, 17.176118508,
      17.1763647199, 17.1764385998, 17.1763891429, 17.1763962209, 17.1764167845,
      17.1764170676, 17.1763980091, 17.1764258146, 17.1764309257, 17.1764714941,
      17.1764735654};
  const std::vector<double> kB = {
      0.786023907363, 0.794096373022, 0.82808060199, 0.896716453135,
      0.944957435131, 1.01580952853,  1.12092409283, 1.21253206581,
      1.2285740152,   1.28021496534,  1.31408706307, 1.40250448138,
      1.43496547639,  1.56761049479,  1.58928894252, 1.625419572,
      1.75357084721,  1.84154048562,  1.86832839996, 1.95380352437,
      2.1095148325,   2.16023526341,  2.18484400213, 2.28308594972,
      2.31572270393,  2.38613734394,  2.49860802293, 2.60578513145,
      2.61851787567,  2.68986533582,  2.74506740272, 2.76168067753,
      2.79189843684,  2.81678304821,  2.88453649729, 3.0745081678,
      3.1932380721,   3.2660240829,   3.40797962248, 3.57862218469,
      3.60157808661,  3.76291733235,  3.79394444078, 3.89906364679,
      3.98167157173,  4.03876070678,  4.06668059528, 4.11071394384,
      4.21492706984,  4.32333242148,  4.49160708487, 4.51581664383,
      4.62705945224,  4.72995480895,  4.75429876149, 4.82816771418,
      4.875228405,    5.12349063158,  5.18169011176, 5.22693697363,
      5.25315319002,  5.35304517299,  5.42232906818, 5.48935075849,
      5.52693009377,  5.54028791189,  5.658176817,   5.71639881283,
      5.74861769378,  5.79424612224,  5.84193527699, 5.86557989568,
      5.86558507383,  5.86557838321,  5.86558629572, 5.86558075249,
      5.91952131689,  6.14681512862,  6.16980638355, 6.36787780374,
      6.40031998605,  6.43031863868,  6.55164381862, 6.61233017594,
      6.83189426363,  6.95950374007,  7.21906011552, 7.38237980008,
      7.61651538312,  7.81037791073,  7.91044896841, 8.4333486259,
      8.5092927888,   8.63593922555,  8.78043901175, 8.82511343807,
      8.93963769823,  8.94801327586,  8.97581564635, 9.08982496709,
      9.20094082505,  9.31399486214,  9.42414142936, 9.47672402114,
      9.6163943857,   9.69092769921,  9.77227725089, 9.79181031883,
      9.93301320076,  10.0176354349,  10.1835782155, 10.2236116603,
      10.2724362388,  10.3579012007,  10.4314313456, 10.6783040911,
      10.8888325989,  10.9247535244,  10.8548102453, 11.0184312388,
      11.0492895395,  11.1338383034,  11.2542805597, 11.3949904069,
      11.4730074033,  11.5279820785,  11.5618183389, 11.6615115404,
      11.7520955056,  11.9066661969,  11.9417809919, 11.9831381515,
      12.0205190629,  12.0511856899,  12.0937747955, 12.1315421984,
      12.2043228149,  12.2043418363,  12.2320583984, 12.2404622063,
      12.3076328784,  12.3508247361,  12.3858991638, 12.4062425569,
      12.4748361558,  12.498863019,   12.6631068885, 12.612825796,
      12.6337861493,  12.6779695079,  12.7237800956, 12.7614808679,
      12.7702853382,  12.83192119,    12.8810638785, 12.8914590031,
      12.8935355023,  12.9411235526,  12.9655247703, 13.0643100813,
      13.1342854127,  13.1776664257,  13.2130600512, 13.2592222244,
      13.3055535331,  13.3305770308,  13.3493298143, 13.4064354599,
      13.4506167099,  13.464251332,   13.5071533099, 13.5499948859,
      13.5766167715,  13.5933089331,  13.6397580877, 13.8311575651,
      13.8890643716,  13.8630529791,  13.8284536153, 13.8285772502,
      13.8284012452,  13.8283628449,  13.8283202723, 13.8283569068,
      13.828378275,   13.8284471184,  13.8286604807, 13.8286468163,
      13.8286847249,  13.8286939934,  13.8287028298, 13.828704156,
      13.8286962435,  13.8286914825,  13.8286925256, 13.8286798969,
      13.8287227824,  13.8287425116,  13.8287317529, 13.8287338987,
      13.8287502602,  13.8287305161,  13.8287424967, 13.8287901208,
      13.8298124149,  13.8550359383,  13.9422123507, 13.9002141505,
      13.9134312496,  13.929920055,   13.9569742903, 13.9698992521,
      13.9829530567,  13.9910034984,  14.0059264153, 14.0057240129,
      14.0352818742,  14.0543343201,  14.0696993768, 14.1042909995,
      14.1221014932,  14.1372745112,  14.1676206663, 14.212420851,
      14.2234660983,  14.2388544381,  14.2516394109, 14.2568499669,
      14.303938143,   14.3114682958,  14.3600661457, 14.3803633898,
      14.3953281045,  14.4004568607,  14.4481085837, 14.4826946482,
      14.6054403558,  14.5044947714,  14.5133052319, 14.5413658544,
      14.555038102,   14.5647895858,  14.6123601049, 14.6226675063,
      14.6334404722,  14.6471436769,  14.7069348395, 14.7189656571,
      14.7371450588,  14.7788667083,  14.8028739393, 14.8154124022,
      14.8424343318,  14.8956428021,  14.9359976351, 14.9594392255,
      14.9987123534,  15.0046695247,  15.064931199,  15.0931895524,
      15.1186977625,  15.1419873834,  15.1613562405, 15.1701782346,
      15.2263251096,  15.2268866375,  15.2293826044, 15.2782927305,
      15.304524608,   15.3337647095,  15.337191768,  15.3829172328,
      15.3922943249,  15.4184742123,  15.4373408034, 15.501227811,
      15.5146511495,  15.5426931381,  15.5760130808, 15.5986872911,
      15.6128944159,  15.667108193,   15.6806910485, 15.7007008642,
      15.7437030971,  15.8078295216,  15.8170703128, 15.8664529473,
      15.8833620548,  15.9487687275,  15.9659837708, 15.9956458285,
      16.0103245229,  16.0245609209,  16.0637303069, 16.0723093674,
      16.2502563968,  16.2809230462,  16.3143286332, 16.2353431061,
      16.2557477131,  16.265727438,   16.2835215852, 16.3520816788,
      16.4172340706,  16.4378354326,  16.4615258202, 16.525843896,
      16.5800288096,  16.6200473383,  16.6380255222, 16.667975679,
      16.7024702057,  16.7183799669,  16.7315529659, 16.7459584698,
      16.8278289139,  16.8702834845,  16.8858979791, 16.9337751046,
      16.9599886462,  16.9789180756,  16.9886897355, 17.0439919904,
      17.0511987209,  17.0666949004,  17.140973717,  17.1723995805,
      17.0929745436,  17.0964335799,  17.0987493992, 17.1048822328,
      17.119304277,   17.1259551719,  17.1345225647, 17.1380867437,
      17.1459127665,  17.1577020288,  17.166432187,  17.1687975377,
      17.170102559,   17.170103319,   17.1701164916, 17.1700912043,
      17.1700694636,  17.1700736061,  17.170073621,  17.1701689214,
      17.1701198965,  17.1701733321,  17.1701620221, 17.1701580137,
      17.1701483876,  17.1701626256,  17.1701636836, 17.2388676107,
      17.290528208,   17.1652897447,  17.1660324931, 17.1663035974,
      17.1665048599,  17.1667667329,  17.1667986289, 17.1667929739,
      17.1670458913,  17.1670277342,  17.1670695171, 17.1670939848,
      17.1671179309,  17.1671360806};
  AnalyzerOutput output;
  HelperCompareTwoSamples(kA, kB, 0, UTestConfig_DirectionBias_IGNORE_DECREASE,
                          0.05, &output);
  UTestAnalyzerOutput utest_output1 = ExtractUTestAnalyzerOutput(output);
  EXPECT_FALSE(output.regression());
  EXPECT_NEAR(0.99998, utest_output1.config_result_list(0).p_value(), kEpsilon);

  HelperCompareTwoSamples(kB, kA, 0, UTestConfig_DirectionBias_IGNORE_INCREASE,
                          0.05, &output);
  UTestAnalyzerOutput utest_output2 = ExtractUTestAnalyzerOutput(output);
  EXPECT_FALSE(output.regression());
  EXPECT_EQ(utest_output1.config_result_list(0).p_value(),
            utest_output2.config_result_list(0).p_value());

  HelperCompareTwoSamples(kB, kA, 0, UTestConfig_DirectionBias_IGNORE_DECREASE,
                          0.05, &output);
  UTestAnalyzerOutput utest_output3 = ExtractUTestAnalyzerOutput(output);
  EXPECT_TRUE(output.regression());
  EXPECT_NEAR(1.918000e-05, utest_output3.config_result_list(0).p_value(),
              kEpsilon);

  HelperCompareTwoSamples(kA, kB, 0, UTestConfig_DirectionBias_IGNORE_INCREASE,
                          0.05, &output);
  UTestAnalyzerOutput utest_output4 = ExtractUTestAnalyzerOutput(output);
  EXPECT_TRUE(output.regression());
  EXPECT_EQ(utest_output3.config_result_list(0).p_value(),
            utest_output4.config_result_list(0).p_value());
}

TEST_F(AnalyzerTest, Ties) {
  // Expected p-values confirmed using scipy.stats.mannwhitneyu.
  const std::vector<double> x = {
      0.0, 0.0, 0.0, 0.0, 0.0, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 2.0};
  const std::vector<double> y = {
      0.0, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0,
      1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 3.0};
  AnalyzerOutput output;
  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_IGNORE_DECREASE,
                          0.05, &output);
  UTestAnalyzerOutput utest_output = ExtractUTestAnalyzerOutput(output);
  EXPECT_NEAR(0.1431016, utest_output.config_result_list(0).p_value(),
              kEpsilon);

  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_IGNORE_INCREASE,
                          0.05, &output);
  utest_output = ExtractUTestAnalyzerOutput(output);
  EXPECT_NEAR(0.8577485, utest_output.config_result_list(0).p_value(),
              kEpsilon);

  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  utest_output = ExtractUTestAnalyzerOutput(output);
  EXPECT_NEAR(0.2862021, utest_output.config_result_list(0).p_value(),
              kEpsilon);
}

TEST_F(AnalyzerTest, AllTiesBeforeBiasing) {
  const std::vector<double> x(20, 10.0);
  const std::vector<double> y(20, 11.0);

  AnalyzerOutput output;
  HelperCompareTwoSamples(x, x, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  EXPECT_FALSE(output.regression());

  HelperCompareTwoSamples(x, x, 1.0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  EXPECT_TRUE(output.regression());

  HelperCompareTwoSamples(x, x, -1.0, UTestConfig_DirectionBias_IGNORE_INCREASE,
                          0.05, &output);
  EXPECT_FALSE(output.regression());

  HelperCompareTwoSamples(x, x, 1.0, UTestConfig_DirectionBias_IGNORE_INCREASE,
                          0.05, &output);
  EXPECT_TRUE(output.regression());

  HelperCompareTwoSamples(x, x, 1.0, UTestConfig_DirectionBias_IGNORE_DECREASE,
                          0.05, &output);
  EXPECT_FALSE(output.regression());

  HelperCompareTwoSamples(x, x, -1.0, UTestConfig_DirectionBias_IGNORE_DECREASE,
                          0.05, &output);
  EXPECT_TRUE(output.regression());
}

TEST_F(AnalyzerTest, AllTiesAfterBiasing) {
  const std::vector<double> x(20, 10.0);
  const std::vector<double> y(20, 11.0);

  AnalyzerOutput output;

  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  EXPECT_TRUE(output.regression());

  HelperCompareTwoSamples(x, y, 1.0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  EXPECT_FALSE(output.regression());

  HelperCompareTwoSamples(x, y, 1.0, UTestConfig_DirectionBias_IGNORE_DECREASE,
                          0.05, &output);
  EXPECT_FALSE(output.regression());

  HelperCompareTwoSamples(x, y, 1.0, UTestConfig_DirectionBias_IGNORE_INCREASE,
                          0.05, &output);
  EXPECT_FALSE(output.regression());
}

TEST_F(AnalyzerTest, AllTiesDifferentSizes) {
  const std::vector<double> x(20, 10.0);
  const std::vector<double> y(10, 10.0);

  AnalyzerOutput output;
  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  EXPECT_FALSE(output.regression());
}

TEST_F(AnalyzerTest, HumanReadableMetricLabels) {
  const std::vector<double> x(20, 10.0);
  const std::vector<double> y(10, 10.0);

  AnalyzerOutput output;
  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  EXPECT_FALSE(output.regression());

  mako::utest_analyzer::UTestAnalyzerOutput utest_output;
  ASSERT_TRUE(
      google::protobuf::TextFormat::ParseFromString(output.output(), &utest_output));
  EXPECT_EQ(utest_output.config_result_list(0).a_metric_label(), kMetricKey1);
  EXPECT_EQ(utest_output.config_result_list(0).b_metric_label(), kMetricKey2);
}

TEST_F(AnalyzerTest, LargeNumberSamplesNoBiasSame) {
  std::vector<double> x(60000);
  std::iota(std::begin(x), std::end(x), 0);
  std::vector<double> y(60000);
  std::iota(std::begin(y), std::end(y), 0);

  AnalyzerOutput output;
  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  EXPECT_FALSE(output.regression());
  UTestAnalyzerOutput utest_output = ExtractUTestAnalyzerOutput(output);
  EXPECT_NEAR(1, utest_output.config_result_list(0).p_value(), kEpsilon);
}

TEST_F(AnalyzerTest, LargeNumberSamplesNoBiasNoRegression) {
  std::vector<double> x(60000);
  std::iota(std::begin(x), std::end(x), 0);
  std::vector<double> y(60000);
  std::iota(std::begin(y), std::end(y), 100);

  AnalyzerOutput output;
  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_NO_BIAS, 0.05,
                          &output);
  EXPECT_FALSE(output.regression());
  UTestAnalyzerOutput utest_output = ExtractUTestAnalyzerOutput(output);
  EXPECT_NEAR(.317, utest_output.config_result_list(0).p_value(), kEpsilon);
}

TEST_F(AnalyzerTest, LargeNumberSamplesBiasRegression) {
  std::vector<double> x(60000);
  std::iota(std::begin(x), std::end(x), 0);
  std::vector<double> y(60000);
  std::iota(std::begin(y), std::end(y), 100);

  AnalyzerOutput output;
  HelperCompareTwoSamples(x, y, 0, UTestConfig_DirectionBias_IGNORE_DECREASE,
                          0.2, &output);
  EXPECT_TRUE(output.regression());
  UTestAnalyzerOutput utest_output = ExtractUTestAnalyzerOutput(output);
  EXPECT_NEAR(.159, utest_output.config_result_list(0).p_value(), kEpsilon);
}

}  // namespace utest_analyzer
}  // namespace mako
