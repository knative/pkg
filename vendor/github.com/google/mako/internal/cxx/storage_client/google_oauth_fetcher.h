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
#ifndef INTERNAL_CXX_STORAGE_CLIENT_GOOGLE_OAUTH_FETCHER_H_
#define INTERNAL_CXX_STORAGE_CLIENT_GOOGLE_OAUTH_FETCHER_H_

#include <memory>
#include <string>

#include "absl/synchronization/mutex.h"
#include "google/cloud/status.h"
#include "google/cloud/status_or.h"
#include "google/cloud/storage/oauth2/credentials.h"
#include "helpers/cxx/status/statusor.h"
#include "internal/cxx/storage_client/oauth_token_provider.h"

namespace mako {
namespace internal {

// Fetches an OAuth2 token using Application Default Credentials. See
// https://cloud.google.com/docs/authentication/production#providing_credentials_to_your_application.
//
// This class is thread-safe (go/thread-safe).
class GoogleOAuthFetcher : public OAuthTokenProvider {
 public:
  GoogleOAuthFetcher();

  helpers::StatusOr<std::string> GetBearerToken() override;

  // Exposed for testing.
  static helpers::StatusOr<std::string> ParseAuthorizationHeader(
      const google::cloud::StatusOr<std::string>&);

 private:
  std::shared_ptr<google::cloud::storage::oauth2::Credentials> credentials_;
  absl::Mutex mutex_;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_STORAGE_CLIENT_GOOGLE_OAUTH_FETCHER_H_
