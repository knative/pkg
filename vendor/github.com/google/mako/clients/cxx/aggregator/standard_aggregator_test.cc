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
#include "clients/cxx/aggregator/standard_aggregator.h"

#include <set>
#include <string>
#include <type_traits>
#include <utility>
#include <vector>

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/cxx/fileio/memory_fileio.h"
#include "absl/memory/memory.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_format.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace aggregator {
namespace {

using mako::AggregatorInput;
using mako::AggregatorOutput;
using mako::MetricAggregate;
using ::testing::ContainsRegex;

SampleRecord HelperCreateSampleRecord(
    double input_value, const std::vector<std::pair<std::string, double>>& metrics) {
  SampleRecord sr;
  SamplePoint* sp = sr.mutable_sample_point();
  sp->set_input_value(input_value);
  for (auto pair : metrics) {
    KeyedValue* kv = sp->add_metric_value_list();
    kv->set_value_key(pair.first);
    kv->set_value(pair.second);
  }
  return sr;
}

AggregatorInput HelperCreateAggregatorInput(const std::vector<std::string>& files) {
  AggregatorInput agg_input;

  for (const auto& file : files) {
    mako::SampleFile* sample_file = agg_input.add_sample_file_list();
    sample_file->set_file_path(file);
    sample_file->set_sampler_name(absl::StrCat("Sampler", file));
  }

  // Create RunInfo
  agg_input.mutable_run_info()->set_benchmark_key("benchmark_key");
  agg_input.mutable_run_info()->set_run_key("run_key");
  agg_input.mutable_run_info()->set_timestamp_ms(123456);

  // Each element of vector is an ignore range the first element of pair the
  // start of range, second element is end of range.
  static const auto& kIgnoreRanges =
      *new std::vector<std::pair<double, double>>{std::make_pair(10, 20),
                                                  std::make_pair(30, 40)};

  // Add ignore ranges to RunInfo
  int i = 0;
  for (const auto& ignore_range : kIgnoreRanges) {
    mako::LabeledRange* r =
        agg_input.mutable_run_info()->mutable_ignore_range_list()->Add();
    r->set_label(std::to_string(i++));
    r->mutable_range()->set_start(ignore_range.first);
    r->mutable_range()->set_end(ignore_range.second);
  }

  // Create BenchmarkInfo
  agg_input.mutable_benchmark_info()->set_benchmark_key("benchmark_key");
  agg_input.mutable_benchmark_info()->set_benchmark_name("bname");
  agg_input.mutable_benchmark_info()->set_project_name("project");
  *agg_input.mutable_benchmark_info()->add_owner_list() = "owner";
  agg_input.mutable_benchmark_info()->mutable_input_value_info()->set_value_key(
      "k");
  agg_input.mutable_benchmark_info()->mutable_input_value_info()->set_label(
      "klabel");

  return agg_input;
}

void WriteFile(const std::string& file_path,
               const std::vector<mako::SampleRecord>& data) {
  mako::memory_fileio::FileIO fileio;

  ASSERT_TRUE(fileio.Open(file_path, mako::FileIO::AccessMode::kWrite));
  for (const auto& d : data) {
    ASSERT_TRUE(fileio.Write(d));
  }
  ASSERT_TRUE(fileio.Close());
}

class StandardAggregatorTest : public ::testing::Test {
 protected:
  StandardAggregatorTest() {}

  ~StandardAggregatorTest() override {}

  void SetUp() override {
    a_.SetFileIO(std::unique_ptr<mako::FileIO>(
        new mako::memory_fileio::FileIO()));
  }

  void TearDown() override {}

