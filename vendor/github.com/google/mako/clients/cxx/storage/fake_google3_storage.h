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

// Provides an in memory fake of the Mako storage system.
//
#ifndef CLIENTS_CXX_STORAGE_FAKE_GOOGLE3_STORAGE_H_
#define CLIENTS_CXX_STORAGE_FAKE_GOOGLE3_STORAGE_H_

#include <string>
#include <vector>

#include "spec/cxx/storage.h"
#include "absl/strings/string_view.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace fake_google3_storage {

// Fake Google3 version of Mako (go/mako) storage system.
//
// See https://github.com/google/mako/blob/master/spec/cxx/storage.h for more
// information about interface.
//
// See https://github.com/google/mako/blob/master/spec/proto/mako.proto for
// information about the protobuf structures used below.
//
// This is useful for unit/integration testing code that depends on
// Mako storage.
//
// Different instances of this class in the same process share a single
// in-memory database. Insertions/Updates/Deletes of items stored in an instance
// of this Fake Storage class will live beteween instances of this class until
// the FakeClear() method is called.
//
// This fake does not support queries involving cursors concurrent with
// updates to any data of the same type.
//
// This fake does not implement all errors that may occur
// when calling a real server. It attempts to succeed on all calls
// and only fails when it cannot satisfy the request.
//
// All functions that are not part of the spec are prefixed with 'Fake',
// and these functions can be used as test helper functions.
//
// This class is Thread-safe.
class Storage : public mako::Storage {
 public:
  struct Options {
    int metric_value_count_max = 50000;
    int error_count_max = 5000;
    int batch_size_max = 1000000;
    int bench_limit_max = 3000;
    int run_limit_max = 3000;
    int batch_limit_max = 100;
    std::string hostname = "example.com";
  };

  Storage();

  Storage(int metric_value_count_max, int error_count_max, int batch_size_max,
          int bench_limit_max, int run_limit_max, int batch_limit_max,
          absl::string_view hostname = "")
      : metric_value_count_max_(metric_value_count_max),
        error_count_max_(error_count_max),
        batch_size_max_(batch_size_max),
        bench_limit_max_(bench_limit_max),
        run_limit_max_(run_limit_max),
        batch_limit_max_(batch_limit_max),
        hostname_(hostname) {}

  explicit Storage(const Options& options)
      : metric_value_count_max_(options.metric_value_count_max),
        error_count_max_(options.error_count_max),
        batch_size_max_(options.batch_size_max),
        bench_limit_max_(options.bench_limit_max),
        run_limit_max_(options.run_limit_max),
        batch_limit_max_(options.batch_limit_max),
        hostname_(options.hostname) {}

