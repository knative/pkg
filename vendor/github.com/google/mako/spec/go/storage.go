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

package mako

import (
	"context"

	pgpb "github.com/google/mako/spec/proto/mako_go_proto"
)

// Storage interface used to store data inside Mako.
type Storage interface {
	// CreateBenchmarkInfo creates a new BenchmarkInfo record.
	//
	// The BenchmarkInfo reference that is passed, should contain all the
	// information that you wish to have stored in the newly created benchmark.
	// (Although benchmark_info.benchmark_key should not be filled in. That value
	// will be returned in the CreationResponse and used on subsequent queries.)
	//
	// The result of the creation request will be placed in the returned
	// CreationResponse message.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as CreationResponse.Status.fail_message during error).
	//
	// NOTE: Some implementations may not implement this function, and the user
	// may need to use a tool or dashboard to create the benchmark.
	//
	// NOTE: You should not be calling this function often. Benchmarks are meant
	// to be reused for extended periods see go/mako-help for more information.
	CreateBenchmarkInfo(context.Context, *pgpb.BenchmarkInfo) (*pgpb.CreationResponse, error)

	// UpdateBenchmarkInfo updates the specified BenchmarkInfo record.
	//
	// The BenchmarkInfo.benchmark_key will be used to select the benchmark that
	// you wish to update. All information in the provided BenchmarkInfo
	// will overwrite the existing record.
	//
	// The results of the operation will be placed in ModificiationResponse.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as ModificationResponse.Status.fail_message during
	// error).
	//
	// This must be called whenever you modify metric or custom aggregation
	// information.
	//
	// See go/mako-help for more information.
	//
	// NOTE: Requires BenchmarkInfo.benchmark_key
	UpdateBenchmarkInfo(context.Context, *pgpb.BenchmarkInfo) (*pgpb.ModificationResponse, error)

	// QueryBenchmarkInfo queries the storage system and returns BenchmarkInfo
	// results.
	//
	// You'll always get the best performance when supplying the benchmark_key, if
	// that is set, all other query params will be ignored.
	//
	// The BenchmarkInfoQueryResponse will be populated with results.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as BenchmarkInfoQueryResponse.Status.fail_message
	// during error).
	QueryBenchmarkInfo(context.Context, *pgpb.BenchmarkInfoQuery) (*pgpb.BenchmarkInfoQueryResponse, error)

	// DeleteBenchmarkInfo deletes the specified BenchmarkInfo record.
	//
	// Delete requests use the same query messages as normal queries to identify
	// data that should be deleted although only the benchmark_key field is used.
	// All other fields are ignored (eg. limit, cursor, project_name, etc).
	//
	// The results of the operation will be placed in ModificiationResponse.
	//
	// NOTE: Deleting a benchmark requires all child (eg. run info, sample
	// batches, etc) to first be deleted.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as ModificationResponse.Status.fail_message during
	// error).
	DeleteBenchmarkInfo(context.Context, *pgpb.BenchmarkInfoQuery) (*pgpb.ModificationResponse, error)

	// CountBenchmarkInfo counts BenchmarkInfo records that match the BenchmarkInfoQuery.
	//
	// Depending on the implementation, the Count operation may be significantly
	// cheaper and/or faster than QueryBenchmarkInfo.
	//
	// CountResponse will be reset to default values and then populated with the
	// results of the count.
	//
	// The boolean returned represents success (true) or failure (false) of the
	// operation. More details about the success/failure will be in
	// CountResponse.Status.
	CountBenchmarkInfo(context.Context, *pgpb.BenchmarkInfoQuery) (*pgpb.CountResponse, error)

	// CreateRunInfo creates a new RunInfo record.
	//
	// The RunInfo reference that is passed, should contain all the
	// information that you wish to have stored in the newly created run info.
	// (Although run_info.run_key should not be filled in. That value
	// will be returned in the CreationResponse and used on subsequent queries.)
	//
	// The result of the creation request will be placed in the returned
	// CreationResponse message.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as CreationResponse.Status.fail_message during error).
	CreateRunInfo(context.Context, *pgpb.RunInfo) (*pgpb.CreationResponse, error)

	// UpdateRunInfo updates the specified RunInfo record.
	//
	// The RunInfo.run_key will be used to select the run info that
	// you wish to update. All information in the provided RunInfo
	// will overwrite the existing record.
	//
	// The results of the operation will be placed in ModificiationResponse.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as ModificationResponse.Status.fail_message during
	// error).
	//
	// This must be called whenever you modify metric or custom aggregation
	// information.
	//
	// NOTE: Requires RunInfo.run_key
	UpdateRunInfo(context.Context, *pgpb.RunInfo) (*pgpb.ModificationResponse, error)

	// QueryRunInfo queries the storage system and returns RunInfo
	// results.
	//
	// You'll always get the best performance when supplying the run_key, if
	// that is set, all other query params will be ignored.
	//
	// The RunInfoQueryResponse will be populated with results.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as RunInfoQueryResponse.Status.fail_message
	// during error).
	QueryRunInfo(context.Context, *pgpb.RunInfoQuery) (*pgpb.RunInfoQueryResponse, error)

	// DeleteRunInfo deletes the specified RunInfo and child SampleBatch data.
	//
	// Delete requests use the same query messages as normal queries to identify
	// data that should be deleted. If run_key is provided then all other fields
	// are ignored. To prevent accidental deletion of large amounts of data the
	// benchmark_key is also required.
	//
	// Also deletes child SampleBatch data for each run that matches the query.
	//
	// The results of the operation will be placed in ModificiationResponse.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as ModificationResponse.Status.fail_message during
	// error).
	DeleteRunInfo(context.Context, *pgpb.RunInfoQuery) (*pgpb.ModificationResponse, error)

	// CountRunInfo counts RunInfo records matching the RunInfoQuery.
	//
	// Depending on the implementation, the Count operation may be significantly
	// cheaper and/or faster than the QueryInfo.
	//
	// The results of the operation will be placed in CountResponse.
	//
	// The boolean returned represents success (true) or failure (false) of the
	// operation. More details about the success/failure will be in
	// CountResponse.Status.
	CountRunInfo(context.Context, *pgpb.RunInfoQuery) (*pgpb.CountResponse, error)

	// CreateSampleBatch creates a new SampleBatch record.
	//
	// The SampleBatch reference that is passed, should contain all the
	// information that you wish to have stored in the newly created run info.
	// (Although sample_batch.batch_key should not be filled in. That value
	// will be returned in the CreationResponse and used on subsequent queries.)
	//
	// The result of the creation request will be placed in the returned
	// CreationResponse message.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as CreationResponse.Status.fail_message during error).
	CreateSampleBatch(context.Context, *pgpb.SampleBatch) (*pgpb.CreationResponse, error)

	// QuerySampleBatch queries the storage system and returns SampleBatch
	// results.
	//
	// You'll always get the best performance when supplying the batch_key, if
	// that is set, all other query params will be ignored.
	//
	// The SampleBatchQueryResponse will be populated with results.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as SampleBatchQueryResponse.Status.fail_message
	// during error).
	QuerySampleBatch(context.Context, *pgpb.SampleBatchQuery) (*pgpb.SampleBatchQueryResponse, error)

	// DeleteSampleBatch deletes the specified SampleBatch record.
	//
	// Delete requests use the same query messages as normal queries to identify
	// data that should be deleted. If batch_key is provided then all other fields
	// are ignored. To prevent accidental deletion of large amounts of data the
	// benchmark_key is also required.
	//
	// Calling this is uncommon. Deleting a RunInfo will delete the child batches.
	//
	// The results of the operation will be placed in ModificiationResponse.
	//
	// The returned error will be nil if the request was successful, otherwise it
	// will contain the reason that the request failed (error.Error() contains
	// the same information as ModificationResponse.Status.fail_message during
	// error).
	DeleteSampleBatch(context.Context, *pgpb.SampleBatchQuery) (*pgpb.ModificationResponse, error)

	// GetMetricValueCountMax returns the max number of metric values that can be
	// saved per run.
	GetMetricValueCountMax(context.Context) (int, error)

	// GetSampleErrorCountMax returns the max number of errors that can be saved
	// per run.
	GetSampleErrorCountMax(context.Context) (int, error)

	// GetBatchSizeMax returns max binary size in base 10 bytes of a SampleBatch.
	// (eg. 1MB == 1,000,000)
	GetBatchSizeMax(context.Context) (int, error)

	// GetHostname returns the hostname backing this Storage implementation.
	GetHostname(context.Context) string
}
