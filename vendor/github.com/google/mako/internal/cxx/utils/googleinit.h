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

#ifndef INTERNAL_CXX_UTILS_GOOGLEINIT_H_
#define INTERNAL_CXX_UTILS_GOOGLEINIT_H_

namespace mako {
namespace internal {

class GoogleInitializer {
 public:
  typedef void (*void_function)(void);
  GoogleInitializer(const char*, void_function f) { f(); }
};

#define MAKO_MODULE_INITIALIZER(name, body)                           \
  namespace {                                                             \
  static void google_init_module_##name() { body; }                       \
  mako::internal::GoogleInitializer google_initializer_module_##name( \
      #name, google_init_module_##name);                                  \
  }

}  // namespace internal
}  // namespace mako

#endif  //  INTERNAL_CXX_UTILS_GOOGLEINIT_H_
