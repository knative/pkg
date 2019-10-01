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

#include "internal/cxx/clock_external.h"

#include "absl/time/clock.h"
#include "absl/time/time.h"

namespace mako {
namespace external_helpers {
namespace {

class SimpleClock : public Clock {
 public:
  ~SimpleClock() override {}
  absl::Time TimeNow() override;
  void Sleep(absl::Duration d) override;
};

}  // namespace

Clock* Clock::RealClock() {
  static Clock* clock = new SimpleClock;
  return clock;
}

absl::Time SimpleClock::TimeNow() {
  return absl::Now();
}

void SimpleClock::Sleep(absl::Duration d) {
  absl::SleepFor(d);
}


}  // namespace external_helpers
}  // namespace mako
