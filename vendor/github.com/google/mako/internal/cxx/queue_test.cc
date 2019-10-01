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
#include "internal/cxx/queue.h"

#include <algorithm>
#include <string>
#include <vector>

#include "glog/logging.h"
#include "benchmark/benchmark.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "internal/proto/mako_internal.pb.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_format.h"
#include "absl/time/time.h"
#include "internal/cxx/load/common/thread_pool_factory.h"
#include "testing/cxx/protocol-buffer-matchers.h"

namespace mako {
namespace internal {
namespace {

using ::mako::EqualsProto;
using ::testing::Eq;
using ::testing::Optional;

TEST(QueueTest, MessagePutAndGet) {
  int size = 100;
  Queue<mako_internal::ErrorMessage> q;
  ASSERT_TRUE(q.empty());
  for (int i = 0; i < size; i++) {
    ASSERT_EQ(i, q.size());

    mako_internal::ErrorMessage message;
    message.set_err(absl::StrCat(i));
    ASSERT_TRUE(q.put(message));

    ASSERT_FALSE(q.empty());
  }
  ASSERT_FALSE(q.empty());
  mako_internal::ErrorMessage message;
  for (int i = 0; i < size; i++) {
    EXPECT_THAT(
        q.get(absl::Milliseconds(100)),
        Optional(EqualsProto(absl::StrFormat(R"proto(err: "%d")proto", i))));
  }
}

void Writer(Queue<int>* q, int start, int finish) {
  LOG(INFO) << "Writing start: " << start << " finish: " << finish;
  while (start <= finish) {
    LOG(INFO) << "put(" << start << ")";
    ASSERT_TRUE(q->put(start));
    start++;
  }
  return;
}

void Reader(Queue<int>* q, std::size_t expected_size, bool blocking_get) {
  LOG(INFO) << "Reading until have " << expected_size << " results";
  std::vector<int> results;
  int result;
  while (expected_size > results.size()) {
    if (blocking_get) {
      result = q->get();
    } else {
      auto maybe_result = q->get(absl::Milliseconds(1000));
      ASSERT_TRUE(maybe_result.has_value());
      result = maybe_result.value();
    }
    LOG(INFO) << "get() = " << result;
    results.push_back(result);
  }

  // Expect this call to timeout
  EXPECT_THAT(q->get(absl::Milliseconds(1000)), Eq(absl::nullopt));

  // Make sure we have all values we wrote
  std::sort(results.begin(), results.end());
  for (std::size_t i = 0; i < expected_size; i++) {
    ASSERT_EQ(i, results[i]);
  }
}

TEST(QueueTest, MultipleThreads) {
  Queue<int> q;

  auto pool = mako::internal::CreateThreadPool(3);
  pool->StartWorkers();
  pool->Schedule([&] { Writer(&q, 0, 10); });
  pool->Schedule([&] { Writer(&q, 11, 20); });
  pool->Schedule([&] { Reader(&q, 21, false); });  // non-blocking get
}

TEST(QueueTest, MultipleThreadsBlockingGet) {
  Queue<int> q;

  auto pool = mako::internal::CreateThreadPool(3);
  pool->StartWorkers();
  pool->Schedule([&] { Writer(&q, 0, 10); });
  pool->Schedule([&] { Writer(&q, 11, 20); });
  pool->Schedule([&] { Reader(&q, 21, true); });  // blocking get
}

// Example class that only supports move semantics.
class MoveOnly {
 public:
  MoveOnly() = default;
  MoveOnly(const MoveOnly&) = delete;
  MoveOnly(MoveOnly&&) = default;

  MoveOnly& operator=(const MoveOnly&) = delete;
  MoveOnly& operator=(MoveOnly&&) = default;
};

TEST(QueueTest, SupportsMoveSemantics) {
  Queue<MoveOnly> q;
  ASSERT_TRUE(q.put(MoveOnly()));
  MoveOnly m = q.get();
  // Prevent unused variable warning from breaking compilation.
  (void)m;
}

// Example class that doesn't support moving.
class CopyOnly {
 public:
  CopyOnly() = default;
  CopyOnly(const CopyOnly&) = default;
  CopyOnly& operator=(const CopyOnly&) = default;
};

TEST(QueueTest, SupportsCopySemantics) {
  Queue<CopyOnly> q;
  ASSERT_TRUE(q.put(CopyOnly()));
  CopyOnly c = q.get();
  // Prevent unused variable warning from breaking compilation.
  (void)c;
}

static void BM_EmptyQueueGetDelay(benchmark::State& state) {
  Queue<int> q;
  for (auto x : state) {
    CHECK(!q.get(absl::Milliseconds(state.range(0))).has_value());
  }
}
BENCHMARK(BM_EmptyQueueGetDelay)->Arg(0)->Arg(1);

static void BM_NonEmptyQueueGetDelay(benchmark::State& state) {
  Queue<int> q;
  for (auto x : state) {
    q.put(5);
    CHECK_EQ(q.get(absl::Milliseconds(state.range(0))).value(), 5);
  }
}
BENCHMARK(BM_NonEmptyQueueGetDelay)->Arg(0)->Arg(1);

static void BM_NonEmptyQueueGet(benchmark::State& state) {
  Queue<int> q;
  for (auto x : state) {
    q.put(5);
    CHECK_EQ(q.get(), 5);
  }
}
BENCHMARK(BM_NonEmptyQueueGet);

}  // namespace
}  // namespace internal
}  // namespace mako