  Aggregator a_;
};

TEST_F(StandardAggregatorTest, SanityChecks) {
  AggregatorOutput out;
  WriteFile("file1", {HelperCreateSampleRecord(
                         1, {std::make_pair("y", 100), std::make_pair("y", 50),
                             std::make_pair("y", 0)})});

  ASSERT_EQ(a_.Aggregate(HelperCreateAggregatorInput({"file1"}), &out), "");

  ASSERT_TRUE(out.has_aggregate());
  ASSERT_TRUE(out.aggregate().has_run_aggregate());
  ASSERT_EQ(out.aggregate().run_aggregate().usable_sample_count(), 1);
  ASSERT_TRUE(out.aggregate().run_aggregate().has_error_sample_count());
  ASSERT_TRUE(out.aggregate().run_aggregate().has_ignore_sample_count());

  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 1);
  const mako::MetricAggregate& ma =
      out.aggregate().metric_aggregate_list(0);
  ASSERT_EQ(ma.metric_key(), "y");
  ASSERT_EQ(ma.min(), 0);
  ASSERT_EQ(ma.max(), 100);
  ASSERT_EQ(ma.mean(), 50);
  ASSERT_EQ(ma.median(), 50);
  ASSERT_GT(ma.standard_deviation(), 0);
  ASSERT_GT(ma.percentile_list_size(), 0);
  ASSERT_EQ(ma.count(), 3);
  ASSERT_EQ(ma.median_absolute_deviation(), 50);
}

TEST_F(StandardAggregatorTest, MultipleFilesMultipleMetrics) {
  AggregatorOutput out;

  WriteFile("file1", {HelperCreateSampleRecord(
                         1, {std::make_pair("y", 100), std::make_pair("y", 50),
                             std::make_pair("y", 0)})});

  WriteFile("file2", {HelperCreateSampleRecord(
                         2, {std::make_pair("x", 100), std::make_pair("x", 50),
                             std::make_pair("x", 0)})});

  ASSERT_EQ(a_.Aggregate(HelperCreateAggregatorInput({"file1", "file2"}), &out),
            "");

  ASSERT_EQ(out.aggregate().run_aggregate().usable_sample_count(), 2);
  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 2);
  for (const mako::MetricAggregate& ma :
       out.aggregate().metric_aggregate_list()) {
    ASSERT_TRUE(ma.metric_key() == "y" || ma.metric_key() == "x");
    ASSERT_EQ(ma.min(), 0);
    ASSERT_EQ(ma.max(), 100);
    ASSERT_EQ(ma.mean(), 50);
    ASSERT_EQ(ma.median(), 50);
    ASSERT_GT(ma.standard_deviation(), 0);
    ASSERT_GT(ma.percentile_list_size(), 0);
    ASSERT_EQ(ma.count(), 3);
    ASSERT_EQ(ma.median_absolute_deviation(), 50);
  }
}

TEST_F(StandardAggregatorTest, AggregateWithoutFileIO) {
  Aggregator a;
  AggregatorOutput out;
  ASSERT_NE(a.Aggregate(HelperCreateAggregatorInput({}), &out), "");
}

TEST_F(StandardAggregatorTest, EmptyFile) {
  AggregatorOutput out;
  WriteFile("file1", {});

  ASSERT_EQ(a_.Aggregate(HelperCreateAggregatorInput({"file1"}), &out), "");

  ASSERT_TRUE(out.aggregate().has_run_aggregate());
  ASSERT_TRUE(out.aggregate().run_aggregate().has_usable_sample_count());
  ASSERT_TRUE(out.aggregate().run_aggregate().has_error_sample_count());
  ASSERT_TRUE(out.aggregate().run_aggregate().has_ignore_sample_count());
  ASSERT_EQ(out.aggregate().run_aggregate().usable_sample_count(), 0);
  ASSERT_EQ(out.aggregate().run_aggregate().error_sample_count(), 0);
  ASSERT_EQ(out.aggregate().run_aggregate().ignore_sample_count(), 0);
  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 0);
}

TEST_F(StandardAggregatorTest, BadPercentile) {
  AggregatorOutput out;

  WriteFile("file1", {
                         HelperCreateSampleRecord(1, {std::make_pair("y", 1)}),
                     });

  mako::AggregatorInput ai = HelperCreateAggregatorInput({"file1"});

  ai.mutable_benchmark_info()->clear_percentile_milli_rank_list();
  ai.mutable_benchmark_info()->add_percentile_milli_rank_list(101 * 1000.0);
  ASSERT_NE(a_.Aggregate(ai, &out), "");
}

