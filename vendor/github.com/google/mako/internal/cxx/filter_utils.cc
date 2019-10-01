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

#include "glog/logging.h"
#include "src/google/protobuf/repeated_field.h"
#include "absl/strings/str_cat.h"

namespace mako {
namespace internal {

constexpr char kNoError[] = "";

// Introduced to distinguish missing data errors.
// NOTE: Could be made more generic if needed. A generic version could hold a
// string message and an int error code. "Missing Data" below, could be
// represented as a specific error code. A wrapper with the same functions could
// be written around the generic version later.
// NOTE: If increase scope of this object, consider using enums instead to
// represent different types instead of different factories.
class FilterError {
 public:
  // Construct FilterError with given error message. Not missing data.
  static FilterError Error(const std::string& err) {
    return FilterError(err, false);
  }

  // Construct FilterError with no error. Not missing data.
  static FilterError NoError() { return FilterError("", false); }

  // Construct FilterError with given error message, Missing data.
  static FilterError MissingData(const std::string& err) {
    return FilterError(err, true);
  }

  // Return the error message.
  std::string error_msg() {
    if (err_.empty()) {
      return kNoError;
    }
    return absl::StrCat("FilterUtils: ", err_);
  }

  // Is this an error
  bool error() { return !err_.empty(); }

  // Does it have missing data
  bool missing_data() { return missing_data_; }

