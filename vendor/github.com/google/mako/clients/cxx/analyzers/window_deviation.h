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
// Mako Window Deviation Analyzer (WDA).
//
// For detailed information on usage, see go/mako-wda
//
// For more information about the interface see
// https://github.com/google/mako/blob/master/spec/cxx/analyzer.h
#ifndef CLIENTS_CXX_ANALYZERS_WINDOW_DEVIATION_H_
#define CLIENTS_CXX_ANALYZERS_WINDOW_DEVIATION_H_

#include <string>
#include <vector>

#include "clients/proto/analyzers/window_deviation.pb.h"
#include "spec/cxx/analyzer.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace window_deviation {

class Analyzer : public mako::Analyzer {
 public:
  // Constructor, takes configuration input.
  explicit Analyzer(
      const mako::window_deviation::WindowDeviationInput&
      config)
    : config_(config) {}

  ~Analyzer() override {}

  // Historical information the analyzer requires, see ifc for more info.
  bool ConstructHistoricQuery(
      const mako::AnalyzerHistoricQueryInput& input,
      mako::AnalyzerHistoricQueryOutput* output) override;

  // Return analyzer type, see ifc for more info
  std::string analyzer_type() override { return "WindowDeviation"; }

  // Return analyzer name, see ifc for more info.
  // If WindowDeviationInput.name is not set, empty std::string will be returned.
  std::string analyzer_name() override { return config_.name(); }


 private:
  // Run the analysis, see ifc for more info.
  bool DoAnalyze(const mako::AnalyzerInput& input,
                 mako::AnalyzerOutput* output) override;

  // A helper method for validating the config. It was passed to the ctor, but
  // we can't return an error there so we check it on subsequent calls to
  // methods.
  std::string ValidateWindowDeviationInput() const;

  WindowDeviationInput config_;
};

}  // namespace window_deviation
}  // namespace mako

#endif  // CLIENTS_CXX_ANALYZERS_WINDOW_DEVIATION_H_
