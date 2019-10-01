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
#ifndef INTERNAL_CXX_PROTO_CACHE_H_
#define INTERNAL_CXX_PROTO_CACHE_H_

#include <stddef.h>
// ProtoCache provided key value cache of protocol buffers.
//
// Common use would be to cache a storage query (key) to the results of that
// query (value).
//
// The max size in bytes passed to the constructor controls how large the
// 'values' of the mapping can get. During a 'Put' that max value may
// momentarily be exceeded, but the largest key->value mapping will be evicted
// until the size is beneath the max.
//
// Assumptions about the 'key' (or query) protocol buffer:
//   - Order of repeated fields does not matter.
//
// Simple example of usage:
//   RunInfoQuery query = <your query>
//   RunInfoQueryResults results = <results from that query>
//
//   ProtoCache<RunInfoQuery, RunInfoQueryResponse> cache(5000);
//   // Map query to results
//   cache.Put(query, results)
//   ...
//   // results2 will have same contents as results.
//   RunInfoQueryResults results2;
//   cache.Get(query, &results2)
//
// Complexity:
//   Put:
//     Average case: O(log(N))
//     Worst case: O(N log(N)) (When everything needs to be evicted).
//   Get:
//     Average case: O(1)
//     Worst case: O(N)
//   Remove:
//     Average case: O(1)
//     Worst case: O(N)
//   Clear:
//     O(1)
//
// Class is not thread-safe.
//
#include <queue>
#include <string>

#include "glog/logging.h"
#include "src/google/protobuf/util/message_differencer.h"
#include "spec/proto/mako.pb.h"
#include "absl/container/flat_hash_map.h"
#include "absl/container/flat_hash_set.h"

namespace mako {
namespace internal {

constexpr int kDefaultMaxSizeEvictedKeysBytes = 1 * 1000 * 1000;

// Functor used to hash protocol buffers for a std::map.
// Using SerializeToString as an input to std::hash<string> is a common
// solution to this problem (g/c-users/07RjbPNFztE/UepzJDL0uwoJ).
//
// We instead use the length of that string below to allow different orderings
// of repeated fields (eg. proto1.tags = ["1", "2"] and proto2.tags = ["2", "1])
// to still match the same hash bucket. Then in ProtoEquals below we can compare
// those protos fully.
//
// There isn't much risk here as we're counting on the ProtoEquals for
// correctness.
template <class T>
struct ProtoHash {
  size_t operator()(const T& key) const {
    std::string a;
    CHECK(key.SerializeToString(&a));
    return hasher(a.size());
  }
  std::hash<int> hasher;
};

// Functor used to compare two protocol buffers for equality. We defer the
// comparions to MessageDifferencer configured to allow repeating fields to be
// treated as sets.
template <class T>
struct ProtoEquals {
  bool operator()(const T& t1, const T& t2) const {
    google::protobuf::util::MessageDifferencer m;
    m.set_repeated_field_comparison(
        google::protobuf::util::MessageDifferencer::RepeatedFieldComparison::AS_SET);
    return m.Compare(t1, t2);
  }
};

// ProtoWrapper allows us to define a sorting of protocol buffers based on their
// ByteSizeLong().
template <class T>
struct ProtoWrapper {
  ProtoWrapper(int size_in_bytes, T in_key)
      : size_bytes(size_in_bytes), key(in_key) {}
  bool operator<(const ProtoWrapper& other) const {
    return this->get_size_bytes() < other.get_size_bytes();
  }
  int get_size_bytes() const { return size_bytes; }
  // Difficulty w/ copy constructor when make this const instead of accessing
  // through a function.
  int size_bytes;
  T key;
};

template <class K, class V>
class ProtoCache {
 public:
  explicit ProtoCache(int max_size_bytes)
      : ProtoCache(max_size_bytes, kDefaultMaxSizeEvictedKeysBytes) {}

  ProtoCache(int max_size_bytes, int max_size_evicted_keys_bytes)
      : hits_(0),
        misses_(0),
        current_size_bytes_(0),
        eviction_count_(0),
        eviction_size_bytes_(0),
        evicted_keys_size_bytes_(0),
        preventable_misses_(0),
        max_size_bytes_(max_size_bytes),
        max_size_evicted_keys_bytes_(max_size_evicted_keys_bytes) {}

  // Lookup a key in cache.
  // If return value is false, then key does not exist in cache and value* has
  // not been modified.
  // If return value is true, then value* has been populated.
  bool Get(const K& key, V* out_value);

  // Put key and value in cache.
  // If value is larger than max_size_bytes then it will not be placed in cache.
  // If the key already exists in the cache the value will not be overwritten.
  void Put(const K& key, const V& in_value);

  // Remove a specific key from cache.
  // Return value is true if key was found, otherwise false.
  bool Remove(const K& key);

  // Number of cache hits since construction or last call to Clear().
  int hits() { return hits_; }

  // Number of cache misses since construction or last call to Clear().
  int misses() { return misses_; }

  // Total size of all results stored in cache.
  int size_bytes() { return current_size_bytes_; }

  // Number of cache entries that were evicted.
  int eviction_count() { return eviction_count_; }

  // Total size of all evicted results.
  int eviction_size_bytes() { return eviction_size_bytes_; }

  // Total size of all evicted keys (may exceed max_size_evicted_keys_bytes_).
  int evicted_keys_size_bytes() { return evicted_keys_size_bytes_; }

