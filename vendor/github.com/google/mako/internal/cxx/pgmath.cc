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
#include "internal/cxx/pgmath.h"

#include <algorithm>
#include <cmath>
#include <cstdlib>
#include <random>
#include <sstream>
#include <string>
#include <vector>

namespace mako {
namespace internal {

Random::Random() {
  // std::random_device is a non-deterministic uniform random number generator,
  // although implementations are allowed to implement std::random_device using
  // a pseudo-random number engine if there is no support for non-deterministic
  // random number generation.
  std::random_device rd;
  // Seed the engine.
  engine_.seed(rd());
}

int Random::ProduceInt(int a, int b) {
  // produces integer values evenly distributed across a range
  std::uniform_int_distribution<int> uniform_dist(a, b);
  return uniform_dist(engine_);
}

RunningStats::RunningStats(const RunningStats::Config& config)
    : n_(0),
      config_(config),
      sorted_(false),
      min_(0.0),
      max_(0.0),
      sum_(0.0),
      old_m_(0.0),
      new_m_(0.0),
      old_s_(0.0),
      new_s_(0.0) {}

std::string RunningStats::Add(double x) {
  // Increment value count
  ++n_;

  sum_ += x;

  // Min/max and Cook/Knuth algo vars
  if (n_ == 1) {
    min_ = x;
    max_ = x;
    old_m_ = new_m_ = x;
    old_s_ = 0.0;
  } else {
    min_ = std::min(min_, x);
    max_ = std::max(max_, x);
    new_m_ = old_m_ + (x - old_m_) / static_cast<double>(n_);
    new_s_ = old_s_ + (x - old_m_) * (x - new_m_);
    old_m_ = new_m_;
    old_s_ = new_s_;
  }

  // Add to sample
  if (config_.max_sample_size < 0) {
    sample_.push_back(x);
  } else if (config_.max_sample_size > 0) {
    // Use reservoir sampling:
    // https://en.wikipedia.org/wiki/Reservoir_sampling
    if (sample_.size() < static_cast<std::size_t>(config_.max_sample_size)) {
      sample_.push_back(x);
    } else {
      // Randomly replace elements in the reservoir with a decreasing
      // probability. Choose an integer between 0 and index (inclusive)
      // of added value.
      if (!config_.random) {
        return "RunningStats was not supplied a Random instance.""";
      }
      int r = config_.random->ProduceInt(0, n_ - 1);
      if (r < config_.max_sample_size) {
        sample_[r] = x;
      }
    }
  }

  // Flag as not sorted, since we may have added/inserted a value
  sorted_ = false;
  return "";
}

std::string RunningStats::AddVector(const std::vector<double>& values) {
  std::string err;
  for (double x : values) {
    err = Add(x);
    if (!err.empty()) {
      return err;
    }
  }
  return "";
}

RunningStats::Result RunningStats::Count() const {
  Result r;
  r.error = CheckCount();
  r.value = n_;
  return r;
}

RunningStats::Result RunningStats::Sum() const {
  Result r;
  r.error = CheckCount();
  r.value = sum_;
  return r;
}

RunningStats::Result RunningStats::Min() const {
  Result r;
  r.error = CheckCount();
  r.value = min_;
  return r;
}

RunningStats::Result RunningStats::Max() const {
  Result r;
  r.error = CheckCount();
  r.value = max_;
  return r;
}

RunningStats::Result RunningStats::Mean() const {
  Result r;
  r.error = CheckCount();
  r.value = new_m_;
  return r;
}

RunningStats::Result RunningStats::Median() {
  return Percentile(0.5);
}

RunningStats::Result RunningStats::Variance() const {
  Result r;
  r.error = CheckCount();
  if (n_ <= 1) {
    // For perf data, a single value has 0 variance
    r.value = 0.0;
  } else {
    r.value = new_s_ / static_cast<double>(n_);
  }
  return r;
}

RunningStats::Result RunningStats::Stddev() const {
  Result r = Variance();
  r.value = std::sqrt(r.value);
  return r;
}

RunningStats::Result RunningStats::Mad() {
  Result median = Median();
  if (!median.error.empty()) {
    return median;
  }
  RunningStats residuals = RunningStats(config_);
  for (double x : sample_) {
    residuals.Add(std::abs(x - median.value));
  }
  return residuals.Median();
}

RunningStats::Result RunningStats::Percentile(double pct) {
  // preconditions
  Result r;
  r.error = CheckCount();
  if (!r.error.empty()) {
    return r;
  }
  r.error = CheckSample();
  if (!r.error.empty()) {
    return r;
  }
  if (pct < 0.0 || pct > 1.0) {
    std::stringstream ss;
    ss << "Bad pct arg: " << pct;
    r.error = ss.str();
    return r;
  }
  // simple case for size 1
  if (sample_.size() == 1) {
    r.value = sample_[0];
    return r;
  }
  // float index that has this value and floor/ceiling of it
  SortSample();
  double k = static_cast<double>(sample_.size() - 1) * pct;
  double f = std::floor(k);
  double c = std::ceil(k);
  if (f == c) {
    r.value = sample_[static_cast<int>(k)];
    return r;
  }
  // interpolate
  double d0 = sample_[static_cast<int>(f)] * (c - k);
  double d1 = sample_[static_cast<int>(c)] * (k - f);
  r.value = d0 + d1;
  return r;
}

void RunningStats::SortSample() {
  if (!sorted_) {
    sorted_ = true;
    std::sort(sample_.begin(), sample_.end());
  }
}

std::string RunningStats::CheckCount() const {
  if (n_ <= 0) {
    return "No data added";
  }
  return "";
}

std::string RunningStats::CheckSample() const {
  if (sample_.empty()) {
    return "No sample data maintained";
  }
  return "";
}

}  // namespace internal
}  // namespace mako
