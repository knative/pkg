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
#ifndef INTERNAL_CXX_STORAGE_CLIENT_OAUTH_TOKEN_PROVIDER_H_
#define INTERNAL_CXX_STORAGE_CLIENT_OAUTH_TOKEN_PROVIDER_H_

#include <string>

#include "helpers/cxx/status/statusor.h"

// A means of obtaining an OAuth2 Bearer token.
class OAuthTokenProvider {
 public:
  // Returns an OAuth2 Bearer token to be used in an "Authorization: Bearer
  // <token>" HTTP header.
  //
  // Implementations should return ::mako::helpers::FailedPreconditionError
  // for non-retryable errors. All other error types may be retried.
  virtual mako::helpers::StatusOr<std::string> GetBearerToken() = 0;
  virtual ~OAuthTokenProvider() = default;
};

#endif  // INTERNAL_CXX_STORAGE_CLIENT_OAUTH_TOKEN_PROVIDER_H_
