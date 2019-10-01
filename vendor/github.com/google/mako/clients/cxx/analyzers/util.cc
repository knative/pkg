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
#include "clients/cxx/analyzers/util.h"

#include "absl/strings/str_format.h"
#include "spec/proto/mako.pb.h"

using ::absl::StrFormat;

namespace mako {
namespace analyzer_util {

std::string GetHumanFriendlyDataFilterString(
    const mako::DataFilter& data_filter,
    const mako::BenchmarkInfo& bench_info) {
  std::string label = data_filter.value_key();
  switch (data_filter.data_type()) {
    case mako::DataFilter::BENCHMARK_SCORE:
      label = "benchmark_score";
      break;
    case mako::DataFilter::ERROR_COUNT:
      label = "error_count";
      break;
    case mako::DataFilter::CUSTOM_AGGREGATE:
      for (const auto& info : bench_info.custom_aggregation_info_list()) {
        if (data_filter.value_key() == info.value_key()) {
          label = info.label();
          break;
        }
      }
      break;
    default:
      for (const auto& info : bench_info.metric_info_list()) {
        if (data_filter.value_key() == info.value_key()) {
          label = info.label();
          break;
        }
      }
      break;
  }
  if (label.empty()) {
    label = "unknown";
  }

  std::string suffix;
  if (data_filter.data_type() == DataFilter::METRIC_AGGREGATE_PERCENTILE) {
    const double percentile =
        static_cast<double>(data_filter.percentile_milli_rank()) / 1000.0;
    suffix = StrFormat("p%4.3f", percentile);
  } else {
    static const auto* kTypeParams =
        new std::map<DataFilter::DataType, std::string>({
            {DataFilter::METRIC_AGGREGATE_MEAN, "mean"},
            {DataFilter::METRIC_AGGREGATE_MEDIAN, "median"},
            {DataFilter::METRIC_AGGREGATE_MAX, "max"},
            {DataFilter::METRIC_AGGREGATE_MIN, "min"},
            {DataFilter::METRIC_AGGREGATE_COUNT, "count"},
            {DataFilter::METRIC_AGGREGATE_STDDEV, "stddev"},
            {DataFilter::METRIC_AGGREGATE_MAD, "mad"},
            {DataFilter::CUSTOM_AGGREGATE, "customagg"},
        });
    auto it = kTypeParams->find(data_filter.data_type());
    suffix = it != kTypeParams->end() ? it->second : "";
  }

  if (suffix.empty()) {
    return label;
  } else {
    return StrFormat("%s\\%s", label, suffix);
  }
}

}  // namespace analyzer_util
}  // namespace mako
