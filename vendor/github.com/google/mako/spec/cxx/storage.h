// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// For more information about Mako see: go/mako.

#ifndef SPEC_CXX_STORAGE_H_
#define SPEC_CXX_STORAGE_H_

#include <string>

#include "spec/proto/mako.pb.h"

namespace mako {

// An abstract class which describes the Mako C++ Storage interface.
// See implementing classes for detailed description on usage. See below for
// details on individual functions.
class Storage {
 public:
  // Creates a new BenchmarkInfo record.
  //
  // The BenchmarkInfo reference that is passed, should contain all the
  // information that you wish to have stored in the newly created benchmark.
  // (Although benchmark_info.benchmark_key should not be filled in. That value
  // will be returned in the CreationResponse and used on subsequent queries.)
  //
  // The result of the creation request will be placed in the supplied
  // CreationResponse message.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // CreationResponse.Status.
  //
  // NOTE: Some implementations may not implement this function, and the user
  // may need to use a tool or dashboard to create the benchmark.
  //
  // NOTE: You should not be calling this function often. Benchmarks are meant
  // to be reused for extended periods. There is a command line tool for this
  // operation, see go/mako-help for more information.
  virtual bool CreateBenchmarkInfo(
      const mako::BenchmarkInfo& benchmark_info,
      mako::CreationResponse* creation_response) = 0;

  // Update a BenchmarkInfo record.
  //
  // The BenchmarkInfo.benchmark_key will be used to select the benchmark that
  // you wish to update. All information in the provided BenchmarkInfo
  // will overwrite the existing record.
  //
  // The results of the operation will be placed in ModificiationResponse.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // ModificationResponse.Status.
  //
  // This must be called whenever you modify metric or custom aggregation
  // information. It is convenient to:
  //
  //  - Hard code the benchmark_info fields in test code.
  //
  // NOTE: Requires BenchmarkInfo.benchmark_key
  virtual bool UpdateBenchmarkInfo(
      const mako::BenchmarkInfo& benchmark_info,
      mako::ModificationResponse* mod_response) = 0;

  // Queries for BenchmarkInfo records that match the BenchmarkInfoQuery.
  //
  // You'll always get the best performance when supplying the benchmark_key, if
  // that is set, all other query params will be ignored.
  //
  // The BenchmarkInfoQueryResponse will be populated with results.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // BenchmarkInfoQueryResponse.Status.
  virtual bool QueryBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::BenchmarkInfoQueryResponse* query_response) = 0;

