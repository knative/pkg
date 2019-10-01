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
#include "spec/cxx/analyzer.h"

#include "absl/flags/flag.h"
#include "absl/strings/str_cat.h"
#include "spec/proto/mako.pb.h"
#include "src/farmhash.h"

ABSL_FLAG(
    bool, mako_emit_text_serialized_input_config, true,
    "Temporary flag to control emitting the text serialized input_config field "
    "in the AnalyzerOutput message. Disable this to work around RunInfo size "
    "exceeded errors. For more context see b/129875625. This flag will be "
    "removed when we deprecate and remove the text serialized input_config "
    "field and replace it with serialized_input_config field.");

namespace mako {

bool Analyzer::Analyze(const mako::AnalyzerInput& analyzer_input,
                       mako::AnalyzerOutput* analyzer_output) {
  // Basic initialization of output
  analyzer_output->Clear();
  analyzer_output->set_analyzer_type(analyzer_type());
  if (!analyzer_name().empty()) {
    analyzer_output->set_analyzer_name(analyzer_name());
  }
  analyzer_output->set_run_key(
      analyzer_input.run_to_be_analyzed().run_info().run_key());

  // Call concrete analyze algorithm implemented by descendant class.
  bool result = DoAnalyze(analyzer_input, analyzer_output);

  // If successful, use the output as a unique key.
  if (result) {
    analyzer_output->set_analysis_key(
        absl::StrCat(util::Fingerprint64(analyzer_output->output())));
  }

  return result;
}

}  // namespace mako
