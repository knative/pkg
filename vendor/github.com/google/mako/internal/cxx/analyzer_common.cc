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
#include "internal/cxx/analyzer_common.h"

#include <functional>

#include "glog/logging.h"
#include "absl/strings/str_cat.h"
#include "helpers/cxx/status/canonical_errors.h"
#include "helpers/cxx/status/status.h"
#include "helpers/cxx/status/statusor.h"
#include "internal/cxx/filter_utils.h"

namespace mako {
namespace internal {

helpers::StatusOr<std::vector<RunData>> ExtractDataAndRemoveEmptyResults(
    const DataFilter& data_filter,
    const std::vector<const RunInfo*>& sorted_runs) {
  std::vector<RunData> data;
  // Optimize for the case where there are not many empty results.
  data.reserve(sorted_runs.size());
  for (const RunInfo* run : sorted_runs) {
    std::vector<const SampleBatch*> unused;
    std::vector<std::pair<double, double>> results;
    std::string err = ::mako::internal::ApplyFilter(*run, unused, data_filter,
                                                   false, &results);
    if (!err.empty()) {
      return helpers::UnknownError(
          absl::StrCat("Run data extraction failed for run_key(",
                       run->run_key(), "): ", err));
    }
    if (results.empty()) {
      VLOG(1) << "No run data found for run key: " << run->run_key();
      continue;
    } else if (results.size() != 1) {
      return helpers::UnknownError(
          absl::StrCat("Run data extraction failed to get one value, got : ",
                       results.size()));
    }
    data.push_back(RunData{run, results[0].second});
  }
  return data;
}

std::vector<const RunInfo*> SortRuns(const AnalyzerInput& input,
                                     const RunOrder& run_order) {
  std::vector<const RunInfo*> runs;
  for (const RunBundle& bundle : input.historical_run_list()) {
    runs.push_back(&bundle.run_info());
  }

  std::function<bool(const RunInfo*, const RunInfo*)> sort_function;
  switch (run_order) {
    case RunOrder::UNSPECIFIED:
    case RunOrder::TIMESTAMP:
    default:
      sort_function = [](const RunInfo* a, const RunInfo* b) {
        return a->timestamp_ms() < b->timestamp_ms();
      };
      break;
    case RunOrder::BUILD_ID:
      sort_function = [](const RunInfo* a, const RunInfo* b) {
        return a->build_id() < b->build_id();
      };
      break;
  }
  std::sort(runs.begin(), runs.end(), sort_function);
  runs.push_back(&input.run_to_be_analyzed().run_info());
  return runs;
}

}  // namespace internal
}  // namespace mako
