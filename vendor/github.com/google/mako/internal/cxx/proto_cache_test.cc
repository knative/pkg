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
#include "internal/cxx/proto_cache.h"

#include <string>

#include "gtest/gtest.h"
#include "absl/strings/str_cat.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace internal {

TEST(ProtoCacheTest, SingleFieldSet) {
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(500);
  mako::RunInfoQueryResponse r;

  mako::RunInfoQuery q;
  *q.add_tags() = "1";

  // Miss
  ASSERT_FALSE(cache.Get(q, &r));
  ASSERT_EQ(1, cache.misses());
  ASSERT_EQ(0, cache.preventable_misses());

  // Put
  mako::RunInfoQueryResponse a;
  a.set_cursor("abc");
  cache.Put(q, a);

  // Hit
  ASSERT_TRUE(cache.Get(q, &r));
  EXPECT_EQ(1, cache.misses());
  EXPECT_EQ(0, cache.preventable_misses());
  EXPECT_EQ(1, cache.hits());
}

TEST(ProtoCacheTest, ProtosOfSameSerialLengthKeepDifferentResults) {
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(500);

  mako::RunInfoQuery q1;
  *q1.add_tags() = "1";
  mako::RunInfoQueryResponse r1;
  *r1.mutable_cursor() = "11";

  mako::RunInfoQuery q2;
  *q2.add_tags() = "2";
  mako::RunInfoQueryResponse r2;
  *r2.mutable_cursor() = "22";

  // Verify that cache queries have the same length.
  std::string a;
  std::string b;
  q1.SerializeToString(&a);
  q2.SerializeToString(&b);
  ASSERT_EQ(a.length(), b.length());

  mako::RunInfoQueryResponse r;
  cache.Put(q1, r1);
  cache.Put(q2, r2);

  ASSERT_TRUE(cache.Get(q1, &r));
  EXPECT_EQ("11", r.cursor());

  ASSERT_TRUE(cache.Get(q2, &r));
  EXPECT_EQ("22", r.cursor());
}

TEST(ProtoCacheTest, RepeatedFieldOrder) {
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(500);
  mako::RunInfoQueryResponse r;

  // q1 and q1 are the same except different order. They should be hashed the
  // same way.
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b";
  *q1.add_tags() = "1";
  *q1.add_tags() = "2";

  mako::RunInfoQuery q2;
  *q2.mutable_benchmark_key() = "b";
  *q2.add_tags() = "2";
  *q2.add_tags() = "1";

  // Put
  mako::RunInfoQueryResponse a1;
  a1.set_cursor("abc");
  cache.Put(q1, a1);

  // Hit
  mako::RunInfoQueryResponse a2;
  ASSERT_TRUE(cache.Get(q2, &a2));
  EXPECT_EQ("abc", a2.cursor());
}

TEST(ProtoCacheTest, MissingRepeatedField) {
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(500);
  mako::RunInfoQueryResponse r;

  // q1 and q2 are mostly the same except q1 has an extra repeated field (tag)
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b";
  *q1.mutable_run_key() = "r";
  *q1.add_tags() = "1";
  *q1.add_tags() = "2";
  *q1.add_tags() = "3";

  mako::RunInfoQuery q2;
  *q2.mutable_benchmark_key() = "b";
  *q2.mutable_run_key() = "r";
  *q2.add_tags() = "1";
  *q2.add_tags() = "2";

  // Put
  mako::RunInfoQueryResponse a1;
  a1.set_cursor("abc");
  cache.Put(q1, a1);

  // Miss
  mako::RunInfoQueryResponse a2;
  ASSERT_FALSE(cache.Get(q2, &a2));

  // Hit
  a1.Clear();
  ASSERT_TRUE(cache.Get(q1, &a1));
  EXPECT_EQ(a1.cursor(), "abc");
}

