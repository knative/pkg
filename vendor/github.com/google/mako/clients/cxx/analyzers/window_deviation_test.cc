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
#include "clients/cxx/analyzers/window_deviation.h"

#include <string>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/text_format.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/proto/analyzers/window_deviation.pb.h"
#include "spec/proto/mako.pb.h"
#include "absl/strings/str_cat.h"
#include "testing/cxx/protocol-buffer-matchers.h"

using ::testing::HasSubstr;

namespace mako {
namespace window_deviation {

TEST(WindowDeviationTest, QueryTest) {
  struct Case {
    std::string name;
    std::string ctor_in;
    std::string func_in;
    std::string want_func_out;
  };
  std::vector<Case> cases = {
      {
          "basic",
          // WindowDeviationInput, note second query
          "run_info_query_list: < "
          "  benchmark_key: \"b1\" "
          "  max_timestamp_ms: 10 "
          "  limit: 5 "
          "> "
          "run_info_query_list: < "
          "  benchmark_key: \"b2\" "
          "  max_timestamp_ms: 10 "
          "  limit: 5 "
          "> "
          "tolerance_check_list: < "
          "  data_filter: < "
          "    data_type: METRIC_AGGREGATE_MEAN "
          "    value_key: \"y1\" "
          "  > "
          "  mean_tolerance_params_list: < "
          "    const_term: 1 "
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
          "override-timestamp-and-bench",
          // WindowDeviationInput, missing benchmark_key and max_timestamp_ms
          "run_info_query_list: < "
          "  limit: 5 "
          "> "
          "tolerance_check_list: < "
          "  data_filter: < "
          "    data_type: METRIC_AGGREGATE_MEAN "
          "    value_key: \"y1\" "
          "  > "
          "  mean_tolerance_params_list: < "
          "    const_term: 1 "
          "  > "
          "> ",
          // AnalyzerHistoricQueryInput
          "benchmark_info: < "
          "  benchmark_key: \"b1\" "
          "> "
          "run_info: < "
          "  benchmark_key: \"b1\" "
          "  run_key: \"r1\" "
          "  timestamp_ms: 5 "
          "> ",
          // AnalyzerHistoricQueryOutput
          "status: < "
          "  code: SUCCESS "
          "> "
          "get_batches: false "
          "run_info_query_list: < "
          "  benchmark_key: \"b1\" "
          "  max_timestamp_ms: 4 "
          "  limit: 5 "
          "> ",
      },
      {
          "override-build-id",
          // WindowDeviationInput, missing max_build_id
          "run_info_query_list: < "
          "  benchmark_key: \"b1\" "
          "  run_order: BUILD_ID "
          "  limit: 5 "
          "> "
          "tolerance_check_list: < "
          "  data_filter: < "
          "    data_type: METRIC_AGGREGATE_MEAN "
          "    value_key: \"y1\" "
          "  > "
          "  mean_tolerance_params_list: < "
          "    const_term: 1 "
          "  > "
          "> ",
          // AnalyzerHistoricQueryInput
          "benchmark_info: < "
          "  benchmark_key: \"b1\" "
          "> "
          "run_info: < "
          "  benchmark_key: \"b1\" "
          "  run_key: \"r1\" "
          "  build_id: 5 "
          "> ",
          // AnalyzerHistoricQueryOutput
          "status: < "
          "  code: SUCCESS "
          "> "
          "get_batches: false "
          "run_info_query_list: < "
          "  benchmark_key: \"b1\" "
          "  run_order: BUILD_ID "
          "  max_build_id: 4 "
          "  limit: 5 "
          "> ",
      },
  };
  for (const Case& c : cases) {
    SCOPED_TRACE(c.name);
    LOG(INFO) << "Case: " << c.name;
    WindowDeviationInput ctor_in;
    google::protobuf::TextFormat::ParseFromString(c.ctor_in, &ctor_in);
    AnalyzerHistoricQueryInput func_in;
    google::protobuf::TextFormat::ParseFromString(c.func_in, &func_in);
    AnalyzerHistoricQueryOutput want_func_out;
    google::protobuf::TextFormat::ParseFromString(c.want_func_out, &want_func_out);
    window_deviation::Analyzer wda(ctor_in);
    AnalyzerHistoricQueryOutput got_func_out;
    bool success = wda.ConstructHistoricQuery(func_in, &got_func_out);
    LOG(INFO) << "Got:\n" << got_func_out.DebugString();
    EXPECT_TRUE(success);
    EXPECT_EQ(got_func_out.DebugString(), want_func_out.DebugString());
  }
}

TEST(WindownDeviationTest, AnalyzeTest) {
  struct Case {
    std::string name;
    std::string ctor_in;
    std::string func_in;
    bool want_regression;
    std::string want_in_output;
    Status_Code want_status;
    std::vector<ToleranceCheck> checks_skipped_for_missing_data;
    std::vector<double> want_min_timestamp_ms;
    std::vector<double> want_max_timestamp_ms;
    std::vector<double> want_min_build_id;
    std::vector<double> want_max_build_id;
  };
  std::string tolerance_check_with_missing_data =
      "  data_filter: < "
      "    data_type: METRIC_AGGREGATE_MEAN "
      "    value_key: \"y1\" "
      "  > "
      "  mean_tolerance_params_list: <const_term: 1> ";
  ToleranceCheck tolerance_check_with_missing_data_proto;
  google::protobuf::TextFormat::ParseFromString(tolerance_check_with_missing_data,
                                      &tolerance_check_with_missing_data_proto);
  std::string tolerance_check_minimum_historical_window_size =
      "  data_filter: < "
      "    data_type: METRIC_AGGREGATE_MEAN "
      "    value_key: \"y1\" "
      "  > "
      "  minimum_historical_window_size: 4"
      "  mean_tolerance_params_list: <const_term: 1> ";
  ToleranceCheck tolerance_check_minimum_historical_window_size_proto;
  google::protobuf::TextFormat::ParseFromString(
      tolerance_check_minimum_historical_window_size,
      &tolerance_check_minimum_historical_window_size_proto);
  std::vector<Case> cases = {
      {"mean-basic",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: <const_term: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 50 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"mean-multiple-recent",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  recent_window_size: 2 "
       "  mean_tolerance_params_list: < "
       "    const_term: 1 "
       "  > "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"r4\" "
       "    timestamp_ms: 4 "  // note recent out of order
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"r5\" "
       "    timestamp_ms: 5 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 55 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"mean-multi-param-all-pass",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: <const_term: 1> "
       "> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: <mean_coeff: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 50 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1, 1},
       // want_max_timestamp_ms
       {3, 3},
       // want_min_build_id
       {0, 0},
       // want_max_build_id
       {0, 0}},
      {"mean-multi-param-one-pass",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: <const_term: 1> "
       "  mean_tolerance_params_list: <mean_coeff: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 52 "  // delta = 2
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"mean-multi-param-all-fail",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: <const_term: 1> "
       "  mean_tolerance_params_list: <mean_coeff: 0.0001> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 52 "  // delta = 2
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       true,
       // want_in_output
       "Found 1 regressed checks",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"run-order-build-id-regression",
       // WindowDeviationInput
       "run_info_query_list: < "
       "  run_order: BUILD_ID "
       "> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  recent_window_size: 2 "
       "  mean_tolerance_params_list: <const_term: 1> "
       "  mean_tolerance_params_list: <mean_coeff: 0.0001> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    build_id: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 10 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h2\" "
       "    timestamp_ms: 2 "
       "    build_id: 2 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 10 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h3\" "
       "    timestamp_ms: 3 "
       "    build_id: 4 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 50 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h4\" "
       "    timestamp_ms: 4 "
       "    build_id: 3 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 10 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"r5\" "
       "    timestamp_ms: 5 "
       "    build_id: 5 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 50 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       true,
       // want_in_output
       "Found 1 regressed checks",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {0},
       // want_max_timestamp_ms
       {0},
       // want_min_build_id
       {1},
       // want_max_build_id
       {3}},
      {"mean-multi-filter-one-pass",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: <const_term: 1> "
       "> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: <mean_coeff: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 52 "  // delta = 2
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       true,
       // want_in_output
       "Found 1 regressed checks",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1, 1},
       // want_max_timestamp_ms
       {3, 3},
       // want_min_build_id
       {0, 0},
       // want_max_build_id
       {0, 0}},
      {"mean-all-terms-barely-pass",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: < "
       "    const_term: 10 "
       "    mean_coeff: 0.2 "          // term = 10
       "    stddev_coeff: 2.449490  "  // term ~ 10
       "  > "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 21 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"mean-all-terms-barely-fail",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: < "
       "    const_term: 10 "
       "    mean_coeff: 0.2 "         // term = 10
       "    stddev_coeff: 2.449490 "  // term ~ 10
       "  > "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 19 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       true,
       // want_in_output
       "Found 1 regressed checks",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"mean-ignore-increase",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  direction_bias: IGNORE_INCREASE "
       "  mean_tolerance_params_list: <const_term: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 60 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"mean-ignore-decrease",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  direction_bias: IGNORE_DECREASE "
       "  mean_tolerance_params_list: <const_term: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 40 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"historic-mean-NaN",
       // WindowDeviationInput
       "name: \"AnalyzerName\" "
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: CUSTOM_AGGREGATE "
       "    value_key: \"z1\" "
       "    ignore_missing_data: false "
       "  > "
       "  mean_tolerance_params_list: <mean_coeff: 0.1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        custom_aggregate_list: < "
       "          value_key: \"z1\" "
       "          value: NaN"
       "        > "
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
       "        custom_aggregate_list: < "
       "          value_key: \"z1\" "
       "          value: NaN"
       "        > "
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
       "        custom_aggregate_list: < "
       "          value_key: \"z1\" "
       "          value: 50.0"
       "        > "
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
       "        custom_aggregate_list: < "
       "          value_key: \"z1\" "
       "          value: NaN"
       "        > "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "Tolerance must be nonnegative",
       // want_status
       Status_Code_FAIL,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"historic-median-NaN",
       // WindowDeviationInput
       "name: \"AnalyzerName\" "
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: CUSTOM_AGGREGATE "
       "    value_key: \"z1\" "
       "    ignore_missing_data: false "
       "  > "
       "  median_tolerance_params_list: <median_coeff: 0.1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        custom_aggregate_list: < "
       "          value_key: \"z1\" "
       "          value: NaN"
       "        > "
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
       "        custom_aggregate_list: < "
       "          value_key: \"z1\" "
       "          value: NaN"
       "        > "
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
       "        custom_aggregate_list: < "
       "          value_key: \"z1\" "
       "          value: 50.0"
       "        > "
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
       "        custom_aggregate_list: < "
       "          value_key: \"z1\" "
       "          value: NaN"
       "        > "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "Tolerance must be nonnegative",
       // want_status
       Status_Code_FAIL,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {
          "mean-missing-data",
          // WindowDeviationInput
          "name: \"AnalyzerName\" "
          "run_info_query_list: <> "
          "tolerance_check_list: < "
          "  data_filter: < "
          "    data_type: BENCHMARK_SCORE "
          "    ignore_missing_data: false "
          "  > "
          "  mean_tolerance_params_list: <const_term: 1> "
          "> ",
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 45 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h2\" "
          "    timestamp_ms: 2 "
          "    aggregate: < "
          "      run_aggregate: <> "  // missing benchmark_score
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h3\" "
          "    timestamp_ms: 3 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 55 "
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
          "        benchmark_score: 50 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_in_output
          "extraction failed",
          // want_status
          Status_Code_FAIL,
      },
      {
          "run-info-query-run-order-mismatch",
          // WindowDeviationInput
          "name: \"AnalyzerName\" "
          "run_info_query_list: < "
          "  run_order: TIMESTAMP "
          "> "
          "run_info_query_list: < "
          "  run_order: BUILD_ID "
          "> "
          "tolerance_check_list: < "
          "  data_filter: < "
          "    data_type: BENCHMARK_SCORE "
          "    ignore_missing_data: false "
          "  > "
          "  mean_tolerance_params_list: <const_term: 1> "
          "> ",
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 45 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h2\" "
          "    timestamp_ms: 2 "
          "    aggregate: < "
          "      run_aggregate: <> "  // missing benchmark_score
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h3\" "
          "    timestamp_ms: 3 "
          "    aggregate: < "
          "      run_aggregate: < "
          "        benchmark_score: 55 "
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
          "        benchmark_score: 50 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_in_output
          "Inconsistent run_order field",
          // want_status
          Status_Code_FAIL,
      },
      {"median-basic'",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  recent_window_size: 3 "
       "  median_tolerance_params_list: <const_term: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 40 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 60 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"r4\" "
       "    timestamp_ms: 4 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 49 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"r5\" "
       "    timestamp_ms: 5 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 50 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"r6\" "
       "    timestamp_ms: 6 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 100 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"median-all-terms-barely-pass",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  recent_window_size: 3 "
       "  median_tolerance_params_list: < "
       "    const_term: 2.1 "
       "    median_coeff: 0.1 "
       "    mad_coeff: 2.9652 "
       "  > "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 49 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 52 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"r4\" "
       "    timestamp_ms: 4 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 49 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"r5\" "
       "    timestamp_ms: 5 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 60 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"r6\" "
       "    timestamp_ms: 6 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 100 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"median-all-terms-barely-fail",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  recent_window_size: 3 "
       "  median_tolerance_params_list: < "
       "    const_term: 2.0 "
       "    median_coeff: 0.1 "
       "    mad_coeff: 2.9652 "
       "  > "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 49 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 52 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"r4\" "
       "    timestamp_ms: 4 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 49 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"r5\" "
       "    timestamp_ms: 5 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 60 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"r6\" "
       "    timestamp_ms: 6 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 100 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       true,
       // want_in_output
       "Found 1 regressed checks",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"median-ignore-increase",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  direction_bias: IGNORE_INCREASE "
       "  median_tolerance_params_list: <const_term: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 60 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"median-ignore-decrease",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       "  direction_bias: IGNORE_DECREASE "
       "  median_tolerance_params_list: <const_term: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
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
       "        benchmark_score: 40 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"ignore-missing-data",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: METRIC_AGGREGATE_MEAN "
       "    value_key: \"y1\" "
       "    ignore_missing_data: true "
       "  > "
       "  recent_window_size: 2 "
       "  mean_tolerance_params_list: <const_term: 1> "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y2\" "  // NOTE: Missing data for 'y1'
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 100 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 100 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 101 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 2 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 3 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y2\" "  // NOTE: Missing data for 'y1'
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"r4\" "
       "    timestamp_ms: 4 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {1},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {
          "ignore-missing-data-too-few-runs",
          // WindowDeviationInput
          absl::StrCat("run_info_query_list: <> "
                       "tolerance_check_list: < ",
                       tolerance_check_with_missing_data, "> "),
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "    aggregate: < "
          "      metric_aggregate_list: < "
          "        metric_key: \"y1\" "
          "        mean: 100 "
          "      > "
          "    > "
          "  > "
          "> "
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"r4\" "
          "    timestamp_ms: 4 "
          "    aggregate: < "
          "      metric_aggregate_list: < "
          "        metric_key: \"y1\" "
          "        mean: 99 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_in_output
          "okay",
          // want_status
          Status_Code_SUCCESS,
          // checks_skipped_for_missing_data
          {tolerance_check_with_missing_data_proto},
      },
      {
          "ignore-missing-data-custom-minimum_historical_window_size",
          // WindowDeviationInput
          absl::StrCat("run_info_query_list: <> "
                       "tolerance_check_list: < ",
                       tolerance_check_minimum_historical_window_size, "> "),
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "    aggregate: < "
          "      metric_aggregate_list: < "
          "        metric_key: \"y1\" "
          "        mean: 100 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h2\" "
          "    timestamp_ms: 2 "
          "    aggregate: < "
          "      metric_aggregate_list: < "
          "        metric_key: \"y1\" "
          "        mean: 100 "
          "      > "
          "    > "
          "  > "
          "> "
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h3\" "
          "    timestamp_ms: 3 "
          "    aggregate: < "
          "      metric_aggregate_list: < "
          "        metric_key: \"y1\" "
          "        mean: 100 "
          "      > "
          "    > "
          "  > "
          "> "
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"h4\" "
          "    timestamp_ms: 4 "
          "    aggregate: < "
          "      metric_aggregate_list: < "
          "        metric_key: \"y1\" "
          "        mean: 100 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_in_output
          "okay",
          // want_status
          Status_Code_SUCCESS,
          // checks_skipped_for_missing_data
          {tolerance_check_minimum_historical_window_size_proto},
      },
      {"non-default-minimum_historical_window_size",
       // WindowDeviationInput
       absl::StrCat("run_info_query_list: <> "
                    "tolerance_check_list: < ",
                    tolerance_check_minimum_historical_window_size, "> "),
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 100 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h2\" "
       "    timestamp_ms: 2 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 100 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h3\" "
       "    timestamp_ms: 3 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 100 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h4\" "
       "    timestamp_ms: 4 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 100 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"h5\" "
       "    timestamp_ms: 5 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 100 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {4},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {
          "too-few-runs",
          // WindowDeviationInput
          "run_info_query_list: <> "
          "tolerance_check_list: < "
          "  data_filter: < "
          "    data_type: METRIC_AGGREGATE_MEAN "
          "    value_key: \"y1\" "
          "    ignore_missing_data: false "
          "  > "
          "  mean_tolerance_params_list: <const_term: 1> "
          "> ",
          // AnalyzerInput
          "historical_run_list: < "
          "  run_info: < "
          "    run_key: \"h1\" "
          "    timestamp_ms: 1 "
          "    aggregate: < "
          "      metric_aggregate_list: < "
          "        metric_key: \"y1\" "
          "        mean: 100 "
          "      > "
          "    > "
          "  > "
          "> "
          "run_to_be_analyzed: < "
          "  run_info: < "
          "    run_key: \"r4\" "
          "    timestamp_ms: 4 "
          "    aggregate: < "
          "      metric_aggregate_list: < "
          "        metric_key: \"y1\" "
          "        mean: 99 "
          "      > "
          "    > "
          "  > "
          "> ",
          // want_regression
          false,
          // want_in_output
          "Failure computing stats",
          // want_status
          Status_Code_FAIL,
      },
      {"median-negative-values",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       // These values make the tolerance very nearly zero if we don't use
       // the absolute value of each term in the sum for the tolerance
       // calculation, which causes an analysis failure.
       // After fixing that, the tolerance is a more acceptable value of 2.
       "  median_tolerance_params_list: < "
       "    median_coeff: 1 "
       "    mad_coeff: 1 "
       "  > "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: -2 "
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
       "        benchmark_score: -1 "
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
       "        benchmark_score: 0 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"h3\" "
       "    timestamp_ms: 3 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 0 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"mean-negative-values",
       // WindowDeviationInput
       "run_info_query_list: <> "
       "tolerance_check_list: < "
       "  data_filter: < "
       "    data_type: BENCHMARK_SCORE "
       "    ignore_missing_data: false "
       "  > "
       // These values make the tolerance very nearly zero if we don't use
       // the absolute value of each term in the sum for the tolerance
       // calculation, which causes an analysis failure.
       // After fixing that, the tolerance is a more acceptable value of 2.
       "  mean_tolerance_params_list: < "
       "    mean_coeff: 1 "
       "    stddev_coeff: 1.224744871391589 "
       "  > "
       "> ",
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: -2 "
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
       "        benchmark_score: -1 "
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
       "        benchmark_score: 0 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"h3\" "
       "    timestamp_ms: 3 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 0 "
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       false,
       // want_in_output
       "okay",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1},
       // want_max_timestamp_ms
       {3},
       // want_min_build_id
       {0},
       // want_max_build_id
       {0}},
      {"b/80141326",
       // WindowDeviationInput
       absl::StrCat("run_info_query_list: <> "
                    "tolerance_check_list: < ",
                    tolerance_check_with_missing_data, "> ",
                    "tolerance_check_list: < "
                    "  data_filter: < "
                    "    data_type: BENCHMARK_SCORE "
                    // ignore_missing_data defaults to true
                    "  > "
                    "  mean_tolerance_params_list: <const_term: 1> "
                    "  mean_tolerance_params_list: <mean_coeff: 0.0001> "
                    "> "),
       // AnalyzerInput
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h1\" "
       "    timestamp_ms: 1 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 45 "
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
       "        benchmark_score: 50 "
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
       "        benchmark_score: 55 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h4\" "
       "    timestamp_ms: 4 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 52 "  // delta = 2
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h5\" "
       "    timestamp_ms: 5 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h6\" "
       "    timestamp_ms: 6 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h7\" "
       "    timestamp_ms: 7 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> "
       "historical_run_list: < "
       "  run_info: < "
       "    run_key: \"h8\" "
       "    timestamp_ms: 8 "
       "    aggregate: < "
       "      metric_aggregate_list: < "
       "        metric_key: \"y1\" "
       "        mean: 99 "
       "      > "
       "    > "
       "  > "
       "> "
       "run_to_be_analyzed: < "
       "  run_info: < "
       "    run_key: \"r9\" "
       "    timestamp_ms: 9 "
       "    aggregate: < "
       "      run_aggregate: < "
       "        benchmark_score: 52 "  // delta = 2
       "      > "
       "    > "
       "  > "
       "> ",
       // want_regression
       true,
       // want_in_output
       "Found 1 regressed checks",
       // want_status
       Status_Code_SUCCESS,
       // checks_skipped_for_missing_data
       {},
       // want_min_timestamp_ms
       {1, 5},
       // want_max_timestamp_ms
       {4, 7},
       // want_min_build_id
       {0, 0},
       // want_max_build_id
       {0, 0}},
  };
  for (const Case& c : cases) {
    SCOPED_TRACE(c.name);
    LOG(INFO) << "Case: " << c.name;
    WindowDeviationInput ctor_in;
    google::protobuf::TextFormat::ParseFromString(c.ctor_in, &ctor_in);
    AnalyzerInput func_in;
    google::protobuf::TextFormat::ParseFromString(c.func_in, &func_in);
    window_deviation::Analyzer wda(ctor_in);
    AnalyzerOutput got;
    bool success = wda.Analyze(func_in, &got);
    LOG(INFO) << "Got:\n" << got.DebugString();
    WindowDeviationOutput wda_output;
    google::protobuf::TextFormat::ParseFromString(got.output(), &wda_output);
    EXPECT_EQ(got.regression(), c.want_regression);
    EXPECT_THAT(wda_output.output_message(), HasSubstr(c.want_in_output));
    EXPECT_EQ(got.status().code(), c.want_status);
    EXPECT_EQ(got.analyzer_type(), wda.analyzer_type());
    EXPECT_EQ(got.run_key(), func_in.run_to_be_analyzed().run_info().run_key());
    WindowDeviationInput parsed_input_config;
    google::protobuf::TextFormat::ParseFromString(got.input_config(),
                                        &parsed_input_config);
    EXPECT_THAT(parsed_input_config, EqualsProto(ctor_in));
    if (c.want_status == Status_Code_SUCCESS) {
      EXPECT_TRUE(success);
    } else {
      EXPECT_FALSE(success);
      continue;
    }
    ASSERT_EQ(wda_output.checks_size(), ctor_in.tolerance_check_list_size());
    int regressed_count = 0;
    int skipped_count = 0;
    int passed_count = 0;
    for (const auto& check_output : wda_output.checks()) {
      // validate that output is sorted REGRESSED > SKIPPED > PASSED
      switch (check_output.result()) {
        case WindowDeviationOutput::ToleranceCheckOutput::REGRESSED:
          regressed_count++;
          EXPECT_EQ(0, skipped_count + passed_count);
          break;
        case WindowDeviationOutput::ToleranceCheckOutput::SKIPPED:
          ASSERT_LT(skipped_count, c.checks_skipped_for_missing_data.size());
          EXPECT_THAT(
              check_output.tolerance_check(),
              EqualsProto(c.checks_skipped_for_missing_data[skipped_count]));
          skipped_count++;
          EXPECT_EQ(0, passed_count);
          break;
        case WindowDeviationOutput::ToleranceCheckOutput::PASSED:
          passed_count++;
          break;
        default:
          // if not successful, sometimes results can be uncategorized.
          if (success) {
            FAIL() << "unknown result type: " << check_output.result();
          }
          break;
      }
      // validate output stats that are populated for regressions and passed
      // checks.
      int output_stats_index = 0;
      switch (check_output.result()) {
        case WindowDeviationOutput::ToleranceCheckOutput::REGRESSED:
        case WindowDeviationOutput::ToleranceCheckOutput::PASSED:
          output_stats_index = passed_count + regressed_count - 1;
          // validate output statistics
          ASSERT_LT(output_stats_index, c.want_min_timestamp_ms.size());
          ASSERT_LT(output_stats_index, c.want_max_timestamp_ms.size());
          EXPECT_EQ(c.want_min_timestamp_ms[output_stats_index],
                    check_output.historical_window_min_timestamp_ms());
          EXPECT_EQ(c.want_max_timestamp_ms[output_stats_index],
                    check_output.historical_window_max_timestamp_ms());
          EXPECT_EQ(c.want_min_build_id[output_stats_index],
                    check_output.historical_window_min_build_id());
          EXPECT_EQ(c.want_max_build_id[output_stats_index],
                    check_output.historical_window_max_build_id());
          break;
          break;
        default:
          break;
      }
    }
    // validate that the expected number of checks skipped matches
    EXPECT_EQ(c.checks_skipped_for_missing_data.size(), skipped_count);
  }
}  // NOLINT(readability/fn_size)

}  // namespace window_deviation
}  // namespace mako
