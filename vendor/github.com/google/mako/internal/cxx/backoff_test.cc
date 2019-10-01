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

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "absl/time/time.h"

namespace mako {
namespace internal {
namespace {

using ::testing::Eq;
using ::testing::Ge;
using ::testing::Gt;
using ::testing::Lt;

TEST(BackoffTest, Limits) {
  absl::Duration min = absl::Seconds(1);
  absl::Duration max = absl::Seconds(10);

  absl::Duration first = ComputeBackoff(min, max, 0);
  absl::Duration big = ComputeBackoff(min, max, 100);

  EXPECT_THAT(first, Ge(min));
  EXPECT_THAT(big, Eq(max));
}

TEST(BackoffTest, Exponential) {
  absl::Duration min = absl::Seconds(1);
  absl::Duration max = absl::Seconds(100);

  absl::Duration first = ComputeBackoff(min, max, 0);
  absl::Duration second = ComputeBackoff(min, max, 1);
  absl::Duration third = ComputeBackoff(min, max, 2);
  absl::Duration fourth = ComputeBackoff(min, max, 3);

  EXPECT_THAT(first, Lt(second));
  EXPECT_THAT(second, Lt(third));
  EXPECT_THAT(third, Lt(fourth));

  absl::Duration first_diff = second - first;
  absl::Duration second_diff = third - second;
  absl::Duration third_diff = fourth - third;

  EXPECT_THAT(second_diff, Gt(first_diff));
  EXPECT_THAT(third_diff, Gt(second_diff));
}

}  // namespace
}  // namespace internal
}  // namespace mako
