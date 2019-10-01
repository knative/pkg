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
#include "internal/cxx/load/common/executor.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "absl/synchronization/barrier.h"
#include "absl/synchronization/notification.h"

namespace mako {
namespace internal {
namespace {

TEST(ExecutorTest, Empty) {
  Executor executor(1);
  executor.Wait();
}

TEST(ExecutorTest, Executes) {
  Executor executor(1);
  absl::Notification notification;
  executor.Schedule([&](){
    notification.Notify();
  });
  notification.WaitForNotification();
}

TEST(ExecutorTest, ExecutesTwo) {
  Executor executor(1);
  absl::Notification notification1;
  absl::Notification notification2;
  executor.Schedule([&](){
    notification1.Notify();
  });
  executor.Schedule([&](){
    notification2.Notify();
  });
  notification2.WaitForNotification();
  notification1.WaitForNotification();
}

TEST(ExecutorTest, WorkIsCompleteWhenWaitReturns) {
  absl::Barrier barrier(2);
  Executor executor(1);

  absl::Notification started;
  absl::Notification finished;

  executor.Schedule([&] {
    started.Notify();
    barrier.Block();
    finished.Notify();
  });

  started.WaitForNotification();
  ASSERT_FALSE(finished.HasBeenNotified());
  barrier.Block();
  executor.Wait();
  EXPECT_TRUE(finished.HasBeenNotified());
}

TEST(ExecutorTest, WorkIsCompleteWhenWaitReturnsX2) {
  absl::Barrier barrier(3);
  Executor executor(2);

  absl::Notification started1;
  absl::Notification started2;
  absl::Notification finished1;
  absl::Notification finished2;

  executor.Schedule([&] {
    started1.Notify();
    barrier.Block();
    finished1.Notify();
  });
  executor.Schedule([&] {
    started2.Notify();
    barrier.Block();
    finished2.Notify();
  });

  started1.WaitForNotification();
  started2.WaitForNotification();
  ASSERT_FALSE(finished1.HasBeenNotified());
  ASSERT_FALSE(finished2.HasBeenNotified());
  barrier.Block();
  executor.Wait();
  EXPECT_TRUE(finished1.HasBeenNotified());
  EXPECT_TRUE(finished2.HasBeenNotified());
}

TEST(ExecutorTest, NoReusingSchedule) {
  Executor executor(1);
  executor.Schedule([&](){});
  executor.Wait();
  ASSERT_DEATH(executor.Schedule([&]() {}),
               "Schedule method should never be called after Wait");
}

TEST(ExecutorTest, NoReusingWait) {
  Executor executor(1);
  executor.Schedule([&](){});
  executor.Wait();
  ASSERT_DEATH(executor.Wait(), "Wait method should never be called twice");
}

TEST(ExecutorTest, WorkIsCompleteWhenDestructed) {
  absl::Notification finished;
  {
    absl::Barrier barrier(2);
    absl::Notification started;
    Executor executor(1);
    executor.Schedule([&] {
      started.Notify();
      barrier.Block();
      finished.Notify();
    });
    started.WaitForNotification();
    ASSERT_FALSE(finished.HasBeenNotified());
    barrier.Block();
  }
  EXPECT_TRUE(finished.HasBeenNotified());
}

}  // namespace
}  // namespace internal
}  // namespace mako
