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
#include "internal/cxx/proto_validation.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace internal {
namespace {

using ::testing::Eq;
using ::testing::Gt;
using ::testing::HasSubstr;
using ::testing::StartsWith;

mako::RunInfo CreateValidRunInfo() {
  mako::RunInfo r;
  r.set_benchmark_key("123");
  r.set_run_key("456");
  r.set_timestamp_ms(789);
  return r;
}

mako::BenchmarkInfo CreateValidBenchmarkInfo() {
  mako::BenchmarkInfo b;
  b.set_benchmark_key("123");
  b.set_benchmark_name("bname");
  b.set_project_name("project");
  *b.add_owner_list() = "owner";
  mako::ValueInfo* v = b.mutable_input_value_info();
  v->set_value_key("k");
  v->set_label("KLabel");
  *(b.add_metric_info_list()) = *v;
  return b;
}

TEST(ProtoValidationTest, AggregatorInput) {
  mako::AggregatorInput f;
  mako::AggregatorInput a;
  *a.mutable_run_info() = CreateValidRunInfo();
  *a.mutable_benchmark_info() = CreateValidBenchmarkInfo();
  mako::SampleFile* sf = a.add_sample_file_list();
  sf->set_file_path("/tmp/");
  sf->set_sampler_name("name");
  ASSERT_EQ("", ValidateAggregatorInput(a));

  f = a;
  f.clear_run_info();
  ASSERT_NE("", ValidateAggregatorInput(f));

  f = a;
  f.clear_benchmark_info();
  ASSERT_NE("", ValidateAggregatorInput(f));

  f = a;
  f.mutable_benchmark_info()->clear_benchmark_key();
  ASSERT_NE("", ValidateAggregatorInput(f));

  // NOTE: An empty sample_file_list is valid.
  f = a;
  f.clear_sample_file_list();
  ASSERT_EQ("", ValidateAggregatorInput(f));

  f = a;
  f.mutable_sample_file_list(0)->set_file_path("");
  ASSERT_NE("", ValidateAggregatorInput(f));

  f = a;
  f.mutable_sample_file_list(0)->clear_sampler_name();
  ASSERT_NE("", ValidateAggregatorInput(f));
}

TEST(ProtoValidationTest, ValidateRunInfo) {
  mako::RunInfo f;
  mako::RunInfo r = CreateValidRunInfo();
  ASSERT_EQ("", ValidateRunInfo(r));

  f = r;
  f.clear_benchmark_key();
  ASSERT_NE("", ValidateRunInfo(f));

  f = r;
  f.clear_run_key();
  ASSERT_NE("", ValidateRunInfo(f));

  f = r;
  f.set_timestamp_ms(-1);
  ASSERT_NE("", ValidateRunInfo(f));
}

TEST(ProtoValidationTest, ValidateBenchmarkInfo) {
  mako::BenchmarkInfo f;
  mako::BenchmarkInfo b = CreateValidBenchmarkInfo();
  ASSERT_EQ("", ValidateBenchmarkInfo(b));

  f = b;
  f.clear_benchmark_key();
  ASSERT_NE("", ValidateBenchmarkInfo(f));

  f = b;
  f.set_benchmark_name("");
  ASSERT_NE("", ValidateBenchmarkInfo(f));

  f = b;
  f.clear_project_name();
  ASSERT_NE("", ValidateBenchmarkInfo(f));

  f = b;
  f.clear_owner_list();
  ASSERT_NE("", ValidateBenchmarkInfo(f));

  f = b;
  f.clear_input_value_info();
  ASSERT_NE("", ValidateBenchmarkInfo(f));

  f = b;
  f.mutable_input_value_info()->clear_label();
  ASSERT_NE("", ValidateBenchmarkInfo(f));

  f = b;
  f.mutable_input_value_info()->clear_value_key();
  ASSERT_NE("", ValidateBenchmarkInfo(f));
}

TEST(ProtoValidationTest, DownsamplerInput) {
  mako::DownsamplerInput f;
  mako::DownsamplerInput d;
  *d.mutable_run_info() = CreateValidRunInfo();
  mako::SampleFile* sf = d.add_sample_file_list();
  sf->set_file_path("/tmp/");
  sf->set_sampler_name("name");
  d.set_metric_value_count_max(1);
  d.set_sample_error_count_max(2);
  d.set_batch_size_max(3);
  ASSERT_EQ("", ValidateDownsamplerInput(d));

  f = d;
  f.clear_run_info();
  ASSERT_NE("", ValidateDownsamplerInput(f));

  f = d;
  f.mutable_run_info()->clear_benchmark_key();
  ASSERT_NE("", ValidateDownsamplerInput(f));

  f = d;
  f.set_sample_error_count_max(-1);
  ASSERT_NE("", ValidateDownsamplerInput(f));

  f = d;
  f.clear_metric_value_count_max();
  ASSERT_NE("", ValidateDownsamplerInput(f));

  f = d;
  f.set_batch_size_max(-2);
  ASSERT_NE("", ValidateDownsamplerInput(f));

  // NOTE: An empty sample_file_list is valid.
  f = d;
  f.clear_sample_file_list();
  ASSERT_EQ("", ValidateDownsamplerInput(f));
}

}  // namespace
}  // namespace internal
}  // namespace mako
