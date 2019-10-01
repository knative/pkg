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
#include "clients/cxx/storage/fake_google3_storage.h"

#include <algorithm>
#include <set>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/repeated_field.h"
#include "absl/base/thread_annotations.h"
#include "absl/strings/str_cat.h"
#include "absl/synchronization/mutex.h"
#include "internal/cxx/proto_validation.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace fake_google3_storage {
namespace {

class SortByDescendingTimeStampsMs {
 public:
  bool operator()(const mako::RunInfo& r1,
                  const mako::RunInfo& r2) const {
    return r1.timestamp_ms() > r2.timestamp_ms();
  }
};

ABSL_CONST_INIT absl::Mutex mutex(absl::kConstInit);
int max_key GUARDED_BY(mutex) = 0;

std::vector<BenchmarkInfo>& Benchmarks() EXCLUSIVE_LOCKS_REQUIRED(mutex) {
  static auto* benchmarks = new std::vector<BenchmarkInfo>();
  return *benchmarks;
}

std::multiset<mako::RunInfo, SortByDescendingTimeStampsMs>& Runs()
    EXCLUSIVE_LOCKS_REQUIRED(mutex) {
  static auto* runs =
      new std::multiset<mako::RunInfo, SortByDescendingTimeStampsMs>();
  return *runs;
}

std::vector<mako::SampleBatch>& Batches() EXCLUSIVE_LOCKS_REQUIRED(mutex) {
  static auto* batches = new std::vector<mako::SampleBatch>();
  return *batches;
}

constexpr char kNoError[] = "";

std::string NextKey() EXCLUSIVE_LOCKS_REQUIRED(mutex) {
  return std::to_string(++max_key);
}

mako::Status SuccessStatus() {
  mako::Status status;
  status.set_code(mako::Status::SUCCESS);
  return status;
}

mako::Status ErrorStatus(const std::string& msg) {
  mako::Status status;
  status.set_code(mako::Status::FAIL);
  status.set_fail_message(msg);
  return status;
}

bool SampleBatchQueryMatch(const mako::SampleBatchQuery& query,
                           const mako::SampleBatch& sample_batch) {
  if (query.has_batch_key()) {
    return sample_batch.batch_key() == query.batch_key();
  }
  if (query.has_benchmark_key() &&
      sample_batch.benchmark_key() != query.benchmark_key()) {
    return false;
  }
  if (query.has_run_key() && sample_batch.run_key() != query.run_key()) {
    return false;
  }
  return true;
}

bool RunInfoQueryMatch(const mako::RunInfoQuery& query,
                       const mako::RunInfo& run_info) {
  if (query.has_run_key() && query.run_key() != "*") {
    return run_info.run_key() == query.run_key();
  }
  if (!query.test_pass_id().empty()) {
    return run_info.test_pass_id() == query.test_pass_id();
  }
  if (query.benchmark_key() != "*" && !query.benchmark_key().empty() &&
      run_info.benchmark_key() != query.benchmark_key()) {
    return false;
  }
  if (query.has_min_timestamp_ms() &&
      run_info.timestamp_ms() < query.min_timestamp_ms()) {
    return false;
  }
  if (query.has_max_timestamp_ms() &&
      run_info.timestamp_ms() > query.max_timestamp_ms()) {
    return false;
  }
  if (query.has_min_build_id() && run_info.build_id() < query.min_build_id()) {
    return false;
  }
  if (query.has_max_build_id() && run_info.build_id() > query.max_build_id()) {
    return false;
  }

  // set is important not only to eliminate dupes but also because
  // std::includes requires sorted elements
  std::set<std::string> query_tags(query.tags().begin(), query.tags().end());
  std::set<std::string> run_info_tags(run_info.tags().begin(),
                                 run_info.tags().end());
  if (query.tags_size() > 0 &&
      !std::includes(run_info_tags.begin(), run_info_tags.end(),
                     query_tags.begin(), query_tags.end())) {
    return false;
  }

  return true;
}

bool BenchmarkInfoQueryMatch(const mako::BenchmarkInfoQuery& query,
                             const mako::BenchmarkInfo& benchmark_info) {
  if (query.has_benchmark_key()) {
    return query.benchmark_key() == benchmark_info.benchmark_key();
  }

  if (!query.project_name().empty() &&
      query.project_name() != benchmark_info.project_name()) {
    return false;
  }
  if (!query.benchmark_name().empty() &&
      query.benchmark_name() != benchmark_info.benchmark_name()) {
    return false;
  }

  auto ol = benchmark_info.owner_list();
  if (query.owner() != "*" && !query.owner().empty() &&
      std::find(ol.begin(), ol.end(), query.owner()) == std::end(ol)) {
    return false;
  }

  return true;
}

int GetSize(const RunInfoQueryResponse& response) {
  return response.run_info_list_size();
}

bool QueryMatch(const RunInfoQuery& query, const RunInfo& run) {
  return RunInfoQueryMatch(query, run);
}

void AddNewElement(const RunInfo& run, RunInfoQueryResponse* response) {
  *response->add_run_info_list() = run;
}

int GetSize(const BenchmarkInfoQueryResponse& response) {
  return response.benchmark_info_list_size();
}

bool QueryMatch(const BenchmarkInfoQuery& query,
                const BenchmarkInfo& benchmark) {
  return BenchmarkInfoQueryMatch(query, benchmark);
}

void AddNewElement(const BenchmarkInfo& benchmark,
                   BenchmarkInfoQueryResponse* response) {
  *response->add_benchmark_info_list() = benchmark;
}

int GetSize(const SampleBatchQueryResponse& response) {
  return response.sample_batch_list_size();
}

bool QueryMatch(const SampleBatchQuery& query, const SampleBatch& batch) {
  return SampleBatchQueryMatch(query, batch);
}

void AddNewElement(const SampleBatch& batch,
                   SampleBatchQueryResponse* response) {
  *response->add_sample_batch_list() = batch;
}

template <typename Container, typename Query, typename Response>
void FindMatchingElements(const Container& container, const Query& query,
                          int limit_max, Response* response) {
  auto iter = container.begin();
  if (query.has_cursor()) {
    typename Container::size_type cursor = stoi(query.cursor());
    CHECK_LT(cursor, container.size());
    std::advance(iter, cursor);
  }

  int limit = limit_max;
  if (query.has_limit() && query.limit() > 0) {
    limit = std::min(static_cast<int>(query.limit()), limit);
  }

  for (; iter != container.end(); iter++) {
    const auto& element = *iter;
    if (limit == GetSize(*response)) {
      // Match datastore behavior where a returned cursor doesn't necessarily
      // mean there are more elements.
      response->set_cursor(
          std::to_string(std::distance(container.begin(), iter)));
      break;
    }
    if (QueryMatch(query, element)) {
      AddNewElement(element, response);
    }
  }
  *response->mutable_status() = SuccessStatus();
}

}  // namespace