TEST(ProtoCacheTest, CheckingSize) {
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(500);
  mako::RunInfoQueryResponse r;
  r.set_cursor("abc");

  mako::RunInfoQuery q;
  *q.add_tags() = "1";

  // Put
  cache.Put(q, r);
  EXPECT_EQ(r.ByteSizeLong(), cache.size_bytes());
}

TEST(ProtoCacheTest, LargestEvicted) {
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";

  mako::RunInfoQueryResponse r1;
  r1.set_cursor("abc");

  mako::RunInfoQuery q2;
  *q2.add_tags() = "b2";

  mako::RunInfoQueryResponse r2;
  r2.set_cursor("efghijklmn");

  mako::RunInfoQueryResponse r;

  // r2 is larger than r1
  ASSERT_GT(r2.ByteSizeLong(), r1.ByteSizeLong());
  // But keys are the same
  ASSERT_EQ(q1.ByteSizeLong(), q2.ByteSizeLong());

  // Set cache size to just under total
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(
      r2.ByteSizeLong() + r1.ByteSizeLong() - 1);

  // Insert largest first, should fit
  cache.Put(q2, r2);
  ASSERT_TRUE(cache.Get(q2, &r));

  // Insert the smaller now, largest should be evicted.
  cache.Put(q1, r1);
  ASSERT_TRUE(cache.Get(q1, &r));
  ASSERT_FALSE(cache.Get(q2, &r));

  // Check counts are correct
  EXPECT_EQ(r1.ByteSizeLong(), cache.size_bytes());
  EXPECT_EQ(2, cache.hits());
  EXPECT_EQ(1, cache.misses());
  EXPECT_EQ(1, cache.preventable_misses());
}

TEST(ProtoCacheTest, MultipleLargeSmallestSaved) {
  // Smallest
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r1;
  r1.set_cursor("1");

  // Three larger.
  mako::RunInfoQuery q2;
  *q2.add_tags() = "b2";
  mako::RunInfoQueryResponse r2;
  r2.set_cursor("2222222");

  mako::RunInfoQuery q3;
  *q3.add_tags() = "b3";
  mako::RunInfoQueryResponse r3;
  r3.set_cursor("3333333");

  mako::RunInfoQuery q4;
  *q4.add_tags() = "b4";
  mako::RunInfoQueryResponse r4;
  r4.set_cursor("4444444");

  int total_result_size = r1.ByteSizeLong() + r2.ByteSizeLong() +
                          r3.ByteSizeLong() + r4.ByteSizeLong();

  // Set cache size to just under total
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(
      total_result_size - 1);

  mako::RunInfoQueryResponse r;

  // Insert all but last, all should stay in cache.
  cache.Put(q1, r1);
  cache.Put(q2, r2);
  cache.Put(q3, r3);
  ASSERT_TRUE(cache.Get(q1, &r));
  ASSERT_TRUE(cache.Get(q2, &r));
  ASSERT_TRUE(cache.Get(q3, &r));

  // Insert last, smallest should stay in cache.
  cache.Put(q4, r4);
  ASSERT_TRUE(cache.Get(q1, &r));
}

TEST(ProtoCacheTest, ExactSize) {
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r1;
  r1.set_cursor("1");

  mako::RunInfoQuery q2;
  *q2.add_tags() = "b2";
  mako::RunInfoQueryResponse r2;
  r2.set_cursor("22");

  mako::RunInfoQuery q3;
  *q3.add_tags() = "b3";
  mako::RunInfoQueryResponse r3;
  r3.set_cursor("333");

  mako::RunInfoQuery q4;
  *q4.add_tags() = "b4";
  mako::RunInfoQueryResponse r4;
  r4.set_cursor("4444");

  int total_result_size = r1.ByteSizeLong() + r2.ByteSizeLong() +
                          r3.ByteSizeLong() + r4.ByteSizeLong();

  // Set cache size to just exactly total, should fit all
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(
      total_result_size);

  // All should fit
  cache.Put(q1, r1);
  cache.Put(q2, r2);
  cache.Put(q3, r3);
  cache.Put(q4, r4);
  mako::RunInfoQueryResponse r;
  ASSERT_TRUE(cache.Get(q1, &r));
  ASSERT_TRUE(cache.Get(q2, &r));
  ASSERT_TRUE(cache.Get(q3, &r));
  ASSERT_TRUE(cache.Get(q4, &r));
}

