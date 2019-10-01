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

#include "clients/cxx/storage/google3_storage.h"

#include <array>
#include <cstring>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "internal/proto/mako_internal.pb.h"
#include "absl/container/flat_hash_map.h"
#include "absl/flags/flag.h"
#include "absl/strings/ascii.h"
#include "absl/strings/str_split.h"
#include "absl/strings/string_view.h"
#include "absl/time/time.h"
#include "internal/cxx/proto_validation.h"
#include "internal/cxx/storage_client/retrying_storage_request.h"
#include "spec/proto/mako.pb.h"

ABSL_FLAG(
    std::string, mako_internal_storage_host, "",
    "If set, overrides the storage host which was passed to constructor.");

ABSL_FLAG(
    std::string, mako_client_tool_tag, "",
    "Allows clients to identify their workload. If this is not set we will "
    "use the google3_environment_collector to generate a tool tag based on "
    "the build target. This data is used for understanding usage patterns.");

ABSL_FLAG(
    std::string, mako_internal_sudo_run_as, "",
    "If set, runs the command as the specified identity. Should be of the "
    "form: user@google.com or group@prod.google.com. The server will check "
    "whether the caller has permission to use this feature.");

ABSL_FLAG(bool, mako_internal_force_trace, false,
          "Force a stackdriver trace on the server for all storage requests.");

ABSL_FLAG(std::string, mako_internal_test_pass_id_override, "",
          "If set, overrides the test_pass_id set by a user or the Mako "
          "framework. Useful for frameworks such as Chamber that need to group "
          "runs. Note this is only applied on RunInfo creation/update. If "
          "provided along with the mako_internal_test_pass_id_override "
          "environment variable, this will take precedence (the envrionment "
          "variable will be ignored).");

ABSL_FLAG(
    std::vector<std::string>, mako_internal_additional_tags, {},
    "Additional tags to attach to all created RunInfos.  Note that these tags "
    "are only added on RunInfo creation/update. Be aware of tag limits "
    "(go/mako-limits) when using this flag - the number of tags in the "
    "original RunInfo plus those added via this flag must not exceed the "
    "limit! If provided along with the mako_internal_additional_tags "
    "environment variable, this will take precedence (the envrionment variable "
    "will be ignored).");

