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
//
// Functions below perform validation on the supplied protobuffers. An error
// string is returned, empty means successful.
//
// NOTE: Some validation is still left to the mako server (eg. valid
// characters in labels, etc). The purpose of this validation library is to
// ensure that protobufs can safely be processed.
#ifndef INTERNAL_CXX_PROTO_VALIDATION_H_
#define INTERNAL_CXX_PROTO_VALIDATION_H_

#include <string>

#include "spec/proto/mako.pb.h"

namespace mako {
namespace internal {

// Validate that all REQUIRED AggregatorInput fields are set. Returns error
// message as string or empty if successful.
std::string ValidateAggregatorInput(const mako::AggregatorInput& input);

// Validate that all REQUIRED RunInfo fields are set. Returns error
// message as string or empty if successful.
std::string ValidateRunInfo(const mako::RunInfo& input);

// Validate that all REQUIRED BenchmarkInfo fields are set. Returns error
// message as string or empty if successful.
std::string ValidateBenchmarkInfo(const mako::BenchmarkInfo& input);

// Validate that all REQUIRED SampleFile fields are set. Returns error
// message as string or empty if successful.
std::string ValidateSampleFile(const mako::SampleFile& input);

// Validate that all REQUIRED DownsamplerInput fields are set. Returns error
// message as string or empty if successful.
std::string ValidateDownsamplerInput(const mako::DownsamplerInput& input);

// Clears the aux_data field so it will not take up space on the
// server.
void StripAuxData(mako::SamplePoint* point);
// Overload included to allow template functions to compile.
void StripAuxData(mako::SampleError* error);

// Validate that all REQUIRED BenchmarkInfo fields are set for Benchmark
// creation. Returns error message as string or empty if successful.
std::string ValidateBenchmarkInfoCreationRequest(
    const mako::BenchmarkInfo& input);

// Validate that all REQUIRED RunInfo fields are set for RunInfo creation.
// Returns error message as string or empty if successful.
std::string ValidateRunInfoCreationRequest(const mako::RunInfo& input);

// Validate that all REQUIRED SampleBatch fields are set for SampleBatch
// creation.  Returns error message as string or empty if successful.
std::string ValidateSampleBatchCreationRequest(const mako::SampleBatch& input);

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_PROTO_VALIDATION_H_
