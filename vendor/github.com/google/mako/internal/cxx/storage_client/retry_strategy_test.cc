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

#include "internal/cxx/storage_client/retry_strategy.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "internal/cxx/clock_mock.h"

namespace mako {
namespace internal {

namespace {

using testing::AllOf;
using testing::AnyNumber;
using testing::AtLeast;
using testing::Ge;
using testing::Le;
using testing::StrictMock;

TEST(RetryStrategyTest, DoesRetry) {
  int count = 0;
  auto f = [&count]() {
    ++count;
    return StorageRetryStrategy::kContinue;
  };

  absl::Duration timeout = absl::Hours(1);
  absl::Duration min_sleep = absl::Seconds(1);
  absl::Duration max_sleep = absl::Minutes(15);

  StrictMock<ClockMock> clock;
  EXPECT_CALL(clock, Sleep(AllOf(Ge(min_sleep), Le(max_sleep))))
      .Times(AtLeast(3));
  EXPECT_CALL(clock, TimeNow()).Times(AnyNumber());

  StorageBackoff retry(timeout, min_sleep, max_sleep, &clock);
  retry.Do(f);

  EXPECT_GE(count, 4);
}

TEST(RetryStrategyTest, DoesExecuteAtLeastOnce) {
  int count = 0;
  auto f = [&count]() {
    ++count;
    return StorageRetryStrategy::kContinue;
  };

  absl::Duration timeout = absl::Milliseconds(1);
  absl::Duration min_sleep = absl::Seconds(1);
  absl::Duration max_sleep = absl::Seconds(5);

  StrictMock<ClockMock> clock;
  EXPECT_CALL(clock, TimeNow()).Times(AnyNumber());

  StorageBackoff retry(timeout, min_sleep, max_sleep, &clock);
  retry.Do(f);

  EXPECT_EQ(count, 1);
}

TEST(RetryStrategyTest, DoesBreak) {
  int count = 0;
  auto f = [&count]() {
    ++count;
    return StorageRetryStrategy::kBreak;
  };

  absl::Duration timeout = absl::Hours(1);
  absl::Duration min_sleep = absl::Seconds(1);
  absl::Duration max_sleep = absl::Minutes(15);

  StrictMock<ClockMock> clock;
  EXPECT_CALL(clock, TimeNow()).Times(AnyNumber());

  StorageBackoff retry(timeout, min_sleep, max_sleep, &clock);
  retry.Do(f);

  EXPECT_GE(count, 1);
}

}  // namespace

}  // namespace internal
}  // namespace mako
