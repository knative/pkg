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

// For more information about Mako see: go/mako.
#ifndef SPEC_CXX_ANALYZER_H_
#define SPEC_CXX_ANALYZER_H_

#include <string>

#include "spec/proto/mako.pb.h"

namespace mako {

// An abstract class which describes the Mako C++ Analyze interface.
// See implementing classes for detailed description on usage. See below for
// details on individual functions.
class Analyzer {
 public:
  // Mako framework will call this to determine if historical data should be
  // passed to Analyze() function below.
  //
  // If analyzer doesn't need any historic data, return without setting any
  // fields inside the AnalyzerHistoricQueryOutput message.
  //
  // In all other cases this function should set
  // AnalyzerHistoricQueryOutput.Status.
  //
  // See Analyzer section of mako.proto for details of proto message.
  //
  // When requesting the last N runs you'll need to set max_timestamp to avoid
  // retrieving the current run. eg:
  //   RunInfoQuery.max_timestamp =
  //     AnalyzerHistoricQueryInput.run_info.timestamp_ms - 1
  //
  // The bool returned represents success (true) or failure (false) of the
  // operation. In the case of failure, AnalyzerHistoricQueryOutput.Status
  // contains the error message.
  virtual bool ConstructHistoricQuery(
      const mako::AnalyzerHistoricQueryInput& query_input,
      mako::AnalyzerHistoricQueryOutput* query_output) = 0;

  // Run the analysis.
  //
  // Analyzer extenders will override the DoAnalyzer() method.
  //
  // This method does generic work common to all Analyzers and then delegates to
  // DoAnalyze() for implementation specific analysis, and expects most fields
  // of AnalyzerOutput to be set in DoAnalyze().
  //
  // The following fields of AnalyzerOutput will be populated before
  // DoAnalyze() is called:
  //   1) analyzer_type -> the return value of analyzer_type()
  //   2) analyzer_name -> the return value of analyzer_name()
  //   3) run_key -> analyzer_input.run_to_be_analyzed.run_info.run_key
  //
  // The analyzer_key field of AnalyzerOutput will be populated after
  // DoAnalyze() returns.
  //
  // The results of the operation will be placed in AnalyzerOutput.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // AnalyzerOutput.Status.
  virtual bool Analyze(const mako::AnalyzerInput& analyzer_input,
                       mako::AnalyzerOutput* analyzer_output);

  // Return the type name of the analyzer (eg. 'TTest')
  virtual std::string analyzer_type() = 0;

  // Return the name of the analyzer (eg. 'MyTTestOfMetric1'). Empty std::string will
  // be returned if the name has not been configured.
  virtual std::string analyzer_name() { return ""; }

  virtual ~Analyzer() {}

 private:
  // Implementation of the analyzer specific logic of running analysis.
  //
  // The results of the operation will be placed in AnalyzerOutput. See
  // comments on the Analyze() method for fields of AnalyzerOutput that the
  // framework populates before/after DoAnalyze() is called.
  //
  // The boolean returned represents success (true) or failure (false) of the
  // operation. More details about the success/failure will be in
  // AnalyzerOutput.Status.
  virtual bool DoAnalyze(const mako::AnalyzerInput& analyzer_input,
                         mako::AnalyzerOutput* analyzer_output) = 0;
};

}  // namespace mako
#endif  // SPEC_CXX_ANALYZER_H_
