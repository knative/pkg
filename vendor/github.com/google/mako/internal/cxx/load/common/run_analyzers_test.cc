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
#include "internal/cxx/load/common/run_analyzers.h"

#include <functional>

#include "glog/logging.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/cxx/analyzers/threshold_analyzer.h"
#include "clients/cxx/dashboard/standard_dashboard.h"
#include "clients/cxx/storage/fake_google3_storage.h"
#include "clients/proto/analyzers/threshold_analyzer.pb.h"
#include "spec/cxx/analyzer.h"
#include "spec/cxx/dashboard.h"
#include "spec/cxx/mock_analyzer.h"
#include "spec/cxx/storage.h"
#include "absl/memory/memory.h"
#include "absl/strings/str_format.h"
#include "spec/proto/mako.pb.h"
#include "testing/cxx/protocol-buffer-matchers.h"

namespace mako {
namespace internal {

// Shutdown only called once
// SamplerBoss returns some samplers who have written data to fileio.
//  - RunInfo's batch keys have actually been written.
// Analyzer tests.

constexpr char kNoError[] = "";
constexpr char kBenchmarkMetricKey[] = "x";
constexpr char kBenchmarkMetricLabel[] = "ThroughputMs";
constexpr char kBenchmarkCustomAggregateKey[] = "c";
constexpr char kBenchmarkCustomAggregateLabel[] = "CustomStuff";
constexpr char kSampler1[] = "s1";
constexpr char kSampler2[] = "s2";
constexpr char kTempDir[] = "/tmp";

using ::mako::TestOutput;

// GMock stuff
using ::mako::EqualsProto;
using ::testing::_;
using ::testing::DoAll;
using ::testing::HasSubstr;
using ::testing::Invoke;
using ::testing::IsEmpty;
using ::testing::Return;
using ::testing::SetArgPointee;
using ::testing::Truly;
using ::testing::UnorderedElementsAre;
using ::testing::UnorderedPointwise;

// NiceMock doesn't print warning for calls you didn't define.
using ::testing::NiceMock;

mako::BenchmarkInfo CreateBenchmarkInfo() {
  mako::BenchmarkInfo benchmark_info;
  benchmark_info.set_benchmark_name("bname");
  benchmark_info.set_project_name("pname");
  *benchmark_info.add_owner_list() = "*";
  benchmark_info.mutable_input_value_info()->set_value_key("t");
  benchmark_info.mutable_input_value_info()->set_label("time");
  mako::ValueInfo* metric = benchmark_info.add_metric_info_list();
  metric->set_label(kBenchmarkMetricLabel);
  metric->set_value_key(kBenchmarkMetricKey);
  mako::ValueInfo* custom_metric =
      benchmark_info.add_custom_aggregation_info_list();
  custom_metric->set_label(kBenchmarkCustomAggregateLabel);
  custom_metric->set_value_key(kBenchmarkCustomAggregateKey);
  return benchmark_info;
}

void CreateSampleBatches(mako::RunInfo* run_info, uint32_t num_batches,
                         std::vector<mako::SampleBatch>* batches) {
  mako::fake_google3_storage::Storage s;
  std::vector<std::string> batch_keys;
  for (uint32_t i = 0; i < num_batches; i++) {
    mako::CreationResponse create_resp;
    mako::SampleBatch sample_batch;
    sample_batch.set_benchmark_key(run_info->benchmark_key());
    sample_batch.set_run_key(run_info->run_key());
    mako::SamplePoint* sample_point = sample_batch.add_sample_point_list();
    sample_point->set_input_value(i);
    mako::KeyedValue* metric = sample_point->add_metric_value_list();
    metric->set_value(i);
    metric->set_value_key(kBenchmarkMetricKey);
    batches->push_back(sample_batch);

    ASSERT_TRUE(s.CreateSampleBatch(sample_batch, &create_resp));
    batch_keys.push_back(create_resp.key());
  }

  for (const auto& batch_key : batch_keys) {
    *run_info->add_batch_key_list() = batch_key;
  }
  mako::ModificationResponse mod_resp;
  ASSERT_TRUE(s.UpdateRunInfo(*run_info, &mod_resp));
  ASSERT_EQ(1, mod_resp.count());
}

void CreateBenchmark(mako::BenchmarkInfo* benchmark_info) {
  mako::fake_google3_storage::Storage s;

  benchmark_info->set_benchmark_name("bname");
  benchmark_info->set_project_name("bproject");
  benchmark_info->mutable_input_value_info()->set_label("time");
  benchmark_info->mutable_input_value_info()->set_value_key("t");
  *benchmark_info->add_owner_list() = "*";

  mako::CreationResponse create_resp;
  mako::ModificationResponse mod_resp;
  ASSERT_TRUE(s.CreateBenchmarkInfo(*benchmark_info, &create_resp));
  benchmark_info->set_benchmark_key(create_resp.key());
}

void CreateRun(const mako::BenchmarkInfo& benchmark_info,
               mako::RunInfo* run_info) {
  mako::fake_google3_storage::Storage s;
  mako::CreationResponse create_resp;
  run_info->set_benchmark_key(benchmark_info.benchmark_key());
  run_info->set_timestamp_ms(1000);
  ASSERT_TRUE(s.CreateRunInfo(*run_info, &create_resp));
  run_info->set_run_key(create_resp.key());
}

class RunAnalyzersTest : public ::testing::Test {
 protected:
  RunAnalyzersTest() {}

