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

#include <algorithm>
#include <cmath>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/repeated_field.h"
#include "src/google/protobuf/text_format.h"
#include "clients/proto/analyzers/window_deviation.pb.h"
#include "spec/proto/mako.pb.h"
#include "absl/strings/str_cat.h"
#include "absl/types/span.h"
#include "clients/cxx/analyzers/util.h"
#include "helpers/cxx/status/status.h"
#include "internal/cxx/analyzer_common.h"
#include "internal/cxx/filter_utils.h"
#include "internal/cxx/pgmath.h"

namespace mako {
namespace window_deviation {

namespace {
constexpr char kNoError[] = "";
}

bool Analyzer::ConstructHistoricQuery(const AnalyzerHistoricQueryInput& input,
                                      AnalyzerHistoricQueryOutput* output) {
  output->Clear();
  output->set_get_batches(false);
  // Set query timestamps and benchmark keys if they are not set
  for (const auto& orig_query : config_.run_info_query_list()) {
    RunInfoQuery* query = output->add_run_info_query_list();
    *query = orig_query;
    switch (query->run_order()) {
      case RunOrder::UNSPECIFIED:
      case RunOrder::TIMESTAMP:
        if (!query->has_max_timestamp_ms()) {
          query->set_max_timestamp_ms(input.run_info().timestamp_ms() - 1);
          LOG(INFO) << "Automatically set run_info_query.max_timestamp_ms to "
                    << query->max_timestamp_ms();
        }
        break;
      case RunOrder::BUILD_ID:
        if (!query->has_max_build_id()) {
          query->set_max_build_id(input.run_info().build_id() - 1);
          LOG(INFO) << "Automatically set run_info_query.max_build_id to "
                    << query->max_build_id();
        }
        break;
      default:
        output->mutable_status()->set_code(Status_Code_FAIL);
        output->mutable_status()->set_fail_message(absl::StrCat(
            "Unknown RunOrder: ", RunOrder_Name(query->run_order()),
            ". This is an internal Mako error, please file a bug "
            "go/mako-bug."));
        return false;
    }
    if (!query->has_benchmark_key()) {
      query->set_benchmark_key(input.benchmark_info().benchmark_key());
      LOG(INFO) << "Automatically set run_info_query.benchmark_key to "
                << query->benchmark_key();
    }
  }
  output->mutable_status()->set_code(Status_Code_SUCCESS);
  return true;
}

namespace {

void SetWDAOutput(const std::string& msg, WindowDeviationOutput* wda_output,
                  AnalyzerOutput* output) {
  wda_output->set_output_message(msg);
  google::protobuf::TextFormat::PrintToString(*wda_output, output->mutable_output());
}

void SetAnalyzerOutputPassing(const std::string& msg,
                              WindowDeviationOutput* wda_output,
                              AnalyzerOutput* output) {
  LOG(INFO) << msg;
  output->set_regression(false);
  output->mutable_status()->set_code(Status_Code_SUCCESS);
  SetWDAOutput(msg, wda_output, output);
}

void SetAnalyzerOutputWithError(const std::string& msg,
                                WindowDeviationOutput* wda_output,
                                AnalyzerOutput* output) {
  LOG(INFO) << msg;
  output->set_regression(false);
  output->mutable_status()->set_code(Status_Code_FAIL);
  output->mutable_status()->set_fail_message(msg);
  SetWDAOutput(msg, wda_output, output);
}

void SetAnalyzerOutputWithRegression(const std::string& msg,
                                     WindowDeviationOutput* wda_output,
                                     AnalyzerOutput* output) {
  LOG(INFO) << msg;
  output->set_regression(true);
  output->mutable_status()->set_code(Status_Code_SUCCESS);
  SetWDAOutput(msg, wda_output, output);
}

std::string ComputeStats(
    const std::vector<internal::RunData>& data,
    int recent_window_size, ToleranceCheckStats* output) {
  absl::Span<const internal::RunData>
      historic_data =
          absl::MakeConstSpan(&data[0], data.size() - recent_window_size);
  absl::Span<const internal::RunData> recent_data =
      absl::MakeConstSpan(&data[data.size() - recent_window_size],
                          recent_window_size);
  output->set_historic_data_length(historic_data.size());
  output->set_recent_data_length(recent_data.size());

  // prep running stats objects
  mako::internal::RunningStats historic_stats;
  mako::internal::RunningStats recent_stats;
  for (const auto& d : historic_data) {
    historic_stats.Add(d.value);
  }
  for (const auto& d : recent_data) {
    recent_stats.Add(d.value);
  }
  mako::internal::RunningStats::Result result;

  // historic_mean
  result = historic_stats.Mean();
  if (!result.error.empty()) {
    return absl::StrCat("Failure computing historic mean: ", result.error);
  }
  output->set_historic_mean(result.value);

  // historic_median
  result = historic_stats.Median();
  if (!result.error.empty()) {
    return absl::StrCat("Failure computing historic median: ", result.error);
  }
  output->set_historic_median(result.value);

  // historic_stddev
  result = historic_stats.Stddev();
  if (!result.error.empty()) {
    return absl::StrCat("Failure computing historic stddev: ", result.error);
  }
  output->set_historic_stddev(result.value);

  // historic_mad
  result = historic_stats.Mad();
  if (!result.error.empty()) {
    return absl::StrCat("Failure computing historic MAD: ", result.error);
  }
  output->set_historic_mad(result.value);

  // recent_mean
  result = recent_stats.Mean();
  if (!result.error.empty()) {
    return absl::StrCat("Failure computing recent mean: ", result.error);
  }
  output->set_recent_mean(result.value);

  // recent_median
  result = recent_stats.Median();
  if (!result.error.empty()) {
    return absl::StrCat("Failure computing recent median: ", result.error);
  }
  output->set_recent_median(result.value);

  // deltas
  output->set_delta_mean(output->recent_mean() - output->historic_mean());
  output->set_delta_median(output->recent_median() - output->historic_median());

  return kNoError;
}

}  // namespace

bool Analyzer::DoAnalyze(const AnalyzerInput& input, AnalyzerOutput* output) {
  LOG(INFO) << "START: Window Deviation Analyzer";

  // Save the analyzer configuration.
  google::protobuf::TextFormat::PrintToString(config_,
                                    output->mutable_input_config());

  // Implementation specific output that will be eventually merged into
  // AnayzerOutput as a text serialized proto.
  WindowDeviationOutput wda_output;

  // Validate config - it was passed to ctor, but we can't return error there
  std::string err = ValidateWindowDeviationInput();
  if (!err.empty()) {
    err = absl::StrCat("Bad WindowDeviationInput provided to WDA ctor: ", err);
    LOG(ERROR) << err;
    SetAnalyzerOutputWithError(err, &wda_output, output);
    LOG(INFO) << "END: Window Deviation Analyzer";
    return false;
  }

  // Determine if we are sorting by timestamp or build_id
  const RunOrder run_order = config_.run_info_query_list(0).run_order();

  // Sort runs ascending by the specified field order.
  const std::vector<const RunInfo*> sorted_runs =
      internal::SortRuns(input, run_order);

  // Iterate over each ToleranceCheck
  std::stringstream regression_output;
  int total_checks_regressed_count = 0;
  bool overall_regression = false;
  const auto& bench_info = input.run_to_be_analyzed().benchmark_info();
  for (int check_index = 0; check_index < config_.tolerance_check_list_size();
       check_index++) {
    WindowDeviationOutput::ToleranceCheckOutput* check_output =
        wda_output.add_checks();
    const ToleranceCheck& check = config_.tolerance_check_list(check_index);
    *check_output->mutable_tolerance_check() = check;
    check_output->set_metric_label(
      analyzer_util::GetHumanFriendlyDataFilterString(
            check.data_filter(), bench_info));

    // Extract data
    auto status_or_data = internal::ExtractDataAndRemoveEmptyResults(
        check.data_filter(), sorted_runs);
    if (!status_or_data.ok()) {
      err = absl::StrCat(
          "Failure extracting data: ",
          helpers::StatusToString(std::move(status_or_data).status()));
      LOG(ERROR) << err;
      SetAnalyzerOutputWithError(err, &wda_output, output);
      LOG(INFO) << "END: Window Deviation Analyzer";
      return false;
    }

    std::vector<internal::RunData> data =
        std::move(status_or_data).value();

    // check length
    if (data.size() <
        static_cast<std::size_t>(check.recent_window_size() +
                                 check.minimum_historical_window_size())) {
      std::stringstream errss;
      errss << "Total data length is " << data.size() << ", recent is "
            << check.recent_window_size() << ", historic must be >= "
            << check.minimum_historical_window_size() << ".";
      err = errss.str();
      if (check.data_filter().ignore_missing_data()) {
        LOG(INFO) << err << " Ignoring missing data.";
        check_output->set_result(
            WindowDeviationOutput::ToleranceCheckOutput::SKIPPED);
        continue;
      }
      err = absl::StrCat("Failure computing stats: ", err);
      LOG(ERROR) << err;
      SetAnalyzerOutputWithError(err, &wda_output, output);
      LOG(INFO) << "END: Window Deviation Analyzer";
      return false;
    }

    // Record some properties about the historical window being used in this
    // check that allow us to approximately re-create the historical window.
    switch (run_order) {
      case RunOrder::UNSPECIFIED:
      case RunOrder::TIMESTAMP:
        check_output->set_historical_window_min_timestamp_ms(
            data.front().run->timestamp_ms());
        check_output->set_historical_window_max_timestamp_ms(
            data[data.size() - check.recent_window_size() - 1]
                .run->timestamp_ms());
        break;
      case RunOrder::BUILD_ID:
        check_output->set_historical_window_min_build_id(
            data.front().run->build_id());
        check_output->set_historical_window_max_build_id(
            data[data.size() - check.recent_window_size() - 1].run->build_id());
        break;
    }

    // Compute historic stats, recent stats, and deltas
    ToleranceCheckStats* stats = check_output->mutable_stats();
    err = ComputeStats(data, check.recent_window_size(), stats);
    if (!err.empty()) {
      err = absl::StrCat("Failure computing stats: ", err);
      LOG(ERROR) << err;
      SetAnalyzerOutputWithError(err, &wda_output, output);
      LOG(INFO) << "END: Window Deviation Analyzer";
      return false;
    }

    // All must exceed tolerance below for check regression, so assume
    // regression and set to false if anything passes.
    bool is_check_regressed = true;
    auto bias = check.direction_bias();

    // Check tolerance for each set of mean params for this check.
    for (int i = 0; i < check.mean_tolerance_params_list_size(); ++i) {
      const MeanToleranceParams& params = check.mean_tolerance_params_list(i);
      MeanToleranceCheckResult* result =
          stats->add_mean_tolerance_check_result();
      *result->mutable_params() = params;
      double tolerance =
          params.const_term() +
          params.mean_coeff() * std::abs(stats->historic_mean()) +
          params.stddev_coeff() * stats->historic_stddev();
      // Tolerance could be NaN, so this is not equivalent to tolerance < 0.
      if (!(tolerance >= 0)) {
        std::stringstream errss;
        errss << "Tolerance must be nonnegative. Got: " << tolerance;
        err = errss.str();
        LOG(ERROR) << err;
        SetAnalyzerOutputWithError(err, &wda_output, output);
        LOG(INFO) << "END: Window Deviation Analyzer";
        return false;
      }
      result->set_tolerance(tolerance);
      result->set_is_regressed(false);
      if (stats->delta_mean() <= 0 &&
          bias == ToleranceCheck_DirectionBias_IGNORE_DECREASE) {
        is_check_regressed = false;
      } else if (stats->delta_mean() >= 0 &&
                 bias == ToleranceCheck_DirectionBias_IGNORE_INCREASE) {
        is_check_regressed = false;
      } else if (std::abs(stats->delta_mean()) <= tolerance) {
        is_check_regressed = false;
      } else {
        result->set_is_regressed(true);
      }
    }

    // Check tolerance for each set of median params for this check.
    for (int i = 0; i < check.median_tolerance_params_list_size(); ++i) {
      const MedianToleranceParams& params =
          check.median_tolerance_params_list(i);
      MedianToleranceCheckResult* result =
          stats->add_median_tolerance_check_result();
      *result->mutable_params() = params;
      double tolerance =
          params.const_term() +
          params.median_coeff() * std::abs(stats->historic_median()) +
          params.mad_coeff() * stats->historic_mad();
      // Tolerance could be NaN, so this is not equivalent to tolerance < 0.
      if (!(tolerance >= 0)) {
        std::stringstream errss;
        errss << "Tolerance must be nonnegative. Got: " << tolerance;
        err = errss.str();
        LOG(ERROR) << err;
        SetAnalyzerOutputWithError(err, &wda_output, output);
        LOG(INFO) << "END: Window Deviation Analyzer";
        return false;
      }
      result->set_tolerance(tolerance);
      result->set_is_regressed(false);
      if (stats->delta_median() <= 0 &&
          bias == ToleranceCheck_DirectionBias_IGNORE_DECREASE) {
        is_check_regressed = false;
      } else if (stats->delta_median() >= 0 &&
                 bias == ToleranceCheck_DirectionBias_IGNORE_INCREASE) {
        is_check_regressed = false;
      } else if (std::abs(stats->delta_median()) <= tolerance) {
        is_check_regressed = false;
      } else {
        result->set_is_regressed(true);
      }
    }

    // Log all details, but only save details to output for check regression.
    if (is_check_regressed) {
      check_output->set_result(
          WindowDeviationOutput::ToleranceCheckOutput::REGRESSED);
      std::string msg = absl::StrCat("Check found REGRESSION:\n",
                                check_output->DebugString());
      LOG(INFO) << msg;
      total_checks_regressed_count++;
      if (overall_regression) {
        // We've already added regression info for a check,
        // so add newline before next
        regression_output << "\n";
      }
      regression_output << msg;
      overall_regression = true;
    } else {
      check_output->set_result(
          WindowDeviationOutput::ToleranceCheckOutput::PASSED);
      LOG(INFO) << absl::StrCat("Check passed:\n", check_output->DebugString());
    }
  }

  // Maintain an intentional ordering for checks so that regressed checks
  // all show up first, followed by skipped checks, followed by passed checks.
  std::sort(wda_output.mutable_checks()->begin(),
            wda_output.mutable_checks()->end(),
            [](const WindowDeviationOutput::ToleranceCheckOutput& a,
               const WindowDeviationOutput::ToleranceCheckOutput& b) {
              return a.result() < b.result();
            });

  // Set output, complete log section, and return true since the analyzer
  // completed its analysis.
  if (overall_regression) {
    LOG(INFO) << regression_output.str();
    SetAnalyzerOutputWithRegression(
        absl::StrCat("Found ", total_checks_regressed_count,
                     " regressed checks. See the 'checks' field for details."),
        &wda_output, output);
  } else {
    SetAnalyzerOutputPassing("okay", &wda_output, output);
  }
  LOG(INFO) << "END: Window Deviation Analyzer";
  return true;
}

std::string Analyzer::ValidateWindowDeviationInput() const {
  if (config_.run_info_query_list_size() == 0) {
    return "WindowDeviationInput.run_info_query_list is empty";
  }
  if (config_.tolerance_check_list_size() == 0) {
    return "WindowDeviationInput.tolerance_check_list is empty";
  }
  // Validate that all run info query have the same RunOrder.
  RunOrder first_run_order = config_.run_info_query_list(0).run_order();
  if (first_run_order == RunOrder::UNSPECIFIED) {
    first_run_order = RunOrder::TIMESTAMP;
  }
  for (int i = 1; i < config_.run_info_query_list().size(); ++i) {
    RunOrder current_run_order = config_.run_info_query_list(i).run_order();
    if (current_run_order == RunOrder::UNSPECIFIED) {
      current_run_order = RunOrder::TIMESTAMP;
    }
    if (first_run_order != current_run_order) {
      return absl::StrCat(
          "Inconsistent run_order field in "
          "WindowDeviationInput.run_info_query_list.");
    }
  }
  for (int i = 0; i < config_.tolerance_check_list_size(); ++i) {
    const ToleranceCheck& check = config_.tolerance_check_list(i);
    if (!check.has_data_filter()) {
      return absl::StrCat("WindowDeviationInput.tolerance_check_list[", i,
                          "].data_filter is missing");
    }
    if (check.data_filter().data_type() ==
        DataFilter_DataType_METRIC_SAMPLEPOINTS) {
      return absl::StrCat("WindowDeviationInput.tolerance_check_list[", i,
                          "].data_filter.data_type must be an aggregate");
    }
    if (check.recent_window_size() < 1) {
      return absl::StrCat("WindowDeviationInput.tolerance_check_list[", i,
                          "].recent_window_size must be > 0");
    }
    if (check.mean_tolerance_params_list_size() == 0 &&
        check.median_tolerance_params_list_size() == 0) {
      return absl::StrCat("WindowDeviationInput.tolerance_check_list[", i,
                          "] missing *ToleranceParamsList");
    }
    for (int j = 0; j < check.mean_tolerance_params_list_size(); ++j) {
      const MeanToleranceParams& params = check.mean_tolerance_params_list(j);
      if (params.const_term() < 0 || params.mean_coeff() < 0 ||
          params.stddev_coeff() < 0) {
        return absl::StrCat("WindowDeviationInput.tolerance_check_list[", i,
                            "].mean_tolerance_params_list[", j,
                            "] has a negative value");
      }
      if (params.const_term() == 0 && params.mean_coeff() == 0 &&
          params.stddev_coeff() == 0) {
        return absl::StrCat("WindowDeviationInput.tolerance_check_list[", i,
                            "].mean_tolerance_params_list[", j,
                            "] has no values set");
      }
    }
    for (int j = 0; j < check.median_tolerance_params_list_size(); ++j) {
      const MedianToleranceParams& params =
          check.median_tolerance_params_list(j);
      if (params.const_term() < 0 || params.median_coeff() < 0 ||
          params.mad_coeff() < 0) {
        return absl::StrCat("WindowDeviationInput.tolerance_check_list[", i,
                            "].median_tolerance_params_list[", j,
                            "] has a negative value");
      }
      if (params.const_term() == 0 && params.median_coeff() == 0 &&
          params.mad_coeff() == 0) {
        return absl::StrCat("WindowDeviationInput.tolerance_check_list[", i,
                            "].median_tolerance_params_list[", j,
                            "] has no values set");
      }
    }
  }
  return "";
}

}  // namespace window_deviation
}  // namespace mako