Storage::Storage() : Storage(Options()) {}

bool Storage::CreateBenchmarkInfo(
    const mako::BenchmarkInfo& benchmark_info,
    mako::CreationResponse* creation_response) {
  VLOG(2) << "FakeStorage.CreateBenchmarkInfo("
          << benchmark_info.ShortDebugString() << ")";
  CHECK(creation_response);
  absl::MutexLock lock(&mutex);
  std::string err =
      mako::internal::ValidateBenchmarkInfoCreationRequest(benchmark_info);
  if (!err.empty()) {
    LOG(ERROR) << err;
    *creation_response->mutable_status() = ErrorStatus(err);
    return false;
  }
  mako::BenchmarkInfo benchmark;
  benchmark = benchmark_info;
  benchmark.set_benchmark_key(NextKey());
  Benchmarks().push_back(benchmark);
  creation_response->set_key(benchmark.benchmark_key());
  *creation_response->mutable_status() = SuccessStatus();
  VLOG(2) << "Created BenchmarkInfo with key: " << benchmark.benchmark_key();
  return true;
}

bool Storage::UpdateBenchmarkInfo(
    const mako::BenchmarkInfo& benchmark_info,
    mako::ModificationResponse* mod_response) {
  VLOG(2) << "FakeStorage.UpdateBenchmarkInfo("
          << benchmark_info.ShortDebugString() << ")";
  CHECK(mod_response);
  absl::MutexLock lock(&mutex);
  std::string err = mako::internal::ValidateBenchmarkInfo(benchmark_info);
  if (!err.empty()) {
    LOG(ERROR) << err;
    *mod_response->mutable_status() = ErrorStatus(err);
    return false;
  }
  for (auto itr = Benchmarks().begin(); itr != Benchmarks().end(); ++itr) {
    if (itr->benchmark_key() == benchmark_info.benchmark_key()) {
      *itr = benchmark_info;
      mod_response->set_count(1);
      *mod_response->mutable_status() = SuccessStatus();
      return true;
    }
  }
  mod_response->set_count(0);
  err = absl::StrCat("Could not find benchmark with key: ",
                     benchmark_info.benchmark_key());
  *mod_response->mutable_status() = ErrorStatus(err);
  LOG(ERROR) << err;
  return false;
}

