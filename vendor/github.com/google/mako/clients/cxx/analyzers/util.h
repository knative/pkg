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
#ifndef CLIENTS_CXX_ANALYZERS_UTIL_H_
#define CLIENTS_CXX_ANALYZERS_UTIL_H_

#include <string>

#include "spec/proto/mako.pb.h"

namespace mako {
namespace analyzer_util {

// Find the most appropriate human readable label for the DataFilter we are
// analyzing for the provided Benchmark. If we can't find an appropriate label,
// this function will return "unknown". This can happen in situations where
// the value_key provided in the data_filter is incorrect or has been removed
// from the BenchmarkInfo.
std::string GetHumanFriendlyDataFilterString(
    const mako::DataFilter& data_filter,
    const mako::BenchmarkInfo& bench_info);

}  // namespace analyzer_util
}  // namespace mako

#endif  // CLIENTS_CXX_ANALYZERS_UTIL_H_
