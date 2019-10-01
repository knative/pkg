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
#include "internal/cxx/storage_client/url.h"

#include <string>

#include "glog/logging.h"
#include "absl/container/flat_hash_map.h"
#include "absl/strings/ascii.h"
#include "absl/strings/numbers.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_format.h"
#include "absl/strings/string_view.h"
#include "absl/types/optional.h"
#include "helpers/cxx/status/canonical_errors.h"
#include "re2/re2.h"

namespace mako {
namespace internal {
namespace {
constexpr absl::string_view kDefaultScheme = "https";

const absl::flat_hash_map<std::string, int>& DefaultPorts() {
  static auto* ports = new absl::flat_hash_map<std::string, int>({
    {"http", 80},
    {"https", 443},
  });
  return *ports;
}

}  // namespace

absl::optional<int> Url::Port() const {
  if (!port_.has_value()) {
    auto it = DefaultPorts().find(scheme_);
    if (it != DefaultPorts().end()) {
      return it->second;
    }
    return absl::nullopt;
  }
  return port_;
}

absl::string_view Url::Path() const {
  if (path_.empty()) {
    return "/";
  }
  return path_;
}

helpers::StatusOr<Url> Url::Parse(absl::string_view url_str) {
  using size_type = ::absl::string_view::size_type;
  constexpr size_type npos = ::absl::string_view::npos;

  absl::string_view scheme = kDefaultScheme;
  size_type scheme_end = url_str.find("://");
  if (scheme_end == 0) {
    return helpers::InvalidArgumentError(
        "Misformed URL, could not parse scheme");
  }
  if (scheme_end != npos) {
    scheme = url_str.substr(0, scheme_end);
    CHECK_GE(url_str.size(), scheme_end + 3);
    url_str = url_str.substr(scheme_end + 3);  // advance past the '://'
  }

  if (url_str.empty()) {
    return helpers::InvalidArgumentError("Misformed URL, no host");
  }

  absl::string_view host;
  if (url_str[0] == '[') {
    // Handle ipv6 hosts.
    size_type host_end = url_str.find("]");
    if (host_end == npos) {
      return helpers::InvalidArgumentError(
          "Misformed URL, found '[' implying the start of an IPv6 host, but "
          "did not find the matching ']'");
    }
    host = url_str.substr(0, host_end + 1);
    CHECK(!host.empty());
    url_str = url_str.substr(host_end + 1);
  } else {
    // Handle non-ipv6 hosts.
    size_type past_host = url_str.find_first_of("/:?");
    if (past_host == 0) {
      return helpers::InvalidArgumentError("Misformed URL, no host");
    }

    host = url_str.substr(0, past_host);
    CHECK(!host.empty());

    if (past_host != npos) {
      url_str = url_str.substr(past_host);
    } else {
      url_str = "";
    }
  }

  if (url_str.empty()) {
    return Url(scheme, host, /*port=*/absl::nullopt, /*path=*/"",
               /*query=*/"");
  }

  absl::optional<int> port;
  if (url_str[0] == ':') {
    size_type port_end = url_str.find('/');
    absl::string_view port_str = url_str.substr(0, port_end);
    port_str.remove_prefix(1);  // remove the ':'
    int iport;
    if (!absl::SimpleAtoi(port_str, &iport)) {
      return helpers::InvalidArgumentError(
          absl::StrFormat("Misformed URL, unrecognized port (%s)", port_str));
    }
    if (iport < 0) {
      return helpers::InvalidArgumentError(
          absl::StrFormat("Misformed URL, bad port: %s", port_str));
    }
    port = iport;
    if (port_end == npos) {
      return Url(scheme, host, port, /*path=*/"", /*query=*/"");
    }
    url_str = url_str.substr(port_end);
  }

  CHECK(!url_str.empty());
  CHECK(url_str[0] == '/' || url_str[0] == '?');

  absl::string_view path;
  if (url_str[0] == '/') {
    size_type path_end = url_str.find('?');
    path = url_str.substr(0, path_end);

    if (path_end == npos) {
      return Url(scheme, host, port, path, /*query=*/"");
    }

    url_str = url_str.substr(path_end + 1);  // advance past the '?'
  }

  if (absl::StartsWith(url_str, "?")) {
    url_str.remove_prefix(1);
  }
  absl::string_view query = url_str;

  return Url(scheme, host, port, path, query);
}

Url Url::WithPath(absl::string_view path) {
  std::string normalized_path(path);
  if (!normalized_path.empty() && normalized_path[0] != '/') {
    normalized_path = absl::StrCat("/", path);
  }
  return Url{scheme_, host_, port_, normalized_path, query_};
}

std::string Url::ToString() const {
  std::string port_str = "";
  if (port_.has_value()) {
    auto it = DefaultPorts().find(scheme_);
    if (it == DefaultPorts().end() || port_.value() != it->second) {
      // If we don't have a default scheme/port combo, we need to print the
      // port.
      port_str = absl::StrCat(":", port_.value());
    }
  }
  std::string query_str = "";
  if (!query_.empty()) {
    query_str = absl::StrCat("?", query_);
  }
  std::string path_str(path_);
  if (path_.empty()) {
    path_str = "/";
  }
  return absl::StrCat(scheme_, "://", host_, port_str, path_str, query_str);
}

}  // namespace internal
}  // namespace mako
