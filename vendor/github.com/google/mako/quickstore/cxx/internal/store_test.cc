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
#include "quickstore/cxx/internal/store.h"

#include <set>
#include <string>

#include "glog/logging.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/cxx/aggregator/standard_aggregator.h"
#include "clients/cxx/downsampler/standard_downsampler.h"
#include "clients/cxx/fileio/memory_fileio.h"
#include "clients/cxx/storage/fake_google3_storage.h"
#include "clients/proto/analyzers/threshold_analyzer.pb.h"
#include "spec/cxx/fileio.h"
#include "absl/memory/memory.h"
#include "proto/quickstore/quickstore.pb.h"
#include "spec/proto/mako.pb.h"
#include "testing/cxx/protocol-buffer-matchers.h"

namespace mako {
namespace quickstore {
namespace internal {
namespace {

constexpr char kM1[] = "m1";
constexpr char kM2[] = "m2";
constexpr char kM3[] = "m3";
constexpr char kC1[] = "c1";
constexpr char kSampleErrorString[] = "Some Error";
constexpr double kM3Mean = 2;
constexpr double kM3Min = 1;


using ::mako::EqualsProto;
using ::mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput;
using ::mako::analyzers::threshold_analyzer::ThresholdConfig;
using ::mako::quickstore::QuickstoreInput;
using ::mako::quickstore::QuickstoreOutput;
using ::testing::IsEmpty;
using ::testing::UnorderedPointwise;

class StoreTest : public ::testing::Test {
 protected:
  StoreTest() {
    // Clear previous data before each call
    mako::fake_google3_storage::Storage s;
    s.FakeClear();

    // Create BenchmarkInfo
    mako::BenchmarkInfo b;
    b.set_benchmark_name("b name");
    b.set_project_name("b project");
    *b.add_owner_list() = "*";
    mako::ValueInfo* m = b.add_metric_info_list();
    m->set_label("Metric 1");
    m->set_value_key(kM1);
    m = b.add_metric_info_list();
    m->set_label("Metric 2");
    m->set_value_key(kM2);
    m->set_label("Metric 3");
    m->set_value_key(kM3);
    m = b.add_custom_aggregation_info_list();
    m->set_label("Custom 1");
    m->set_value_key(kC1);
    b.mutable_input_value_info()->set_label("Time");
    b.mutable_input_value_info()->set_value_key("t");
    mako::CreationResponse c;
    CHECK(s.CreateBenchmarkInfo(b, &c)) << c.status().fail_message();
    // Save benchmark_key
    benchmark_key_ = c.key();

    // Create SamplePoints
    for (int i = 100; i < 200; i++) {
      mako::SamplePoint p;
      p.set_input_value(i);
      for (const auto& m : {kM1, kM2}) {
        mako::KeyedValue* k = p.add_metric_value_list();
        k->set_value(i * 5);
        k->set_value_key(m);
      }
      points_.push_back(p);
    }
    for (int i = 10; i < 20; i++) {
      mako::SamplePoint p;
      p.set_input_value(i);
      for (const auto& m : {kM3}) {
        mako::KeyedValue* k = p.add_metric_value_list();
        k->set_value(i * 5);
        k->set_value_key(m);
      }
      points_.push_back(p);
    }

    // Create SampleErrors
    for (int i = 0; i < 100; i++) {
      mako::SampleError e;
      e.set_input_value(i);
      e.set_error_message(kSampleErrorString);
      errors_.push_back(e);
    }

    // Custom metric aggregates for kM3
    agg_met_keys_.push_back(kM3);
    agg_types_.push_back("min");
    agg_values_.push_back(kM3Min);
    agg_met_keys_.push_back(kM3);
    agg_types_.push_back("mean");
    agg_values_.push_back(kM3Mean);

    // Create custom aggregates
    mako::KeyedValue k;
    k.set_value(1000);
    k.set_value_key(kC1);
    run_aggs_.push_back(k);

    input_.set_benchmark_key(benchmark_key_);
  }

