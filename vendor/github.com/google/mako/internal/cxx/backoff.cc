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

#include "internal/cxx/backoff.h"

#include <cmath>

#include "absl/time/time.h"

namespace mako {
namespace internal {

absl::Duration ComputeBackoff(absl::Duration min_delay,
                              absl::Duration max_delay, int previous_retries) {
  // 1.3 was picked based on the graph of (1.3^x - 1) seeming reasonable.
  // Casting to double avoid ClangTidy warnings about using IPow.
  return std::min(
      max_delay,
      min_delay +
          min_delay * (pow(1.3, static_cast<double>(previous_retries)) - 1));
}

}  // namespace internal
}  // namespace mako