 private:
  FilterError(const std::string& err, bool missing_data)
      : err_(err), missing_data_(missing_data) {}
  std::string err_;
  bool missing_data_;
};

FilterError ProcessSamplePoint(
    const SamplePoint& sample_point, const std::string& value_key,
    const google::protobuf::RepeatedPtrField<LabeledRange>& ignore_ranges,
    std::vector<std::pair<double, double>>* results) {
  for (const auto& keyed_value : sample_point.metric_value_list()) {
    if (!keyed_value.has_value() || !keyed_value.has_value_key()) {
      return FilterError::Error(
          absl::StrCat("SamplePoint missing keyed value. Input value:",
                       sample_point.input_value()));
    }

    if (keyed_value.value_key() == value_key) {
      // Check if sample_point was taken inside an ignore region.
      for (const LabeledRange& ignore_range : ignore_ranges) {
        if (!ignore_range.has_range()) {
          return FilterError::Error(absl::StrCat("IgnoreRange with label ",
                                                 ignore_range.label(),
                                                 " missing range."));
        }
        const Range& range = ignore_range.range();
        if (sample_point.input_value() >= range.start() &&
            sample_point.input_value() <= range.end()) {
          // We don't need to check any more points inside this sample point
          // because the entire point is at the same InputValue().
          // Performing this check here to avoid iterating through
          // ignoreRegions for SampleBatches which don't contain metricKey.
          return FilterError::NoError();
        }
      }
      results->push_back(
          std::make_pair(sample_point.input_value(), keyed_value.value()));
    }
  }
  return FilterError::NoError();
}

FilterError FilterSamplePoints(
    const std::vector<const SampleBatch*>& sample_batches,
    const std::string& value_key,
    const google::protobuf::RepeatedPtrField<LabeledRange>& ignore_ranges, bool sort_data,
    std::vector<std::pair<double, double>>* results) {
  for (const auto sample_batch : sample_batches) {
    if (sample_batch == nullptr) {
      return FilterError::Error("nullptr SampleBatch found.");
    }

    for (const auto& sample_point : sample_batch->sample_point_list()) {
      if (!sample_point.has_input_value()) {
        return FilterError::Error("SamplePoint missing input value.");
      }
      FilterError error =
          ProcessSamplePoint(sample_point, value_key, ignore_ranges, results);
      if (error.error()) {
        return error;
      }
    }
  }

  if (sort_data) {
    std::sort(results->begin(), results->end());
  }

  return FilterError::NoError();
}

FilterError PackAndPush(const RunInfo& run_info, double value,
                        std::vector<std::pair<double, double>>* results) {
  results->push_back(std::make_pair(run_info.timestamp_ms(), value));
  return FilterError::NoError();
}

FilterError FilterBenchmarkScore(
    const RunInfo& run_info, std::vector<std::pair<double, double>>* results) {
  if (!run_info.aggregate().has_run_aggregate()) {
    return FilterError::MissingData("RunInfo missing RunAggregate");
  }
  if (!run_info.aggregate().run_aggregate().has_benchmark_score()) {
    return FilterError::MissingData("RunInfo missing BenchmarkScore.");
  }
  return PackAndPush(run_info,
                     run_info.aggregate().run_aggregate().benchmark_score(),
                     results);
}

FilterError FilterErrorCount(const RunInfo& run_info,
                             std::vector<std::pair<double, double>>* results) {
  if (!run_info.aggregate().has_run_aggregate()) {
    return FilterError::MissingData("RunInfo missing RunAggregate");
  }
  if (!run_info.aggregate().run_aggregate().has_error_sample_count()) {
    return FilterError::MissingData("RunInfo missing SampleCount.");
  }
  return PackAndPush(run_info,
                     run_info.aggregate().run_aggregate().error_sample_count(),
                     results);
}

FilterError FilterCustomAggregate(
    const RunInfo& run_info, const std::string& custom_aggregate_key,
    std::vector<std::pair<double, double>>* results) {
  if (!run_info.aggregate().has_run_aggregate()) {
    return FilterError::MissingData("RunInfo missing RunAggregate");
  }
  // Search for our custom aggregate
  for (const auto& keyed_value :
       run_info.aggregate().run_aggregate().custom_aggregate_list()) {
    if (keyed_value.value_key() == custom_aggregate_key) {
      if (!keyed_value.has_value()) {
        return FilterError::MissingData("KeyedValue missing value.");
      }
      return PackAndPush(run_info, keyed_value.value(), results);
    }
  }
  return FilterError::MissingData(absl::StrCat(
      "could not find custom aggregate with key:", custom_aggregate_key));
}

int FindMetricAggregateIndex(
    const google::protobuf::RepeatedPtrField<MetricAggregate>& metric_aggregate_list,
    const std::string& value_key) {
  for (int i = 0; i < metric_aggregate_list.size(); i++) {
    const MetricAggregate& metric_aggregate = metric_aggregate_list.Get(i);
    if (metric_aggregate.metric_key() == value_key) {
      return i;
    }
  }
  return -1;
}

FilterError FilterMetricAggregate(
    const RunInfo& run_info, const DataFilter& data_filter,
    std::vector<std::pair<double, double>>* results) {
  int index = FindMetricAggregateIndex(
      run_info.aggregate().metric_aggregate_list(), data_filter.value_key());

  if (index < 0) {
    return FilterError::MissingData(
        absl::StrCat("could not find metric aggregate with value key:",
                     data_filter.value_key()));
  }

  MetricAggregate metric_aggregate =
      run_info.aggregate().metric_aggregate_list(index);

  switch (data_filter.data_type()) {
    case mako::DataFilter::METRIC_AGGREGATE_COUNT:
      return PackAndPush(run_info, metric_aggregate.count(), results);
    case mako::DataFilter::METRIC_AGGREGATE_MIN:
      return PackAndPush(run_info, metric_aggregate.min(), results);
    case mako::DataFilter::METRIC_AGGREGATE_MAX:
      return PackAndPush(run_info, metric_aggregate.max(), results);
    case mako::DataFilter::METRIC_AGGREGATE_MEAN:
      return PackAndPush(run_info, metric_aggregate.mean(), results);
    case mako::DataFilter::METRIC_AGGREGATE_MEDIAN:
      return PackAndPush(run_info, metric_aggregate.median(), results);
    case mako::DataFilter::METRIC_AGGREGATE_STDDEV:
      return PackAndPush(run_info, metric_aggregate.standard_deviation(),
                         results);
    case mako::DataFilter::METRIC_AGGREGATE_MAD:
      return PackAndPush(run_info, metric_aggregate.median_absolute_deviation(),
                         results);
    default:
      return FilterError::Error("unknown DataType()");
  }
}

FilterError FilterPercentileAggregate(
    const RunInfo& run_info, const DataFilter& data_filter,
    std::vector<std::pair<double, double>>* results) {
  int metric_index = FindMetricAggregateIndex(
      run_info.aggregate().metric_aggregate_list(), data_filter.value_key());

  if (metric_index < 0) {
    return FilterError::MissingData(absl::StrCat(
        "could not find metric aggregate with key:", data_filter.value_key()));
  }
  if (!data_filter.has_percentile_milli_rank()) {
    return FilterError::Error(
        "DataFilter does not contain percentile_milli_rank.");
  }

  MetricAggregate metric_aggregate =
      run_info.aggregate().metric_aggregate_list().Get(metric_index);
  const Aggregate& aggregate = run_info.aggregate();
  double desired_pmr = data_filter.percentile_milli_rank();

  if (aggregate.percentile_milli_rank_list_size() !=
      metric_aggregate.percentile_list_size()) {
    return FilterError::Error(
        absl::StrCat("size of RunInfo.Aggregate.percentile_milli_rank_list (",
                     aggregate.percentile_milli_rank_list_size(),
                     ") does not match size of "
                     "RunInfo.Aggregate.MetricAggregate. percentile_list (",
                     metric_aggregate.percentile_list_size(), ")"));
  }

  int m_idx = -1;
  for (int i = 0; i < aggregate.percentile_milli_rank_list_size(); i++) {
    if (aggregate.percentile_milli_rank_list(i) == desired_pmr) {
      m_idx = i;
      break;
    }
  }
  if (m_idx == -1) {
    return FilterError::MissingData(
        absl::StrCat("could not find percentile: ", desired_pmr,
                     " in RunInfo.Aggregate.percentile_milli_rank_list"));
  }

  return PackAndPush(run_info, metric_aggregate.percentile_list(m_idx),
                     results);
}

FilterError FilterAggregates(const RunInfo& run_info,
                             const DataFilter& data_filter,
                             std::vector<std::pair<double, double>>* results) {
  if (!run_info.has_aggregate()) {
    return FilterError::MissingData("RunInfo missing aggregate");
  }
  switch (data_filter.data_type()) {
    case mako::DataFilter::ERROR_COUNT:
      return FilterErrorCount(run_info, results);
    case mako::DataFilter::BENCHMARK_SCORE:
      return FilterBenchmarkScore(run_info, results);
    case mako::DataFilter::CUSTOM_AGGREGATE:
      return FilterCustomAggregate(run_info, data_filter.value_key(), results);
    case mako::DataFilter::METRIC_AGGREGATE_PERCENTILE:
      return FilterPercentileAggregate(run_info, data_filter, results);
    default:
      return FilterMetricAggregate(run_info, data_filter, results);
  }
}

std::string ApplyFilter(const RunInfo& run_info,
                   const std::vector<const SampleBatch*>& sample_batches,
                   const DataFilter& data_filter, bool sort_data,
                   std::vector<std::pair<double, double>>* results) {
  if (!data_filter.has_data_type()) {
    return FilterError::Error("DataFilter is missing data_type").error_msg();
  }
  if (!data_filter.has_value_key()) {
    if (data_filter.data_type() != mako::DataFilter::ERROR_COUNT &&
        data_filter.data_type() != mako::DataFilter::BENCHMARK_SCORE) {
      return FilterError::Error("DataFilter is missing value_key").error_msg();
    }
  }

  FilterError err =
      (data_filter.data_type() == mako::DataFilter::METRIC_SAMPLEPOINTS)
          ? FilterSamplePoints(sample_batches, data_filter.value_key(),
                               run_info.ignore_range_list(), sort_data, results)
          : FilterAggregates(run_info, data_filter, results);
  if (err.error() && err.missing_data() && data_filter.ignore_missing_data()) {
    VLOG(1)
        << "ignoring error because DataFilter.ignore_missing_data = true; err: "
        << err.error_msg();
    return kNoError;
  }
  return err.error_msg();
}

}  // namespace internal
}  // namespace mako