bool Storage::QueryBenchmarkInfo(
    const mako::BenchmarkInfoQuery& benchmark_info_query,
    mako::BenchmarkInfoQueryResponse* query_response) {
  VLOG(2) << "FakeStorage.QueryBenchmarkInfo("
          << benchmark_info_query.ShortDebugString() << ")";
  CHECK(query_response);
  absl::MutexLock lock(&mutex);
  FindMatchingElements(Benchmarks(), benchmark_info_query, bench_limit_max_,
                       query_response);
  VLOG(2) << query_response->benchmark_info_list_size() << " benchmarks found";
  return true;
}

bool Storage::DeleteBenchmarkInfo(
    const mako::BenchmarkInfoQuery& benchmark_info_query,
    mako::ModificationResponse* mod_response) {
  VLOG(2) << "FakeStorage.DeleteBenchmarkInfo("
          << benchmark_info_query.ShortDebugString() << ")";
  CHECK(mod_response);
  absl::MutexLock lock(&mutex);
  auto itr = Benchmarks().begin();
  while (itr != Benchmarks().end()) {
    if (BenchmarkInfoQueryMatch(benchmark_info_query, *itr)) {
      itr = Benchmarks().erase(itr);
      mod_response->set_count(mod_response->count() + 1);
    } else {
      ++itr;
    }
  }
  *mod_response->mutable_status() = SuccessStatus();
  return true;
}

bool Storage::CountBenchmarkInfo(
    const mako::BenchmarkInfoQuery& benchmark_info_query,
    mako::CountResponse* count_response) {
  VLOG(2) << "FakeStorage.CountBenchmarkInfo("
          << benchmark_info_query.ShortDebugString() << ")";
  CHECK(count_response);

  count_response->Clear();

  absl::MutexLock lock(&mutex);
  auto itr = Benchmarks().begin();
  while (itr != Benchmarks().end()) {
    if (BenchmarkInfoQueryMatch(benchmark_info_query, *itr)) {
      count_response->set_count(count_response->count() + 1);
    }
    ++itr;
  }
  *count_response->mutable_status() = SuccessStatus();
  return true;
}

bool Storage::CreateRunInfo(const mako::RunInfo& run_info,
                            mako::CreationResponse* creation_response) {
  VLOG(2) << "FakeStorage.CreateRunInfo(" << run_info.ShortDebugString() << ")";
  CHECK(creation_response);
  absl::MutexLock lock(&mutex);
  std::string err = mako::internal::ValidateRunInfoCreationRequest(run_info);
  if (!err.empty()) {
    LOG(ERROR) << err;
    *creation_response->mutable_status() = ErrorStatus(err);
    return false;
  }
  mako::RunInfo run;
  run = run_info;
  run.set_run_key(NextKey());
  Runs().insert(run);
  creation_response->set_key(run.run_key());
  *creation_response->mutable_status() = SuccessStatus();
  VLOG(2) << "Created RunInfo with key: " << run.run_key();
  return true;
}

