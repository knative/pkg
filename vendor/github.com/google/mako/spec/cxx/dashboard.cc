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
#include "spec/cxx/dashboard.h"

namespace mako {

namespace {
constexpr char kNeedsImplementedStr[] =
    "This dashboard method is not supported by this implementation.";
}  // namespace

std::string Dashboard::AggregateChart(
    const mako::DashboardAggregateChartInput& input, std::string* link) const {
  return kNeedsImplementedStr;
}
std::string Dashboard::RunChart(const mako::DashboardRunChartInput& input,
                           std::string* link) const {
  return kNeedsImplementedStr;
}
std::string Dashboard::CompareAggregateChart(
    const mako::DashboardCompareAggregateChartInput& input,
    std::string* link) const {
  return kNeedsImplementedStr;
}
std::string Dashboard::CompareRunChart(
    const mako::DashboardCompareRunChartInput& input, std::string* link) const {
  return kNeedsImplementedStr;
}
std::string Dashboard::VisualizeAnalysis(
    const mako ::DashboardVisualizeAnalysisInput& input,
    std::string* link) const {
  return kNeedsImplementedStr;
}

}  // namespace mako
