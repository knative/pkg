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
#ifndef INTERNAL_CXX_STORAGE_CLIENT_MOCK_TRANSPORT_H_
#define INTERNAL_CXX_STORAGE_CLIENT_MOCK_TRANSPORT_H_

#include "src/google/protobuf/message.h"
#include "gmock/gmock.h"
#include "absl/strings/string_view.h"
#include "absl/time/time.h"
#include "helpers/cxx/status/status.h"
#include "internal/cxx/storage_client/transport.h"

namespace mako {
namespace internal {

class MockTransport : public mako::internal::StorageTransport {
 public:
  MOCK_METHOD0(Connect, helpers::Status());
  MOCK_METHOD4(Call, helpers::Status(absl::string_view path,
                                     const google::protobuf::Message& request,
                                     absl::Duration deadline,
                                     google::protobuf::Message* response));
  MOCK_CONST_METHOD0(last_call_server_elapsed_time, absl::Duration());
  MOCK_METHOD1(set_client_tool_tag, void(absl::string_view));
  MOCK_METHOD0(GetHostname, std::string());
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_STORAGE_CLIENT_MOCK_TRANSPORT_H_