TEST_F(StandardAggregatorTest, SampleError) {
  AggregatorOutput out;

  SampleRecord sr = HelperCreateSampleRecord(1, {std::make_pair("y", 1)});
  sr.mutable_sample_error()->set_input_value(1);
  sr.mutable_sample_error()->set_error_message("Some Error");
  sr.mutable_sample_error()->set_sampler_name("Sample1");

  WriteFile("file1", {sr});

  ASSERT_EQ(a_.Aggregate(HelperCreateAggregatorInput({"file1"}), &out), "");
  ASSERT_EQ(out.aggregate().run_aggregate().error_sample_count(), 1);
  ASSERT_EQ(out.aggregate().run_aggregate().usable_sample_count(), 1);
}

TEST_F(StandardAggregatorTest, Percentile) {
  AggregatorOutput out;

  // By having an odd number of samples we find exact index matches when
  // processing the median and need to interpolate when finding the other
  // percentiles.
  WriteFile("file1", {
                         HelperCreateSampleRecord(1, {std::make_pair("y", 1)}),
                         HelperCreateSampleRecord(2, {std::make_pair("y", 2)}),
                         HelperCreateSampleRecord(3, {std::make_pair("y", 3)}),
                         HelperCreateSampleRecord(4, {std::make_pair("y", 4)}),
                         HelperCreateSampleRecord(5, {std::make_pair("y", 5)}),
                         HelperCreateSampleRecord(6, {std::make_pair("y", 6)}),
                         HelperCreateSampleRecord(7, {std::make_pair("y", 7)}),
                         HelperCreateSampleRecord(8, {std::make_pair("y", 8)}),
                         HelperCreateSampleRecord(9, {std::make_pair("y", 9)}),
                     });

  mako::AggregatorInput ai = HelperCreateAggregatorInput({"file1"});

  ai.mutable_benchmark_info()->clear_percentile_milli_rank_list();
  for (int percent : {10, 20, 30, 40, 50, 60, 70, 80, 90}) {
    ai.mutable_benchmark_info()->add_percentile_milli_rank_list(percent *
                                                                1000.0);
  }

  ASSERT_EQ(a_.Aggregate(ai, &out), "");

  ASSERT_EQ(out.aggregate().run_aggregate().usable_sample_count(), 9);
  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 1);
  const mako::MetricAggregate& ma =
      out.aggregate().metric_aggregate_list(0);
  for (int i = 10; i < ma.percentile_list_size(); i++) {
    ASSERT_EQ(i * 10, ma.percentile_list(i)) << ma.DebugString();
  }
}

TEST_F(StandardAggregatorTest, IgnoreRanges) {
  AggregatorOutput out;

  // Points between 10,20 and 30,40 are in kignore_range
  WriteFile("file1",
            {
                // honored
                HelperCreateSampleRecord(1, {std::make_pair("y", 0)}),
                // ignored
                HelperCreateSampleRecord(10, {std::make_pair("y", 50)}),
                // ignored
                HelperCreateSampleRecord(15, {std::make_pair("y", 50)}),
                // ignored
                HelperCreateSampleRecord(20, {std::make_pair("y", 50)}),
                // honored
                HelperCreateSampleRecord(45, {std::make_pair("y", 100)}),
                // honored
                HelperCreateSampleRecord(55, {std::make_pair("y", 200)}),
            });

  ASSERT_EQ(a_.Aggregate(HelperCreateAggregatorInput({"file1"}), &out), "");

  ASSERT_EQ(out.aggregate().run_aggregate().usable_sample_count(), 3);
  ASSERT_EQ(out.aggregate().run_aggregate().ignore_sample_count(), 3);

  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 1);
  const mako::MetricAggregate& ma =
      out.aggregate().metric_aggregate_list(0);

  // Metrics should only account for 0 and 100.
  ASSERT_EQ(ma.metric_key(), "y");
  ASSERT_EQ(ma.min(), 0);
  ASSERT_EQ(ma.max(), 200);
  ASSERT_EQ(ma.mean(), 100);
  ASSERT_EQ(ma.median(), 100);
  ASSERT_EQ(ma.count(), 3);
  ASSERT_EQ(ma.median_absolute_deviation(), 100);
  ASSERT_NEAR(ma.standard_deviation(), 81.6496581, 0.000001);
}