  // Creates a new BenchmarkInfo record in Mako storage via RPC sent to
  // server specified in constructor.
  //
  // BenchmarkInfoQuery is used for filtering, and must contain all required
  // fields described in mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CreateBenchmarkInfo(
      const mako::BenchmarkInfo& benchmark_info,
      mako::CreationResponse* creation_response) override;

  // Updates an existing BenchmarkInfo record in Mako storage via RPC sent
  // to server specified in constructor.
  //
  // BenchmarkInfoQuery is used for filtering, and must contain all required
  // fields described in mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool UpdateBenchmarkInfo(
      const mako::BenchmarkInfo& benchmark_info,
      mako::ModificationResponse* mod_response) override;

  // Queries for existing BenchmarkInfo records in Mako storage via RPC sent
  // to server specified in constructor.
  //
  // BenchmarkInfoQuery is used for filtering, and must contain all required
  // fields described in mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool QueryBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::BenchmarkInfoQueryResponse* query_response) override;

  // Deletes existing BenchmarkInfo records in Mako storage via RPC sent to
  // server specified in constructor.
  //
  // BenchmarkInfoQuery is used for filtering, and must contain all required
  // fields described in mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool DeleteBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::ModificationResponse* mod_response) override;

  // Queries for the count of matching BenchmarkInfo records in Mako storage
  // via RPC sent to server specified in constructor.
  //
  // BenchmarkInfoQuery is used for filtering, and must contain all required
  // fields described in mako.proto.
  //
  // CountResponse will be cleared and then populated with the results of the
  // count.
  //
  // More details can be found in interface documentation.
  bool CountBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::CountResponse* count_response) override;

  // Creates a new RunInfo records in Mako storage via RPC sent to
  // server specified in constructor.
  //
  // RunInfo arg must contain all required fields described in mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CreateRunInfo(const mako::RunInfo& run_info,
                     mako::CreationResponse* creation_response) override;

  // Updates an existing RunInfo record in Mako storage via RPC sent to
  // server specified in constructor.
  //
  // RunInfo arg must contain all required fields described in mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool UpdateRunInfo(const mako::RunInfo& run_info,
                     mako::ModificationResponse* mod_response) override;

  // Queries for existing RunInfo records in Mako storage via RPC sent to
  // server specified in constructor.
  //
  // RunInfoQuery is used for filtering, and must contain all required fields
  // described in mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool QueryRunInfo(const mako::RunInfoQuery& run_info_query,
                    mako::RunInfoQueryResponse* query_response) override;

  // Deletes existing RunInfo records in Mako storage via RPC sent to
  // server specified in constructor.
  //
  // RunInfoQuery is used for filtering, and must contain all required fields
  // described in mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool DeleteRunInfo(const mako::RunInfoQuery& run_info_query,
                     mako::ModificationResponse* mod_response) override;

  // Queries for the count of matching RunInfo records in Mako storage via
  // RPC sent to server specified in constructor.
  //
  // RunInfoQuery is used for filtering, and must contain all required fields
  // described in mako.proto.
  //
  // CountResponse will be cleared and then populated with the results of the
  // count.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CountRunInfo(const mako::RunInfoQuery& run_info_query,
                    mako::CountResponse* count_response) override;

  // Creates a new SampleBatch record in Mako storage via RPC sent to
  // server specified in constructor.
  //
  // SampleBatch arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CreateSampleBatch(
      const mako::SampleBatch& sample_batch,
      mako::CreationResponse* creation_response) override;

  // Queries for existing SampleBatch records in Mako storage via RPC sent
  // to server specified in constructor.
  //
  // SampleBatchQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool QuerySampleBatch(
      const mako::SampleBatchQuery& sample_batch_query,
      mako::SampleBatchQueryResponse* query_response) override;

  // Deletes existing SampleBatch records in Mako storage via RPC sent to
  // server specified in constructor.
  //
  // SampleBatchQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if RPC fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool DeleteSampleBatch(const mako::SampleBatchQuery& sample_batch_query,
                         mako::ModificationResponse* mod_response) override;

  // Max number of metrics that can be saved per run. String returned
  // contains error message, or empty std::string if successful.
  //
  // More details can be found in interface documentation.
  std::string GetMetricValueCountMax(int* metric_value_count_max) override;

  // Max number of errors that can be saved per run. String returned contains
  // error message, or empty if successful.
  //
  // More details can be found in interface documentation.
  std::string GetSampleErrorCountMax(int* sample_error_max) override;

  // Max binary size (in base 10 bytes, eg 1MB == 1,000,000) of a SampleBatch.
  // String returned contains error message or empty if successful.
  //
  // More details can be found in interface documentation.
  std::string GetBatchSizeMax(int* batch_size_max) override;

  // The hostname backing this Storage implementation.
  //
  // More details can be found in interface documentation.
  std::string GetHostname() override;

  // Clear the storage system of all known data.
  //
  // Because different instances of this class share the same in-memory
  // database, this method deletes all data for all instances of this fake
  // Storage class in this process.
  void FakeClear();

  // Adds all provided benchmarks to stored data
  void FakeStageBenchmarks(
      const std::vector<mako::BenchmarkInfo>& benchmark_info_list);

  // Adds all provided benchmarks to stored data
  void FakeStageRuns(const std::vector<mako::RunInfo>& run_info_list);

  // Adds all provided batches to stored data
  void FakeStageBatches(
      const std::vector<mako::SampleBatch>& sample_batch_list);

  ~Storage() override {}

 private:
  // Will be returned by GetMetricValueCountMax()
  int metric_value_count_max_;
  // Will be returned by GetSampleErrorCountMax()
  int error_count_max_;
  // Will be returned by GetBatchSizeMax()
  int batch_size_max_;
  // Default/max limit for BenchmarkInfo queries
  int bench_limit_max_;
  // Default/max limit for RunInfo queries
  int run_limit_max_;
  // Default/max limit for SampleBatch queries
  int batch_limit_max_;
  // What to return from GetHostname(). This is used by the Mako framework
  // to create links.
  std::string hostname_;

#ifndef SWIG
  // Not copyable.
  Storage(const Storage&) = delete;
  Storage& operator=(const Storage&) = delete;

  // Not movable.
  Storage(Storage&&) = delete;
  Storage& operator=(Storage&&) = delete;
#endif  // #ifndef SWIG
};

}  // namespace fake_google3_storage
}  // namespace mako

#endif  // CLIENTS_CXX_STORAGE_FAKE_GOOGLE3_STORAGE_H_
