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

#ifndef HELPERS_CXX_QUICKSTORE_QUICKSTORE_H_
#define HELPERS_CXX_QUICKSTORE_QUICKSTORE_H_

#include <list>
#include <map>
#include <string>

#include "spec/cxx/storage.h"
#include "proto/quickstore/quickstore.pb.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace quickstore {

// Quickstore offers a way to utilize Mako storage, downsampling,
// aggregation and analyzers in a simple way. This is most helpful when you have
// a pre-existing benchmarking tool which exports data that you would like to
// save in Mako.
//
// Basic usage:
//  // Store data, metric and run aggregates will be calculated automatically.
//  Quickstore q(kBenchmarkKey);
//  for (int data : {1, 2, 3, 4, 5, 6}) {
//    q.AddSamplePoint(time_ms, {{"y", data}});
//  }
//  QuickstoreOutput output = q.Store();
//  if(!IsOK(output)) {
//    LOG(ERROR) << "Failed to store: " << output.summary_output();
//  }
//
// More information about quickstore: go/mako-quickstore
//
// Class is not thread-safe per go/thread-safe.

// Convenience function checking Store() was successful.
inline bool IsOK(
    const mako::quickstore::QuickstoreOutput& output) {
  return output.status() ==
         mako::quickstore::QuickstoreOutput::SUCCESS;
}

class Quickstore {
 public:
  // Will create a new Mako Run under this benchmark key in the default
  // Mako storage system.
  explicit Quickstore(const std::string& benchmark_key);

  // Will create a new Mako Run under this benchmark key in the storage
  // provided.
  // It does not take ownership of the Storage client object.
  Quickstore(const std::string& benchmark_key, Storage* storage);

  // Provide extra metadata about the Run (eg. such as a description).
  // See QuickstoreInput for more information.
  explicit Quickstore(
      const mako::quickstore::QuickstoreInput& input)
      : input_(input), storage_(nullptr) {}

  // Provide extra metadata about the Run (eg. such as a description).
  // See QuickstoreInput for more information.
  // Takes a pointer to a mako::Storage client implementation.
  // It does not take ownership of the Storage client object.
  Quickstore(const mako::quickstore::QuickstoreInput& input,
             Storage* storage)
      : input_(input), storage_(storage) {}

  virtual ~Quickstore() {}

  // Add a sample at the specified xval.
  //
  // The map represents a mapping from metric to value.
  // It is more efficient for Mako to store multiple metrics collected at
  // the same xval together, but it is optional.
  //
  // When adding data via this function, calling the Add*Aggregate() functions
  // is optional, as the aggregates will get computed by this class.
  //
  // A std::string is returned with an error if the operation was unsucessful.
  virtual std::string AddSamplePoint(double xval,
                                const std::map<std::string, double>& yvals);
  virtual std::string AddSamplePoint(const mako::SamplePoint& point);

  // Add an error at the specified xval.
  //
  // When adding errors via this function, the aggregate error count will be set
  // automatically.
  //
  // A std::string is returned with an error if the operation was unsucessful.
  virtual std::string AddError(double xval, const std::string& error_msg);
  virtual std::string AddError(const mako::SampleError& error);

  // Add an aggregate value over the entire run.
  // If value_key is:
  //  * "~ignore_sample_count"
  //  * "~usable_sample_count"
  //  * "~error_sample_count"
  //  * "~benchmark_score"
  //  The corresponding value will be overwritten by the Mako aggregator. If
  //  none of these values are provided, they will be calculated automatically
  //  by the framework based on SamplePoints/Errors provided before Store() is
  //  called.
  //
  // Otherwise the value_key will be set to a custom aggregate (see
  // RunAggregate.custom_aggregate_list in mako.proto).
  //
  // If no run aggregates are manully set with this method, values are
  // automatically calculated.
  //
  // A std::string is returned with an error if the operation was unsucessful.
  virtual std::string AddRunAggregate(const std::string& value_key, double value);

  // Add an aggregate for a specific metric.
  // If value_key is:
  //  * "min"
  //  * "max"
  //  * "mean"
  //  * "median"
  //  * "standard_deviation"
  //  * "median_absolute_deviation"
  //  * "count"
  //  The corresponding value inside the MetricAggregate (defined in
  //  mako.proto) will be set.
  //
  // The value_key can also represent a percentile in
  // MetricAggregate.percentile_list (defined in mako.proto).
  //
  // For example "p98000" would be interpreted as the 98th percentile. These
  // need to correspond to the percentiles that your benchmark has set.
  // It is an error to supply an percentile that is not part of your benchmark.
  // If any percentiles are provided, the automatically calculated percentiles
  // will be cleared to 0.
  //
  // If any aggregate_types (eg. "min") are set for a value_key it will
  // overwrite the entire MetricAggregate for that value_key. If no
  // aggregate_types are provided for a value_key metric aggregates (including
  // percentiles) will be calculated automatically based on data provided via
  // calls to AddSamplePoint.
  //
  // A std::string is returned with an error if the operation was unsucessful.
  virtual std::string AddMetricAggregate(const std::string& value_key,
                                    const std::string& aggregate_type, double value);

  // Store all the values that you have added. You cannot save if no Add*()
  // functions have been called.
  //
  // Each call to Store() will create a new unique Mako Run and store all
  // Aggregate and SamplePoint data registered using the Add* methods since the
  // last call to Store() as a part of that new Run.
  //
  // Data can be added via Add* calls in any order.
  //
  // After a call to Store() all data added via Add* will be cleared.
  virtual mako::quickstore::QuickstoreOutput Store();

 private:
  mako::quickstore::QuickstoreInput input_;
  Storage* storage_;
  std::list<mako::SamplePoint> points_;
  std::list<mako::SampleError> errors_;
  std::list<mako::KeyedValue> run_aggregates_;
  std::list<std::string> metric_aggregate_value_keys_;
  std::list<std::string> metric_aggregate_types_;
  std::list<double> metric_aggregate_values_;
};

}  // namespace quickstore
}  // namespace mako

#endif  // HELPERS_CXX_QUICKSTORE_QUICKSTORE_H_
