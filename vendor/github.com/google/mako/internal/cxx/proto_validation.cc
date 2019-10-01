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

// Perform validation on REQUIRED fields in mako protocol buffers.
#include "internal/cxx/proto_validation.h"

#include <string>
#include <unordered_set>

#include "absl/strings/str_cat.h"
#include "absl/strings/str_format.h"
#include "spec/proto/mako.pb.h"

namespace {
constexpr char kNoError[] = "";

constexpr int kMaxUrlLength = 2000;

std::string ValidateRunInfoSharedFields(const mako::RunInfo& input) {
  if (!input.has_benchmark_key() || input.benchmark_key().empty()) {
    return "RunInfo.benchmark_key empty";
  }
  if (!input.has_timestamp_ms() || input.timestamp_ms() < 0) {
    return "RunInfo.timestamp_ms missing/invalid";
  }
  return kNoError;
}

std::string ValidateBenchmarkInfoSharedFields(const mako::BenchmarkInfo& input) {
  if (!input.has_benchmark_name() || input.benchmark_name().empty()) {
    return "BenchmarkInfo.benchmark_name missing/empty";
  }
  if (!input.has_project_name() || input.project_name().empty()) {
    return "BenchmarkInfo.project_name missing/empty";
  }
  if (input.owner_list_size() == 0) {
    return "BenchmarkInfo.owner_list missing";
  }
  if (!input.has_input_value_info()) {
    return "BenchmarkInfo.input_value_info missing/empty";
  }
  // NOTE: Allow server to enforce label and value key restrictions.
  if (!input.input_value_info().has_label() ||
      input.input_value_info().label().empty()) {
    return "BenchmarkInfo.input_value_info.label missing/empty";
  }
  if (!input.input_value_info().has_value_key() ||
      input.input_value_info().value_key().empty()) {
    return "BenchmarkInfo.input_value_info.value_info missing/empty";
  }
  return kNoError;
}

std::string ValidateSampleBatchSharedFields(const mako::SampleBatch& input) {
  if (!input.has_benchmark_key() || input.benchmark_key().empty()) {
    return "SampleBatch.benchmark_key missing";
  }
  if (!input.has_run_key() || input.run_key().empty()) {
    return "SampleBatch.run_key missing";
  }
  int total = input.sample_point_list_size() + input.sample_error_list_size();
  if (total == 0) {
    return "Either SampleBatch.sample_point_list or "
           "SampleBatch.sample_error_list must have data";
  }
  return kNoError;
}
}  // namespace

namespace mako {
namespace internal {

std::string ValidateRunInfo(const mako::RunInfo& input) {
  if (!input.has_run_key() || input.run_key().empty()) {
    return "RunInfo.run_key empty";
  }
  return ValidateRunInfoSharedFields(input);
}

std::string ValidateBenchmarkInfo(const mako::BenchmarkInfo& input) {
  if (!input.has_benchmark_key() || input.benchmark_key().empty()) {
    return "BenchmarkInfo.benchmark_key missing/empty";
  }
  return ValidateBenchmarkInfoSharedFields(input);
}

std::string ValidateAggregatorInput(const mako::AggregatorInput& input) {
  if (!input.has_benchmark_info()) {
    return "AggregatorInput.benchmark_info missing";
  }
  std::string err = ValidateBenchmarkInfo(input.benchmark_info());
  if (!err.empty()) {
    return absl::StrCat("AggregatorInput.benchmark_info error: ", err);
  }
  if (!input.has_run_info()) {
    return "AggregatorInput.run_info missing";
  }
  err = ValidateRunInfo(input.run_info());
  if (!err.empty()) {
    return absl::StrCat("AggregatorInput.run_info error: ", err);
  }

  for (int i = 0; i < input.sample_file_list_size(); i++) {
    err = ValidateSampleFile(input.sample_file_list(i));
    if (!err.empty()) {
      return absl::StrCat("AggregatorInput.sample_file_list[", i,
                          "] error: ", err);
    }
  }
  return kNoError;
}

std::string ValidateSampleFile(const mako::SampleFile& input) {
  if (!input.has_sampler_name() || input.sampler_name().empty()) {
    return "SampleFile.sampler_name missing/empty";
  }
  if (!input.has_file_path() || input.file_path().empty()) {
    return "SampleFile.file_path missing/empty";
  }
  return kNoError;
}

std::string ValidateDownsamplerInput(const mako::DownsamplerInput& input) {
  if (!input.has_run_info()) {
    return "DownsamplerInput.run_info missing";
  }
  std::string err = ValidateRunInfo(input.run_info());
  if (!err.empty()) {
    return absl::StrCat("DownsamplerInput.run_info error: ", err);
  }

  for (int i = 0; i < input.sample_file_list_size(); i++) {
    err = ValidateSampleFile(input.sample_file_list(i));
    if (!err.empty()) {
      return absl::StrCat("DownsamplerInput.sample_file_list[", i,
                          "] error: ", err);
    }
  }
  if (!input.has_metric_value_count_max() ||
      input.metric_value_count_max() < 0) {
    return "DownsamplerInput.metric_value_count_max missing/invalid";
  }
  if (!input.has_sample_error_count_max() ||
      input.sample_error_count_max() < 0) {
    return "DownsamplerInput.sample_error_count_max missing/invalid";
  }
  if (!input.has_batch_size_max() || input.batch_size_max() < 0) {
    return "DownsamplerInput.batch_size_max missing/invalid";
  }
  return kNoError;
}

void StripAuxData(mako::SamplePoint* point) {
  point->mutable_aux_data()->clear();
}
void StripAuxData(mako::SampleError* error) {}

std::string ValidateBenchmarkInfoCreationRequest(
    const mako::BenchmarkInfo& input) {
  if (input.has_benchmark_key() && !input.benchmark_key().empty()) {
    return "BenchmarkInfo.benchmark_key should be missing/empty";
  }
  return ValidateBenchmarkInfoSharedFields(input);
}

std::string ValidateRunInfoCreationRequest(const mako::RunInfo& input) {
  if (input.has_run_key() && !input.run_key().empty()) {
    return "RunInfo.run_key should be missing/empty.";
  }

  return ValidateRunInfoSharedFields(input);
}

std::string ValidateSampleBatchCreationRequest(const mako::SampleBatch& input) {
  if (input.has_batch_key() && !input.batch_key().empty()) {
    return "SampleBatch.batch_key should be missing.";
  }
  return ValidateSampleBatchSharedFields(input);
}

}  // namespace internal
}  // namespace mako
