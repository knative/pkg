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

#include "internal/cxx/utils/stringutil.h"

#include "absl/strings/ascii.h"
#include "absl/strings/string_view.h"

namespace mako {
namespace internal {

void TrimWhitespace(absl::string_view* str) {
  int trim_count = 0;
  for (auto it = str->begin(); it != str->end(); ++it) {
    if (!absl::ascii_isspace(*it)) {
      break;
    }
    trim_count++;
  }
  if (trim_count > 0) {
    str->remove_prefix(trim_count);
  }

  trim_count = 0;
  for (auto it = str->rbegin(); it != str->rend(); ++it) {
    if (!absl::ascii_isspace(*it)) {
      break;
    }
    trim_count++;
  }
  if (trim_count > 0) {
    str->remove_suffix(trim_count);
  }
}

}  // namespace internal
}  // namespace mako
