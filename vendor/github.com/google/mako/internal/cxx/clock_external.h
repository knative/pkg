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

#ifndef INTERNAL_CXX_CLOCK_EXTERNAL_H_
#define INTERNAL_CXX_CLOCK_EXTERNAL_H_

#include "absl/time/time.h"

namespace mako {
namespace external_helpers {

class Clock {
 public:
  static Clock* RealClock();

  virtual ~Clock() {}
  virtual absl::Time TimeNow() = 0;
  virtual void Sleep(absl::Duration d) = 0;
};

}  // namespace external_helpers
}  // namespace mako

#endif  // INTERNAL_CXX_CLOCK_EXTERNAL_H_