TEST(ProtoCacheTest, Remove) {
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r1;
  r1.set_cursor("1");

  mako::RunInfoQuery q2;
  *q2.add_tags() = "b2";
  mako::RunInfoQueryResponse r2;
  r2.set_cursor("22");

  mako::RunInfoQuery q3;
  *q3.add_tags() = "b3";
  mako::RunInfoQueryResponse r3;
  r3.set_cursor("333");

  mako::RunInfoQuery q4;
  *q4.add_tags() = "b4";
  mako::RunInfoQueryResponse r4;
  r4.set_cursor("4444");

  int total_result_size = r1.ByteSizeLong() + r2.ByteSizeLong() +
                          r3.ByteSizeLong() + r4.ByteSizeLong();

  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(
      total_result_size);

  // Nothing to remove yet
  ASSERT_FALSE(cache.Remove(q1));

  // All should fit
  cache.Put(q1, r1);
  cache.Put(q2, r2);
  cache.Put(q3, r3);
  cache.Put(q4, r4);

  mako::RunInfoQueryResponse r;
  int expected_size = total_result_size;

  // Remove 1 (smallest)
  ASSERT_TRUE(cache.Remove(q1));
  expected_size -= r1.ByteSizeLong();
  ASSERT_EQ(expected_size, cache.size_bytes());

  // Others should still exist
  ASSERT_FALSE(cache.Get(q1, &r));
  ASSERT_TRUE(cache.Get(q2, &r));
  ASSERT_TRUE(cache.Get(q3, &r));
  ASSERT_TRUE(cache.Get(q4, &r));

  // Remove 3
  ASSERT_TRUE(cache.Remove(q3));
  expected_size -= r3.ByteSizeLong();
  ASSERT_EQ(expected_size, cache.size_bytes());

  ASSERT_FALSE(cache.Get(q1, &r));
  ASSERT_TRUE(cache.Get(q2, &r));
  ASSERT_FALSE(cache.Get(q3, &r));
  ASSERT_TRUE(cache.Get(q4, &r));

  // Remove 4 (largest)
  ASSERT_TRUE(cache.Remove(q4));
  expected_size -= r4.ByteSizeLong();
  ASSERT_EQ(expected_size, cache.size_bytes());

  ASSERT_FALSE(cache.Get(q1, &r));
  ASSERT_TRUE(cache.Get(q2, &r));
  ASSERT_FALSE(cache.Get(q3, &r));
  ASSERT_FALSE(cache.Get(q4, &r));

  // Remove 2 (last)
  ASSERT_TRUE(cache.Remove(q2));
  expected_size -= r2.ByteSizeLong();
  ASSERT_EQ(expected_size, cache.size_bytes());

  ASSERT_FALSE(cache.Get(q1, &r));
  ASSERT_FALSE(cache.Get(q2, &r));
  ASSERT_FALSE(cache.Get(q3, &r));
  ASSERT_FALSE(cache.Get(q4, &r));

  // Can't remove them twice
  ASSERT_FALSE(cache.Remove(q1));
  ASSERT_FALSE(cache.Remove(q2));
  ASSERT_FALSE(cache.Remove(q3));
  ASSERT_FALSE(cache.Remove(q4));
}

