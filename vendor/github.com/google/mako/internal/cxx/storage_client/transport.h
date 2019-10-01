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
#ifndef INTERNAL_CXX_STORAGE_CLIENT_TRANSPORT_H_
#define INTERNAL_CXX_STORAGE_CLIENT_TRANSPORT_H_

#include "src/google/protobuf/message.h"
#include "absl/strings/string_view.h"
#include "absl/time/time.h"
#include "helpers/cxx/status/status.h"

namespace mako {
namespace internal {

// `mako::google3_storage::Storage` uses a `StorageTransport` to communicate
// with the server.
//
// Implementations should be thread-safe as per go/thread-safe.
class StorageTransport {
 public:
  virtual ~StorageTransport() = default;

  // Do whatever connection is necessary for the transport. Will be called at
  // least once before Call is called. Calls to Connect() after a successful
  // connection should be no-ops.
  //
  // mako::helpers::StatusCode::kFailedPrecondition indicates an error that
  // is not retryable.
  virtual helpers::Status Connect() = 0;

  virtual void set_client_tool_tag(absl::string_view) = 0;

  // Make the specified call. Return Status codes other than
  // mako::helpers::StatusCode::kOk indicate a transport-layer error (e.g.
  // failed to send an RPC). Storage API errors will be returned via the
  // response with an OK Status.
  //
  // mako::helpers::StatusCode::kFailedPrecondition indicates an error that
  // is not retryable.
  virtual helpers::Status Call(absl::string_view path,
                               const google::protobuf::Message& request,
                               absl::Duration deadline,
                               google::protobuf::Message* response) = 0;

  // Returns the number of seconds that the last RPC call took (according to the
  // server). This is exposed for tests of this library to use and should not
  // otherwise be relied on. This is also not guaranteed to be correct in
  // multi-threaded situations.
  virtual absl::Duration last_call_server_elapsed_time() const = 0;

  // The hostname backing this Storage implementation.
  // Returns a URL without the trailing slash.
  virtual std::string GetHostname() = 0;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_STORAGE_CLIENT_TRANSPORT_H_