  // Delete BenchmarkInfo records that match the BenchmarkInfoQuery.
  //
  // Delete requests use the same query messages as normal queries to identify
  // data that should be deleted although only the benchmark_key field is used.
  // All other fields are ignored (eg. limit, cursor, project_name, etc).
  //
  // *NOTE* To delete a benchmark, all child data (eg. sample-batches and
  // run-infos) must first be deleted.
  //
  // The results of the operation will be placed in ModificiationResponse.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // ModificationResponse.Status.
  virtual bool DeleteBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::ModificationResponse* mod_response) = 0;

  // Count BenchmarkInfo records that match the BenchmarkInfoQuery.
  //
  // Depending on the implementation, the Count operation may be significantly
  // cheaper and/or faster than QueryBenchmarkInfo.
  // CountResponse will be reset to default values and then populated with the
  // results of the count.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // CountResponse.Status.
  virtual bool CountBenchmarkInfo(
      const mako::BenchmarkInfoQuery& benchmark_info_query,
      mako::CountResponse* count_response) = 0;

  // Creates a new RunInfo record.
  //
  // The RunInfo reference that is passed, should contain all the
  // information that you wish to have stored in the newly created run info.
  // (Although run_info.run_key should not be filled in. That value
  // will be returned in the CreationResponse and used on subsequent queries.)
  //
  // The result of the creation request will be placed in the supplied
  // CreationResponse message.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // CreationResponse.Status.
  //
  // NOTE: See mako.proto for more information about required fields.
  virtual bool CreateRunInfo(const mako::RunInfo& run_info,
                             mako::CreationResponse* creation_response) = 0;

  // Update a RunInfo record.
  //
  // The RunInfo.run_key will be used to select the run info that
  // you wish to update. All information in the provided RunInfo
  // will overwrite the existing record.
  //
  // The results of the operation will be placed in ModificiationResponse.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // ModificationResponse.Status.
  //
  // NOTE: Requires RunInfo.run_key
  virtual bool UpdateRunInfo(const mako::RunInfo& run_info,
                             mako::ModificationResponse* mod_response) = 0;

  // Queries for RunInfo records that match the RunInfoQuery.
  //
  // You'll always get the best performance when supplying the run_key, if
  // that is set, all other query params will be ignored.
  //
  // The RunInfoQueryResponse will be populated with results.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // RunInfoQueryResponse.Status.
  virtual bool QueryRunInfo(const mako::RunInfoQuery& run_info_query,
                            mako::RunInfoQueryResponse* query_response) = 0;

  // Delete RunInfo and child SampleBatch data for runs matching the
  // RunInfoQuery.
  //
  // Delete requests use the same query messages as normal queries to identify
  // data that should be deleted. If you supply the run_key, only that field is
  // used. To prevent accidental deletion of data, benchmark_key is always
  // required.
  //
  // Also deletes child SampleBatch data for each run that matches the query.
  //
  // The results of the operation will be placed in ModificiationResponse.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // ModificationResponse.Status.
  virtual bool DeleteRunInfo(const mako::RunInfoQuery& run_info_query,
                             mako::ModificationResponse* mod_response) = 0;

  // Count RunInfo records matching the RunInfoQuery.
  //
  // Depending on the implementation, the Count operation may be significantly
  // cheaper and/or faster than QueryRunInfo.
  //
  // The results of the operation will be placed in CountResponse.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // CountResponse.Status.
  virtual bool CountRunInfo(const mako::RunInfoQuery& run_info_query,
                            mako::CountResponse* count_response) = 0;

  // Creates a new SampleBatch record.
  //
  // The SampleBatch reference that is passed, should contain all the
  // information that you wish to have stored in the newly created batch.
  // (Although sample_batch.batch_key should not be filled in. That value
  // will be returned in the CreationResponse and used on subsequent queries.)
  //
  // The result of the creation request will be placed in the supplied
  // CreationResponse message.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // CreationResponse.Status.
  //
  // NOTE: See mako.proto for more information about required fields.
  virtual bool CreateSampleBatch(
      const mako::SampleBatch& sample_batch,
      mako::CreationResponse* creation_response) = 0;

  // Queries for SampleBatch records that match the SampleBatchQuery.
  //
  // You'll always get the best performance when supplying the batch_key, if
  // that is set, all other query params will be ignored.
  //
  // The SampleBatchQueryResponse will be populated with results.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // SampleBatchQueryResponse.Status.
  virtual bool QuerySampleBatch(
      const mako::SampleBatchQuery& sample_batch_query,
      mako::SampleBatchQueryResponse* query_response) = 0;

  // Delete SampleBatch records that match the SampleBatchQuery.
  //
  // Delete requests use the same query messages as normal queries to identify
  // data that should be deleted. If you supply the batch_key, only that field
  // is used. To prevent deletion of accidental data, benchmark_key is always
  // required.
  //
  // Calling this is uncommon. Deleting a RunInfo will delete the child batches.
  //
  // The results of the operation will be placed in ModificiationResponse.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // ModificationResponse.Status.
  virtual bool DeleteSampleBatch(
      const mako::SampleBatchQuery& sample_batch_query,
      mako::ModificationResponse* mod_response) = 0;

  // Max number of metrics that can be saved per run
  // String contains error message, or empty if successful.
  virtual std::string GetMetricValueCountMax(int* metric_value_count_max) = 0;

  // Max number of errors that can be saved per run
  // String contains error message, or empty if successful.
  virtual std::string GetSampleErrorCountMax(int* sample_error_max) = 0;

  // Max binary size (in base 10 bytes, eg 1MB == 1,000,000) of a SampleBatch.
  // String contains error message, or empty if successful.
  virtual std::string GetBatchSizeMax(int* batch_size_max) = 0;

  // The hostname backing this Storage implementation.
  // Used to tell a Dashboard implementation what hostname to use when
  // generating chart URLs.
  // TODO(b/124764372) Make pure virtual once all subclasses updated.
  virtual std::string GetHostname() { return "GetHostnameNotImplemented"; }

  virtual ~Storage() {}
};

}  // namespace mako
#endif  // SPEC_CXX_STORAGE_H_
