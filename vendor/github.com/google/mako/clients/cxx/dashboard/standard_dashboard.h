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
#ifndef CLIENTS_CXX_DASHBOARD_STANDARD_DASHBOARD_H_
#define CLIENTS_CXX_DASHBOARD_STANDARD_DASHBOARD_H_

#include <string>

#include "spec/cxx/dashboard.h"
#include "spec/proto/mako.pb.h"
#include "absl/strings/string_view.h"

namespace mako {
namespace standard_dashboard {

// Dashboard implementation. See interface for docs.
class Dashboard : public mako::Dashboard {
 public:
  Dashboard();
  // Provided hostname must not have a trailing slash.
  // TODO(b/128004122): Use a URL type to make comparisons smarter and less
  // error prone.
  explicit Dashboard(absl::string_view hostname);

  std::string AggregateChart(
      const mako::DashboardAggregateChartInput& input,
      std::string* link) const override;

  std::string RunChart(
      const mako::DashboardRunChartInput& input,
      std::string* link) const override;

  std::string CompareAggregateChart(
      const mako::DashboardCompareAggregateChartInput& input,
      std::string* link) const override;

  std::string CompareRunChart(
      const mako::DashboardCompareRunChartInput& input,
      std::string* link) const override;

  std::string VisualizeAnalysis(
      const mako::DashboardVisualizeAnalysisInput& input,
      std::string* link) const override;

 private:
  std::string hostname_;
};

}  // namespace standard_dashboard
}  // namespace mako

#endif  // CLIENTS_CXX_DASHBOARD_STANDARD_DASHBOARD_H_
