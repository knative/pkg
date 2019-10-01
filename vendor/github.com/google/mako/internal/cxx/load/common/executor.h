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

// A way to run functions concurrently and wait for their completion.
//
// This class is thread-unsafe, as per go/thread-unsafe.
//
// This class is intended for a single use. After `Wait` returns the instance is
// unusable, and a subsequent call to `Schedule` will cause program termination.
//
// If `Wait` is not called explicitly, the destructor will block until the
// scheduled work completes.
//
// Example 1:
//
//   // Set up an executor with 5 worker threads
//   Executor executor(5);
//
//   // Submit a lambda to be executed
//   executor.Schedule([] { /* Do some work */ });
//
//   // Submit a method to be executed
//   executor.Schedule([&foo, params] { foo.DoSomething(params); });
//
//   // Block until both the previous have finished.
//   executor.Wait();
//
// Example 2:
//   {
//     // Set up an executor with 5 worker threads
//     Executor executor(5);
//
//     // Submit a lambda to be executed
//     executor.Schedule([] { /* Do some work */ });
//
//     // Rely on the Executor's destructor rather than call Wait() explicitly.
//   }

#ifndef INTERNAL_CXX_LOAD_COMMON_EXECUTOR_H_
#define INTERNAL_CXX_LOAD_COMMON_EXECUTOR_H_

#include <functional>
#include <utility>

#include "glog/logging.h"
#include "absl/synchronization/mutex.h"
#include "internal/cxx/load/common/thread_pool_factory.h"

namespace mako {
namespace internal {

class Executor {
 public:
  // Construct an Executor, specifying the number of threads that scheduled
  // functions should run on.
  explicit Executor(int num_threads)
      : thread_pool_(CreateThreadPool(num_threads)) {
    thread_pool_->StartWorkers();
  }

  ~Executor() {
    if (!did_wait_) {
      Wait();
    }
  }

  // Schedules a function to be executed and returns immediately. The executor
  // will run this as soon as it has a free thread. This method cannot be called
  // after `Wait`.
  void Schedule(const std::function<void()>& func) {
    CHECK(!did_wait_) << "mako::internal::Executor Schedule method should "
                      << "never be called after Wait().";
    {
      absl::MutexLock lock(&mutex_);
      count_++;
    }
    auto f = [func, this]() {
      func();
      absl::MutexLock lock(&mutex_);
      count_--;
    };
    thread_pool_->Schedule(std::move(f));
  }

  // Block until all previously added functions have finished executing. After
  // this call returns, the Executor instance is unusable. If not called
  // explicitly, the destructor will similarly block until the work is complete.
  // instance.
  void Wait() {
    CHECK(!did_wait_) << "mako::internal::Executor Wait method should "
                      << "never be called twice.";
    did_wait_ = true;
    absl::MutexLock lock(&mutex_);
    mutex_.Await(absl::Condition(this, &Executor::Done));
  }

 private:
  bool Done() EXCLUSIVE_LOCKS_REQUIRED(mutex_) { return count_ == 0; }

  std::unique_ptr<mako::internal::ThreadPool> thread_pool_;
  absl::Mutex mutex_;
  int count_ GUARDED_BY(mutex_) = 0;
  bool did_wait_ = false;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_LOAD_COMMON_EXECUTOR_H_