TEST(ProtoCacheTest, Overwrite) {
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(900);

  // Should remain unchanged.
  mako::RunInfoQuery q0;
  *q0.mutable_benchmark_key() = "b0";
  mako::RunInfoQueryResponse r0;
  r0.set_cursor("0");
  cache.Put(q0, r0);

  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r1;
  r1.set_cursor("1");
  cache.Put(q1, r1);

  // We get results from q1
  mako::RunInfoQueryResponse r;
  ASSERT_TRUE(cache.Get(q1, &r));
  ASSERT_EQ(r.cursor(), "1");

  // Same key as q1 before but put different results.
  mako::RunInfoQuery q2;
  *q2.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r2;
  r2.set_cursor("2");
  cache.Put(q2, r2);

  // Now get results from q1.
  ASSERT_TRUE(cache.Get(q1, &r));
  ASSERT_EQ(r.cursor(), "1");
  ASSERT_TRUE(cache.Get(q2, &r));
  ASSERT_EQ(r.cursor(), "1");

  // q0 is still in cache.
  ASSERT_TRUE(cache.Get(q0, &r));
  ASSERT_EQ(r.cursor(), "0");
}

TEST(ProtoCacheTest, TooLarge) {
  // Small
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r1;
  r1.set_cursor("1");

  // Large
  mako::RunInfoQuery q2;
  *q2.add_tags() = "b2";
  mako::RunInfoQueryResponse r2;
  r2.set_cursor("2222222");

  // Limit is smaller than large
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(
      r2.ByteSizeLong() - 1);

  cache.Put(q1, r1);
  cache.Put(q2, r2);

  mako::RunInfoQueryResponse r;

  // Small is still in, but large has not been added.
  EXPECT_TRUE(cache.Get(q1, &r));
  EXPECT_FALSE(cache.Get(q2, &r));
}

TEST(ProtoCacheTest, Clear) {
  // Small
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r1;
  r1.set_cursor("1");

  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(500);

  mako::RunInfoQueryResponse r;
  cache.Put(q1, r1);
  EXPECT_TRUE(cache.Get(q1, &r));
  EXPECT_EQ(1, cache.hits());

  // Create a miss
  mako::RunInfoQuery q2;
  *q2.mutable_benchmark_key() = "b2";
  EXPECT_FALSE(cache.Get(q2, &r));
  EXPECT_EQ(1, cache.misses());
  EXPECT_EQ(0, cache.preventable_misses());

  // Size is non-zero
  EXPECT_GT(cache.size_bytes(), 0);

  // Now clear and everything should be set back to 0.
  cache.Clear();
  EXPECT_EQ(0, cache.hits());
  EXPECT_EQ(0, cache.misses());
  EXPECT_EQ(0, cache.preventable_misses());
  EXPECT_EQ(0, cache.size_bytes());
  // And q1 is no longer in cache.
  EXPECT_FALSE(cache.Get(q1, &r));
}

TEST(ProtoCacheTest, QueueUpdated) {
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r1;
  r1.set_cursor("11");

  // r2 is larger than r1
  mako::RunInfoQuery q2;
  *q2.add_tags() = "b2";
  mako::RunInfoQueryResponse r2;
  r2.set_cursor("222");

  mako::RunInfoQueryResponse r;

  // Cache just big enough for q1 and q2
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(
      r1.ByteSizeLong() + r2.ByteSizeLong());

  // Put first two
  cache.Put(q1, r1);
  cache.Put(q2, r2);

  // Remove larger - q2
  ASSERT_TRUE(cache.Remove(q2));

  // q1 still in cache
  ASSERT_TRUE(cache.Get(q1, &r));

  // Add 4 results, all smaller than q1 which should evict it.
  int expected_size = 0;
  for (int i = 0; i < 3; i++) {
    mako::RunInfoQuery q;
    // +10 to avoid name conflict with b1 and b2 above.
    *q.add_tags() = absl::StrCat("b", i + 10);
    r.set_cursor("3");
    cache.Put(q, r);
    expected_size += r.ByteSizeLong();
  }

  ASSERT_FALSE(cache.Get(q1, &r));
  EXPECT_EQ(expected_size, cache.size_bytes());
}

