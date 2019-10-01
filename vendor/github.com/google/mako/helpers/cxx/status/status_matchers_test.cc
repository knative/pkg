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
#include "helpers/cxx/status/status_matchers.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "helpers/cxx/status/canonical_errors.h"
#include "helpers/cxx/status/status.h"
#include "helpers/cxx/status/statusor.h"

namespace mako {
namespace helpers {
namespace {

using ::testing::Eq;
using ::testing::Test;

TEST(StatusMatchersTest, IsOkAndHolds) {
  StatusOr<int> status_or_int = 3;
  EXPECT_THAT(status_or_int, IsOkAndHolds(Eq(3)));
}

TEST(StatusMatchersTest, StatusIs) {
  StatusOr<int> status_or_int = AbortedError("aborted");

  EXPECT_THAT(status_or_int, StatusIs(Eq(StatusCode::kAborted)));
  EXPECT_THAT(status_or_int.status(), StatusIs(Eq(StatusCode::kAborted)));
}

TEST(StatusMatchersTest, ExpectOk) {
  EXPECT_OK(OkStatus());
}

TEST(StatusMatchersTest, AssertOk) {
  ASSERT_OK(OkStatus());
}

}  // namespace
}  // namespace helpers
}  // namespace mako
