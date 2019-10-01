// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "helpers/cxx/status/status.h"
#include <string>

#include "absl/strings/str_cat.h"
#include "absl/strings/string_view.h"

namespace mako {
namespace helpers {

Status OkStatus() { return Status(); }

std::string StatusToString(const Status& status) {
  return absl::StrCat(StatusCodeToString(status.code()), ": ",
                      status.message());
}

bool HasErrorCode(const Status& status, StatusCode code) {
  return status.code() == code;
}

}  // namespace helpers
}  // namespace mako

