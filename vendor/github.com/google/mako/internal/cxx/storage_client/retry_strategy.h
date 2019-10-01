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

#ifndef INTERNAL_CXX_STORAGE_CLIENT_RETRY_STRATEGY_H_
#define INTERNAL_CXX_STORAGE_CLIENT_RETRY_STRATEGY_H_

#include <functional>

#include "glog/logging.h"
#include "absl/time/time.h"
#include "internal/cxx/backoff.h"
#include "internal/cxx/clock.h"

namespace mako {
namespace internal {

// An abstraction representing a policy for retrying a Mako Storage API call
// until it succeeds. `mako::google3_storage::Storage` uses an
// implementation of this interface to retry storage calls.
//
// Implementations should be thread-safe as per go/thread-safe.
class StorageRetryStrategy {
 public:
  enum Step {
    kBreak = 0,
    kContinue,
  };
  // Call `f`, retrying as needed until it returns kBreak or the
  // implementation's policy determines it should stop.
  //
  // Note: Implementations should not extend the lifetime of `f` beyond the
  // scope of the Do call.
  virtual void Do(std::function<Step()> f) = 0;
  virtual ~StorageRetryStrategy() {}
};

// Sleep between retries, with the backoff amount computed using
// util_time::ComputeBackoff.
class StorageBackoff : public StorageRetryStrategy {
 public:
  StorageBackoff(absl::Duration timeout, absl::Duration min_sleep,
                 absl::Duration max_sleep)
      : StorageBackoff(timeout, min_sleep, max_sleep,
                       mako::helpers::Clock::RealClock()) {}
  StorageBackoff(absl::Duration timeout, absl::Duration min_sleep,
                 absl::Duration max_sleep, mako::helpers::Clock *clock)
      : clock_(clock),
        timeout_(timeout),
        min_sleep_(min_sleep),
        max_sleep_(max_sleep) {}

  void Do(std::function<Step()> f) override {
    absl::Time deadline = clock_->TimeNow() + timeout_;

    for (int retries = 0;; retries++) {
      if (f() == StorageRetryStrategy::kBreak) {
        return;
      }

      absl::Duration sleep_duration =
          ComputeBackoff(min_sleep_, max_sleep_, retries);
      if ((clock_->TimeNow() + sleep_duration) > deadline) {
        LOG(WARNING) << "After " << (retries + 1) << " failed tries, could not "
                     << "complete storage operation within timeout of "
                     << timeout_ << "; aborting.";
        break;
      }
      LOG(WARNING) << "Sleeping " << sleep_duration << " before try number "
                   << (retries + 2);
      clock_->Sleep(sleep_duration);
    }
  }

 private:
  mako::helpers::Clock *clock_;
  absl::Duration timeout_;
  absl::Duration min_sleep_;
  absl::Duration max_sleep_;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_STORAGE_CLIENT_RETRY_STRATEGY_H_
