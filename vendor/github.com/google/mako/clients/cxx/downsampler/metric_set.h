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
#ifndef CLIENTS_CXX_DOWNSAMPLER_METRIC_SET_H_
#define CLIENTS_CXX_DOWNSAMPLER_METRIC_SET_H_

#include <stddef.h>

#include <forward_list>
#include <string>

#include "absl/strings/str_cat.h"
#include "absl/strings/str_join.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace downsampler {

// Metric keys for a sample point sorted  by name so that a point with metrics
// m1 & m2 is the same as a point with metrics m2 & m1.
inline std::string GetKey(mako::SamplePoint* point) {
  // Microbenchmarks showed std::forward_list to be most performant here.
  // If we increase maximum number of metrics allowed we will want to revisit
  // this.
  std::forward_list<std::string> keys;
  for (const auto& kv : point->metric_value_list()) {
    keys.push_front(kv.value_key());
  }
  keys.sort();
  return absl::StrJoin(keys, ",");
}

struct MetricSet {
  explicit MetricSet(mako::SamplePoint* point)
      : slot_count(point->metric_value_list_size()), key(GetKey(point)) {}
  explicit MetricSet(mako::SampleError* error)
      : slot_count(1), key(error->sampler_name()) {}

  std::string ToString() const {
    return absl::StrCat("MetricSet{key=", key, ",slot_count=", slot_count, "}");
  }

  const int slot_count;
  const std::string key;
};

inline bool operator==(const MetricSet& lhs, const MetricSet& rhs) {
  return lhs.slot_count == rhs.slot_count && lhs.key == rhs.key;
}

inline bool operator!=(const MetricSet& lhs, const MetricSet& rhs) {
  return !(lhs == rhs);
}

struct HashMetricSet {
  size_t operator()(const MetricSet& metric_set) const {
    return std::hash<std::string>()(metric_set.key);
  }
};

}  // namespace downsampler
}  // namespace mako
#endif  // CLIENTS_CXX_DOWNSAMPLER_METRIC_SET_H_
