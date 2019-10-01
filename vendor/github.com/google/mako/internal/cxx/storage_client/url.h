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
#ifndef INTERNAL_CXX_STORAGE_CLIENT_URL_H_
#define INTERNAL_CXX_STORAGE_CLIENT_URL_H_

// A class for parsing URLs and providing access to the subcomponents.
#include <string>

#include "absl/strings/match.h"
#include "absl/strings/string_view.h"
#include "absl/types/optional.h"
#include "helpers/cxx/status/statusor.h"

namespace mako {
namespace internal {

class Url {
 public:
  static helpers::StatusOr<Url> Parse(absl::string_view);

  // If no scheme was specified in the parsed URL, returns 'https'.
  absl::string_view Scheme() const { return scheme_; }

  absl::string_view Host() const { return host_; }

  // If no port was specified in the parsed URL, returns the default port for
  // recognized schemes (port 80 for the 'http' scheme, 443 for the 'https'
  // scheme) and absl::nullopt otherwise.
  absl::optional<int> Port() const;

  // If no path was specified in the parsed URL, returns '/'.
  absl::string_view Path() const;

  // If no query was specified in the parsed URL, returns an empty std::string.
  absl::string_view Query() const { return query_; }

  // Returns a new Url with the given path. Any preexisting path will be
  // forgotten/ignored/replaced in the new Url.
  Url WithPath(absl::string_view);

  std::string ToString() const;

 private:
  Url(absl::string_view scheme, absl::string_view host,
      absl::optional<int> port, absl::string_view path, absl::string_view query)
      : scheme_(scheme), host_(host), port_(port), path_(path), query_(query) {}
  const std::string scheme_;
  const std::string host_;
  const absl::optional<int> port_;
  const std::string path_;
  const std::string query_;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_STORAGE_CLIENT_URL_H_
