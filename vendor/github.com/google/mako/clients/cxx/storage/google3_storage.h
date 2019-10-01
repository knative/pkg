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

//
// Provides access to the Google3 version of the Mako storage system.
//
#ifndef CLIENTS_CXX_STORAGE_GOOGLE3_STORAGE_H_
#define CLIENTS_CXX_STORAGE_GOOGLE3_STORAGE_H_

#include <memory>
#include <string>
#include <vector>

#include "spec/cxx/storage.h"
#include "absl/flags/flag.h"
#include "internal/cxx/storage_client/retry_strategy.h"
#include "internal/cxx/storage_client/transport.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace google3_storage {

// Mako (go/mako) base storage client.
//
// Mako customers (http://mako.dev) should instead use NewMakoClient() in
// https://github.com/google/mako/blob/master/clients/cxx/storage/mako_client.h.
//
// See https://github.com/google/mako/blob/master/spec/cxx/storage.h for more
// information about interface.
//
// See https://github.com/google/mako/blob/master/spec/proto/mako.proto for
// information about the protobuf structures used below.
//
// Direct use of this class isn't needed when using the normal flow of Mako.
// Consider using Quickstore if you want to write performance test data to the
// storage and dashboard service.
//
// This class is thread-safe (as per go/thread-safe) if the `StorageTransport`
// and the `StorageRetryStrategy` are thread-safe.
class Storage : public mako::Storage {
 public:

  // Create Storage object with the given storage transport, and the default
  // retry strategy.
  explicit Storage(
      std::unique_ptr<mako::internal::StorageTransport> transport);

  // Create Storage object with the given storage transport and retry strategy.
  //
  // This constructor is primarily useful for testing.
  Storage(
      std::unique_ptr<mako::internal::StorageTransport> transport,
      std::unique_ptr<mako::internal::StorageRetryStrategy> retry_strategy);

  // SWIG does not parse the "= delete" syntax.
#ifndef SWIG
  // Not copyable.
  Storage(const Storage&) = delete;
  Storage& operator=(const Storage&) = delete;

  // Movable.
  Storage(Storage&&) = default;
  Storage& operator=(Storage&&) = default;
#endif  // #ifndef SWIG

  // Fetch the storage transport. Exposed for testing.
  mako::internal::StorageTransport* transport() { return transport_.get(); }

  // Creates a new BenchmarkInfo record in Mako storage via the
  // StorageTransport.
  //
  // BenchmarkInfoQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CreateBenchmarkInfo(
      const mako::BenchmarkInfo& benchmark_info,
      mako::CreationResponse* creation_response) override;

  // Updates an existing BenchmarkInfo record in Mako storage via the
  // StorageTransport.
  //
  // BenchmarkInfoQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool UpdateBenchmarkInfo(
      const mako::BenchmarkInfo& benchmark_info,
      mako::ModificationResponse* mod_response) override;

  // Queries for existing BenchmarkInfo records in Mako storage via the
  // StorageTransport.
  //
  // BenchmarkInfoQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool QueryBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::BenchmarkInfoQueryResponse* query_response) override;

  // Deletes existing BenchmarkInfo records in Mako storage via the
  // StorageTransport.
  //
  // BenchmarkInfoQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool DeleteBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::ModificationResponse* mod_response) override;

  // Queries for the count of matching BenchmarkInfo records in Mako storage
  // via the StorageTransport.
  //
  // BenchmarkInfoQuery arg must contain all required fields described in
  // mako.proto.
  //
  // CountResponse will be cleared before issuing the query, and will be
  // populated with the response from Mako storage.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CountBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::CountResponse* count_response) override;

  // Creates a new RunInfo records in Mako storage via the StorageTransport.
  //
  // RunInfo arg must contain all required fields described in mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CreateRunInfo(const mako::RunInfo& run_info,
                     mako::CreationResponse* creation_response) override;

  // Updates an existing RunInfo record in Mako storage via the
  // StorageTransport.
  //
  // RunInfo arg must contain all required fields described in mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool UpdateRunInfo(const mako::RunInfo& run_info,
                     mako::ModificationResponse* mod_response) override;

  // Queries for existing RunInfo records in Mako storage via the
  // StorageTransport.
  //
  // RunInfoQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool QueryRunInfo(const mako::RunInfoQuery& run_info_query,
                    mako::RunInfoQueryResponse* query_response) override;

  // Deletes existing RunInfo records in Mako storage via the
  // StorageTransport.
  //
  // RunInfoQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool DeleteRunInfo(const mako::RunInfoQuery& run_info_query,
                     mako::ModificationResponse* mod_response) override;

  // Queries for the count of matching RunInfo records in Mako storage via
  // the StorageTransport.
  //
  // RunInfoQuery arg must contain all required fields described in
  // mako.proto.
  //
  // CountResponse will be cleared before issuing the query, and will be
  // populated with the response from Mako storage.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CountRunInfo(const mako::RunInfoQuery& run_info_query,
                    mako::CountResponse* count_response) override;

  // Creates a new SampleBatch record in Mako storage via the
  // StorageTransport.
  //
  // SampleBatch arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool CreateSampleBatch(
      const mako::SampleBatch& sample_batch,
      mako::CreationResponse* creation_response) override;

  // Queries for existing SampleBatch records in Mako storage via the
  // StorageTransport.
  //
  // SampleBatchQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool QuerySampleBatch(
      const mako::SampleBatchQuery& sample_batch_query,
      mako::SampleBatchQueryResponse* query_response) override;

  // Deletes existing SampleBatch records in Mako storage via the
  // StorageTransport.
  //
  // SampleBatchQuery arg must contain all required fields described in
  // mako.proto.
  //
  // Returns false if message fails or backend fails. More details found in
  // response's Status protobuf.
  //
  // More details can be found in interface documentation.
  bool DeleteSampleBatch(const mako::SampleBatchQuery& sample_batch_query,
                         mako::ModificationResponse* mod_response) override;

  // Max number of metric values that can be saved per run. String returned
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
  std::string GetHostname() override { return transport_->GetHostname(); }

  // Returns the number of seconds that the last message call took (according to
  // the server). This is exposed for tests of this library to use and should
  // not otherwise be relied on. This is also not guaranteed to be correct in
  // multi-threaded situations.
  // TODO(b/73734783): Remove this.
  double last_call_server_elapsed_time() const {
    return absl::ToDoubleSeconds(transport_->last_call_server_elapsed_time());
  }

 protected:
  std::unique_ptr<mako::internal::StorageTransport> transport_;

  std::unique_ptr<mako::internal::StorageRetryStrategy> retry_strategy_;
};

std::string ApplyHostnameFlagOverrides(const std::string& hostname);
}  // namespace google3_storage
}  // namespace mako

// Exposed for testing only.
extern absl::Flag<std::string> FLAGS_mako_internal_storage_host;
extern absl::Flag<std::string> FLAGS_mako_internal_sudo_run_as;

extern absl::Flag<std::string> FLAGS_mako_internal_test_pass_id_override;
extern absl::Flag<std::vector<std::string> > FLAGS_mako_internal_additional_tags;
#endif  // CLIENTS_CXX_STORAGE_GOOGLE3_STORAGE_H_
