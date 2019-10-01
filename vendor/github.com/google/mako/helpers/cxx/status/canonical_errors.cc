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

#include "helpers/cxx/status/canonical_errors.h"

#include "absl/strings/string_view.h"
#include "helpers/cxx/status/status.h"

namespace mako {
namespace helpers {

Status AbortedError(absl::string_view message) {
  return Status(StatusCode::kAborted, std::string(message));
}

Status AlreadyExistsError(absl::string_view message) {
  return Status(StatusCode::kAlreadyExists, std::string(message));
}

Status CancelledError(absl::string_view message) {
  return Status(StatusCode::kCancelled, std::string(message));
}

Status DataLossError(absl::string_view message) {
  return Status(StatusCode::kDataLoss, std::string(message));
}

Status DeadlineExceededError(absl::string_view message) {
  return Status(StatusCode::kDeadlineExceeded, std::string(message));
}

Status FailedPreconditionError(absl::string_view message) {
  return Status(StatusCode::kFailedPrecondition, std::string(message));
}

Status InternalError(absl::string_view message) {
  return Status(StatusCode::kInternal, std::string(message));
}

Status InvalidArgumentError(absl::string_view message) {
  return Status(StatusCode::kInvalidArgument, std::string(message));
}

Status NotFoundError(absl::string_view message) {
  return Status(StatusCode::kNotFound, std::string(message));
}

Status OutOfRangeError(absl::string_view message) {
  return Status(StatusCode::kOutOfRange, std::string(message));
}

Status PermissionDeniedError(absl::string_view message) {
  return Status(StatusCode::kPermissionDenied, std::string(message));
}

Status ResourceExhaustedError(absl::string_view message) {
  return Status(StatusCode::kResourceExhausted, std::string(message));
}

Status UnavailableError(absl::string_view message) {
  return Status(StatusCode::kUnavailable, std::string(message));
}

Status UnimplementedError(absl::string_view message) {
  return Status(StatusCode::kUnimplemented, std::string(message));
}

Status UnknownError(absl::string_view message) {
  return Status(StatusCode::kUnknown, std::string(message));
}

bool IsAborted(const Status& status) {
  return status.code() == StatusCode::kAborted;
}

bool IsAlreadyExists(const Status& status) {
  return status.code() == StatusCode::kAlreadyExists;
}

bool IsCancelled(const Status& status) {
  return status.code() == StatusCode::kCancelled;
}

bool IsDataLoss(const Status& status) {
  return status.code() == StatusCode::kDataLoss;
}

bool IsDeadlineExceeded(const Status& status) {
  return status.code() == StatusCode::kDeadlineExceeded;
}

bool IsFailedPrecondition(const Status& status) {
  return status.code() == StatusCode::kFailedPrecondition;
}

bool IsInternal(const Status& status) {
  return status.code() == StatusCode::kInternal;
}

bool IsInvalidArgument(const Status& status) {
  return status.code() == StatusCode::kInvalidArgument;
}

bool IsNotFound(const Status& status) {
  return status.code() == StatusCode::kNotFound;
}

bool IsOutOfRange(const Status& status) {
  return status.code() == StatusCode::kOutOfRange;
}

bool IsPermissionDenied(const Status& status) {
  return status.code() == StatusCode::kPermissionDenied;
}

bool IsResourceExhausted(const Status& status) {
  return status.code() == StatusCode::kResourceExhausted;
}

bool IsUnavailable(const Status& status) {
  return status.code() == StatusCode::kUnavailable;
}

bool IsUnimplemented(const Status& status) {
  return status.code() == StatusCode::kUnimplemented;
}

bool IsUnknown(const Status& status) {
  return status.code() == StatusCode::kUnknown;
}

}  // namespace helpers
}  // namespace mako
