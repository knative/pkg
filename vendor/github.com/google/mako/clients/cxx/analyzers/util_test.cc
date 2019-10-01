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
#include "clients/cxx/analyzers/util.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"

#include "spec/proto/mako.pb.h"

namespace {

using mako::BenchmarkInfo;
using mako::DataFilter;

class UtilTest : public ::testing::Test {
 protected:
  UtilTest() {
    // Add metric aggregates
    auto metric_1_aggregate = benchmark_info_.add_metric_info_list();
    metric_1_aggregate->set_value_key("m1");
    metric_1_aggregate->set_label("MetricOne");

    auto metric_2_aggregate = benchmark_info_.add_metric_info_list();
    metric_2_aggregate->set_value_key("m2");
    metric_2_aggregate->set_label("MetricTwo");

    auto custom_1_aggregate =
        benchmark_info_.add_custom_aggregation_info_list();
    custom_1_aggregate->set_value_key("cagg1");
    custom_1_aggregate->set_label("CustomAggregateOne");
  }

  BenchmarkInfo benchmark_info_;
};

TEST_F(UtilTest, MissingKeyReturnsKey) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
  data_filter.set_value_key("key_not_present_in_benchmark");

  EXPECT_EQ("key_not_present_in_benchmark",
            ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
                data_filter, benchmark_info_));
}

TEST_F(UtilTest, EmptyKeyReturnsUnknown) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_METRIC_SAMPLEPOINTS);
  data_filter.set_value_key("");

  EXPECT_EQ("unknown",
            ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
                data_filter, benchmark_info_));
}

TEST_F(UtilTest, BenchmarkScore) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_BENCHMARK_SCORE);

  EXPECT_EQ("benchmark_score",
            ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
                data_filter, benchmark_info_));
}

TEST_F(UtilTest, CustomAggregate) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_CUSTOM_AGGREGATE);
  data_filter.set_value_key("cagg1");

  EXPECT_EQ("CustomAggregateOne\\customagg",
            ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
                data_filter, benchmark_info_));
}

TEST_F(UtilTest, ErrorCount) {
  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter_DataType_ERROR_COUNT);

  EXPECT_EQ("error_count",
            ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
                data_filter, benchmark_info_));
}

TEST_F(UtilTest, Metric) {
  DataFilter data_filter;
  data_filter.set_data_type(
      mako::DataFilter_DataType_METRIC_AGGREGATE_MEDIAN);
  data_filter.set_value_key("m1");

  EXPECT_EQ("MetricOne\\median",
            ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
                data_filter, benchmark_info_));
}

TEST_F(UtilTest, Percentiles) {
  DataFilter data_filter;
  data_filter.set_data_type(
      mako::DataFilter_DataType_METRIC_AGGREGATE_PERCENTILE);
  data_filter.set_value_key("m1");
  data_filter.set_percentile_milli_rank(99999);

  EXPECT_EQ("MetricOne\\p99.999",
            ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
                data_filter, benchmark_info_));

  data_filter.set_percentile_milli_rank(1);

  EXPECT_EQ("MetricOne\\p0.001",
            ::mako::analyzer_util::GetHumanFriendlyDataFilterString(
                data_filter, benchmark_info_));
}


}  // namespace