TEST_F(StandardAggregatorTest, MultiValueMath) {
  AggregatorOutput out;

  WriteFile("file1",
            {
                HelperCreateSampleRecord(1, {std::make_pair("y", 1)}),
                HelperCreateSampleRecord(2, {std::make_pair("y", 100)}),
                HelperCreateSampleRecord(3, {std::make_pair("y", 1000)}),
                HelperCreateSampleRecord(4, {std::make_pair("y", 10000)}),
                HelperCreateSampleRecord(5, {std::make_pair("y", 100000)}),
                HelperCreateSampleRecord(6, {std::make_pair("y", 1000000)}),
            });

  ASSERT_EQ(a_.Aggregate(HelperCreateAggregatorInput({"file1"}), &out), "");

  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 1);
  const mako::MetricAggregate& ma =
      out.aggregate().metric_aggregate_list(0);

  ASSERT_EQ(6, ma.count());

  // Verified with
  // https://www.google.com/webhp?sourceid=chrome-instant&rlz=1CAGGAB_enUS552US552&ion=1&espv=2&es_th=1&ie=UTF-8#q=(1%2B%20100%20%2B%201000%20%2B%2010000%20%2B%20100000%20%2B%201000000)%2F6&es_th=1
  ASSERT_NEAR(185183.5, ma.mean(), 0.0001);

  // Verified with
  // https://www.easycalculation.com/statistics/standard-deviation.php
  // 'Standard Deviation'
  ASSERT_NEAR(366138.2794263, ma.standard_deviation(), 0.000001);

  ASSERT_EQ(5449.5, ma.median_absolute_deviation());

  ASSERT_NEAR(5.95, ma.percentile_list(0), 0.0001);
  ASSERT_NEAR(10.9, ma.percentile_list(1), 0.0001);
  ASSERT_NEAR(25.75, ma.percentile_list(2), 0.0001);
  ASSERT_NEAR(50.5, ma.percentile_list(3), 0.0001);
  ASSERT_NEAR(550000, ma.percentile_list(4), 0.0001);
  ASSERT_NEAR(775000, ma.percentile_list(5), 0.0001);
  ASSERT_NEAR(910000, ma.percentile_list(6), 0.0001);
  ASSERT_NEAR(955000, ma.percentile_list(7), 0.0001);
}

TEST_F(StandardAggregatorTest, SingleValueMath) {
  AggregatorOutput out;

  WriteFile("file1", {
                         HelperCreateSampleRecord(1, {std::make_pair("y", 1)}),
                     });

  ASSERT_EQ(a_.Aggregate(HelperCreateAggregatorInput({"file1"}), &out), "");

  EXPECT_EQ(out.aggregate().metric_aggregate_list_size(), 1);
  const mako::MetricAggregate& ma =
      out.aggregate().metric_aggregate_list(0);
  EXPECT_EQ(ma.count(), 1);
  EXPECT_EQ(ma.mean(), 1);
  EXPECT_EQ(ma.standard_deviation(), 0);
  EXPECT_EQ(ma.median_absolute_deviation(), 0);
  for (const auto& percent : ma.percentile_list()) {
    EXPECT_EQ(percent, 1);
  }
}

