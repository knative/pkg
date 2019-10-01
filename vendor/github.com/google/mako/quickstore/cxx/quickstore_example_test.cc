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

#include <string>

#include "glog/logging.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/cxx/storage/base_storage_client.h"
#include "clients/cxx/storage/google3_storage.h"
#include "clients/cxx/storage/mako_client.h"
#include "clients/proto/analyzers/threshold_analyzer.pb.h"
#include "clients/proto/analyzers/utest_analyzer.pb.h"
#include "clients/proto/analyzers/window_deviation.pb.h"
#include "absl/flags/flag.h"
#include "absl/time/clock.h"
#include "absl/time/time.h"
#include "proto/quickstore/quickstore.pb.h"
#include "quickstore/cxx/quickstore.h"
#include "spec/proto/mako.pb.h"

ABSL_FLAG(bool, mako_storage, false,
          "Use the Mako client instead of the internal client.");

namespace {

using ::mako::BaseStorageClient;
using ::mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput;
using ::mako::analyzers::threshold_analyzer::ThresholdConfig;
using ::mako::quickstore::QuickstoreInput;
using ::mako::quickstore::QuickstoreOutput;
using ::mako::quickstore::IsOK;
using ::mako::quickstore::Quickstore;
using ::mako::utest_analyzer::UTestAnalyzerInput;
using ::mako::window_deviation::WindowDeviationInput;

constexpr char kMakoBenchmarkKey[] = "4911226874232832";
constexpr char kMetric[] = "y";
constexpr char kCustomAggregate[] = "c1";

// Convienient Historic Run Key
constexpr char kMakoHistoricRunKey[] = "4625301237661696";

double NowInMs() {
  return static_cast<double>(::absl::ToUnixMillis(::absl::Now()));
}

std::string BenchmarkKey() {
  return kMakoBenchmarkKey;
}

std::string HistoricRunKey() {
  return kMakoHistoricRunKey;
}

Quickstore GetQuickstore(const std::string& benchmark_key) {
  Quickstore q(BenchmarkKey());
  return q;
}

Quickstore GetQuickstore(const QuickstoreInput& input) {
  Quickstore q(input);
  return q;
}

// ====== Simplest use of quickstore ======
// Supply your data, everything else is done for you.
TEST(QuickstoreTest, SimpleTest) {
  Quickstore q = GetQuickstore(BenchmarkKey());

  // When the point was collected
  double t = NowInMs();
  for (int data : {1, 2, 3, 4, 5, 6}) {
    q.AddSamplePoint(t++, {{kMetric, data}});
  }
  QuickstoreOutput output = q.Store();
  ASSERT_TRUE(IsOK(output));
  LOG(INFO) << "View chart at: " << output.run_chart_link();
}

// ====== Customize run information ======
// If you'd like more control over what gets recorded for your run (eg. you'd
// like a description to be set, for instance), set those fields on the
// provided QuickstoreInput.
TEST(QuickstoreTest, CustomQuickstoreInput) {
  QuickstoreInput input;
  input.set_benchmark_key(BenchmarkKey());
  input.set_description("This will show up as part of the Mako run chart.");

  Quickstore q = GetQuickstore(input);

  // Add sample data
  double t = NowInMs();
  for (int data : {1, 2, 3, 4, 5, 6}) {
    q.AddSamplePoint(t++, {{kMetric, data}});
  }

  // Also report that an error occurred here
  q.AddError(t, "Something happened here");

  QuickstoreOutput output = q.Store();
  ASSERT_TRUE(IsOK(output));

  LOG(INFO) << "View chart at: " << output.run_chart_link()
            << " NOTE the 'Run Information.Description' has been modified, and "
               "a SampleError was reported.";
}

// ====== Custom metric aggregates ======
// Some benchmarking tools only report aggregates for certain metrics.
TEST(QuickstoreTest, CustomMetricAggregates) {
  Quickstore q = GetQuickstore(BenchmarkKey());

  // We got the mean, 98th and 99th percentile for metric 'y' from our tool.
  double p98 = 100;
  double p99 = 101;
  double mean = 90;
  q.AddMetricAggregate(kMetric, "p98000", p98);
  q.AddMetricAggregate(kMetric, "p99000", p99);
  q.AddMetricAggregate(kMetric, "mean", mean);
  QuickstoreOutput output = q.Store();
  ASSERT_TRUE(IsOK(output));

  LOG(INFO) << "View chart at: " << output.run_chart_link()
            << " NOTE the other percentiles for metric 'y' are set to 0.";
}

// ====== Custom run aggregates ======
// Some benchmarking tools only report a single aggregate for each run.
// You might also want to manually set meta-data information about the data such
// as the amount of sample points that were collected (NOTE: Would be
// automatically calculated for you otherwise).
TEST(QuickstoreTest, CustomRunAggregates) {
  Quickstore q = GetQuickstore(BenchmarkKey());

  // Single value for the entire run.
  q.AddRunAggregate(kCustomAggregate, 500);
  // Let Mako know that this aggregate was based from 100 benchmark
  // iterations.
  q.AddRunAggregate("~usable_sample_count", 100);
  QuickstoreOutput output = q.Store();
  ASSERT_TRUE(IsOK(output));

  LOG(INFO) << "View chart at: " << output.run_chart_link();
}

// ====== Run analyzers on data ======
// IT IS A VERY GOOD IDEA TO SETUP ANALYZERS TO VERIFY YOUR DATA!!
TEST(QuickstoreTest, RunAnalyzers) {
  QuickstoreInput input;
  input.set_benchmark_key(BenchmarkKey());

  // Add a Threshold analyzer to check our data.
  ThresholdAnalyzerInput* threshold_input = input.add_threshold_inputs();
  threshold_input->set_name("BadThresholdAnalyzer");
  ThresholdConfig* config = threshold_input->add_configs();
  // Data should be between 100 and 200 so this should fail
  config->set_max(200);
  config->set_min(101);
  config->mutable_data_filter()->set_data_type(
      mako::DataFilter::METRIC_SAMPLEPOINTS);
  config->mutable_data_filter()->set_value_key(kMetric);

  // Add a Window Deviation analyzer as well.
  // See window_deviation.proto for full documentation.
  // Verify that the median for metric 'y' hasn't increased by more than 10%
  // in the last 5 runs, compared to the 10 runs before those.
  WindowDeviationInput* wda_input = input.add_wda_inputs();
  auto* wda_query = wda_input->add_run_info_query_list();
  wda_query->set_benchmark_key(BenchmarkKey());
  *wda_query->add_tags() = "analyzer_example";
  wda_query->set_limit(14);
  auto* wda_check = wda_input->add_tolerance_check_list();
  wda_check->mutable_data_filter()->set_data_type(
      mako::DataFilter::METRIC_AGGREGATE_MEDIAN);
  wda_check->mutable_data_filter()->set_value_key(kMetric);
  wda_check->set_recent_window_size(5);
  wda_check->set_direction_bias(
      mako::window_deviation::ToleranceCheck::IGNORE_DECREASE);
  wda_check->add_median_tolerance_params_list()->set_median_coeff(0.05);

  // Add a UTestAnalyzer.
  // See utest_analyzer.proto for full documentation.
  // Verify that the central tendency of metric _METRIC1 between the
  // current run and a historic run are not significantly different.
  UTestAnalyzerInput* utest_input = input.add_utest_inputs();
  utest_input->mutable_a_sample()->set_include_current_run(true);
  utest_input->mutable_b_sample()->add_run_query_list()->set_run_key(
      HistoricRunKey());
  auto* utest_config = utest_input->add_config_list();
  utest_config->set_a_metric_key(kMetric);
  utest_config->set_b_metric_key(kMetric);
  utest_config->set_significance_level(0.05);

  Quickstore q = GetQuickstore(input);
  double t = NowInMs();
  // Our data is between 100 and 200
  for (int i = 100; i < 200; i++) {
    q.AddSamplePoint(t++, {{kMetric, i}});
  }

  QuickstoreOutput output = q.Store();

  LOG(INFO) << "View chart at: " << output.run_chart_link()
            << " NOTE: Even if analyzers fails we still get data";

  // Three analyzers should have returned status
  ASSERT_EQ(3, output.analyzer_output_list_size());

  int num_regressions = 0;
  for (const auto& result : output.analyzer_output_list()) {
    ASSERT_EQ(result.status().code(), mako::Status::SUCCESS);
    // We only expect the ThresholdAnalyzer to fail
    if (result.regression()) {
      num_regressions++;
      ASSERT_FALSE(IsOK(output));
      ASSERT_EQ("Threshold", result.analyzer_type());
      ASSERT_EQ("BadThresholdAnalyzer", result.analyzer_name());
    }
  }
  ASSERT_EQ(1, num_regressions);
  EXPECT_EQ(mako::quickstore::QuickstoreOutput::ANALYSIS_FAIL,
            output.status());
}

}  // namespace
