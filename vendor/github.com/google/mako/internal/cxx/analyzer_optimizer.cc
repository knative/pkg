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
#include "internal/cxx/analyzer_optimizer.h"

#include <algorithm>
#include <sstream>
#include <utility>

#include "glog/logging.h"
#include "spec/proto/mako.pb.h"
#include "absl/strings/str_cat.h"

namespace mako {
namespace internal {

constexpr char kNoError[] = "";
constexpr int kDefaultQueryLimit = 100;

std::string AnalyzerOptimizer::AddAnalyzer(
    const mako::AnalyzerHistoricQueryOutput& query_output,
    mako::Analyzer* analyzer) {
  VLOG(1) << "Analyzer: " << analyzer << " AnalyzerHistoricQueryOutput: "
          << query_output.ShortDebugString();
  analyzer_to_query_[analyzer] = query_output;
  return kNoError;
}

std::string AnalyzerOptimizer::OrderAnalyzers(
    std::vector<mako::Analyzer*>* analyzers) {
  for (auto pair : analyzer_to_query_) {
    analyzers->push_back(pair.first);
  }
  return kNoError;
}

std::string AnalyzerOptimizer::AddSampleBatchesToRunBundle(
    mako::RunBundle* run_bundle) {
  mako::SampleBatchQuery query;
  mako::SampleBatchQueryResponse response;
  query.set_run_key(run_bundle->run_info().run_key());
  query.set_benchmark_key(run_bundle->benchmark_info().benchmark_key());

  if (!sample_batch_cache_.Get(query, &response)) {
    VLOG(1) << "Cache miss for SampleBatchQuery";
    // Need to query for results
    if (!storage_->QuerySampleBatch(query, &response)) {
      std::string err = absl::StrCat(
          "Error running SampleBatch query: ", query.ShortDebugString(),
          ". Error: ", response.status().fail_message());
      LOG(ERROR) << err;
      return err;
    }
    // Place in cache for next time.
    sample_batch_cache_.Put(query, response);
  } else {
    VLOG(1) << "Cache hit for SampleBatchQuery.";
  }
  // Add to results
  for (const mako::SampleBatch& batch : response.sample_batch_list()) {
    *run_bundle->add_batch_list() = batch;
  }
  return kNoError;
}

std::string AnalyzerOptimizer::QueryRuns(
    const mako::RunInfoQuery& in_query,
    mako::RunInfoQueryResponse* out_response) {
  // Copy the query, because we want to change it without affecting caller
  mako::RunInfoQuery query(in_query);
  out_response->Clear();
  const int queryLimit = query.limit();
  if (queryLimit < 0 || queryLimit > 1000) {
    std::string err = absl::StrCat("Query limit is not in range [0,1000]: ",
                                   query.ShortDebugString());
    LOG(ERROR) << err;
    return err;
  }
  const int initLimit = queryLimit > 0 ? queryLimit : kDefaultQueryLimit;
  while (true) {
    query.set_limit(initLimit - out_response->run_info_list_size());
    if (!out_response->cursor().empty()) {
      query.set_cursor(out_response->cursor());
    }
    mako::RunInfoQueryResponse response;
    if (!storage_->QueryRunInfo(query, &response)) {
      std::string err =
          absl::StrCat("Error running RunInfoQuery: ", query.ShortDebugString(),
                       ". Error: ", response.status().fail_message());
      LOG(ERROR) << err;
      return err;
    }
    *out_response->mutable_status() = response.status();
    out_response->set_cursor(response.cursor());
    // NOTE: must maintain decending timestamp order for final response
    std::copy(response.run_info_list().begin(), response.run_info_list().end(),
              google::protobuf::RepeatedPtrFieldBackInserter(
                  out_response->mutable_run_info_list()));
    if (out_response->run_info_list_size() == initLimit ||
        response.run_info_list_size() == 0 || response.cursor().empty()) {
      break;
    }
  }
  return kNoError;
}

std::string AnalyzerOptimizer::AddDataForQuery(
    std::vector<std::string>* warnings, bool need_batches,
    const mako::RunInfoQuery& query, std::set<std::string>* seen_run_keys,
    const std::string& sample_key, mako::AnalyzerInput* analyzer_input) {
  VLOG(1) << "Processing RunInfoQuery: " << query.ShortDebugString()
          << " need batches: " << need_batches;
  mako::RunInfoQueryResponse response;
  std::string err = GetResponseForQuery(query, &response);
  if (!err.empty()) {
    err = absl::StrCat("Error querying for RunInfo: ", err);
    LOG(ERROR) << err;
    return err;
  }
  // Log first/last keys
  if (!response.run_info_list_size()) {
    LOG_STRING(WARNING, warnings)
        << "No results were returned from RunInfoQuery passed to analyzer: "
        << query.ShortDebugString();
    return kNoError;
  }
  LOG(INFO) << "Analyzer's run query: " << query.ShortDebugString();
  LOG(INFO) << "Returned " << response.run_info_list_size() << " runs";
  LOG(INFO) << "First run key from query: "
            << response.run_info_list(0).run_key();
  if (response.run_info_list_size() > 1) {
    LOG(INFO)
        << "Last run key from query: "
        << response.run_info_list(response.run_info_list_size() - 1).run_key();
  }

  return AddHistoricalRuns(response, query, need_batches, seen_run_keys,
                           warnings, sample_key, analyzer_input);
}

std::string AnalyzerOptimizer::AddHistoricalRuns(
    const mako::RunInfoQueryResponse& response,
    const mako::RunInfoQuery& query, bool need_batches,
    std::set<std::string>* seen_run_keys, std::vector<std::string>* warnings,
    const std::string& sample_key, mako::AnalyzerInput* analyzer_input) {
  // Copy results into analyzer_input
  for (const mako::RunInfo& run_info : response.run_info_list()) {
    auto inserted = (seen_run_keys->insert(run_info.run_key())).second;
    if (!inserted) {
      LOG(WARNING) << "Run " << run_info.run_key()
                   << " has already been seen; ignoring";
      continue;
    }

    if (run_info.run_key() == current_run_bundle_.run_info().run_key()) {
      LOG_STRING(WARNING, warnings)
          << "Query: " << query.ShortDebugString()
          << " returned current run_key: " << run_info.run_key()
          << ". This is probably not what you want.";
    }

    mako::RunBundle* run_bundle;
    if (sample_key.empty()) {
      run_bundle = analyzer_input->add_historical_run_list();
    } else {
      run_bundle = (*analyzer_input->mutable_historical_run_map())[sample_key]
                       .add_historical_run_list();
    }

    if (run_info.benchmark_key() ==
        current_run_bundle_.benchmark_info().benchmark_key()) {
      *run_bundle->mutable_benchmark_info() =
          current_run_bundle_.benchmark_info();
    } else {
      LOG_STRING(ERROR, warnings)
          << "Query: " << query.ShortDebugString()
          << " returned a RunInfo result from a different benchmark key ("
          << run_info.benchmark_key() << ") than you are currently testing ("
          << current_run_bundle_.benchmark_info().benchmark_key()
          << "). This is probably not what you want.";
      run_bundle->mutable_benchmark_info()->set_benchmark_key(
          run_info.benchmark_key());
    }
    *run_bundle->mutable_run_info() = run_info;
    if (need_batches) {
      std::string err = AddSampleBatchesToRunBundle(run_bundle);
      if (!err.empty()) {
        err = absl::StrCat("Error querying for SampleBatches. Error: ", err);
        LOG(ERROR) << err;
        return err;
      }
    }
  }
  return kNoError;
}

std::string AnalyzerOptimizer::GetResponseForQuery(
    const mako::RunInfoQuery& query,
    mako::RunInfoQueryResponse* response) {
  if (!run_info_cache_.Get(query, response)) {
    VLOG(1) << "Cache miss for RunInfoQuery.";
    // Cache miss, query storage then insert into cache.
    std::string err = QueryRuns(query, response);
    if (!err.empty()) {
      LOG(ERROR) << err;
      return err;
    }
    // Place into cache
    run_info_cache_.Put(query, *response);
  } else {
    VLOG(1) << "Cache hit for RunInfoQuery.";
  }
  return kNoError;
}

std::string AnalyzerOptimizer::GetDataForAnalyzer(
    mako::Analyzer* analyzer, std::vector<std::string>* warnings,
    mako::AnalyzerInput* analyzer_input) {
  auto it = analyzer_to_query_.find(analyzer);
  if (it == analyzer_to_query_.end()) {
    std::string err =
        "Could not find Analyzer instance. Did you call AddAnalyzer() with "
        "it?";
    LOG(ERROR) << err;
    return err;
  }
  analyzer_input->Clear();

  // Copy the current run to be analyzed into input.
  *analyzer_input->mutable_run_to_be_analyzed() = current_run_bundle_;

  // Get queries this analyzer needs to execute.
  mako::AnalyzerHistoricQueryOutput queries = it->second;

  if (queries.run_info_query_map_size()) {
    for (auto& it : queries.run_info_query_map()) {
      std::set<std::string> seen_run_keys;
      for (const mako::RunInfoQuery& query :
           it.second.run_info_query_list()) {
        std::string err =
            AddDataForQuery(warnings, queries.get_batches(), query,
                            &seen_run_keys, it.first, analyzer_input);
        if (!err.empty()) {
          LOG(ERROR) << err;
          return err;
        }
      }
    }
  } else if (queries.run_info_query_list_size()) {
    std::set<std::string> seen_run_keys;
    for (const mako::RunInfoQuery& query : queries.run_info_query_list()) {
      std::string err = AddDataForQuery(warnings, queries.get_batches(), query,
                                        &seen_run_keys, "", analyzer_input);
      if (!err.empty()) {
        LOG(ERROR) << err;
        return err;
      }
    }
  }

  return kNoError;
}

std::string AnalyzerOptimizer::GetOptimizerSummary() {
  std::stringstream ss;
  ss << "\n--AnalyzerOptimizer stats--\n";
  ss << run_info_cache_.Stats("RunInfoCache");
  ss << sample_batch_cache_.Stats("SampleBatchCache");
  return ss.str();
}

}  // namespace internal
}  // namespace mako
