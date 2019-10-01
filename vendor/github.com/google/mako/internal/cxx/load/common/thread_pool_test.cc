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
#include "internal/cxx/load/common/thread_pool.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "absl/synchronization/blocking_counter.h"
#include "absl/synchronization/mutex.h"

namespace mako {
namespace threadpool_internal {
namespace {

using ::testing::Eq;

TEST(ThreadPoolTest, SimpleCounter) {
  absl::Mutex mutex;
  int count = 0;
  int num_threads = 5;
  int iterations = 100;

  absl::BlockingCounter block(num_threads);

  ThreadPool pool(num_threads);

  auto work = [&count, &mutex, &block, iterations]() {
    for (int i=0; i < iterations; ++i) {
      absl::MutexLock lock(&mutex);
      count += 1;
    }
    block.DecrementCount();
  };

  for (int i=0; i < num_threads; ++i) {
    pool.Schedule(work);
  }
  pool.StartWorkers();
  block.Wait();
  EXPECT_THAT(count, Eq(num_threads * iterations));
}

}  // namespace
}  // namespace threadpool_internal
}  // namespace mako
