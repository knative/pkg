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

// An interface describing an HTTP client, and a simple concrete implementation.
//
#ifndef INTERNAL_CXX_STORAGE_CLIENT_HTTP_CLIENT_H_
#define INTERNAL_CXX_STORAGE_CLIENT_HTTP_CLIENT_H_

#include <string>
#include <utility>
#include <vector>

#include "absl/strings/string_view.h"
#include "helpers/cxx/status/statusor.h"

namespace mako {
namespace internal {

// An HTTP client interface, capable of handling GET and POST requests.
//
// HttpClientInterface implementations must be thread-safe (as per
// go/thread-safe). All methods block until the request and response are
// transmitted.
//
// In the case of success, returns the response body.
// TODO(b/138086731): Readability: Consider a struct instead of std::pair<s,s>.
// TODO(b/75454419): Support timeouts.
class HttpClientInterface {
 public:
  virtual ~HttpClientInterface() = default;

  // Sends a GET request to `url` and returns the response. Possible errors are:
  // - Status(StatusCode::kUnavailable) - retryable errors.
  // - Status(StatusCode::kFailedPrecondition) - non-retryable errors.
  // TODO(b/74948849): Rethink mapping of HTTP status codes to canonical errors.
  virtual helpers::StatusOr<std::string> Get(absl::string_view url) = 0;

  // Sends a GET request to `url` with the given `headers` (as key-value pairs),
  // and returns the response. Possible errors are:
  // - Status(StatusCode::kUnavailable) - retryable errors.
  // - Status(StatusCode::kFailedPrecondition) - non-retryable errors.
  // TODO(b/74948849): Rethink mapping of HTTP status codes to canonical errors.
  virtual helpers::StatusOr<std::string> Get(
      absl::string_view url,
      const std::vector<std::pair<std::string, std::string>>& headers) = 0;

  // Sends a POST request to `url` with `data` as the request body, and
  // returns the response. Possible errors are:
  // - Status(StatusCode::kUnavailable) - retryable errors.
  // - Status(StatusCode::kFailedPrecondition) - non-retryable errors.
  // TODO(b/74948849): Rethink mapping of HTTP status codes to canonical errors.
  virtual helpers::StatusOr<std::string> Post(absl::string_view url,
                                              absl::string_view data) = 0;

  // Sends a POST request to `url` with the given `headers` (as key-value pairs)
  // and with `data` as the request body, and returns the response. Possible
  // errors are:
  // - Status(StatusCode::kUnavailable) - retryable errors.
  // - Status(StatusCode::kFailedPrecondition) - non-retryable errors.
  // TODO(b/74948849): Rethink mapping of HTTP status codes to canonical errors.
  virtual helpers::StatusOr<std::string> Post(
      absl::string_view url,
      const std::vector<std::pair<std::string, std::string>>& headers,
      absl::string_view data) = 0;
};

// A simple and stateless implementation of HttpClientInterface.
class HttpClient : public HttpClientInterface {
 public:
  HttpClient();

  // Override the default CA Certificate path. See
  // https://curl.haxx.se/libcurl/c/CURLOPT_CAINFO.html.
  explicit HttpClient(absl::string_view ca_certificate_path);

  helpers::StatusOr<std::string> Get(absl::string_view url) override;

  helpers::StatusOr<std::string> Get(
      absl::string_view url,
      const std::vector<std::pair<std::string, std::string>>& headers) override;

  helpers::StatusOr<std::string> Post(absl::string_view url,
                                      absl::string_view data) override;

  helpers::StatusOr<std::string> Post(
      absl::string_view url,
      const std::vector<std::pair<std::string, std::string>>& headers,
      absl::string_view data) override;

 private:
  std::string ca_certificate_path_;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_STORAGE_CLIENT_HTTP_CLIENT_H_
