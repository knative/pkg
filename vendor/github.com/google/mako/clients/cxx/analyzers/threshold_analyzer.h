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
// See the License for the specific language governing permissions and
// limitations under the License.
//
//
// Threshold analyzer determines if data is within threshold bounds.
//
// See docs at https://github.com/google/mako/blob/master/clients/proto/analyzers/threshold_analyzer.proto
// See interface at: https://github.com/google/mako/blob/master/spec/cxx/analyzer.h
#ifndef CLIENTS_CXX_ANALYZERS_THRESHOLD_ANALYZER_H_
#define CLIENTS_CXX_ANALYZERS_THRESHOLD_ANALYZER_H_

#include <string>

#include "clients/proto/analyzers/threshold_analyzer.pb.h"
#include "spec/cxx/analyzer.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace threshold_analyzer {

class Analyzer : public mako::Analyzer {
 public:
  explicit Analyzer(
      const mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput&
          analyzer_input);

  bool ConstructHistoricQuery(
      const mako::AnalyzerHistoricQueryInput& query_input,
      mako::AnalyzerHistoricQueryOutput* query_output) override;

  std::string analyzer_type() override { return "Threshold"; }

  // If config_ does not have name field empty std::string will be returned.
  std::string analyzer_name() override { return config_.name(); }

  ~Analyzer() override {}

 private:
  bool DoAnalyze(const mako::AnalyzerInput& analyzer_input,
                 mako::AnalyzerOutput* analyzer_output) override;

  static bool SetAnalyzerError(const std::string& error_message,
                               mako::AnalyzerOutput* output);

  mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput
      config_;
};

}  // namespace threshold_analyzer
}  // namespace mako

#endif  // CLIENTS_CXX_ANALYZERS_THRESHOLD_ANALYZER_H_
