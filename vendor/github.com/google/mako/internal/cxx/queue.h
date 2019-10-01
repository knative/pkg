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
//
// Thread safe Queue
//
// See mako/internal/cxx/queue_ifc.h for more information.
#ifndef INTERNAL_CXX_QUEUE_H_
#define INTERNAL_CXX_QUEUE_H_

#include <cstddef>
#include <queue>
#include <utility>

#include "absl/synchronization/mutex.h"
#include "absl/time/time.h"
#include "absl/types/optional.h"
#include "internal/cxx/queue_ifc.h"

namespace mako {
namespace internal {

// A typed Queue that is thread-safe.
template <typename T>
class Queue : public QueueInterface<T> {
 public:
  Queue<T>() {}

  ~Queue<T>() override {}

  // See interface for documentation
  bool put(T item) override;

  // See interface for documentation
  absl::optional<T> get(absl::Duration timeout) override;

  // See interface for documentation
  T get() override;

  // See interface for documentation
  std::size_t size() override;

  // See interface for documentation
  bool empty() override;

  // See interface for documentation
  void clear() override;

 private:
  absl::Mutex queue_mutex_;
  // GUARDED_BY queue_mutex_ but LockWhenWithTimeout doesn't support these
  // checks.
  std::queue<T> q_;
};

template <typename T>
bool Queue<T>::put(T item) {
  absl::MutexLock lock(&queue_mutex_);
  q_.push(std::move(item));
  return true;
}

template <typename T>
T Queue<T>::get() {
  auto not_empty = [this] { return !q_.empty(); };
  queue_mutex_.LockWhen(absl::Condition(&not_empty));
  T item = std::move(q_.front());
  q_.pop();
  queue_mutex_.Unlock();
  return std::move(item);
}

template <typename T>
absl::optional<T> Queue<T>::get(absl::Duration timeout) {
  auto not_empty = [this] { return !q_.empty(); };
  if (!queue_mutex_.LockWhenWithTimeout(absl::Condition(&not_empty), timeout)) {
    queue_mutex_.Unlock();
    // cond false on return, must have timed out.
    return absl::nullopt;
  }
  T item = std::move(q_.front());
  q_.pop();
  queue_mutex_.Unlock();
  return std::move(item);
}

template <typename T>
std::size_t Queue<T>::size() {
  absl::MutexLock lock(&queue_mutex_);
  return q_.size();
}

template <typename T>
bool Queue<T>::empty() {
  absl::MutexLock lock(&queue_mutex_);
  return q_.empty();
}

template <typename T>
void Queue<T>::clear() {
  absl::MutexLock lock(&queue_mutex_);
  // std::queue does not define a clear method
  q_ = std::queue<T>();
}

}  // namespace internal
}  // namespace mako
#endif  // INTERNAL_CXX_QUEUE_H_
