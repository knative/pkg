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
#ifndef INTERNAL_CXX_ANALYZER_OPTIMIZER_H_
#define INTERNAL_CXX_ANALYZER_OPTIMIZER_H_

#include <map>
#include <set>
#include <string>
#include <vector>

#include "spec/cxx/analyzer.h"
#include "spec/cxx/storage.h"
#include "spec/proto/mako.pb.h"
#include "internal/cxx/proto_cache.h"

namespace mako {
namespace internal {

constexpr int kDefaultRunInfoCacheSizeBytes = 10 * 1000 * 1000;
constexpr int kDefaultSampleBatchCacheSizeBytes = 16 * 1000 * 1000;

// AnalyzerOptimizer provides a more efficient ordering of Mako analyzers.
// It uses a storage query cache and reordering of analyzer calls to accomplish
// this.
//
// More information at go/mako-analyzer-optimizer
class AnalyzerOptimizer {
 public:
  // Specify the storage impl to delegate queries to.
  // Use default cache sizes.
  AnalyzerOptimizer(mako::Storage* storage,
                    const mako::RunBundle& current_run_bundle)
      : AnalyzerOptimizer(storage, current_run_bundle,
                          kDefaultRunInfoCacheSizeBytes,
                          kDefaultSampleBatchCacheSizeBytes) {}

  // Specify all options.
  AnalyzerOptimizer(mako::Storage* storage,
                    const mako::RunBundle& current_run_bundle,
                    int run_info_cache_size_bytes,
                    int sample_batch_cache_size_bytes)
      : storage_(storage),
        current_run_bundle_(current_run_bundle),
        run_info_cache_(run_info_cache_size_bytes),
        sample_batch_cache_(sample_batch_cache_size_bytes) {}

  // Called for each analyzer that you wish to call along with its output from
  // ConstructHistoricQuery().
  //
  // If return std::string is not empty then it contains an error message and no
  // output arguments are valid.
  std::string AddAnalyzer(
      const mako::AnalyzerHistoricQueryOutput& query_output,
      mako::Analyzer* analyzer);

  // Called once to obtain the optimal ordering of analyzers.
  //
  // If return std::string is not empty then it contains an error message and no
  // output arguments are valid.
  //
  // Otherwise analyzers vector is filled with pointers in the order they should
  // be called.
  std::string OrderAnalyzers(std::vector<mako::Analyzer*>* analyzers);

  // Called once per Analyzer to obtain data for that analyzer.
  //
  // If return std::string is not empty then it contains an error message and no
  // output arguments are valid.
  //
  // Otherwise analyzer_input is ready to pass into analyzer.Analyze() and
  // 'warnings' contains any warnings that occurred.
  //
  // A warning will be added when:
  //  - The results of a historic query includes the current run.
  //  - The results of a historic query are empty (eg. no runs match).
  //  - The results of a historic query match a different benchmark.
  //
  // This function will also filter out duplicate run keys from results. So only
  // the first occurrence of each run will be added to analyzer_input.
  std::string GetDataForAnalyzer(mako::Analyzer* analyzer,
                                 std::vector<std::string>* warnings,
                                 mako::AnalyzerInput* analyzer_input);

  // Returns a std::string summary of cache savings.
  std::string GetOptimizerSummary();

 private:
  std::string QueryRuns(const mako::RunInfoQuery& in_query,
                        mako::RunInfoQueryResponse* out_response);
  std::string AddDataForQuery(std::vector<std::string>* warnings,
                              bool need_batches,
                              const mako::RunInfoQuery& query,
                              std::set<std::string>* seen_run_keys,
                              const std::string& sample_key,
                              mako::AnalyzerInput* analyzer_input);
  std::string GetResponseForQuery(const mako::RunInfoQuery& query,
                                  mako::RunInfoQueryResponse* response);
  std::string AddHistoricalRuns(const mako::RunInfoQueryResponse& response,
                                const mako::RunInfoQuery& query,
                                bool need_batches,
                                std::set<std::string>* seen_run_keys,
                                std::vector<std::string>* warnings,
                                const std::string& sample_key,
                                mako::AnalyzerInput* analyzer_input);
  std::string AddSampleBatchesToRunBundle(mako::RunBundle* run_bundle);

  mako::Storage* storage_;
  mako::RunBundle current_run_bundle_;
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse>
      run_info_cache_;
  ProtoCache<mako::SampleBatchQuery, mako::SampleBatchQueryResponse>
      sample_batch_cache_;
  std::map<mako::Analyzer*, mako::AnalyzerHistoricQueryOutput>
      analyzer_to_query_;

  friend class AnalyzerOptimizerTest;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_ANALYZER_OPTIMIZER_H_
