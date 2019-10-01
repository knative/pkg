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
#ifndef CLIENTS_CXX_AGGREGATOR_THREADSAFE_RUNNING_STATS_H_
#define CLIENTS_CXX_AGGREGATOR_THREADSAFE_RUNNING_STATS_H_

#include <memory>
#include <string>
#include <vector>

#include "absl/synchronization/mutex.h"
#include "internal/cxx/pgmath.h"

namespace mako {
namespace aggregator {

class ThreadsafeRunningStats {
 public:
  explicit ThreadsafeRunningStats(const int max_sample_size):
    // We only need a Random instance when max_sample_size > 0. They're somewhat
    // heavyweight, so if max_sample_size <= 0, don't bother creating it.
    random_(max_sample_size > 0 ? new mako::internal::Random : nullptr),
    rs_(mako::internal::RunningStats::Config{
      max_sample_size, random_.get()}) {}

  std::string AddVector(const std::vector<double>& values) {
    absl::MutexLock l(&mutex_);
    return rs_.AddVector(values);
  }

  mako::internal::RunningStats::Result Count() const {
    return rs_.Count();
  }

  mako::internal::RunningStats::Result Min() const {
    return rs_.Min();
  }

  mako::internal::RunningStats::Result Max() const {
    return rs_.Max();
  }

  mako::internal::RunningStats::Result Mean() const {
    return rs_.Mean();
  }

  mako::internal::RunningStats::Result Median() {
    absl::MutexLock l(&mutex_);
    return rs_.Median();
  }

  mako::internal::RunningStats::Result Stddev() const {
    return rs_.Stddev();
  }

  mako::internal::RunningStats::Result Mad() {
    absl::MutexLock l(&mutex_);
    return rs_.Mad();
  }

  mako::internal::RunningStats::Result Percentile(double pct) {
    absl::MutexLock l(&mutex_);
    return rs_.Percentile(pct);
  }

 private:
  // Used to synchronize calls that modify rs_. The const methods are safe to
  // call concurrently, so no synchronization is needed there.
  absl::Mutex mutex_;
  std::unique_ptr<mako::internal::Random> random_;
  mako::internal::RunningStats rs_;
};
}  // namespace aggregator
}  // namespace mako
#endif  // CLIENTS_CXX_AGGREGATOR_THREADSAFE_RUNNING_STATS_H_
