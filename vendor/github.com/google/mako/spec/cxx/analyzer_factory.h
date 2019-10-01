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
#ifndef SPEC_CXX_ANALYZER_FACTORY_H_
#define SPEC_CXX_ANALYZER_FACTORY_H_

#include <memory>
#include <string>
#include <vector>

#include "spec/cxx/analyzer.h"
#include "spec/proto/mako.pb.h"

namespace mako {

// AnalyzerFactory interface to be implemented by test authors.
//
// An implementation is provided to the framework via LoadMain and the framework
// uses it to create and run user-configured analyzers.
class AnalyzerFactory {
 public:
  // Given the input, creates all analyzers.
  //
  // For an associated LoadMain execution, the framework will call this function
  // once.
  //
  // Args:
  //    input: Input which may or may not be needed to construct analyzers.
  //           See the AnalyzerFactoryInput proto documentation.
  //    analyzers: The output analyzers.
  //
  // Returned std::string is empty for success or an error message for failure.
  virtual std::string NewAnalyzers(
      const mako::AnalyzerFactoryInput& input,
      std::vector<std::unique_ptr<mako::Analyzer>>* analyzers) = 0;

  virtual ~AnalyzerFactory() {}
};
}  // namespace mako
#endif  // SPEC_CXX_ANALYZER_FACTORY_H_
