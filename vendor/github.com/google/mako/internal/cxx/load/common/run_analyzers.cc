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
#include "internal/cxx/load/common/run_analyzers.h"

#include <algorithm>
#include <functional>
#include <iterator>
#include <string>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/map.h"
#include "spec/cxx/aggregator.h"
#include "spec/cxx/storage.h"
#include "absl/strings/match.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_format.h"
#include "helpers/cxx/status/status.h"
#include "internal/cxx/analyzer_optimizer.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace internal {

namespace {
constexpr char kNoError[] = "";

static std::string join(const std::vector<std::string>& strings) {
  std::ostringstream output;
  std::copy(strings.begin(), strings.end(),
            std::ostream_iterator<std::string>(output, "\n"));
  return output.str();
}

void SetAnalyzerTypeAndName(mako::Analyzer* analyzer,
                            mako::AnalyzerOutput* output) {
  if (!output->has_analyzer_type()) {
    output->set_analyzer_type(analyzer->analyzer_type());
  }
  if (!output->has_analyzer_name()) {
    output->set_analyzer_name(analyzer->analyzer_name());
  }
}
}  // namespace

std::string RunAnalyzers(
    const mako::BenchmarkInfo& benchmark_info,
    const mako::RunInfo& run_info,
    const std::vector<mako::SampleBatch>& sample_batches,
    bool attach_e_divisive_regressions_to_changepoints,
    mako::Storage* storage, mako::Dashboard* dashboard,
    const std::vector<mako::Analyzer*>& analyzers,
    mako::TestOutput* test_output) {
  LOG(INFO) << "RunAnalyzers()";
  std::string err;

  // Create the current run_bundle
  mako::RunBundle current_run_bundle;
  *current_run_bundle.mutable_benchmark_info() = benchmark_info;
  *current_run_bundle.mutable_run_info() = run_info;
  for (const auto& sample_batch : sample_batches) {
    *current_run_bundle.add_batch_list() = sample_batch;
  }

  // Create AnalyzerHistoricQueryInput
  mako::AnalyzerHistoricQueryInput historic_query_input;
  *historic_query_input.mutable_benchmark_info() = benchmark_info;
  *historic_query_input.mutable_run_info() = run_info;

  // Create AnalyzerOptimizer
  AnalyzerOptimizer optimizer(storage, current_run_bundle);

  // Keep track of different AnalyzerOutputs from each analyzer.
  std::vector<mako::AnalyzerOutput> failures;
  std::vector<mako::AnalyzerOutput> regressions;
  std::vector<mako::AnalyzerOutput> successes;

  // These warnings will be added to TestOutput.summary_output
  std::vector<std::string> warnings;

  for (mako::Analyzer* analyzer : analyzers) {
    CHECK(analyzer);

    // Collect historic query output
    mako::AnalyzerHistoricQueryOutput historic_query_output;
    if (!analyzer->ConstructHistoricQuery(historic_query_input,
                                          &historic_query_output)) {
      err =
          absl::StrCat("Analyzer ConstructHistoricQuery failed with message: ",
                       historic_query_output.status().fail_message());
      LOG(ERROR) << err;
      mako::AnalyzerOutput err_output;
      err_output.mutable_status()->set_code(mako::Status::FAIL);
      err_output.mutable_status()->set_fail_message(err);
      SetAnalyzerTypeAndName(analyzer, &err_output);
      failures.push_back(err_output);
      continue;
    }

    // Associate this analyzer with these queries
    err = optimizer.AddAnalyzer(historic_query_output, analyzer);
    if (!err.empty()) {
      err = absl::StrCat("AnalyzerOptimizer error adding analyzer: ", err);
      LOG(ERROR) << err;
      return err;
    }
  }

  std::vector<mako::Analyzer*> optimal_order;
  err = optimizer.OrderAnalyzers(&optimal_order);
  if (!err.empty()) {
    err = absl::StrCat("AnalyzerOptimizer error ordering analyzers: ", err);
    LOG(ERROR) << err;
    return err;
  }

  // Run each analyzer
  for (mako::Analyzer* analyzer : optimal_order) {
    mako::AnalyzerInput analyzer_input;
    mako::AnalyzerOutput analyzer_output;

    SetAnalyzerTypeAndName(analyzer, &analyzer_output);

    err = optimizer.GetDataForAnalyzer(analyzer, &warnings, &analyzer_input);
    // This could be a storage error. Package into an AnalyzerOutput.
    if (!err.empty()) {
      LOG(ERROR) << err;
      analyzer_output.mutable_status()->set_code(mako::Status::FAIL);
      analyzer_output.mutable_status()->set_fail_message(err);
      failures.push_back(std::move(analyzer_output));
      continue;
    }

    bool success;
    {
      success = analyzer->Analyze(analyzer_input, &analyzer_output);
    }

    if (!success ||
        analyzer_output.status().code() != mako::Status::SUCCESS) {
      LOG(ERROR) << analyzer_output.status().fail_message();
      failures.push_back(std::move(analyzer_output));
      continue;
    }

    std::vector<mako::AnalyzerOutput> analyzer_outputs;
    analyzer_outputs.push_back(std::move(analyzer_output));
    for (auto& analyzer_output : analyzer_outputs) {
      if (analyzer_output.status().code() != mako::Status::SUCCESS) {
        LOG(ERROR) << analyzer_output.status().fail_message();
        failures.push_back(std::move(analyzer_output));
      } else if (analyzer_output.has_regression() &&
                 analyzer_output.regression()) {
        regressions.push_back(std::move(analyzer_output));
      } else {
        successes.push_back(std::move(analyzer_output));
      }
    }
  }

  // sort regressions by run_key
  std::sort(regressions.begin(), regressions.end(),
            [](mako::AnalyzerOutput a, mako::AnalyzerOutput b) {
              return a.run_key() > b.run_key();
            });
  LOG(INFO) << optimizer.GetOptimizerSummary();

  test_output->set_test_status(mako::TestOutput::PASS);
  std::string summary_output;
  if (!failures.empty() || !regressions.empty()) {
    test_output->set_test_status(mako::TestOutput::ANALYSIS_FAIL);
    if (!failures.empty()) {
      absl::StrAppend(&summary_output, failures.size(),
                      " analyzers failed to execute successfully\n");
    }
    if (!regressions.empty()) {
      // Count regressions for the current run_key versus regressions for
      // other run_key so we can better describe to users what regressions we
      // found.
      std::unordered_map<std::string, std::vector<AnalyzerOutput>>
          run_key_to_regressions;
      for (auto& regression : regressions) {
        run_key_to_regressions[regression.run_key()].push_back(regression);
      }
      for (auto it : run_key_to_regressions) {
        const std::string& run_key = it.first;
        const std::vector<AnalyzerOutput>& regressions = it.second;
        const std::string run_key_desc =
            run_key == run_info.run_key()
                ? ""
                : absl::StrCat(" in results from historical run_key ", run_key);
        absl::StrAppend(&summary_output, regressions.size(),
                        " analyzers discovered regressions", run_key_desc,
                        "\n");
      }
    }
  }
  if (!successes.empty()) {
    absl::StrAppend(&summary_output, successes.size(),
                    " analyzers ran successfully\n");
  }
  if (dashboard != nullptr) {
    int unnnamed_analyzer_counter = 0;
    for (const auto& regression : regressions) {
      std::string link;
      mako::DashboardVisualizeAnalysisInput input;
      input.set_run_key(regression.run_key());
      input.set_analysis_key(regression.analysis_key());
      std::string err = dashboard->VisualizeAnalysis(input, &link);

      if (!err.empty()) {
        LOG(WARNING) << "Failed to generate analysis visualization url: "
                     << err;
        continue;
      }

      std::string analyzer_name = regression.analyzer_name();
      if (analyzer_name.empty()) {
        analyzer_name = absl::StrCat("unnamed_#", ++unnnamed_analyzer_counter);
      }
      absl::StrAppend(&summary_output,
                      absl::StrFormat("visualize regression '%s': %s\n",
                                      analyzer_name, link));
    }
  }
  absl::StrAppend(&summary_output, join(warnings));
  test_output->set_summary_output(summary_output);
  for (const std::vector<mako::AnalyzerOutput>& v :
       {failures, regressions, successes}) {
    for (const auto& output : v) {
      *test_output->add_analyzer_output_list() = output;
    }
  }
  return kNoError;
}  // namespace internal

}  // namespace internal
}  // namespace mako