TEST_F(StandardAggregatorTest, MaxSampleSize) {
  // Testing here that the standard aggregator respects the max sample size
  // constructor argument. When that value is positive, it specifies the maximum
  // number of values per metric that should be saved to use for the statistical
  // calculations. Since the internals of the aggregator and RunningStats
  // objects are not exposed, we can't verify this directly.
  //
  // Instead, what we do here is set the max sample size to 1. With a sample
  // size of only 1, all of the calculated percentiles will be identical. The
  // exact value will change from run to run as it is randomly chosen, but as
  // long as they're all identical, we know the max sample size argument was
  // respected.
  AggregatorOutput out;

  std::vector<std::string> files;

  // Create 10 files
  for (int i = 0; i < 10; ++i) {
    files.push_back(absl::StrCat("file", i));
    WriteFile(files.back(),
              {
                  HelperCreateSampleRecord(1, {std::make_pair("y", 1)}),
                  HelperCreateSampleRecord(2, {std::make_pair("y", 100)}),
                  HelperCreateSampleRecord(3, {std::make_pair("y", 1000)}),
                  HelperCreateSampleRecord(4, {std::make_pair("y", 10000)}),
                  HelperCreateSampleRecord(5, {std::make_pair("y", 100000)}),
                  HelperCreateSampleRecord(6, {std::make_pair("y", 1000000)}),
              });
  }

  int max_sample_size = 1;
  Aggregator a(max_sample_size, kDefaultMaxThreads, kDefaultBufferSize);
  a.SetFileIO(absl::make_unique<mako::memory_fileio::FileIO>());

  ASSERT_EQ(a.Aggregate(HelperCreateAggregatorInput(files), &out), "");

  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 1);
  const mako::MetricAggregate& ma =
      out.aggregate().metric_aggregate_list(0);

  ASSERT_EQ(60, ma.count());
  // Verified with
  // https://www.google.com/webhp?sourceid=chrome-instant&rlz=1CAGGAB_enUS552US552&ion=1&espv=2&es_th=1&ie=UTF-8#q=(1%2B%20100%20%2B%201000%20%2B%2010000%20%2B%20100000%20%2B%201000000)%2F6&es_th=1
  EXPECT_NEAR(185183.5, ma.mean(), 0.0001);

  // Verified with
  // https://www.easycalculation.com/statistics/standard-deviation.php
  // 'Standard Deviation'
  EXPECT_NEAR(366138.2794263, ma.standard_deviation(), 0.000001);

  EXPECT_EQ(0, ma.median_absolute_deviation());

  std::set<double> percentiles;
  for (const auto& percent : ma.percentile_list()) {
    percentiles.insert(percent);
  }
  // Since the max sample size is 1, all of the percentiles will be identical.
  EXPECT_EQ(percentiles.size(), 1);
}

TEST_F(StandardAggregatorTest, AggregateMultipleTimes) {
  AggregatorOutput out;

  WriteFile("file1", {HelperCreateSampleRecord(1, {std::make_pair("y", 1)})});
  AggregatorInput aggregator_input = HelperCreateAggregatorInput({"file1"});
  mako::LabeledRange* r =
      aggregator_input.mutable_run_info()->mutable_ignore_range_list()->Add();
  r->set_label("label");
  r->mutable_range()->set_start(100);
  r->mutable_range()->set_end(101);

  ASSERT_EQ(a_.Aggregate(aggregator_input, &out), "");
  ASSERT_EQ(out.aggregate().run_aggregate().usable_sample_count(), 1);
  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 1);
  ASSERT_EQ(out.aggregate().metric_aggregate_list(0).mean(), 1);

  out.Clear();
  WriteFile("file2",
            {HelperCreateSampleRecord(100, {std::make_pair("y", 100)})});
  ASSERT_EQ(a_.Aggregate(HelperCreateAggregatorInput({"file2"}), &out), "");
  ASSERT_EQ(out.aggregate().run_aggregate().usable_sample_count(), 1);
  ASSERT_EQ(out.aggregate().metric_aggregate_list_size(), 1);
  ASSERT_EQ(out.aggregate().metric_aggregate_list(0).mean(), 100);
}

TEST_F(StandardAggregatorTest, FileIOCloseErrorIsReturned) {
  std::string close_error = "close error";
  auto memory_fileio = absl::make_unique<mako::memory_fileio::FileIO>();
  memory_fileio->set_close_error(close_error);
  a_.SetFileIO(std::move(memory_fileio));

  AggregatorOutput out;
  WriteFile("file1", {HelperCreateSampleRecord(1, {std::make_pair("y", 100),
                                                   std::make_pair("y", 50)})});

  EXPECT_THAT(a_.Aggregate(HelperCreateAggregatorInput({"file1"}), &out),
              ContainsRegex(absl::StrFormat(".*%s.*", close_error)));
}

}  // namespace
}  // namespace aggregator
}  // namespace mako
