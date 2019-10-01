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

#include "internal/cxx/storage_client/http_client.h"

#include <stddef.h>

#include <functional>
#include <string>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "absl/strings/match.h"
#include "absl/strings/numbers.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_join.h"
#include "absl/strings/string_view.h"
#include "absl/types/optional.h"
#include "curl/curl.h"
#include "helpers/cxx/status/canonical_errors.h"
#include "internal/cxx/utils/cleanup.h"
#include "internal/cxx/utils/googleinit.h"
#include "internal/cxx/utils/stringutil.h"

MAKO_MODULE_INITIALIZER(libcurlinit, {
  // Since libcurl's global initialization is not thread-safe, libcurl docs
  // recommend it be initialized as early as possible in the process. Other
  // modules in the process may do their own libcurl initialization. The library
  // itself ensures only the first one counts and subsequent calls are no-ops.
  int result = curl_global_init(CURL_GLOBAL_ALL);
  if (result != 0) {
    LOG(FATAL) << "Failed to initialize libcurl; curl_global_init() returned "
               << result << ".";
  }
});

namespace mako {
namespace internal {
namespace {

constexpr int kPayloadLogCharLimit = 1000;

enum class Methods { kGet, kPost };

// Read the status code from an HTTP header status line (aka "first line").
absl::optional<int> ReadStatusCode(absl::string_view line) {
  // From: https://tools.ietf.org/html/rfc2616
  // Status-Line = HTTP-Version SP Status-Code SP Reason-Phrase CRLF
  // HTTP-Version   = "HTTP" "/" 1*DIGIT "." 1*DIGIT
  const std::string kHttp = "HTTP/";
  if (!absl::StartsWith(line, kHttp)) {
    return absl::nullopt;
  }
  line = line.substr(kHttp.size());

  // RFC says to ignore leading 0s, but I've never witnessed an HTTP-Version
  // with leading 0s.
  if (!absl::StartsWith(line, "1.1 ") && !absl::StartsWith(line, "1.2 ")) {
    // We only support HTTP1.x
    return absl::nullopt;
  }
  line = line.substr(4);  // len("1.1 "|"1.2 ") == 4

  absl::string_view::size_type space_index = line.find(' ');
  if (space_index == absl::string_view::npos || space_index == 0) {
    return absl::nullopt;
  }

  int status_code;
  absl::string_view status_line = line.substr(0, space_index);
  if (!absl::SimpleAtoi(status_line, &status_code)) {
    return absl::nullopt;
  }

  return absl::make_optional(status_code);
}

// An instance of this will be stored in the HeaderCallback `userdata`.
struct HeaderCallbackHelper {
  std::vector<std::string> lines;
};

// This gets called to process each HTTP response header line.
size_t HeaderCallback(char* ptr, size_t size, size_t nitems, void* userdata) {
  auto helper = static_cast<HeaderCallbackHelper*>(userdata);

  const int data_length = size * nitems;
  absl::string_view view(ptr, data_length);

  // Deal with \r\n
  TrimWhitespace(&view);

  if (view.empty()) {
    return data_length;
  }

  VLOG(2) << "Received header line: " << view;
  helper->lines.emplace_back(view);

  return data_length;
}

// An instance of this will be stored in the BodyCallback `userdata`.
struct BodyCallbackHelper {
  std::string body;
};

// This gets called to process the HTTP response body.
size_t BodyCallback(char* ptr, size_t size, size_t nmemb, void* userdata) {
  auto helper = static_cast<BodyCallbackHelper*>(userdata);

  const int data_length = size * nmemb;
  absl::string_view body(ptr, data_length);

  // This should be the only copy of the data we make.
  absl::StrAppend(&helper->body, body);

  VLOG(2) << "Received response body (truncated to " << kPayloadLogCharLimit
          << " chars):\n"
          << body.substr(0, kPayloadLogCharLimit);

  return data_length;
}

// Sets some headers particular to our needs. Returns an RAII object which will
// ensure memory gets cleaned up. The object should be kept in scope until
// curl_handle is guaranteed to no longer be used.
//
// curl_handle must not be nullptr. token_provider may be.
Cleanup<std::function<void()>> SetHeaders(
    CURL* curl_handle,
    const std::vector<std::pair<std::string, std::string>>& headers) {
  // Typically one would use `curlslist* http_headers = nullptr` to hold the
  // head of the headers linked list. But since we want to capture it in a
  // cleanup function while allowing it to change values over the course of the
  // function, we need to capture it indirectly. Since the closure escapes this
  // function, we cannot accomplish that by simply capturing http_headers by
  // reference. Instead we allocate space for the pointer dynamically, capture
  // the pointer to that space by value, and delete the space in the cleanup
  // along with the linked list.
  curl_slist** http_headers = new curl_slist*;
  *http_headers = nullptr;
  std::function<void()> cleanup_function = [http_headers]() {
    curl_slist_free_all(*http_headers);
    delete http_headers;
  };
  auto cleanup = MakeCleanup(cleanup_function);

  for (auto const& kv : headers) {
    *http_headers = curl_slist_append(
        *http_headers, absl::StrCat(kv.first, ": ", kv.second).c_str());
  }

  // Give curl the pointer to the headers.
  curl_easy_setopt(curl_handle, CURLOPT_HTTPHEADER, *http_headers);

  return cleanup;
}

helpers::StatusOr<std::string> Request(
    Methods method, absl::string_view url,
    const std::vector<std::pair<std::string, std::string>>& headers,
    absl::string_view post_data, absl::string_view ca_cert_path) {
  CHECK(post_data.empty() || method == Methods::kPost)
      << "Cannot send post data if method is not kPost. This is an internal "
         "Mako error, please file a bug at go/mako-bug";

  // The use of a new handle per Request ensures thread safety.
  CURL* curl_handle = curl_easy_init();
  if (curl_handle == nullptr) {
    return helpers::FailedPreconditionError(
        "Unknown error initializing curl lib. Check stderr for potential "
        "output from libcurl.");
  }
  auto curl_handle_cleaner =
      MakeCleanup([curl_handle] { curl_easy_cleanup(curl_handle); });

  // Set the certificate authority path if one was provided.
  if (!ca_cert_path.empty()) {
    curl_easy_setopt(curl_handle, CURLOPT_CAINFO, ca_cert_path.data());
  }

  // curl_easy_setopt does not take ownership of the std::string returned by Assemble
  curl_easy_setopt(curl_handle, CURLOPT_URL, url.data());

  char error_buffer[CURL_ERROR_SIZE] = {0};
  curl_easy_setopt(curl_handle, CURLOPT_ERRORBUFFER, error_buffer);

  HeaderCallbackHelper header_helper;
  BodyCallbackHelper body_helper;

  // Set up response callbacks.
  curl_easy_setopt(curl_handle, CURLOPT_HEADERFUNCTION, HeaderCallback);
  curl_easy_setopt(curl_handle, CURLOPT_HEADERDATA, &header_helper);
  curl_easy_setopt(curl_handle, CURLOPT_WRITEFUNCTION, BodyCallback);
  curl_easy_setopt(curl_handle, CURLOPT_WRITEDATA, &body_helper);

  // Set request body.
  if (!post_data.empty()) {
    curl_easy_setopt(curl_handle, CURLOPT_POSTFIELDS, post_data.data());
    curl_easy_setopt(curl_handle, CURLOPT_POSTFIELDSIZE, post_data.size());
  }

  // Set headers.
  auto header_cleanup = SetHeaders(curl_handle, headers);

  VLOG(2) << "libcurl is set up, making the curl_easy_perform call";
  CURLcode err = curl_easy_perform(curl_handle);
  VLOG(2) << "curl_easy_perform call returned " << err;
  if (err == CURLE_UNSUPPORTED_PROTOCOL) {
    const std::string error = absl::StrCat(
        "HTTP request had problem with HTTP status line: ", error_buffer);
    LOG(ERROR) << error;
    return helpers::FailedPreconditionError(error);
  } else if (err != CURLE_OK) {
    // Other errors from libcurl.
    const std::string error = absl::StrCat(
        "HTTP request had error: ", error_buffer, "\nfor request to ", url);
    LOG(ERROR) << error;
    // Rather than try to enumerate which of the possible libcurl errors
    // (https://curl.haxx.se/libcurl/c/libcurl-errors.html) should be considered
    // retryable, assume all are retryable.
    return helpers::UnavailableError(error);
  }

  if (header_helper.lines.empty()) {
    const std::string error =
        absl::StrCat("HTTP response from ", url,
                     " does not appear valid: no HTTP status line.");
    LOG(ERROR) << error;
    // TODO(b/74948849) Maybe should be mako::helpers::InternalError(error).
    return helpers::FailedPreconditionError(error);
  }

  absl::optional<int> status_code = ReadStatusCode(header_helper.lines[0]);
  if (!status_code.has_value()) {
    const std::string error =
        absl::StrCat("HTTP response from ", url,
                     " does not appear valid: HTTP status line (should be the "
                     "first header line) is missing or malformed. Headers:\n",
                     absl::StrJoin(header_helper.lines, "\n"));
    LOG(ERROR) << error;
    // TODO(b/74948849) Maybe should be mako::helpers::InternalError(error).
    return helpers::FailedPreconditionError(error);
  }

  if (status_code.value() >= 200 && status_code.value() < 300) {
    if (status_code.value() != 200) {
      LOG(WARNING) << "HTTP response code (" << status_code.value()
                   << ") was not an error code but was not 200/OK. This client "
                      "treats all 2XX codes identically.";
    }
    return body_helper.body;
  }

  std::string error = absl::StrCat("HTTP request to ", url,
                                   " received an unsuccessful HTTP status ",
                                   "code (", status_code.value(), ").");
  if (status_code.value() >= 400 && status_code.value() < 500) {
    // In this situation the response body often contains a human-readable
    // description of the problem. It's useful to stuff it into the error we
    // return, rather than just LOG(ERROR) it (like we do above when we expect a
    // binary payload) and force the user to consult the logs.
    absl::StrAppend(&error, ", payload (first ", kPayloadLogCharLimit,
                    " chars):\n",
                    body_helper.body.substr(0, kPayloadLogCharLimit));
    // 4XX errors are not retryable. Something is wrong with the request.
    // TODO(b/74948849): We may want a more careful mapping to canonical error
    // space.
    return helpers::FailedPreconditionError(error);
  }

  return helpers::UnavailableError(error);
}

}  // namespace

HttpClient::HttpClient() : HttpClient("") {}

HttpClient::HttpClient(absl::string_view ca_certificate_path) {
  ca_certificate_path_ = std::string(ca_certificate_path);
}

helpers::StatusOr<std::string> HttpClient::Get(absl::string_view url) {
  // The tests in http_client_test.cc assume that all requests go through the
  // same logic (the `Request` function), so for brevity many scenarios are
  // tested only for Get. If the validity of the assumption ever changes, update
  // the tests.
  return Get(url, /*headers=*/{});
}

helpers::StatusOr<std::string> HttpClient::Get(
    absl::string_view url,
    const std::vector<std::pair<std::string, std::string>>& headers) {
  // The tests in http_client_test.cc assume that all requests go through the
  // same logic (the `Request` function), so for brevity many scenarios are
  // tested only for Get. If the validity of the assumption ever changes, update
  // the tests.
  return Request(Methods::kGet, url, headers, /*post_data=*/"",
                 ca_certificate_path_);
}

helpers::StatusOr<std::string> HttpClient::Post(absl::string_view url,
                                                absl::string_view data) {
  // The tests in http_client_test.cc assume that all requests go through the
  // same logic (the `Request` function), so for brevity many scenarios are
  // tested only for Get. If the validity of the assumption ever changes, update
  // the tests.
  return Post(url, /*headers=*/{}, data);
}

helpers::StatusOr<std::string> HttpClient::Post(
    // The tests in http_client_test.cc assume that all requests go through the
    // same logic (the `Request` function), so for brevity many scenarios are
    // tested only for Get. If the validity of the assumption ever changes,
    // update the tests.
    absl::string_view url,
    const std::vector<std::pair<std::string, std::string>>& headers,
    absl::string_view data) {
  return Request(Methods::kPost, url, headers, data, ca_certificate_path_);
}

}  // namespace internal
}  // namespace mako
