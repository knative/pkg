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

#include "internal/cxx/storage_client/google_oauth_fetcher.h"

#include "glog/logging.h"
#include "absl/strings/str_cat.h"
#include "absl/types/optional.h"
#include "google/cloud/status.h"
#include "google/cloud/storage/oauth2/credentials.h"
#include "google/cloud/storage/oauth2/google_credentials.h"
#include "helpers/cxx/status/canonical_errors.h"
#include "re2/re2.h"

namespace mako {
namespace internal {
namespace {
constexpr char adc_link[] =
    "https://developers.google.com/identity/protocols/"
    "application-default-credentials";
}

GoogleOAuthFetcher::GoogleOAuthFetcher() {
  using ::google::cloud::StatusOr;
  using ::google::cloud::storage::oauth2::
      CreateServiceAccountCredentialsFromDefaultPaths;
  using ::google::cloud::storage::oauth2::Credentials;
  using ::google::cloud::storage::oauth2::GoogleDefaultCredentials;

  // TODO(b/123657925): Don't CHECK-fail here.

  // The Application Default Credentials protocol decides which type of
  // credential to create based on the environment. Some of those credentials
  // types (e.g. a service account credential created from an exported JSON
  // service account file) take the scope as an argument, and some don't (e.g.
  // credentials derived from the GCE VM's service account, which had scopes set
  // up at VM creation time).
  //
  // The google::cloud::storage::oauth2 lib doesn't have a single function which
  // takes `scopes` as an argument, and uses those scopes iff they're needed.
  //
  // So, instead, we first try to call
  // CreateServiceAccountCredentialsFromDefaultPaths, passing it the scope
  // argument it needs to create that type of credential. If that fails, we then
  // fall back on GoogleDefaultCredentials which will try all Application
  // Default Credentials credential types.

  LOG(INFO) << "Attempting to use Application Default Credentials (" << adc_link
            << ") for authentication with the Mako service.";
  StatusOr<std::shared_ptr<Credentials>> maybe_creds =
      CreateServiceAccountCredentialsFromDefaultPaths(
          {{"https://www.googleapis.com/auth/userinfo.email"}},
          /*subject=*/{});
  if (maybe_creds.ok()) {
    LOG(INFO) << "Successfully initialize service account credentials using "
                 "the GOOGLE_APPLICATION_CREDENTIALS environment variable or "
                 "well-known paths.";
    credentials_ = std::move(maybe_creds).value();
    return;
  }
  LOG(WARNING)
      << "Failure attempting to create service account credentials from the "
         "GOOGLE_APPLICATION_CREDENTIALS environment variable or well-known "
         "paths. Error was: "
      << maybe_creds.status().message()
      << "\nWill try to other credentials types, such as Google Cloud SDK "
         "login (gcloud auth application-default login) or the "
         "environment-based (e.g. GCE/GKE/AppEngine) "
         "default service account.";

  maybe_creds = GoogleDefaultCredentials();
  CHECK(maybe_creds.ok())
      << "Failure attempting to create credentials. Error was:  "
      << maybe_creds.status().message();
  credentials_ = maybe_creds.value();
  LOG(INFO) << "Success creating Google credentials.";
}

helpers::StatusOr<std::string> GoogleOAuthFetcher::GetBearerToken() {
  google::cloud::StatusOr<std::string> result;
  {
    // google::cloud::storage::oauth2::Credentials makes no thread safety
    // guarantees.
    absl::MutexLock _(&mutex_);
    result = credentials_->AuthorizationHeader();
  }
  return ParseAuthorizationHeader(result);
}

helpers::StatusOr<std::string> GoogleOAuthFetcher::ParseAuthorizationHeader(
    const google::cloud::StatusOr<std::string>& header) {
  if (!header.ok()) {
    std::stringstream error;
    error << "Problem fetching OAuth token: " << header.status();
    LOG(ERROR) << error.str();
    return helpers::InternalError(error.str());
  }

  std::string token;
  if (!RE2::FullMatch(header.value(),
                      "Authorization:\\s+Bearer\\s+([\\S]+)\\s*", &token)) {
    return helpers::InternalError(absl::StrCat(
        "Unexpected authorization header std::string ('", header.value(),
        "', expected 'Authorization: Bearer <token>'."));
  }
  return token;
}

}  // namespace internal
}  // namespace mako
