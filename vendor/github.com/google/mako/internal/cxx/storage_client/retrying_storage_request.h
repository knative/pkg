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

// Helper module for making retrying storage calls.
#ifndef INTERNAL_CXX_STORAGE_CLIENT_RETRYING_STORAGE_REQUEST_H_
#define INTERNAL_CXX_STORAGE_CLIENT_RETRYING_STORAGE_REQUEST_H_

#include <string>

#include "glog/logging.h"
#include "src/google/protobuf/map.h"
#include "src/google/protobuf/message.h"
#include "internal/proto/mako_internal.pb.h"
#include "absl/strings/str_cat.h"
#include "absl/time/time.h"
#include "helpers/cxx/status/canonical_errors.h"
#include "helpers/cxx/status/status.h"
#include "internal/cxx/storage_client/retry_strategy.h"
#include "internal/cxx/storage_client/transport.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace internal {

inline helpers::Status ConnectAndCall(
    mako::internal::StorageTransport* transport, const std::string& path,
    const google::protobuf::Message& request, absl::Duration deadline,
    google::protobuf::Message* response) {
  auto status = transport->Connect();
  if (!status.ok()) {
    return status;
  }
  return transport->Call(path, request, deadline, response);
}

template <typename RequestMessage, typename ReturnMessage>
bool RetryingStorageRequest(
    const RequestMessage& request, const std::string& url,
    const std::string& telemetry_action, ReturnMessage* response,
    mako::internal::StorageTransport* transport,
    mako::internal::StorageRetryStrategy* retry_strategy) {
  constexpr absl::Duration kRPCDeadline = absl::Seconds(65);

  std::string result = "FAIL";
  int attempt = 0;

  retry_strategy->Do([&]() {
    ++attempt;
    response->Clear();

    // Doing both the Connect and the Call in this loop allows us to
    // conveniently use the same retry logic for connection and transmission.
    // Once the transport has been connected once, further calls to Connect are
    // no-ops.
    helpers::Status status =
        ConnectAndCall(transport, url, request, kRPCDeadline, response);
    if (!status.ok()) {
      // Insert transport-level errors into the response message. This makes
      // the user aware of what happened, regardless of the originating layer of
      // the layer, and it unifies our error handling below.
      bool retryable = !helpers::IsFailedPrecondition(status);
      response->mutable_status()->set_retry(retryable);
      response->mutable_status()->set_fail_message(status.message());
      response->mutable_status()->set_code(Status::FAIL);
    }

    if (!response->has_status()) {
      std::string error = absl::StrCat(
          "Unknown error in Storage. Response was returned without a Status. "
          "This is probably a Mako bug. Please report it at "
          "go/mako-bug. Will try retrying. Request: ",
          request.ShortDebugString(),
          "\nResponse: ", response->ShortDebugString());
      LOG(ERROR) << error;
      response->mutable_status()->set_retry(true);
      response->mutable_status()->set_fail_message(error);
      response->mutable_status()->set_code(Status::FAIL);
    }

    for (const auto& warning_msg : response->status().warning_messages()) {
      LOG(WARNING) << "Response has warning: " << warning_msg;
    }

    if (response->status().code() == Status::SUCCESS) {
      result = "SUCCESS";
      return mako::internal::StorageRetryStrategy::kBreak;
    }

    LOG(WARNING) << "Response has failure; response status: "
                 << response->status().ShortDebugString();
    if (!response->status().retry()) {
      LOG(WARNING) << "Failure is not marked retryable. Returning error to "
                   << "user, after " << attempt << " tries.";
      return mako::internal::StorageRetryStrategy::kBreak;
    }

    return mako::internal::StorageRetryStrategy::kContinue;
  });

  return response->status().code() == Status::SUCCESS;
}

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_STORAGE_CLIENT_RETRYING_STORAGE_REQUEST_H_