  void TearDown() override {
  }

  // Returns all runs when run_key is empty, and matching runs otherwise.
  std::vector<mako::RunInfo> GetRuns(const std::string& run_key) {
    mako::fake_google3_storage::Storage s;
    mako::RunInfoQuery q;
    q.set_benchmark_key(benchmark_key_);
    if (!run_key.empty()) {
      q.set_run_key(run_key);
    }
    mako::RunInfoQueryResponse r;
    CHECK(s.QueryRunInfo(q, &r)) << r.status().fail_message();
    return {r.run_info_list().begin(), r.run_info_list().end()};
  }

  mako::RunInfo FindRun(const std::string& run_key) {
    auto runs = GetRuns(run_key);
    CHECK(1 == runs.size()) << "Got " << runs.size() << " runs; want 1";
    return runs[0];
  }

  std::string benchmark_key_;
  std::vector<mako::SamplePoint> points_;
  std::vector<mako::SampleError> errors_;
  std::vector<mako::KeyedValue> run_aggs_;
  std::vector<std::string> agg_met_keys_;
  std::vector<std::string> agg_types_;
  std::vector<double> agg_values_;
  QuickstoreInput input_;
};

QuickstoreOutput Call(const QuickstoreInput& input,
                      const std::vector<mako::SamplePoint>& points,
                      const std::vector<mako::SampleError>& errors,
                      const std::vector<mako::KeyedValue>& run_aggregates,
                      const std::vector<std::string>& aggregate_metric_keys,
                      const std::vector<std::string>& aggregate_types,
                      const std::vector<double>& aggregate_values,
                      mako::Storage* storage = nullptr) {
  mako::fake_google3_storage::Storage default_storage;
  if (storage == nullptr) {
    storage = &default_storage;
  }

  return SaveWithStorage(storage, input, points, errors, run_aggregates,
                         aggregate_metric_keys, aggregate_types,
                         aggregate_values);
}

TEST_F(StoreTest, NoBenchmarkInfo) {
  input_.clear_benchmark_key();
  QuickstoreOutput output = Call(input_, points_, {}, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::ERROR, output.status());
  ASSERT_NE("", output.summary_output());
}

TEST_F(StoreTest, BadBenchmarkInfo) {
  input_.set_benchmark_key("bad_key");
  QuickstoreOutput output = Call(input_, points_, {}, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::ERROR, output.status());
  ASSERT_NE("", output.summary_output());
}

TEST_F(StoreTest, InvalidMetricAggregates) {
  // Must all be the same length
  agg_met_keys_.pop_back();
  QuickstoreOutput output =
      Call(input_, {}, {}, {}, agg_met_keys_, agg_types_, agg_values_);
  ASSERT_EQ(QuickstoreOutput::ERROR, output.status());
  ASSERT_NE("", output.summary_output());
}

TEST_F(StoreTest, PointsOnly) {
  QuickstoreOutput output = Call(input_, points_, {}, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
}

TEST_F(StoreTest, ErrorsOnly) {
  QuickstoreOutput output = Call(input_, {}, errors_, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
}

TEST_F(StoreTest, CustomAggregatesOnly) {
  QuickstoreOutput output = Call(input_, {}, {}, run_aggs_, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
}

TEST_F(StoreTest, MetricAggregatesOnly) {
  QuickstoreOutput output =
      Call(input_, {}, {}, {}, agg_met_keys_, agg_types_, agg_values_);
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
}

TEST_F(StoreTest, AutomaticPercentiles) {
  QuickstoreOutput output = Call(input_, points_, errors_, run_aggs_,
                                 agg_met_keys_, agg_types_, agg_values_);
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());

  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_TRUE(actual_run.has_aggregate());
  int num_percentiles =
      actual_run.aggregate().percentile_milli_rank_list_size();
  ASSERT_GT(num_percentiles, 0);
  ASSERT_GT(actual_run.aggregate().metric_aggregate_list_size(), 0);
  for (const auto& ma : actual_run.aggregate().metric_aggregate_list()) {
    EXPECT_EQ(num_percentiles, ma.percentile_list_size());
    // In our static data above we set custom metric values for kM3 but do not
    // set percentiles so they should be all 0.
    if (ma.metric_key() == kM3) {
      // All percentiles should be filled out.
      for (const auto& p : ma.percentile_list()) {
        EXPECT_EQ(p, 0);
      }
    } else {
      // All percentiles should be filled out.
      for (const auto& p : ma.percentile_list()) {
        EXPECT_GT(p, 0) << ma.metric_key();
      }
    }
  }
}

TEST_F(StoreTest, InvalidPercentile) {
  agg_met_keys_.push_back(kM3);
  agg_types_.push_back("pnosuch");
  agg_values_.push_back(100);
  QuickstoreOutput output = Call(input_, points_, errors_, run_aggs_,
                                 agg_met_keys_, agg_types_, agg_values_);
  ASSERT_EQ(QuickstoreOutput::ERROR, output.status());
  ASSERT_NE("", output.summary_output());
}

TEST_F(StoreTest, InvalidMetricAggregate) {
  agg_met_keys_.push_back(kM3);
  agg_types_.push_back("nosuch");
  agg_values_.push_back(100);
  QuickstoreOutput output = Call(input_, points_, errors_, run_aggs_,
                                 agg_met_keys_, agg_types_, agg_values_);
  ASSERT_EQ(QuickstoreOutput::ERROR, output.status());
  ASSERT_NE("", output.summary_output());
}

TEST_F(StoreTest, SetSingleDefaultPercentile) {
  double percentile_value = 1.56;
  agg_met_keys_.push_back(kM3);
  agg_types_.push_back("p98000");
  agg_values_.push_back(percentile_value);

  QuickstoreOutput output = Call(input_, points_, errors_, run_aggs_,
                                 agg_met_keys_, agg_types_, agg_values_);
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_TRUE(actual_run.has_aggregate());
  ASSERT_GT(actual_run.aggregate().metric_aggregate_list_size(), 0);
  for (const auto& ma : actual_run.aggregate().metric_aggregate_list()) {
    if (ma.metric_key() != kM3) {
      continue;
    }
    // 98 percentile is the 6th index in default
    EXPECT_EQ(percentile_value, ma.percentile_list(6));
    // Others should be 0
    for (const auto& d : {0, 1, 2, 3, 4, 5, 7}) {
      EXPECT_EQ(0, ma.percentile_list(d)) << d;
    }
  }
}

TEST_F(StoreTest, CustomPercentile) {
  int p1 = 10;
  int p2 = 15;

  int m3_p1 = 1;
  int m2_p1 = 2;
  int m2_p2 = 3;

  // Update benchmark w/ custom percentile list
  mako::BenchmarkInfoQuery q;
  mako::BenchmarkInfoQueryResponse r;
  q.set_benchmark_key(benchmark_key_);
  mako::fake_google3_storage::Storage s;
  ASSERT_TRUE(s.QueryBenchmarkInfo(q, &r));
  ASSERT_EQ(1, r.benchmark_info_list_size());
  mako::BenchmarkInfo b(r.benchmark_info_list(0));
  b.clear_percentile_milli_rank_list();
  b.add_percentile_milli_rank_list(p1);
  b.add_percentile_milli_rank_list(p2);
  mako::ModificationResponse mr;
  ASSERT_TRUE(s.UpdateBenchmarkInfo(b, &mr));
  ASSERT_EQ(1, mr.count());

  std::vector<std::string> agg_met_keys;
  std::vector<std::string> agg_types;
  std::vector<double> agg_values;

  // kM1 sets none, should both be filled out by default
  // kM2 sets both p2 and p1
  agg_met_keys.push_back(kM2);
  agg_types.push_back("p10");
  agg_values.push_back(m2_p1);
  agg_met_keys.push_back(kM2);
  agg_types.push_back("p15");
  agg_values.push_back(m2_p2);
  // kM3 sets p1
  agg_met_keys.push_back(kM3);
  agg_types.push_back("p10");
  agg_values.push_back(m3_p1);

  QuickstoreOutput output = Call(input_, points_, errors_, run_aggs_,
                                 agg_met_keys, agg_types, agg_values);
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_TRUE(actual_run.has_aggregate());
  ASSERT_GT(actual_run.aggregate().metric_aggregate_list_size(), 0);
  for (const auto& ma : actual_run.aggregate().metric_aggregate_list()) {
    ASSERT_EQ(2, ma.percentile_list_size());
    if (ma.metric_key() == kM1) {
      EXPECT_GT(ma.percentile_list(0), 0);
      EXPECT_GT(ma.percentile_list(1), 0);
    }
    if (ma.metric_key() == kM2) {
      EXPECT_EQ(ma.percentile_list(0), m2_p1);
      EXPECT_EQ(ma.percentile_list(1), m2_p2);
    }
    if (ma.metric_key() == kM3) {
      EXPECT_EQ(ma.percentile_list(0), m3_p1);
      EXPECT_EQ(ma.percentile_list(1), 0);
    }
  }
}

TEST_F(StoreTest, TestAllCustomMetricAggregates) {
  struct CustomAggregateTest {
    std::string aggregate_type;
    double aggregate_value;
    std::function<double(mako::MetricAggregate)> get_aggregate_value;
  };

  std::vector<CustomAggregateTest> custom_aggregate_tests = {
      {"min", .2, [](mako::MetricAggregate ma) { return ma.min(); }},
      {"max", 5, [](mako::MetricAggregate ma) { return ma.max(); }},
      {"mean", 3, [](mako::MetricAggregate ma) { return ma.mean(); }},
      {"median", 3, [](mako::MetricAggregate ma) { return ma.median(); }},
      {"standard_deviation", 1,
       [](mako::MetricAggregate ma) { return ma.standard_deviation(); }},
      {"median_absolute_deviation", 1,
       [](mako::MetricAggregate ma) {
         return ma.median_absolute_deviation();
       }},
      {"count", 100, [](mako::MetricAggregate ma) { return ma.count(); }},
  };

  // Overwrite kM2
  std::vector<std::string> agg_met_keys;
  std::vector<std::string> agg_types;
  std::vector<double> agg_values;

  for (const auto& test : custom_aggregate_tests) {
    agg_met_keys.push_back(kM2);
    agg_types.push_back(test.aggregate_type);
    agg_values.push_back(test.aggregate_value);
  }

  QuickstoreOutput output =
      Call(input_, points_, errors_, {}, agg_met_keys, agg_types, agg_values);
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());

  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_TRUE(actual_run.has_aggregate());
  for (const auto& ma : actual_run.aggregate().metric_aggregate_list()) {
    if (ma.metric_key() == kM2) {
      for (const auto& test : custom_aggregate_tests) {
        EXPECT_EQ(test.aggregate_value, test.get_aggregate_value(ma));
      }
    }
  }
}

TEST_F(StoreTest, MixedCustomAndAutomaticMetricAggregates) {
  // Allow kM1 and kM3 to use automatic custom metrics
  // Overwrite kM2
  std::vector<std::string> agg_met_keys;
  std::vector<std::string> agg_types;
  std::vector<double> agg_values;

  double min = 1;
  double max = 2;
  double mean = 3;
  double median = 3;

  agg_met_keys.push_back(kM2);
  agg_types.push_back("min");
  agg_values.push_back(min);
  agg_met_keys.push_back(kM2);
  agg_types.push_back("max");
  agg_values.push_back(max);
  agg_met_keys.push_back(kM2);
  agg_types.push_back("mean");
  agg_values.push_back(mean);
  agg_met_keys.push_back(kM2);
  agg_types.push_back("median");
  agg_values.push_back(median);

  QuickstoreOutput output =
      Call(input_, points_, errors_, {}, agg_met_keys, agg_types, agg_values);
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());

  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_TRUE(actual_run.has_aggregate());
  ASSERT_EQ(3, actual_run.aggregate().metric_aggregate_list_size());
  std::set<std::string> keys = {kM1, kM2, kM3};
  for (const auto& ma : actual_run.aggregate().metric_aggregate_list()) {
    ASSERT_TRUE(keys.count(ma.metric_key())) << "KEY: " << ma.metric_key();
    keys.erase(ma.metric_key());
    // Few spot checks
    EXPECT_TRUE(ma.has_mean());
    EXPECT_TRUE(ma.has_min());
    EXPECT_TRUE(ma.has_max());

    if (ma.metric_key() == kM2) {
      EXPECT_EQ(min, ma.min());
      EXPECT_EQ(max, ma.max());
      EXPECT_EQ(mean, ma.mean());
      EXPECT_EQ(median, ma.median());
      // Unset b/c we didn't manually set it above.
      EXPECT_FALSE(ma.has_standard_deviation());
    }
  }
  EXPECT_EQ(0, keys.size());
}

TEST_F(StoreTest, AutoMetricAggregates) {
  QuickstoreOutput output = Call(input_, points_, errors_, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());

  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_TRUE(actual_run.has_aggregate());
  ASSERT_EQ(3, actual_run.aggregate().metric_aggregate_list_size());
  std::set<std::string> keys = {kM1, kM2, kM3};
  for (const auto& ma : actual_run.aggregate().metric_aggregate_list()) {
    ASSERT_TRUE(keys.count(ma.metric_key())) << ma.metric_key();
    keys.erase(ma.metric_key());
    // Few spot checks
    EXPECT_TRUE(ma.has_mean());
    EXPECT_TRUE(ma.has_min());
    EXPECT_TRUE(ma.has_standard_deviation());
  }
  EXPECT_EQ(0, keys.size());
}

TEST_F(StoreTest, AutomaticRunAggregates) {
  QuickstoreOutput output = Call(input_, points_, errors_, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());

  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_TRUE(actual_run.has_aggregate());
  ASSERT_TRUE(actual_run.aggregate().has_run_aggregate());
  mako::RunAggregate a = actual_run.aggregate().run_aggregate();
  EXPECT_TRUE(a.has_usable_sample_count());
  EXPECT_TRUE(a.has_ignore_sample_count());
  EXPECT_TRUE(a.has_error_sample_count());
  EXPECT_TRUE(a.has_error_sample_count());
  EXPECT_EQ(0, a.custom_aggregate_list_size());
}

TEST_F(StoreTest, OneRunCreated) {
  QuickstoreOutput output = Call(input_, points_, errors_, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
  EXPECT_EQ(1, GetRuns("").size());
}

TEST_F(StoreTest, NoDefaultDurationTimestamp) {
  ASSERT_FALSE(input_.has_duration_time_ms());
  QuickstoreOutput output = Call(input_, points_, errors_, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_FALSE(actual_run.has_duration_time_ms());
}

TEST_F(StoreTest, BuildId) {
  const int64_t build_id = 12345;
  input_.set_build_id(build_id);
  QuickstoreOutput output = Call(input_, points_, errors_, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_EQ(build_id, actual_run.build_id());
}

TEST_F(StoreTest, NoDefaultBuildId) {
  ASSERT_FALSE(input_.has_build_id());
  QuickstoreOutput output = Call(input_, points_, errors_, {}, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_FALSE(actual_run.has_build_id());
}

TEST_F(StoreTest, CustomRunAggregates) {
  int64_t usable_sample_count = 1;
  int64_t ignore_sample_count = 2;
  int64_t error_sample_count = 3;
  int64_t benchmark_score = 4;
  std::string custom_value_key = "5";
  double custom_value = 6;

  std::vector<mako::KeyedValue> aggs;
  mako::KeyedValue k;

  k.set_value_key("~usable_sample_count");
  k.set_value(usable_sample_count);
  aggs.push_back(k);
  k.set_value_key("~ignore_sample_count");
  k.set_value(ignore_sample_count);
  aggs.push_back(k);
  k.set_value_key("~error_sample_count");
  k.set_value(error_sample_count);
  aggs.push_back(k);
  k.set_value_key("~benchmark_score");
  k.set_value(benchmark_score);
  aggs.push_back(k);
  k.set_value_key(custom_value_key);
  k.set_value(custom_value);
  aggs.push_back(k);

  QuickstoreOutput output = Call(input_, points_, errors_, aggs, {}, {}, {});
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());

  mako::RunInfo actual_run = FindRun(output.run_key());
  ASSERT_TRUE(actual_run.has_aggregate());
  ASSERT_TRUE(actual_run.aggregate().has_run_aggregate());
  mako::RunAggregate a = actual_run.aggregate().run_aggregate();
  EXPECT_EQ(usable_sample_count, a.usable_sample_count());
  EXPECT_EQ(ignore_sample_count, a.ignore_sample_count());
  EXPECT_EQ(error_sample_count, a.error_sample_count());
  EXPECT_EQ(error_sample_count, a.error_sample_count());
  ASSERT_EQ(1, a.custom_aggregate_list_size());
  EXPECT_EQ(custom_value_key, a.custom_aggregate_list(0).value_key());
  EXPECT_EQ(custom_value, a.custom_aggregate_list(0).value());
}

TEST_F(StoreTest, VerifyQuickstoreInput) {
  double duration_time_ms = 500;
  double timestamp_ms = 946684800001;
  std::string hover = "run hover text";
  std::string description = "A good test description";
  std::string tag = "tag1";
  std::string annotation_label = "big problem";
  std::string hyperlink_name = "Mako";
  std::string auxdata_name = "Aux";
  std::string auxdata_data = "Dater";
  std::string err = "A big error happened here";
  std::string option_name = "An important option";
  std::string ignore_range_label = "An ignore range";

  QuickstoreInput input;
  input.set_benchmark_key(benchmark_key_);
  input.set_duration_time_ms(duration_time_ms);
  input.set_timestamp_ms(timestamp_ms);
  input.set_hover_text(hover);
  input.set_description(description);
  *input.add_tags() = tag;
  mako::RunAnnotation* an = input.add_annotation_list();
  an->set_label(annotation_label);
  an->set_description("Something happened here");
  an->set_value_key(kM1);
  mako::NamedData* l = input.add_hyperlink_list();
  l->set_name(hyperlink_name);
  l->set_data("http://mako.com");
  mako::NamedData* ad = input.add_aux_data();
  ad->set_name(auxdata_name);
  ad->set_data(auxdata_data);
  mako::LabeledRange* ig = input.add_ignore_range_list();
  ig->set_label(ignore_range_label);
  ig->mutable_range()->set_end(2);
  ig->mutable_range()->set_start(1);

  QuickstoreOutput output = Call(input, points_, errors_, run_aggs_,
                                 agg_met_keys_, agg_types_, agg_values_);

  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
  EXPECT_EQ(0, output.analyzer_output_list_size());
  EXPECT_TRUE(output.has_summary_output());
  EXPECT_NE("", output.run_chart_link());
  EXPECT_NE("", output.run_key());
  mako::RunInfo actual_run = FindRun(output.run_key());
  EXPECT_EQ(output.run_key(), actual_run.run_key());
  EXPECT_EQ(duration_time_ms, actual_run.duration_time_ms());
  EXPECT_EQ(timestamp_ms, actual_run.timestamp_ms());
  EXPECT_EQ(hover, actual_run.hover_text());
  EXPECT_EQ(description, actual_run.description());
  ASSERT_EQ(1, actual_run.tags_size());
  EXPECT_EQ(tag, actual_run.tags(0));
  ASSERT_EQ(1, actual_run.annotation_list_size());
  EXPECT_EQ(annotation_label, actual_run.annotation_list(0).label());
  ASSERT_EQ(1, actual_run.hyperlink_list_size());
  EXPECT_EQ(hyperlink_name, actual_run.hyperlink_list(0).name());
  ASSERT_EQ(1, actual_run.aux_data_size());
  EXPECT_EQ(auxdata_name, actual_run.aux_data(0).name());
  EXPECT_EQ(auxdata_data, actual_run.aux_data(0).data());
  ASSERT_EQ(1, actual_run.ignore_range_list_size());
  EXPECT_EQ(ignore_range_label, actual_run.ignore_range_list(0).label());
}

TEST_F(StoreTest, SuccessTagsApplied) {
  *input_.add_tags() = "always";
  *input_.mutable_analysis_fail()->add_tags() = "fail";
  *input_.mutable_analysis_pass()->add_tags() = "pass";
  // Add a Threshold analyzer to check our data.
  // analyzer should pass.
  ThresholdAnalyzerInput* threshold_input = input_.add_threshold_inputs();
  ThresholdConfig* config = threshold_input->add_configs();
  config->set_max(10000);
  config->set_min(0);
  config->mutable_data_filter()->set_data_type(
      mako::DataFilter::METRIC_SAMPLEPOINTS);
  config->mutable_data_filter()->set_value_key(kM1);

  QuickstoreOutput output = Call(input_, points_, errors_, run_aggs_,
                                 agg_met_keys_, agg_types_, agg_values_);
  ASSERT_EQ(QuickstoreOutput::SUCCESS, output.status());
  mako::RunInfo actual_run = FindRun(output.run_key());
  bool pass_found = false;
  bool always_found = false;
  for (const auto& t : actual_run.tags()) {
    if ("always" == t) {
      always_found = true;
    }
    if ("pass" == t) {
      pass_found = true;
    }
    ASSERT_NE("fail", t);
  }
  ASSERT_EQ(true, pass_found);
  ASSERT_EQ(true, always_found);
}

TEST_F(StoreTest, FailureTagsApplied) {
  *input_.add_tags() = "always";
  *input_.mutable_analysis_fail()->add_tags() = "fail";
  *input_.mutable_analysis_pass()->add_tags() = "pass";
  // Analyzer should fail
  ThresholdAnalyzerInput* threshold_input = input_.add_threshold_inputs();
  ThresholdConfig* config = threshold_input->add_configs();
  config->set_max(1);
  config->set_min(0);
  config->mutable_data_filter()->set_data_type(
      mako::DataFilter::METRIC_SAMPLEPOINTS);
  config->mutable_data_filter()->set_value_key(kM1);

  QuickstoreOutput output = Call(input_, points_, errors_, run_aggs_,
                                 agg_met_keys_, agg_types_, agg_values_);
  ASSERT_EQ(QuickstoreOutput::ANALYSIS_FAIL, output.status());
  ASSERT_GT(output.analyzer_output_list_size(), 0);
  mako::RunInfo actual_run = FindRun(output.run_key());
  bool fail_found = false;
  bool always_found = false;
  for (const auto& t : actual_run.tags()) {
    if ("always" == t) {
      always_found = true;
    }
    if ("fail" == t) {
      fail_found = true;
    }
    ASSERT_NE("pass", t);
  }
  ASSERT_EQ(true, fail_found);
  ASSERT_EQ(true, always_found);
}

}  // namespace
}  // namespace internal
}  // namespace quickstore
}  // namespace mako