namespace mako {
namespace google3_storage {
namespace {

using ::mako_internal::SudoStorageRequest;

constexpr char kNoError[] = "";
constexpr char kMakoStorageServer[] = "mako.dev";
constexpr char kCreateBenchmarkPath[] = "/storage/benchmark-info/create";
constexpr char kQueryBenchmarkPath[] = "/storage/benchmark-info/query";
constexpr char kModificationBenchmarkPath[] = "/storage/benchmark-info/update";
constexpr char kDeleteBenchmarkPath[] = "/storage/benchmark-info/delete";
constexpr char kCountBenchmarkPath[] = "/storage/benchmark-info/count";
constexpr char kCreateRunInfoPath[] = "/storage/run-info/create";
constexpr char kQueryRunInfoPath[] = "/storage/run-info/query";
constexpr char kModificationRunInfoPath[] = "/storage/run-info/update";
constexpr char kDeleteRunInfoPath[] = "/storage/run-info/delete";
constexpr char kCountRunInfoPath[] = "/storage/run-info/count";
constexpr char kCreateSampleBatchPath[] = "/storage/sample-batch/create";
constexpr char kQuerySampleBatchPath[] = "/storage/sample-batch/query";
constexpr char kDeleteSampleBatchPath[] = "/storage/sample-batch/delete";
constexpr char kSudoPath[] = "/storage/sudo";

// NOTE: Total time may exceed this by up to kRPCDeadline + kMaxSleep.
constexpr absl::Duration kDefaultOperationTimeout = absl::Minutes(3);
// Min and max amount of time we sleep between storage request retries.
constexpr absl::Duration kMinSleep = absl::Seconds(1);
constexpr absl::Duration kMaxSleep = absl::Seconds(30);

constexpr int kMetricValueCountMax = 50000;
constexpr int kSampleErrorCountMax = 5000;
constexpr int kBatchSizeMax = 1000000;

std::string ResolveClientToolTag() {
  std::string client_tool_tag = absl::GetFlag(FLAGS_mako_client_tool_tag);
  if (client_tool_tag.empty()) {
    if (client_tool_tag.empty()) {
      client_tool_tag = "unknown";
    }
  }
  return client_tool_tag;
}

void SetSudoStorageRequestPayload(const BenchmarkInfo& benchmark,
                                  SudoStorageRequest* request) {
  *request->mutable_benchmark() = benchmark;
}

void SetSudoStorageRequestPayload(const BenchmarkInfoQuery& benchmark_query,
                                  SudoStorageRequest* request) {
  *request->mutable_benchmark_query() = benchmark_query;
}

void SetSudoStorageRequestPayload(const RunInfo& run,
                                  SudoStorageRequest* request) {
  *request->mutable_run() = run;
}

void SetSudoStorageRequestPayload(const RunInfoQuery& run_query,
                                  SudoStorageRequest* request) {
  *request->mutable_run_query() = run_query;
}

void SetSudoStorageRequestPayload(const SampleBatch& batch,
                                  SudoStorageRequest* request) {
  *request->mutable_batch() = batch;
}

void SetSudoStorageRequestPayload(const SampleBatchQuery& batch_query,
                                  SudoStorageRequest* request) {
  *request->mutable_batch_query() = batch_query;
}

template <typename Request, typename Response>
bool RetryingStorageRequest(const Request& request, const std::string& url,
                            Response* response,
                            internal::StorageTransport* transport,
                            internal::StorageRetryStrategy* retry_strategy,
                            SudoStorageRequest::Type type) {
  static auto* actions = new absl::flat_hash_map<std::string, std::string>(
      {{kCreateBenchmarkPath, "Storage.CreateBenchmarkInfo"},
       {kQueryBenchmarkPath, "Storage.QueryBenchmarkInfo"},
       {kModificationBenchmarkPath, "Storage.UpdateBenchmarkInfo"},
       {kDeleteBenchmarkPath, "Storage.DeleteBenchmarkInfo"},
       {kCountBenchmarkPath, "Storage.CountBenchmarkInfo"},
       {kCreateRunInfoPath, "Storage.CreateRunInfo"},
       {kQueryRunInfoPath, "Storage.QueryRunInfo"},
       {kModificationRunInfoPath, "Storage.UpdateRunInfo"},
       {kDeleteRunInfoPath, "Storage.DeleteRunInfo"},
       {kCountRunInfoPath, "Storage.CountRunInfo"},
       {kCreateSampleBatchPath, "Storage.CreateSampleBatch"},
       {kQuerySampleBatchPath, "Storage.QuerySampleBatch"},
       {kDeleteSampleBatchPath, "Storage.DeleteSampleBatch"}});

  std::string telemetry_action = actions->at(url);
  const std::string& run_as = absl::GetFlag(FLAGS_mako_internal_sudo_run_as);
  if (run_as.empty()) {
    return internal::RetryingStorageRequest(
        request, url, telemetry_action, response, transport, retry_strategy);
  }

  SudoStorageRequest sudo_req;
  sudo_req.set_type(type);
  sudo_req.set_run_as(run_as);
  SetSudoStorageRequestPayload(request, &sudo_req);
  return internal::RetryingStorageRequest(sudo_req, kSudoPath, telemetry_action,
                                          response, transport, retry_strategy);
}

template <typename Response>
bool UploadRunInfo(
    const RunInfo& run_info, const std::string& path,
    internal::StorageTransport* transport,
    internal::StorageRetryStrategy* retry_strategy,
    Response* response, SudoStorageRequest::Type type) {
  // Look for mako_internal_additional_tags in both flags and environment
  // variables.  If found in both places, prefer the value from the flags.
  const char* env_var_additional_tags =
      std::getenv("mako_internal_additional_tags");
  std::vector<std::string> additional_tags =
      absl::GetFlag(FLAGS_mako_internal_additional_tags);
  if (additional_tags.empty() && env_var_additional_tags) {
    additional_tags = absl::StrSplit(env_var_additional_tags, ',',
                                     absl::SkipWhitespace());
  }
  // Look for mako_internal_test_pass_id_override in both flags and
  // environment variables.  If found in both places, prefer the value from the
  // flags.
  const char* env_var_test_pass_id_override =
      std::getenv("mako_internal_test_pass_id_override");
  std::string test_pass_id_override =
      absl::GetFlag(FLAGS_mako_internal_test_pass_id_override);
  if (test_pass_id_override.empty() && env_var_test_pass_id_override) {
    test_pass_id_override = env_var_test_pass_id_override;
  }
  if (additional_tags.empty() && test_pass_id_override.empty()) {
    return RetryingStorageRequest(run_info, path, response, transport,
                                  retry_strategy, type);
  }
  // copy proto to modify it based on global flags
  RunInfo final_run_info = run_info;
  // add any additional tags requested by flag, ensuring that all tags are
  // unique and ordered as given using a std::set (vs an absl::flat_hash_set,
  // which is unordered)
  std::vector<absl::string_view> tags(run_info.tags().begin(),
                                      run_info.tags().end());
  std::set<absl::string_view> unique_tags(run_info.tags().begin(),
                                          run_info.tags().end());
  for (const std::string& tag : additional_tags) {
    absl::string_view no_whitespace_tag = absl::StripAsciiWhitespace(tag);
    LOG(INFO) << "Adding new tag " << no_whitespace_tag << " to run";
    auto res = unique_tags.insert(no_whitespace_tag);
    if (res.second) {
      tags.emplace_back(no_whitespace_tag);
    }
  }
  final_run_info.clear_tags();
  for (const auto& tag : tags) {
    *final_run_info.add_tags() = std::string(tag);
  }
  // TODO(b/136285571) reference this limit from some common location where it
  // is defined (instead of redefining it here)
  int tag_limit = 20;
  if (final_run_info.tags_size() > tag_limit) {
    std::string err_msg =
        "This run has too many tags; cannot add it to mako storage!";
    LOG(ERROR) << err_msg;
    response->mutable_status()->set_fail_message(err_msg);
    response->mutable_status()->set_code(mako::Status::FAIL);
    return false;
  }

  if (!test_pass_id_override.empty()) {
    LOG(INFO) << "Overriding test pass id for run: changing "
              << final_run_info.test_pass_id() << " to "
              << test_pass_id_override << ".";
    final_run_info.set_test_pass_id(test_pass_id_override);
  }
  return RetryingStorageRequest(final_run_info, path, response, transport,
                                retry_strategy, type);
}

}  // namespace

Storage::Storage(
    std::unique_ptr<mako::internal::StorageTransport> transport)
    : Storage(std::move(transport),
              absl::make_unique<mako::internal::StorageBackoff>(
                  kDefaultOperationTimeout, kMinSleep, kMaxSleep)) {}

Storage::Storage(
    std::unique_ptr<mako::internal::StorageTransport> transport,
    std::unique_ptr<mako::internal::StorageRetryStrategy> retry_strategy)
    : transport_(std::move(transport)),
      retry_strategy_(std::move(retry_strategy)) {
  transport_->set_client_tool_tag(ResolveClientToolTag());
}

bool Storage::CreateBenchmarkInfo(const BenchmarkInfo& benchmark_info,
                                  CreationResponse* creation_response) {
  const std::string& path = kCreateBenchmarkPath;
  return RetryingStorageRequest(benchmark_info, path, creation_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::CREATE_BENCHMARK_INFO);
}

bool Storage::UpdateBenchmarkInfo(const BenchmarkInfo& benchmark_info,
                                  ModificationResponse* mod_response) {
  const std::string& path = kModificationBenchmarkPath;
  return RetryingStorageRequest(benchmark_info, path, mod_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::UPDATE_BENCHMARK_INFO);
}

bool Storage::QueryBenchmarkInfo(const BenchmarkInfoQuery& benchmark_info_query,
                                 BenchmarkInfoQueryResponse* query_response) {
  const std::string& path = kQueryBenchmarkPath;
  return RetryingStorageRequest(benchmark_info_query, path, query_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::QUERY_BENCHMARK_INFO);
}

bool Storage::DeleteBenchmarkInfo(
    const BenchmarkInfoQuery& benchmark_info_query,
    ModificationResponse* mod_response) {
  const std::string& path = kDeleteBenchmarkPath;
  return RetryingStorageRequest(benchmark_info_query, path, mod_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::DELETE_BENCHMARK_INFO);
}

bool Storage::CountBenchmarkInfo(const BenchmarkInfoQuery& benchmark_info_query,
                                 CountResponse* count_response) {
  const std::string& path = kCountBenchmarkPath;
  return RetryingStorageRequest(benchmark_info_query, path, count_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::COUNT_BENCHMARK_INFO);
}

bool Storage::CreateRunInfo(const RunInfo& run_info,
                            CreationResponse* creation_response) {
  return UploadRunInfo(run_info, kCreateRunInfoPath, transport_.get(),
                       retry_strategy_.get(), creation_response,
                       SudoStorageRequest::CREATE_RUN_INFO);
}

bool Storage::UpdateRunInfo(const RunInfo& run_info,
                            ModificationResponse* mod_response) {
  return UploadRunInfo(run_info, kModificationRunInfoPath, transport_.get(),
                       retry_strategy_.get(), mod_response,
                       SudoStorageRequest::UPDATE_RUN_INFO);
}

bool Storage::QueryRunInfo(const RunInfoQuery& run_info_query,
                           RunInfoQueryResponse* query_response) {
  const std::string& path = kQueryRunInfoPath;
  return RetryingStorageRequest(run_info_query, path, query_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::QUERY_RUN_INFO);
}

bool Storage::DeleteRunInfo(const RunInfoQuery& run_info_query,
                            ModificationResponse* mod_response) {
  const std::string& path = kDeleteRunInfoPath;
  return RetryingStorageRequest(run_info_query, path, mod_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::DELETE_RUN_INFO);
}

bool Storage::CountRunInfo(const RunInfoQuery& run_info_query,
                           CountResponse* count_response) {
  const std::string& path = kCountRunInfoPath;
  return RetryingStorageRequest(run_info_query, path, count_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::COUNT_RUN_INFO);
}

bool Storage::CreateSampleBatch(const SampleBatch& sample_batch,
                                CreationResponse* creation_response) {
  const std::string& path = kCreateSampleBatchPath;
  // Make a copy every time to keep code simpler and readable. Cost of copy in
  // the absolute worst case (50000 metrics) (~4ms) is small
  // compared to the cost of the request to the server (~200ms).
  SampleBatch modified_batch = sample_batch;
  for (auto& point : *modified_batch.mutable_sample_point_list()) {
    if (point.aux_data_size() > 0) {
      LOG_FIRST_N(WARNING, 1)
          << "Attempting to create a SampleBatch which contains SamplePoints "
             "with "
             "Aux Data. Aux Data is not displayed on the server, and should be "
             "stripped out before being sent as not to take up valuable space. "
             "This normally happens in the default Downsampler. If your "
             "Mako "
             "test is using the default downsampler and you are seeing this "
             "message, please file a bug at go/mako-bug.";
      internal::StripAuxData(&point);
    }
  }
  return RetryingStorageRequest(modified_batch, path, creation_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::CREATE_SAMPLE_BATCH);
}

bool Storage::QuerySampleBatch(const SampleBatchQuery& sample_batch_query,
                               SampleBatchQueryResponse* query_response) {
  const std::string& path = kQuerySampleBatchPath;
  return RetryingStorageRequest(sample_batch_query, path, query_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::QUERY_SAMPLE_BATCH);
}

bool Storage::DeleteSampleBatch(const SampleBatchQuery& sample_batch_query,
                                ModificationResponse* mod_response) {
  const std::string& path = kDeleteSampleBatchPath;
  return RetryingStorageRequest(sample_batch_query, path, mod_response,
                                transport_.get(), retry_strategy_.get(),
                                SudoStorageRequest::DELETE_SAMPLE_BATCH);
}

std::string Storage::GetMetricValueCountMax(int* metric_count_max) {
  *metric_count_max = kMetricValueCountMax;
  return kNoError;
}

std::string Storage::GetSampleErrorCountMax(int* sample_error_max) {
  *sample_error_max = kSampleErrorCountMax;
  return kNoError;
}

std::string Storage::GetBatchSizeMax(int* batch_size_max) {
  *batch_size_max = kBatchSizeMax;
  return kNoError;
}

std::string ApplyHostnameFlagOverrides(const std::string& hostname) {
  const std::string hostname_override =
      absl::GetFlag(FLAGS_mako_internal_storage_host);
  if (!hostname_override.empty()) {
    LOG(WARNING) << "Overriding constructor-supplied hostname of '" << hostname
                 << "' with flag value '" << hostname_override << "'";
    return hostname_override;
  }
  return hostname;
}
}  // namespace google3_storage
}  // namespace mako
