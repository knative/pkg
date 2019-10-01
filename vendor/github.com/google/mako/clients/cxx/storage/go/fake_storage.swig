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
// Wraps Mako's C++ Fake Storage client for Go.
//
// This should not be used directly, but rather through the fake_google3_storage.go
// wrapper, which provides a more Go-friendly API and documentation.
%include "swig/base.swig"
%include "swig/go/protofunc.swig"
%include "swig/go/strings.swig"

%{
#include "clients/cxx/storage/fake_google3_storage.h"
%}

PROTO_INPUT(
  mako::BenchmarkInfo,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.BenchmarkInfo)

PROTO_INPUT(
  mako::RunInfo,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.RunInfo)

PROTO_INPUT(
  mako::BenchmarkInfoQuery,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.BenchmarkInfoQuery)

PROTO_INPUT(
  mako::RunInfoQuery,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.RunInfoQuery)

PROTO_INPUT(
  mako::SampleBatch,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.SampleBatch)

PROTO_INPUT(
  mako::SampleBatchQuery,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.SampleBatchQuery)

PROTO_IN_OUT(
  mako::CreationResponse,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.CreationResponse)

PROTO_IN_OUT(
  mako::ModificationResponse,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.ModificationResponse)

PROTO_IN_OUT(
  mako::CountResponse,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.CountResponse)

PROTO_IN_OUT(
  mako::BenchmarkInfoQueryResponse,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.BenchmarkInfoQueryResponse)

PROTO_IN_OUT(
  mako::RunInfoQueryResponse,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.RunInfoQueryResponse)

PROTO_IN_OUT(
  mako::SampleBatchQueryResponse,
  pgpb "github.com/google/mako/spec/proto/mako_go_proto",
  pgpb.SampleBatchQueryResponse)

C_PROTO_VECTOR_INPUT_SLICE(mako::BenchmarkInfo)
C_PROTO_VECTOR_INPUT_SLICE(mako::RunInfo)
C_PROTO_VECTOR_INPUT_SLICE(mako::SampleBatch)

// Exposed API
%ignoreall

%unignore mako;
%unignore mako::fake_google3_storage;
%unignore mako::fake_google3_storage::Storage;
%unignore mako::fake_google3_storage::Storage::Storage();
%unignore mako::fake_google3_storage::Storage::Storage(int, int, int, int, int, int);
%unignore mako::fake_google3_storage::Storage::~Storage;

%unignore mako::fake_google3_storage::Storage::CreateBenchmarkInfo;
%unignore mako::fake_google3_storage::Storage::UpdateBenchmarkInfo;
%unignore mako::fake_google3_storage::Storage::QueryBenchmarkInfo;
%unignore mako::fake_google3_storage::Storage::DeleteBenchmarkInfo;
%unignore mako::fake_google3_storage::Storage::CountBenchmarkInfo;

%unignore mako::fake_google3_storage::Storage::CreateRunInfo;
%unignore mako::fake_google3_storage::Storage::UpdateRunInfo;
%unignore mako::fake_google3_storage::Storage::QueryRunInfo;
%unignore mako::fake_google3_storage::Storage::DeleteRunInfo;
%unignore mako::fake_google3_storage::Storage::CountRunInfo;

%unignore mako::fake_google3_storage::Storage::CreateSampleBatch;
%unignore mako::fake_google3_storage::Storage::QuerySampleBatch;
%unignore mako::fake_google3_storage::Storage::DeleteSampleBatch;

%unignore mako::fake_google3_storage::Storage::GetMetricValueCountMax;
%unignore mako::fake_google3_storage::Storage::GetSampleErrorCountMax;
%unignore mako::fake_google3_storage::Storage::GetBatchSizeMax;
%unignore mako::fake_google3_storage::Storage::GetHostname;

%unignore mako::fake_google3_storage::Storage::FakeClear;
%unignore mako::fake_google3_storage::Storage::FakeStageBenchmarks;
%unignore mako::fake_google3_storage::Storage::FakeStageRuns;
%unignore mako::fake_google3_storage::Storage::FakeStageBatches;

%import "spec/cxx/storage.h"
%include "clients/cxx/storage/fake_google3_storage.h"

%unignoreall
