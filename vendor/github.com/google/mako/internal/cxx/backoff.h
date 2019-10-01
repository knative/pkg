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

#ifndef INTERNAL_CXX_BACKOFF_H_
#define INTERNAL_CXX_BACKOFF_H_

#include "absl/time/time.h"

namespace mako {
namespace internal {

// Return a backoff duration given the specified minimum and maximum delay, and
// a count of how many times it has previously been retried.
//
// NOTE: This function makes no attempt to introduce randomization/jitter,
// and thus should not be used in scenarios where synchronized attempts might
// be problematic.
absl::Duration ComputeBackoff(absl::Duration min_delay,
                              absl::Duration max_delay, int previous_retries);

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_BACKOFF_H_