bool Storage::UpdateRunInfo(const mako::RunInfo& run_info,
                            mako::ModificationResponse* mod_response) {
  VLOG(2) << "FakeStorage.UpdateRunInfo(" << run_info.ShortDebugString() << ")";
  CHECK(mod_response);
  absl::MutexLock lock(&mutex);
  std::string err = mako::internal::ValidateRunInfo(run_info);
  if (!err.empty()) {
    LOG(ERROR) << err;
    *mod_response->mutable_status() = ErrorStatus(err);
    return false;
  }
  for (auto itr = Runs().begin(); itr != Runs().end(); ++itr) {
    if (itr->run_key() == run_info.run_key()) {
      Runs().erase(itr);
      Runs().insert(run_info);
      mod_response->set_count(1);
      *mod_response->mutable_status() = SuccessStatus();
      return true;
    }
  }
  mod_response->set_count(0);
  err = absl::StrCat("Could not find run with key: ", run_info.run_key());
  *mod_response->mutable_status() = ErrorStatus(err);
  LOG(ERROR) << err;
  return false;
}

bool Storage::QueryRunInfo(const mako::RunInfoQuery& run_info_query,
                           mako::RunInfoQueryResponse* query_response) {
  VLOG(2) << "FakeStorage.QueryRunInfo(" << run_info_query.ShortDebugString()
          << ")";
  CHECK(query_response);
  absl::MutexLock lock(&mutex);

  // Make sure RunInfoQuery has the correct RunOrder set if we are filtering by
  // timestamp or build id, assuming this is actually important
  if ((run_info_query.has_min_timestamp_ms() ||
       run_info_query.has_max_timestamp_ms()) &&
      run_info_query.run_order() != mako::RunOrder::TIMESTAMP) {
    *query_response->mutable_status() = ErrorStatus(
        "Attempted to filter query by timestamp range without run_order set "
        "to TIMESTAMP");
    return false;
  }
  if ((run_info_query.has_min_build_id() ||
       run_info_query.has_max_build_id()) &&
      run_info_query.run_order() != mako::RunOrder::BUILD_ID) {
    *query_response->mutable_status() = ErrorStatus(
        "Attempted to filter query by build_id range without run_order set "
        "to BUILD_ID");
    return false;
  }

  FindMatchingElements(Runs(), run_info_query, run_limit_max_, query_response);
  VLOG(2) << query_response->run_info_list_size() << " runs found";
  return true;
}

bool Storage::DeleteRunInfo(const mako::RunInfoQuery& run_info_query,
                            mako::ModificationResponse* mod_response) {
  VLOG(2) << "FakeStorage.DeleteRunInfo(" << run_info_query.ShortDebugString()
          << ")";
  CHECK(mod_response);
  absl::MutexLock lock(&mutex);
  auto itr = Runs().begin();
  while (itr != Runs().end()) {
    if (RunInfoQueryMatch(run_info_query, *itr)) {
      itr = Runs().erase(itr);
      mod_response->set_count(mod_response->count() + 1);
    } else {
      ++itr;
    }
  }
  *mod_response->mutable_status() = SuccessStatus();
  return true;
}

bool Storage::CountRunInfo(const mako::RunInfoQuery& run_info_query,
                           mako::CountResponse* count_response) {
  VLOG(2) << "FakeStorage.CountRunInfo(" << run_info_query.ShortDebugString()
          << ")";
  CHECK(count_response);
  CHECK_EQ(0, count_response->count());
  absl::MutexLock lock(&mutex);
  auto itr = Runs().begin();
  while (itr != Runs().end()) {
    if (RunInfoQueryMatch(run_info_query, *itr)) {
      count_response->set_count(count_response->count() + 1);
    }
    ++itr;
  }
  *count_response->mutable_status() = SuccessStatus();
  return true;
}