TEST(ProtoCacheTest, ResultsCopied) {
  // Make sure that results are totally overwritten.
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";
  mako::RunInfoQueryResponse r1;
  r1.set_cursor("1");

  // Where we're going to store results has a RunInfo added.
  mako::RunInfoQueryResponse r;
  r.add_run_info_list()->set_run_key("blah");
  ASSERT_EQ(1, r.run_info_list_size());

  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(500);
  cache.Put(q1, r1);
  EXPECT_TRUE(cache.Get(q1, &r));

  // RunInfos have been cleared by Get().
  EXPECT_EQ(0, r.run_info_list_size());
  EXPECT_EQ("1", r.cursor());
}

TEST(ProtoCacheTest, Stats) {
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";

  mako::RunInfoQueryResponse r1;
  r1.set_cursor("abc");

  mako::RunInfoQuery q2;
  *q2.add_tags() = "b2";

  mako::RunInfoQueryResponse r2;
  r2.set_cursor("efghijklmn");

  mako::RunInfoQueryResponse r;

  // Set cache size to just under total
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(
      r2.ByteSizeLong() + r1.ByteSizeLong() - 1);

  // Insert largest first, should fit
  cache.Put(q2, r2);
  ASSERT_TRUE(cache.Get(q2, &r));

  // Insert the smaller now, largest should be evicted.
  cache.Put(q1, r1);
  ASSERT_TRUE(cache.Get(q1, &r));
  ASSERT_FALSE(cache.Get(q2, &r));

  std::stringstream ss;
  ss << " --name--\n";
  ss << "  hits: 2\n";
  ss << "  misses: 1\n";
  ss << "  size(bytes): 5\n";
  ss << "  entries evicted: 1\n";
  ss << "  evicted size(bytes): 12\n";
  ss << "  preventable misses: 1\n";
  ss << "  evicted key size(bytes): 4\n";
  EXPECT_EQ(ss.str(), cache.Stats("name"));
}

TEST(ProtoCacheTest, EvictedKeysSizeExceeded) {
  mako::RunInfoQuery q1;
  *q1.mutable_benchmark_key() = "b1";

  mako::RunInfoQueryResponse r1;
  r1.set_cursor("abc");

  mako::RunInfoQuery q2;
  *q2.add_tags() = "b2";

  mako::RunInfoQueryResponse r2;
  r2.set_cursor("de");

  mako::RunInfoQuery q3;
  *q3.add_tags() = "b3";

  mako::RunInfoQueryResponse r3;
  r3.set_cursor("i");

  mako::RunInfoQueryResponse r;

  // Set cache size such that only r1 fits
  // Set evicted keys size size such that only q2 fits
  ProtoCache<mako::RunInfoQuery, mako::RunInfoQueryResponse> cache(
      r1.ByteSizeLong(), q2.ByteSizeLong() + 1);

  EXPECT_FALSE(cache.evicted_keys_size_exceeded());
  cache.Put(q1, r1);
  EXPECT_FALSE(cache.evicted_keys_size_exceeded());
  EXPECT_EQ(0, cache.evicted_keys_size_bytes());

  // Evicts q1
  cache.Put(q2, r2);
  EXPECT_FALSE(cache.evicted_keys_size_exceeded());
  EXPECT_EQ(q1.ByteSizeLong(), cache.evicted_keys_size_bytes());

  // Evicts q2
  cache.Put(q3, r3);
  EXPECT_TRUE(cache.evicted_keys_size_exceeded());
  EXPECT_EQ(q1.ByteSizeLong() + q2.ByteSizeLong(),
            cache.evicted_keys_size_bytes());

  cache.Get(q3, &r);
  EXPECT_EQ(0, cache.misses());
  EXPECT_EQ(0, cache.preventable_misses());

  // q2 isn't a preventable miss because it didn't fit in evicted_keys
  cache.Get(q2, &r);
  EXPECT_EQ(1, cache.misses());
  EXPECT_EQ(0, cache.preventable_misses());

  // q1 is a preventable miss because it fit in evicted_keys
  cache.Get(q1, &r);
  EXPECT_EQ(2, cache.misses());
  EXPECT_EQ(1, cache.preventable_misses());
}

}  // namespace internal
}  // namespace mako