  // Number of misses that could have been prevented by with infinitely large
  // cache. If evicted_keys_size_exceeded() == false, this may underestimate the
  // number of preventable misses as we were limited by the size of evicted
  // keys that we could store.
  int preventable_misses() { return preventable_misses_; }

  // Indicates whether we ran out of space to store evicted keys. When
  // evicted_keys_size_exceeded() == false, preventable_misses() may
  // underestimate the number of preventable misses.
  bool evicted_keys_size_exceeded() {
    return evicted_keys_size_bytes() > max_size_evicted_keys_bytes_;
  }

  // Clear all cache contents and state.
  void Clear();

  // Gets relevant stats as a std::string and write them to the event service
  std::string Stats(const std::string& cache_name);

 private:
  void EvictIfNeeded();
  int hits_;
  int misses_;
  int current_size_bytes_;
  // Keep track of evictions so we can better understand when we might want
  // to increase cache sizes.
  int eviction_count_;
  int eviction_size_bytes_;
  int evicted_keys_size_bytes_;
  int preventable_misses_;
  const int max_size_bytes_;
  const int max_size_evicted_keys_bytes_;
  // Holds mapping from key to value.
  absl::flat_hash_map<K, V, ProtoHash<K>, ProtoEquals<K> > cache_;
  absl::flat_hash_set<K, ProtoHash<K>, ProtoEquals<K> > evicted_keys_;
  // Sorted by proto ByteSizeLong().
  std::priority_queue<ProtoWrapper<K> > q_by_size_;
};

template <class K, class V>
void ProtoCache<K, V>::Put(const K& key, const V& value) {
  // Check if key has already been entered
  if (cache_.count(key)) {
    return;
  }
  // If an inserted key is in the evicted set, remove it as it is no longer
  // evicted.
  if (evicted_keys_.find(key) != evicted_keys_.end()) {
    evicted_keys_.erase(key);
  }
  // Taking ByteSizeLong() is expensive, do it once.
  int size_of_results_bytes = value.ByteSizeLong();
  // Avoid abuse
  if (size_of_results_bytes > max_size_bytes_) {
    return;
  }
  cache_[key] = value;
  ProtoWrapper<K> p(size_of_results_bytes, key);
  q_by_size_.push(p);
  current_size_bytes_ += size_of_results_bytes;
  EvictIfNeeded();
  return;
}

template <class K, class V>
bool ProtoCache<K, V>::Remove(const K& key) {
  auto it = cache_.find(key);
  // Cache doesn't contain key
  if (it == cache_.end()) {
    return false;
  }
  current_size_bytes_ -= it->second.ByteSizeLong();
  // Remove key from cache
  //
  // No good way to remove random element from priority_queue. Use the cache_
  // as an indicator of what has been removed from cache.
  cache_.erase(it);
  return true;
}

template <class K, class V>
void ProtoCache<K, V>::EvictIfNeeded() {
  while (current_size_bytes_ > max_size_bytes_ && !q_by_size_.empty()) {
    ProtoWrapper<K> largest = q_by_size_.top();
    q_by_size_.pop();
    // If key doesn't exist in map, then was Removed(). Try again..
    auto it = cache_.find(largest.key);
    if (it == cache_.end()) {
      continue;
    }
    eviction_count_++;
    eviction_size_bytes_ += largest.get_size_bytes();
    cache_.erase(it);
    current_size_bytes_ -= largest.get_size_bytes();

    evicted_keys_size_bytes_ += largest.key.ByteSizeLong();
    if (evicted_keys_size_bytes_ <= max_size_evicted_keys_bytes_) {
      evicted_keys_.insert(largest.key);
    }
  }
}

template <class K, class V>
bool ProtoCache<K, V>::Get(const K& key, V* value) {
  if (!cache_.count(key)) {
    misses_++;
    if (evicted_keys_.find(key) != evicted_keys_.end()) {
      preventable_misses_++;
    }
    return false;
  }
  value->CopyFrom(cache_[key]);
  hits_++;
  return true;
}

template <class K, class V>
void ProtoCache<K, V>::Clear() {
  hits_ = 0;
  misses_ = 0;
  current_size_bytes_ = 0;
  eviction_count_ = 0;
  eviction_size_bytes_ = 0;
  evicted_keys_size_bytes_ = 0;
  preventable_misses_ = 0;
  // No clear method on priority queue.
  q_by_size_ = std::priority_queue<ProtoWrapper<K>>();
  evicted_keys_.clear();
  cache_.clear();
}

template <class K, class V>
std::string ProtoCache<K, V>::Stats(const std::string& cache_name) {

  std::stringstream ss;
  ss << " --" << cache_name << "--\n";
  ss << "  hits: " << hits() << "\n";
  ss << "  misses: " << misses() << "\n";
  ss << "  size(bytes): " << size_bytes() << "\n";
  ss << "  entries evicted: " << eviction_count() << "\n";
  ss << "  evicted size(bytes): " << eviction_size_bytes() << "\n";
  ss << "  preventable misses: " << preventable_misses() << "\n";
  ss << "  evicted key size(bytes): " << evicted_keys_size_bytes() << "\n";
  return ss.str();
}
#ifndef SWIG

// Specialize ProtoCache for Python.
using SampleBatchQueryProtoCache
      = ProtoCache<SampleBatchQuery, SampleBatchQueryResponse>;
using RunInfoQueryProtoCache
      = ProtoCache<RunInfoQuery, RunInfoQueryResponse>;

#endif  // SWIG
}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_PROTO_CACHE_H_
