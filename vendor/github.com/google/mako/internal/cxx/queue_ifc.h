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
#ifndef INTERNAL_CXX_QUEUE_IFC_H_
#define INTERNAL_CXX_QUEUE_IFC_H_
#include <cstddef>

#include "absl/base/macros.h"
#include "absl/time/time.h"
#include "absl/types/optional.h"

namespace mako {
namespace internal {

// A typed Queue that is thread-safe.
template <typename T>
class ConsumerQueueInterface {
 public:
  virtual ~ConsumerQueueInterface() {}

  // Wait up to timeout for an item to become available on the queue and
  // remove it.
  virtual absl::optional<T> get(absl::Duration timeout) = 0;

  // Wait until an item is avaiable on the queue and remove it.
  virtual T get() = 0;

  // Return the current size of the queue.
  virtual std::size_t size() = 0;

  // Bool if the queue is currently empty.
  virtual bool empty() = 0;
};

template <typename T>
class ProducerQueueInterface {
 public:
  virtual ~ProducerQueueInterface() {}

  // Put an item into the queue, return true for success/false otherwise.
  virtual bool put(T item) = 0;
};

template <typename T>
class QueueInterface : public ConsumerQueueInterface<T>,
                       public ProducerQueueInterface<T> {
 public:
  ~QueueInterface() override {}

  // Make the queue empty
  virtual void clear() = 0;
};

}  // namespace internal
}  // namespace mako
#endif  // INTERNAL_CXX_QUEUE_IFC_H_