  ~RunAnalyzersTest() override {}

  void SetUp() override {
    s_.FakeClear();

    // Create BenchmarkInfo, RunInfo and SampleBatches
    CreateBenchmark(&benchmark_info_);
    CreateRun(benchmark_info_, &run_info_);
  }

  mako::fake_google3_storage::Storage s_;
  mako::standard_dashboard::Dashboard d_;
  mako::BenchmarkInfo benchmark_info_;
  mako::RunInfo run_info_;
};

TEST_F(RunAnalyzersTest, Success) {
  mako::TestOutput test_output;
  std::vector<SampleBatch> batches;
  NiceMock<MockAnalyzer> mock_analyzer;
  EXPECT_CALL(mock_analyzer, ConstructHistoricQuery(_, _))
      .WillRepeatedly(Return(true));
  EXPECT_CALL(mock_analyzer, Analyze(_, _)).WillRepeatedly(Return(true));

  std::string err = RunAnalyzers(
      benchmark_info_, run_info_, batches,
      /*attach_e_divisive_regressions_to_changepoints=*/false, &s_, &d_,
      {&mock_analyzer}, &test_output);

  // Validate
  ASSERT_EQ("", err);
  ASSERT_EQ(mako::TestOutput::PASS, test_output.test_status());
  ASSERT_NE("", test_output.summary_output());
  ASSERT_EQ(1, test_output.analyzer_output_list_size());
  ASSERT_FALSE(test_output.analyzer_output_list(0).regression());
  ASSERT_EQ("", test_output.analyzer_output_list(0).output());
}

TEST_F(RunAnalyzersTest, ConstructHistoryErrorStatus) {
  std::string mock_error = "something went wrong";
  mako::TestOutput test_output;
  std::vector<SampleBatch> batches;
  NiceMock<MockAnalyzer> mock_analyzer;
  mako::AnalyzerHistoricQueryOutput mock_output;
  mock_output.mutable_status()->set_code(mako::Status::FAIL);
  mock_output.mutable_status()->set_fail_message(mock_error);
  EXPECT_CALL(mock_analyzer, ConstructHistoricQuery(_, _))
      .WillRepeatedly(DoAll(SetArgPointee<1>(mock_output), Return(false)));
  EXPECT_CALL(mock_analyzer, Analyze(_, _)).WillRepeatedly(Return(true));

  std::string err = RunAnalyzers(
      benchmark_info_, run_info_, batches,
      /*attach_e_divisive_regressions_to_changepoints=*/false, &s_, &d_,
      {&mock_analyzer}, &test_output);

  // Validate
  ASSERT_EQ("", err);
  ASSERT_EQ(mako::TestOutput::ANALYSIS_FAIL, test_output.test_status());
  ASSERT_NE("", test_output.summary_output());
  ASSERT_EQ(1, test_output.analyzer_output_list_size());
  ASSERT_EQ(mako::Status::FAIL,
            test_output.analyzer_output_list(0).status().code());
  EXPECT_THAT(test_output.analyzer_output_list(0).status().fail_message(),
              HasSubstr(mock_error));
}

TEST_F(RunAnalyzersTest, AnalyzeError) {
  std::string mock_error = "something went wrong";
  mako::TestOutput test_output;
  std::vector<SampleBatch> batches;
  NiceMock<MockAnalyzer> mock_analyzer;
  mako::AnalyzerOutput mock_output;
  mock_output.mutable_status()->set_code(mako::Status::FAIL);
  mock_output.mutable_status()->set_fail_message(mock_error);
  mock_output.set_analyzer_type("JustMyType");
  mock_output.set_analyzer_name("Me");
  EXPECT_CALL(mock_analyzer, ConstructHistoricQuery(_, _))
      .WillRepeatedly(Return(true));
  EXPECT_CALL(mock_analyzer, Analyze(_, _))
      .WillRepeatedly(DoAll(SetArgPointee<1>(mock_output), Return(false)));

  std::string err = RunAnalyzers(
      benchmark_info_, run_info_, batches,
      /*attach_e_divisive_regressions_to_changepoints=*/false, &s_, &d_,
      {&mock_analyzer}, &test_output);

  // Validate
  ASSERT_EQ("", err);
  ASSERT_EQ(mako::TestOutput::ANALYSIS_FAIL, test_output.test_status());
  ASSERT_NE("", test_output.summary_output());
  ASSERT_EQ(1, test_output.analyzer_output_list_size());
  mako::AnalyzerOutput analyzer_output =
      test_output.analyzer_output_list(0);
  ASSERT_EQ(mako::Status::FAIL, analyzer_output.status().code());
  EXPECT_THAT(analyzer_output.status().fail_message(), HasSubstr(mock_error));
  EXPECT_EQ("JustMyType", analyzer_output.analyzer_type());
  EXPECT_EQ("Me", analyzer_output.analyzer_name());
}

TEST_F(RunAnalyzersTest, RunAnalyzersSimpleThresholdPass) {
  std::vector<mako::Analyzer*> analyzers;
  mako::TestOutput test_output;
  std::vector<SampleBatch> batches;
  CreateSampleBatches(&run_info_, 100, &batches);
  // Configure ThresholdAnalyzer to ensure all points are between 0 and 100
  mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput input;
  mako::analyzers::threshold_analyzer::ThresholdConfig* config =
      input.add_configs();
  config->set_min(0);
  config->set_max(100);
  config->set_outlier_percent_max(0);
  config->mutable_data_filter()->set_value_key(kBenchmarkMetricKey);
  config->mutable_data_filter()->set_data_type(
      mako::DataFilter::METRIC_SAMPLEPOINTS);
  mako::threshold_analyzer::Analyzer threshold_analyzer(input);
  analyzers.push_back(&threshold_analyzer);

  std::string err = RunAnalyzers(
      benchmark_info_, run_info_, batches,
      /*attach_e_divisive_regressions_to_changepoints=*/false, &s_, &d_,
      analyzers, &test_output);

  // Validate
  ASSERT_EQ("", err);
  ASSERT_EQ(mako::TestOutput::PASS, test_output.test_status());
  ASSERT_NE("", test_output.summary_output());
  ASSERT_EQ(1, test_output.analyzer_output_list_size());
  ASSERT_FALSE(test_output.analyzer_output_list(0).regression());
}

TEST_F(RunAnalyzersTest,
       RunAnalyzersSimpleThresholdRegression) {
  std::vector<mako::Analyzer*> analyzers;
  mako::TestOutput test_output;
  std::vector<SampleBatch> batches;
  // Create SampleBatches
  CreateSampleBatches(&run_info_, 100, &batches);
  // Configure ThresholdAnalyzer to ensure all points are between 0 and 50
  mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput input;
  mako::analyzers::threshold_analyzer::ThresholdConfig* config =
      input.add_configs();
  config->set_min(0);
  config->set_max(50);
  config->set_outlier_percent_max(1.0);
  config->mutable_data_filter()->set_value_key(kBenchmarkMetricKey);
  config->mutable_data_filter()->set_data_type(
      mako::DataFilter::METRIC_SAMPLEPOINTS);
  mako::threshold_analyzer::Analyzer threshold_analyzer(input);
  analyzers.push_back(&threshold_analyzer);

  std::string err = RunAnalyzers(
      benchmark_info_, run_info_, batches,
      /*attach_e_divisive_regressions_to_changepoints=*/false, &s_, &d_,
      analyzers, &test_output);

  // Validate
  ASSERT_EQ("", err);
  ASSERT_EQ(mako::TestOutput::ANALYSIS_FAIL, test_output.test_status());
  ASSERT_NE("", test_output.summary_output());
  ASSERT_EQ(1, test_output.analyzer_output_list_size());
  ASSERT_TRUE(test_output.analyzer_output_list(0).regression());
  ASSERT_NE("", test_output.analyzer_output_list(0).output());
}

TEST_F(RunAnalyzersTest,
       RunAnalyzersSimpleThresholdConfigError) {
  std::vector<mako::Analyzer*> analyzers;
  mako::TestOutput test_output;
  std::vector<SampleBatch> batches;
  // Create SampleBatches
  CreateSampleBatches(&run_info_, 100, &batches);
  // ThresholdAnalyzerInput missing config's configuration to force a failure.
  mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput input;
  input.add_configs();
  mako::threshold_analyzer::Analyzer threshold_analyzer(input);
  analyzers.push_back(&threshold_analyzer);

  std::string err = RunAnalyzers(
      benchmark_info_, run_info_, batches,
      /*attach_e_divisive_regressions_to_changepoints=*/false, &s_, &d_,
      analyzers, &test_output);

  // Validate
  ASSERT_EQ("", err);
  ASSERT_EQ(mako::TestOutput::ANALYSIS_FAIL, test_output.test_status());
  ASSERT_NE("", test_output.summary_output());
  ASSERT_EQ(1, test_output.analyzer_output_list_size());
}

// Regardless of what runs the Analyzer provided queries return, we de-duplicate
// the results that are provided to the Analyzer. Duplicate runs could unfairly
// bias the analysis.
TEST_F(RunAnalyzersTest, NoDuplicateRuns) {
  // Create benchmark, create run infos with overlapping tags
  mako::RunInfo run1;
  run1.add_tags("A");
  run1.add_tags("B");
  CreateRun(benchmark_info_, &run1);

  mako::RunInfo run2;
  run2.add_tags("B");
  CreateRun(benchmark_info_, &run2);

  mako::RunInfo run3;
  run3.add_tags("A");
  CreateRun(benchmark_info_, &run3);

  // Construct analyzer queries that query for "A" and "B" tags
  mako::AnalyzerHistoricQueryOutput mock_output;
  mock_output.mutable_status()->set_code(mako::Status::SUCCESS);
  mock_output.add_run_info_query_list()->add_tags("A");
  mock_output.add_run_info_query_list()->add_tags("B");

  NiceMock<MockAnalyzer> mock_analyzer;
  EXPECT_CALL(mock_analyzer, ConstructHistoricQuery(_, _))
      .WillRepeatedly(DoAll(SetArgPointee<1>(mock_output), Return(true)));

  // Expect duplicate runs to be filtered correctly
  auto expect_three_runs = [](const mako::AnalyzerInput& input) {
    return input.historical_run_list_size() == 3;
  };
  EXPECT_CALL(mock_analyzer, Analyze(Truly(expect_three_runs), _))
      .WillRepeatedly(Return(true));

  mako::TestOutput test_output;
  std::vector<SampleBatch> batches;
  std::string err = RunAnalyzers(
      benchmark_info_, run_info_, batches,
      /*attach_e_divisive_regressions_to_changepoints=*/false, &s_, &d_,
      {&mock_analyzer}, &test_output);
  ASSERT_EQ("", err);
}

TEST_F(RunAnalyzersTest, RunAnalyzersVisualizeAnalysis) {
  std::vector<mako::Analyzer*> analyzers;
  mako::TestOutput test_output;
  std::vector<SampleBatch> batches;
  // Create SampleBatches
  CreateSampleBatches(&run_info_, 100, &batches);
  // Configure ThresholdAnalyzer to ensure all points are between 0 and 50
  mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput input;
  mako::analyzers::threshold_analyzer::ThresholdConfig* config =
      input.add_configs();
  config->set_min(0);
  config->set_max(50);
  config->set_outlier_percent_max(1.0);
  config->mutable_data_filter()->set_value_key(kBenchmarkMetricKey);
  config->mutable_data_filter()->set_data_type(
      mako::DataFilter::METRIC_SAMPLEPOINTS);
  mako::threshold_analyzer::Analyzer threshold_analyzer(input);
  // add two analyzers which will regress.
  analyzers.push_back(&threshold_analyzer);
  input.set_name("threshold_analyzer_two_name");
  mako::threshold_analyzer::Analyzer threshold_analyzer_two(input);
  analyzers.push_back(&threshold_analyzer_two);

  std::string err = RunAnalyzers(
      benchmark_info_, run_info_, batches,
      /*attach_e_divisive_regressions_to_changepoints=*/false, &s_, &d_,
      analyzers, &test_output);

  // Validate
  ASSERT_EQ("", err);
  ASSERT_EQ(mako::TestOutput::ANALYSIS_FAIL, test_output.test_status());
  ASSERT_NE("", test_output.summary_output());
  EXPECT_THAT(test_output.summary_output(),
              HasSubstr("visualize regression 'unnamed_#1': https://"));
  EXPECT_THAT(
      test_output.summary_output(),
      HasSubstr(
          "visualize regression 'threshold_analyzer_two_name': https://"));
  ASSERT_EQ(2, test_output.analyzer_output_list_size());
  ASSERT_TRUE(test_output.analyzer_output_list(0).regression());
  ASSERT_NE("", test_output.analyzer_output_list(0).output());
  ASSERT_TRUE(test_output.analyzer_output_list(1).regression());
  ASSERT_NE("", test_output.analyzer_output_list(1).output());
}

}  // namespace internal
}  // namespace mako
