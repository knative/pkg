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

#include <numeric>
#include <sstream>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/text_format.h"
#include "absl/strings/str_cat.h"
#include "clients/cxx/analyzers/util.h"
#include "internal/cxx/analyzer_common.h"
#include "internal/cxx/filter_utils.h"

namespace mako {
namespace threshold_analyzer {

namespace {

constexpr char kBugLink[] = "https://github.com/google/mako/issues";

}  // namespace

using mako::AnalyzerInput;
using mako::AnalyzerOutput;
using mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput;
using mako::analyzers::threshold_analyzer::ThresholdAnalyzerOutput;
using mako::analyzers::threshold_analyzer::ThresholdConfigResult;

Analyzer::Analyzer(const ThresholdAnalyzerInput& analyzer_input)
    : config_(analyzer_input) {}

bool Analyzer::ConstructHistoricQuery(
    const mako::AnalyzerHistoricQueryInput& query_input,
    mako::AnalyzerHistoricQueryOutput* query_output) {
  query_output->Clear();

  if (config_.has_cross_run_config()) {
    // Determine if we can skip fetching sample batches.
    query_output->set_get_batches(false);
    // We will need to get sample batches if any of the threshold
    // configs wants to use sample points
    for (const auto& config : config_.configs()) {
      if (config.data_filter().data_type() ==
          mako::DataFilter::METRIC_SAMPLEPOINTS) {
        query_output->set_get_batches(true);
        break;
      }
    }

    // Set query timestamps and benchmark keys if they are not set
    for (const auto& orig_query :
         config_.cross_run_config().run_info_query_list()) {
      RunInfoQuery* query = query_output->add_run_info_query_list();
      *query = orig_query;
      switch (query->run_order()) {
        case RunOrder::UNSPECIFIED:
        case RunOrder::TIMESTAMP:
          if (!query->has_max_timestamp_ms()) {
            query->set_max_timestamp_ms(query_input.run_info().timestamp_ms() -
                                        1);
            LOG(INFO) << "Automatically set run_info_query.max_timestamp_ms to "
                      << query->max_timestamp_ms();
          }
          break;
        case RunOrder::BUILD_ID:
          if (!query->has_max_build_id()) {
            query->set_max_build_id(query_input.run_info().build_id() - 1);
            LOG(INFO) << "Automatically set run_info_query.max_build_id to "
                      << query->max_build_id();
          }
          break;
        default:
          query_output->mutable_status()->set_code(Status_Code_FAIL);
          query_output->mutable_status()->set_fail_message(absl::StrCat(
              "Unknown RunOrder: ", RunOrder_Name(query->run_order()),
              ". This is an internal Mako error, please file a bug at: ",
              kBugLink));
          return false;
      }
      if (!query->has_benchmark_key()) {
        query->set_benchmark_key(query_input.benchmark_info().benchmark_key());
        LOG(INFO) << "Automatically set run_info_query.benchmark_key to "
                  << query->benchmark_key();
      }
    }
  }

  query_output->mutable_status()->set_code(mako::Status::SUCCESS);
  return true;
}

bool Analyzer::SetAnalyzerError(const std::string& error_message,
                                mako::AnalyzerOutput* output) {
  auto status = output->mutable_status();
  status->set_code(mako::Status_Code_FAIL);
  status->set_fail_message(
      absl::StrCat("threshold_analyzer::Analyzer Error: ", error_message));
  return false;
}

void EvaluateThresholdConfig(
    const std::vector<std::pair<double, double>>& data,
    const mako::analyzers::threshold_analyzer::ThresholdConfig& config,
    ThresholdConfigResult* result) {
  double points_under_min = 0;
  double points_above_max = 0;
  double total_points = data.size();

  for (auto& pair : data) {
    if (config.has_min() && pair.second < config.min()) {
      points_under_min++;
    } else if (config.has_max() && pair.second > config.max()) {
      points_above_max++;
    }
  }

  double actual_percent_outside_range =
      (points_under_min + points_above_max) / total_points * 100;

  LOG(INFO) << "Points above max: " << points_above_max;
  LOG(INFO) << "Points below min: " << points_under_min;
  LOG(INFO) << "Actual percent outliers: " << actual_percent_outside_range
            << "%";

  // Record analysis values for visualization.
  result->set_percent_above_max(points_above_max / total_points * 100);
  result->set_percent_below_min(points_under_min / total_points * 100);
  if (data.size() == 1) {
    result->set_value_outside_threshold(data[0].second);
  }

  result->set_regression(actual_percent_outside_range >
                         config.outlier_percent_max());
}

bool Analyzer::DoAnalyze(const mako::AnalyzerInput& analyzer_input,
                         mako::AnalyzerOutput* analyzer_output) {
  // Save the analyzer configuration.
  google::protobuf::TextFormat::PrintToString(config_,
                                    analyzer_output->mutable_input_config());

  if (!analyzer_input.has_run_to_be_analyzed()) {
    return SetAnalyzerError("AnalyzerInput missing run_to_be_analyzed.",
                            analyzer_output);
  }
  if (!analyzer_input.run_to_be_analyzed().has_run_info()) {
    return SetAnalyzerError("RunBundle missing run_info.", analyzer_output);
  }

  auto& run_bundle = analyzer_input.run_to_be_analyzed();
  auto& run_info = run_bundle.run_info();

  bool regression_found = false;
  ThresholdAnalyzerOutput config_out;

  if (config_.has_cross_run_config()) {
    // Determine if we are sorting by timestamp or build_id
    const RunOrder run_order =
        config_.cross_run_config().run_info_query_list(0).run_order();
    // Sort runs ascending by the specified field order.
    const std::vector<const RunInfo*> sorted_runs =
        internal::SortRuns(analyzer_input, run_order);

    // Record some properties about the historical window being used in this
    // analysis that allow us to approximately re-create the historical window
    // when executing the same collection of RunInfoQuery messages in the
    // future.
    switch (run_order) {
      case RunOrder::UNSPECIFIED:
      case RunOrder::TIMESTAMP:
        config_out.set_min_timestamp_ms(sorted_runs.front()->timestamp_ms());
        config_out.set_max_timestamp_ms(sorted_runs.back()->timestamp_ms());
        break;
      case RunOrder::BUILD_ID:
        config_out.set_min_build_id(sorted_runs.front()->build_id());
        config_out.set_max_build_id(sorted_runs.back()->build_id());
        break;
    }
  }

  for (auto& config : config_.configs()) {
    if (!config.has_data_filter()) {
      return SetAnalyzerError("ThresholdConfig missing DataFilter.",
                              analyzer_output);
    }

    if (!config.has_max() && !config.has_min()) {
      return SetAnalyzerError("ThresholdConfig must have at least max or min.",
                              analyzer_output);
    }

    std::vector<std::pair<double, double>> results;

    std::vector<const SampleBatch*> batches;
    for (const SampleBatch& sample_batch : run_bundle.batch_list()) {
      batches.push_back(&sample_batch);
    }

    auto error_string = mako::internal::ApplyFilter(
        run_info, batches, config.data_filter(), false, &results);

    if (error_string.length() > 0) {
      std::stringstream ss;
      ss << "Error attempting to retrieve data using data_filter: "
         << config.data_filter().ShortDebugString()
         << ". Error message: " << error_string;
      return SetAnalyzerError(ss.str(), analyzer_output);
    }
    if (results.empty()) {
      std::stringstream ss;
      ss << "Did not find any data using data_filter: "
         << config.data_filter().DebugString();
      if (config.data_filter().ignore_missing_data()) {
        // In the default case we do not emit an error.
        LOG(INFO) << ss.str() << " Ignoring missing data.";
        continue;
      }
      return SetAnalyzerError(ss.str(), analyzer_output);
    }

    ThresholdConfigResult* config_result = config_out.add_config_results();
    // Record threshold config and human friendly metric label.
    config_result->set_metric_label(
        ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
            config.data_filter(), run_bundle.benchmark_info()));
    *config_result->mutable_config() = config;

    LOG(INFO) << "----------";
    LOG(INFO) << "Starting Threshold Config analysis";
    LOG(INFO) << "Threshold Config: " << config.ShortDebugString();
    EvaluateThresholdConfig(results, config, config_result);

    if (config_result->regression() && config_.has_cross_run_config()) {
      config_result->set_cross_run_config_exercised(true);

      // Add in the extra data sets from the historical runs to do a broader
      // evaluation of the same ThresholdConfig.
      int run_with_data_count = 0;
      for (const auto& cross_run_bundle :
           analyzer_input.historical_run_list()) {
        std::vector<std::pair<double, double>> run_results;
        std::vector<const SampleBatch*> cross_run_batch;
        for (const SampleBatch& sample_batch : cross_run_bundle.batch_list()) {
          cross_run_batch.push_back(&sample_batch);
        }

        auto error_string = mako::internal::ApplyFilter(
            cross_run_bundle.run_info(), cross_run_batch, config.data_filter(),
            false, &run_results);

        if (error_string.length() > 0) {
          std::stringstream ss;
          ss << "Error attempting to retrieve data using data_filter: "
             << config.data_filter().ShortDebugString()
             << ". Error message: " << error_string;
          return SetAnalyzerError(ss.str(), analyzer_output);
        }
        if (!run_results.empty()) {
          run_with_data_count++;
          results.insert(results.end(), run_results.begin(), run_results.end());
        }
      }

      // Skip analysis if we don't have enough history and the user set the
      // min_run_count field.
      if (config_.cross_run_config().has_min_run_count() &&
          run_with_data_count < config_.cross_run_config().min_run_count()) {
        LOG(INFO) << "Skipping cross run analysis. Only got "
                  << run_with_data_count
                  << " historical runs, not enough for analyzing. The minimum "
                  << "historical run count is: "
                  << config_.cross_run_config().min_run_count();

        config_result->set_regression(false);
      } else {
        if (config.data_filter().data_type() !=
            mako::DataFilter::METRIC_SAMPLEPOINTS) {
          // If we are not using all sample points, instead we are using a set
          // of pre-computed aggregates across the various runs then go ahead
          // and compute the median of the collection of aggregate values that
          // we have.
          std::sort(results.begin(), results.end());
          double median = results.size() % 2 == 1
                              ? results[results.size() / 2].second
                              : ((results[results.size() / 2 - 1].second +
                                  results[results.size() / 2].second) /
                                 2);
          results.clear();
          results.push_back({0XBADF00D, median});
        }

        LOG(INFO) << "++++++++++";
        LOG(INFO) << "Running Cross Run Threshold Config analysis";
        EvaluateThresholdConfig(results, config, config_result);
      }
    }

    if (config_result->regression()) {
      LOG(INFO) << "REGRESSION found!";
      regression_found = true;
    }

    LOG(INFO) << "Analysis complete for config";
    LOG(INFO) << "----------";
  }

  analyzer_output->mutable_status()->set_code(mako::Status_Code_SUCCESS);
  analyzer_output->set_regression(regression_found);
  google::protobuf::TextFormat::PrintToString(config_out,
                                    analyzer_output->mutable_output());
  return true;
}

}  // namespace threshold_analyzer
}  // namespace mako
