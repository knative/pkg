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

#include "quickstore/cxx/quickstore.h"

#include <utility>
#include <vector>

#include "glog/logging.h"
#include "spec/cxx/storage.h"
#include "quickstore/cxx/internal/store.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace quickstore {

constexpr char kNoError[] = "";

Quickstore::Quickstore(const std::string& benchmark_key) {
  storage_ = nullptr;
  input_.set_benchmark_key(benchmark_key);
}

Quickstore::Quickstore(const std::string& benchmark_key, Storage* storage) {
  storage_ = storage;
  input_.set_benchmark_key(benchmark_key);
}

std::string Quickstore::AddSamplePoint(double input_value,
                                  const std::map<std::string, double>& data) {
  mako::SamplePoint s;
  s.set_input_value(input_value);
  for (const auto& pair : data) {
    mako::KeyedValue* k = s.add_metric_value_list();
    k->set_value_key(pair.first);
    k->set_value(pair.second);
  }
  points_.push_back(s);
  return kNoError;
}

std::string Quickstore::AddSamplePoint(const mako::SamplePoint& point) {
  points_.push_back(point);
  return kNoError;
}

std::string Quickstore::AddError(double input_value, const std::string& error_msg) {
  mako::SampleError e;
  e.set_input_value(input_value);
  e.set_error_message(error_msg);
  errors_.push_back(e);
  return kNoError;
}

std::string Quickstore::AddError(const mako::SampleError& error) {
  errors_.push_back(error);
  return kNoError;
}

std::string Quickstore::AddRunAggregate(const std::string& value_key, double value) {
  mako::KeyedValue k;
  k.set_value_key(value_key);
  k.set_value(value);
  run_aggregates_.push_back(k);
  return kNoError;
}

std::string Quickstore::AddMetricAggregate(const std::string& value_key,
                                      const std::string& aggregate_type,
                                      double value) {
  metric_aggregate_value_keys_.push_back(value_key);
  metric_aggregate_types_.push_back(aggregate_type);
  metric_aggregate_values_.push_back(value);
  return kNoError;
}

mako::quickstore::QuickstoreOutput Quickstore::Store() {
  LOG(INFO) << "Attempting to store:";
  LOG(INFO) << points_.size() << " SamplePoints";
  LOG(INFO) << errors_.size() << " SampleErrors";
  LOG(INFO) << run_aggregates_.size() << " Run Aggregates";
  LOG(INFO) << metric_aggregate_value_keys_.size() << " Metric Aggregates";

  // SWIG likes vectors, clear our internal data structures same time we are
  // converting to vectors.
  std::vector<mako::SamplePoint> points;
  points.reserve(points_.size());
  while (!points_.empty()) {
    points.push_back(points_.front());
    points_.pop_front();
  }
  std::vector<mako::SampleError> errors;
  errors.reserve(errors_.size());
  while (!errors_.empty()) {
    errors.push_back(errors_.front());
    errors_.pop_front();
  }
  std::vector<mako::KeyedValue> run_aggregates;
  run_aggregates.reserve(run_aggregates_.size());
  while (!run_aggregates_.empty()) {
    run_aggregates.push_back(run_aggregates_.front());
    run_aggregates_.pop_front();
  }
  std::vector<std::string> metric_aggregate_value_keys;
  metric_aggregate_value_keys.reserve(metric_aggregate_value_keys_.size());
  while (!metric_aggregate_value_keys_.empty()) {
    metric_aggregate_value_keys.push_back(metric_aggregate_value_keys_.front());
    metric_aggregate_value_keys_.pop_front();
  }
  std::vector<std::string> metric_aggregate_types;
  metric_aggregate_types.reserve(metric_aggregate_types_.size());
  while (!metric_aggregate_types_.empty()) {
    metric_aggregate_types.push_back(metric_aggregate_types_.front());
    metric_aggregate_types_.pop_front();
  }
  std::vector<double> metric_aggregate_values;
  metric_aggregate_values.reserve(metric_aggregate_values_.size());
  while (!metric_aggregate_values_.empty()) {
    metric_aggregate_values.push_back(metric_aggregate_values_.front());
    metric_aggregate_values_.pop_front();
  }
  if (storage_) {
    return mako::quickstore::internal::SaveWithStorage(
        storage_, input_, points, errors, run_aggregates,
        metric_aggregate_value_keys, metric_aggregate_types,
        metric_aggregate_values);
  } else {
    return mako::quickstore::internal::Save(
        input_, points, errors, run_aggregates, metric_aggregate_value_keys,
        metric_aggregate_types, metric_aggregate_values);
  }
}

}  // namespace quickstore
}  // namespace mako
