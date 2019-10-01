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
#include "clients/cxx/dashboard/standard_dashboard.h"

#include <cstddef>

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "absl/strings/match.h"
#include "absl/strings/str_cat.h"
#include "absl/types/optional.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace standard_dashboard {
namespace {

using ::testing::HasSubstr;
using ::testing::Not;

// Utility to check whether a field is present in the link. If:
// - expected_present
//   - !field_value.empty():
//     - Presence of &<field_name>=<field_value> is verified.
//   - Otherwise:
//     - Presence of &<field_name> is verified.
// - Otherwise:
//   - Absence of <field_name>.
void ExpectFieldPresentOrAbsent(const std::string& link, bool expected_present,
                                const std::string& field_name,
                                const std::string& field_value) {
  SCOPED_TRACE(::testing::Message()
               << absl::StrCat("field_name=", field_name,
                               ", expected_present=", expected_present));
  if (expected_present) {
    std::string expected_substr = absl::StrCat("&", field_name);
    if (!field_value.empty()) {
      absl::StrAppend(&expected_substr, "=", field_value);
    }
    EXPECT_THAT(link, HasSubstr(expected_substr));
  } else {
    EXPECT_THAT(link, Not(HasSubstr(field_name)));
  }
}

class Google3DashboardTest
    : public ::testing::TestWithParam<
          std::tuple<bool /* highlight_series  */,
                     absl::optional<std::string> /* hostname */>> {
 public:
  Google3DashboardTest() {
    std::tie(highlight_series_, hostname_) = GetParam();
  }

 protected:
  bool GetHighlightSeries() const { return highlight_series_; }
  // Regardless of whether an URL like "mako.dev" or "http://mako.dev" is
  // passed into the dashboard, we want to test the dashboard output against an
  // URL qualified with the scheme. GetHostname() is used for that purpose.
  // TODO(b/128004122): Use a URL type to make comparisons smarter and less
  // error prone.
  std::string GetHostname() const {
    if (hostname_.has_value()) {
      if (!absl::StartsWith(hostname_.value(), "http://") &&
          !absl::StartsWith(hostname_.value(), "https://")) {
        return absl::StrCat("https://", hostname_.value());
      }
      return hostname_.value();
    }
    return "https://mako.dev";
  }

  Dashboard GetDashboard() {
    if (hostname_.has_value()) {
      return Dashboard(hostname_.value());
    }
    return Dashboard();
  }

 private:
  bool highlight_series_;
  absl::optional<std::string> hostname_;
};

INSTANTIATE_TEST_SUITE_P(Base, Google3DashboardTest,
                         testing::Combine(testing::Bool(),
                                          testing::Values(absl::nullopt,
                                                          "https://mako.dev",
                                                          "mako.dev")));

TEST_P(Google3DashboardTest, AggregateChartTimestamp) {
  std::string link;
  DashboardAggregateChartInput input;
  input.set_benchmark_key("123");
  ASSERT_TRUE(GetDashboard().AggregateChart(input, &link).empty());
  ASSERT_EQ(link, absl::StrCat(GetHostname(), "/benchmark?benchmark_key=123"));
  auto countF = input.add_value_selections();
  countF->set_data_type(DataFilter::METRIC_AGGREGATE_COUNT);
  countF->set_value_key("y");
  auto minF = input.add_value_selections();
  minF->set_data_type(DataFilter::METRIC_AGGREGATE_MIN);
  minF->set_value_key("y");
  auto maxF = input.add_value_selections();
  maxF->set_data_type(DataFilter::METRIC_AGGREGATE_MAX);
  maxF->set_value_key("y");
  auto meanF = input.add_value_selections();
  meanF->set_data_type(DataFilter::METRIC_AGGREGATE_MEAN);
  meanF->set_value_key("y");
  auto medianF = input.add_value_selections();
  medianF->set_data_type(DataFilter::METRIC_AGGREGATE_MEDIAN);
  medianF->set_value_key("y");
  auto stddevF = input.add_value_selections();
  stddevF->set_data_type(DataFilter::METRIC_AGGREGATE_STDDEV);
  stddevF->set_value_key("y");
  auto madF = input.add_value_selections();
  madF->set_data_type(DataFilter::METRIC_AGGREGATE_MAD);
  madF->set_value_key("y");
  auto pctlF = input.add_value_selections();
  pctlF->set_data_type(DataFilter::METRIC_AGGREGATE_PERCENTILE);
  pctlF->set_value_key("y");
  pctlF->set_percentile_milli_rank(1000);
  auto custF = input.add_value_selections();
  custF->set_data_type(DataFilter::CUSTOM_AGGREGATE);
  custF->set_value_key("c");
  auto scoreF = input.add_value_selections();
  scoreF->set_data_type(DataFilter::BENCHMARK_SCORE);
  auto errF = input.add_value_selections();
  errF->set_data_type(DataFilter::ERROR_COUNT);
  input.add_tags("a=b");
  // Use values larger than std::numeric_limits<int>::max() to verify they're
  // handled properly.
  input.set_min_timestamp_ms(1099511627776);  // 2^40
  input.set_max_timestamp_ms(1099511627779);
  input.set_max_runs(7);
  input.set_highlight_series_on_hover(GetHighlightSeries());
  ASSERT_TRUE(GetDashboard().AggregateChart(input, &link).empty());
  EXPECT_THAT(link, HasSubstr(absl::StrCat(GetHostname(),
                                           "/benchmark"
                                           "?benchmark_key=123"
                                           "&~y=count"
                                           "&~y=min"
                                           "&~y=max"
                                           "&~y=mean"
                                           "&~y=median"
                                           "&~y=stddev"
                                           "&~y=mad"
                                           "&~y=p1000"
                                           "&~c=1"
                                           "&benchmark-score=1"
                                           "&error-count=1"
                                           "&tag=a%3Db"
                                           "&tmin=1099511627776"
                                           "&tmax=1099511627779"
                                           "&maxruns=7")));
  ExpectFieldPresentOrAbsent(link, GetHighlightSeries(), "hiser", "1");
}

TEST_P(Google3DashboardTest, AggregateChartBuildID) {
  std::string link;
  DashboardAggregateChartInput input;
  input.set_benchmark_key("123");
  ASSERT_TRUE(GetDashboard().AggregateChart(input, &link).empty());
  ASSERT_EQ(link, absl::StrCat(GetHostname(), "/benchmark?benchmark_key=123"));
  auto countF = input.add_value_selections();
  countF->set_data_type(DataFilter::METRIC_AGGREGATE_COUNT);
  countF->set_value_key("y");
  input.set_run_order(mako::RunOrder::BUILD_ID);
  input.set_min_build_id(200000000);
  input.set_max_build_id(200000100);
  input.set_max_runs(7);
  ASSERT_TRUE(GetDashboard().AggregateChart(input, &link).empty());
  EXPECT_THAT(link, HasSubstr(absl::StrCat(GetHostname(),
                                           "/benchmark"
                                           "?benchmark_key=123"
                                           "&~y=count"
                                           "&maxruns=7"
                                           "&order=bid"
                                           "&bidmin=200000000"
                                           "&bidmax=200000100")));
}

TEST_P(Google3DashboardTest, RunChart) {
  std::string link;
  DashboardRunChartInput input;
  input.set_run_key("123");
  ASSERT_TRUE(GetDashboard().RunChart(input, &link).empty());
  ASSERT_EQ(link, absl::StrCat(GetHostname(), "/run?run_key=123"));
  input.add_metric_keys("y");
  input.set_highlight_series_on_hover(GetHighlightSeries());
  ASSERT_TRUE(GetDashboard().RunChart(input, &link).empty());
  EXPECT_THAT(link,
              HasSubstr(absl::StrCat(GetHostname(), "/run?run_key=123&~y=1")));
  ExpectFieldPresentOrAbsent(link, GetHighlightSeries(), "hiser", "1");
}

TEST_P(Google3DashboardTest, CompareAggregateChart) {
  std::string link;
  DashboardCompareAggregateChartInput input;
  auto mSeries = input.add_series_list();
  mSeries->set_series_label("yL");
  mSeries->set_benchmark_key("123");
  auto mFilter = mSeries->mutable_value_selection();
  mFilter->set_data_type(DataFilter::METRIC_AGGREGATE_MEAN);
  mFilter->set_value_key("y");
  mSeries->add_tags("a=b");
  auto bSeries = input.add_series_list();
  bSeries->set_series_label("score");
  bSeries->set_benchmark_key("456");
  auto bFilter = bSeries->mutable_value_selection();
  bFilter->set_data_type(DataFilter::BENCHMARK_SCORE);
  auto eSeries = input.add_series_list();
  eSeries->set_series_label("err");
  eSeries->set_benchmark_key("456");
  auto eFilter = eSeries->mutable_value_selection();
  eFilter->set_data_type(DataFilter::ERROR_COUNT);
  // Use values larger than std::numeric_limits<int>::max() to verify they're
  // handled properly.
  input.set_min_timestamp_ms(1099511627776);  // 2^40
  input.set_max_timestamp_ms(1099511627779);
  input.set_highlight_series_on_hover(GetHighlightSeries());
  ASSERT_TRUE(GetDashboard().CompareAggregateChart(input, &link).empty());
  EXPECT_THAT(link, HasSubstr(absl::StrCat(GetHostname(),
                                           "/cmpagg"
                                           "?0n=yL"
                                           "&0b=123"
                                           "&0~y=mean"
                                           "&0t=a%3Db"
                                           "&1n=score"
                                           "&1b=456"
                                           "&1s=1"
                                           "&2n=err"
                                           "&2b=456"
                                           "&2e=1"
                                           "&tmin=1099511627776"
                                           "&tmax=1099511627779")));
  ExpectFieldPresentOrAbsent(link, GetHighlightSeries(), "hiser", "1");
}

TEST_P(Google3DashboardTest, CompareRunChart) {
  std::string link;
  DashboardCompareRunChartInput input;
  input.add_run_keys("123");
  input.add_metric_keys("y");
  input.set_highlight_series_on_hover(GetHighlightSeries());
  ASSERT_TRUE(GetDashboard().CompareRunChart(input, &link).empty());
  EXPECT_THAT(link,
              HasSubstr(absl::StrCat(GetHostname(), "/cmprun?0r=123&~y=1")));
  ExpectFieldPresentOrAbsent(link, GetHighlightSeries(), "hiser", "1");
}

TEST_P(Google3DashboardTest, VisualizeAnalysis) {
  std::string link;
  mako::DashboardVisualizeAnalysisInput input;
  input.set_run_key("test_run_key");
  ASSERT_TRUE(GetDashboard().VisualizeAnalysis(input, &link).empty());
  EXPECT_THAT(
      link, HasSubstr(absl::StrCat(GetHostname(),
                                   "/analysis-results?run_key=test_run_key")));
}

TEST_P(Google3DashboardTest, VisualizeAnalysisWithAnalysisKey) {
  std::string link;
  mako::DashboardVisualizeAnalysisInput input;;
  input.set_run_key("test_run_key");
  input.set_analysis_key("test_analysis_key");
  ASSERT_TRUE(GetDashboard().VisualizeAnalysis(input, &link).empty());
  EXPECT_THAT(
      link,
      HasSubstr(absl::StrCat(
          GetHostname(),
          "/analysis-results?run_key=test_run_key#analysistest_analysis_key")));
}

}  // namespace
}  // namespace standard_dashboard
}  // namespace mako