bool Storage::CreateSampleBatch(const mako::SampleBatch& sample_batch,
                                mako::CreationResponse* creation_response) {
  VLOG(2) << "FakeStorage.CreateSampleBatch(" << sample_batch.ShortDebugString()
          << ")";
  CHECK(creation_response);
  absl::MutexLock lock(&mutex);
  std::string err =
      mako::internal::ValidateSampleBatchCreationRequest(sample_batch);
  if (!err.empty()) {
    LOG(ERROR) << err;
    *creation_response->mutable_status() = ErrorStatus(err);
    return false;
  }
  mako::SampleBatch batch;
  batch = sample_batch;
  batch.set_batch_key(NextKey());
  Batches().push_back(batch);
  creation_response->set_key(batch.batch_key());
  *creation_response->mutable_status() = SuccessStatus();
  VLOG(2) << "Created SampleBatch with key: " << batch.batch_key();
  return true;
}

bool Storage::QuerySampleBatch(
    const mako::SampleBatchQuery& sample_batch_query,
    mako::SampleBatchQueryResponse* query_response) {
  VLOG(2) << "FakeStorage.QuerySampleBatch("
          << sample_batch_query.ShortDebugString() << ")";
  CHECK(query_response);
  absl::MutexLock lock(&mutex);
  FindMatchingElements(Batches(), sample_batch_query, batch_limit_max_,
                       query_response);
  VLOG(2) << query_response->sample_batch_list_size()
          << " sample batches found";
  return true;
}

bool Storage::DeleteSampleBatch(
    const mako::SampleBatchQuery& sample_batch_query,
    mako::ModificationResponse* mod_response) {
  VLOG(2) << "FakeStorage.DeleteSampleBatch("
          << sample_batch_query.ShortDebugString() << ")";
  CHECK(mod_response);
  absl::MutexLock lock(&mutex);
  auto itr = Batches().begin();
  while (itr != Batches().end()) {
    if (SampleBatchQueryMatch(sample_batch_query, *itr)) {
      itr = Batches().erase(itr);
      mod_response->set_count(mod_response->count() + 1);
    } else {
      ++itr;
    }
  }
  *mod_response->mutable_status() = SuccessStatus();
  return true;
}

std::string Storage::GetMetricValueCountMax(int* metric_value_max) {
  *metric_value_max = metric_value_count_max_;
  return kNoError;
}

std::string Storage::GetSampleErrorCountMax(int* sample_error_max) {
  *sample_error_max = error_count_max_;
  return kNoError;
}

std::string Storage::GetBatchSizeMax(int* batch_size_max) {
  *batch_size_max = batch_size_max_;
  return kNoError;
}

std::string Storage::GetHostname() {
  if (hostname_.empty()) {
    return "example.com";
  }
  return hostname_;
}

void Storage::FakeClear() {
  VLOG(2) << "FakeStorage.FakeClear()";
  absl::MutexLock lock(&mutex);
  Benchmarks().clear();
  Runs().clear();
  Batches().clear();
  max_key = 0;
}

void Storage::FakeStageBenchmarks(
    const std::vector<mako::BenchmarkInfo>& benchmark_info_list) {
  VLOG(2) << "FakeStorage.FakeStageBenchmarks()";
  absl::MutexLock lock(&mutex);
  for (auto& benchmark : benchmark_info_list) {
    Benchmarks().push_back(benchmark);
  }
}

void Storage::FakeStageRuns(
    const std::vector<mako::RunInfo>& run_info_list) {
  VLOG(2) << "FakeStorage.FakeStageRuns()";
  absl::MutexLock lock(&mutex);
  for (auto& run : run_info_list) {
    Runs().insert(run);
  }
}

void Storage::FakeStageBatches(
    const std::vector<mako::SampleBatch>& sample_batch_list) {
  VLOG(2) << "FakeStorage.FakeStageBatches()";
  absl::MutexLock lock(&mutex);
  for (auto& batch : sample_batch_list) {
    Batches().push_back(batch);
  }
}

}  // namespace fake_google3_storage
}  // namespace mako
