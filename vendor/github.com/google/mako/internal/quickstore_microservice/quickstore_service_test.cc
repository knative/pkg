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
#include "internal/quickstore_microservice/quickstore_service.h"

#include "include/grpcpp/security/server_credentials.h"
#include "include/grpcpp/server_context.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/cxx/storage/fake_google3_storage.h"
#include "proto/quickstore/quickstore.pb.h"
#include "internal/quickstore_microservice/proto/quickstore.pb.h"
#include "absl/memory/memory.h"
#include "internal/cxx/queue.h"
#include "spec/proto/mako.pb.h"
#include "testing/cxx/protocol-buffer-matchers.h"

namespace mako {
namespace internal {
namespace quickstore_microservice {
namespace {

constexpr char kM1[] = "m1";
constexpr char kM2[] = "m2";
constexpr char kM3[] = "m3";
constexpr char kC1[] = "c1";
constexpr char kSampleErrorString[] = "Some Error";
constexpr double kM3Mean = 2;
constexpr double kM3Min = 1;

using ::mako::EqualsProto;
using ::mako::proto::Partially;
using ::testing::Eq;

// Basic test just to verify everything is being piped through correctly.
TEST(QuickstoreServiceTest, Store) {
  mako::internal::Queue<bool> shutdown_queue;
  QuickstoreService service(
      &shutdown_queue,
      absl::make_unique<mako::fake_google3_storage::Storage>());

  StoreInput input;
  input.mutable_quickstore_input()->set_benchmark_key("12345");

  mako::fake_google3_storage::Storage s;

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
  std::string benchmark_key = c.key();
  input.mutable_quickstore_input()->set_benchmark_key(benchmark_key);

  // Create SamplePoints
  for (int i = 100; i < 200; i++) {
    mako::SamplePoint* p = input.add_sample_points();
    p->set_input_value(i);
    for (const auto& m : {kM1, kM2}) {
      mako::KeyedValue* k = p->add_metric_value_list();
      k->set_value(i * 5);
      k->set_value_key(m);
    }
  }
  for (int i = 10; i < 20; i++) {
    mako::SamplePoint* p = input.add_sample_points();
    p->set_input_value(i);
    for (const auto& m : {kM3}) {
      mako::KeyedValue* k = p->add_metric_value_list();
      k->set_value(i * 5);
      k->set_value_key(m);
    }
  }

  // Create SampleErrors
  for (int i = 0; i < 100; i++) {
    mako::SampleError* e = input.add_sample_errors();
    e->set_input_value(i);
    e->set_error_message(kSampleErrorString);
  }

  // Custom metric aggregates for kM3
  input.add_aggregate_value_keys(kM3);
  input.add_aggregate_value_types("min");
  input.add_aggregate_value_values(kM3Min);
  input.add_aggregate_value_keys(kM3);
  input.add_aggregate_value_types("mean");
  input.add_aggregate_value_values(kM3Mean);

  // Create custom aggregates
  mako::KeyedValue* k = input.add_run_aggregates();
  k->set_value(1000);
  k->set_value_key(kC1);

  StoreOutput output;

  // If we instantiate a grpc::ServerContext, its destructor segfaults when run
  // externally. Until we solve this issue, let's just pass in a pointer. The
  // service implementation doesn't use the context anyway.
  grpc::ServerContext* context = nullptr;
  EXPECT_OK(service.Store(context, &input, &output));
  EXPECT_THAT(
      output.quickstore_output().status(),
      Eq(mako::quickstore::QuickstoreOutput::SUCCESS));

  mako::RunInfoQuery query;
  query.set_benchmark_key(benchmark_key);
  query.set_run_key(output.quickstore_output().run_key());

  mako::RunInfoQueryResponse response;
  ASSERT_TRUE(s.QueryRunInfo(query, &response));
  ASSERT_THAT(response.run_info_list_size(), Eq(1));
  EXPECT_THAT(response.run_info_list(0), Partially(EqualsProto(R"proto(
                aggregate {
                  metric_aggregate_list {
                    metric_key: "m1"
                    min: 500
                    max: 995
                    mean: 747.5
                    median: 747.5
                    standard_deviation: 144.33035023861058
                    percentile_list: 504.95
                    percentile_list: 509.90000000000003
                    percentile_list: 524.75
                    percentile_list: 549.5
                    percentile_list: 945.5
                    percentile_list: 970.25
                    percentile_list: 985.1
                    percentile_list: 990.05
                    count: 100
                    median_absolute_deviation: 125
                  }
                  metric_aggregate_list {
                    metric_key: "m2"
                    min: 500
                    max: 995
                    mean: 747.5
                    median: 747.5
                    standard_deviation: 144.33035023861058
                    percentile_list: 504.95
                    percentile_list: 509.90000000000003
                    percentile_list: 524.75
                    percentile_list: 549.5
                    percentile_list: 945.5
                    percentile_list: 970.25
                    percentile_list: 985.1
                    percentile_list: 990.05
                    count: 100
                    median_absolute_deviation: 125
                  }
                  metric_aggregate_list {
                    metric_key: "m3"
                    min: 1
                    mean: 2
                    percentile_list: 0
                    percentile_list: 0
                    percentile_list: 0
                    percentile_list: 0
                    percentile_list: 0
                    percentile_list: 0
                    percentile_list: 0
                    percentile_list: 0
                  }
                  run_aggregate {
                    usable_sample_count: 110
                    ignore_sample_count: 0
                    error_sample_count: 100
                    custom_aggregate_list { value_key: "c1" value: 1000 }
                  }
                  percentile_milli_rank_list: 1000
                  percentile_milli_rank_list: 2000
                  percentile_milli_rank_list: 5000
                  percentile_milli_rank_list: 10000
                  percentile_milli_rank_list: 90000
                  percentile_milli_rank_list: 95000
                  percentile_milli_rank_list: 98000
                  percentile_milli_rank_list: 99000
                }
                test_output { test_status: PASS summary_output: "" }
              )proto")));
}

TEST(QuickstoreServiceTest, Shutdown) {
  mako::internal::Queue<bool> shutdown_queue;
  QuickstoreService service(
      &shutdown_queue,
      absl::make_unique<mako::fake_google3_storage::Storage>());
  ShutdownInput input;
  ShutdownOutput output;
  // If we instantiate a grpc::ServerContext, its destructor segfaults when run
  // externally. Until we solve this issue, let's just pass in a pointer. The
  // service implementation doesn't use the context anyway.
  grpc::ServerContext* context = nullptr;
  EXPECT_OK(service.ShutdownMicroservice(context, &input, &output));
  EXPECT_TRUE(shutdown_queue.get());
}

}  // namespace
}  // namespace quickstore_microservice
}  // namespace internal
}  // namespace mako
