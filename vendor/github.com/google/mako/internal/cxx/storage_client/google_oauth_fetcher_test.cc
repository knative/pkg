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

#include "internal/cxx/storage_client/google_oauth_fetcher.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "absl/strings/str_cat.h"
#include "google/cloud/status.h"
#include "helpers/cxx/status/status_matchers.h"

// Tests that we properly handle the result of
// google::cloud::storage::oauth2::Credentials::AuthorizationHeader().
//
// This does not test the ability to fetch oauth tokens in different Application
// Default Credentials environments/configurations. That's handled by
// google::cloud::storage::oauth2::GoogleDefaultCredentials()'s tests, and will
// be covered by Mako's Mako end-to-end tests (b/122905108).

namespace mako {
namespace internal {
namespace {

using ::mako::helpers::IsOkAndHolds;
using ::mako::helpers::StatusIs;
using ::testing::AllOf;
using ::testing::HasSubstr;

using HeaderResult = google::cloud::StatusOr<std::string>;

HeaderResult SuccessfulResult(const std::string& header) { return {header}; }

HeaderResult FailureResult(google::cloud::StatusCode code,
                           const std::string& message) {
  return google::cloud::Status(code, message);
}

TEST(GoogleOauthFetcherTest, ParsesHeader) {
  const std::string token =
      "ya29.c.El2TBkjVCI-32GrKp9lDJDptZn5NHnIL9NF7Oc_zstErD1EiR6_"
      "fUJRzxTiWbV79DCJmbdHSAnYYt1QQBP4yV42FLXn3qYVLHHUfvPyJIfEIKyD6vVjQVEwW0TN"
      "A8yk";
  auto status_or_token = GoogleOAuthFetcher::ParseAuthorizationHeader(
      SuccessfulResult(absl::StrCat("Authorization: Bearer ", token)));
  EXPECT_THAT(status_or_token, IsOkAndHolds(token));
}

TEST(GoogleOauthFetcherTest, Empty) {
  auto status_or_token =
      GoogleOAuthFetcher::ParseAuthorizationHeader(SuccessfulResult(""));
  EXPECT_THAT(status_or_token, StatusIs(helpers::StatusCode::kInternal));
}

TEST(GoogleOauthFetcherTest, MissingToken) {
  auto status_or_token = GoogleOAuthFetcher::ParseAuthorizationHeader(
      SuccessfulResult("Authorization: Bearer "));
  EXPECT_THAT(status_or_token, StatusIs(helpers::StatusCode::kInternal));
}

TEST(GoogleOauthFetcherTest, TwoTokens) {
  const std::string token = "token1 token2";
  auto status_or_token = GoogleOAuthFetcher::ParseAuthorizationHeader(
      SuccessfulResult(absl::StrCat("Authorization: Bearer ", token)));
  EXPECT_THAT(status_or_token, StatusIs(helpers::StatusCode::kInternal));
}

TEST(GoogleOauthFetcherTest, TokenFailure) {
  const std::string error = "Internal Server Error";
  auto status_or_token = GoogleOAuthFetcher::ParseAuthorizationHeader(
      FailureResult(google::cloud::StatusCode::kInternal, error));
  EXPECT_THAT(status_or_token,
              StatusIs(helpers::StatusCode::kInternal,
                       AllOf(HasSubstr(error), HasSubstr("INTERNAL"))));
}

}  // namespace
}  // namespace internal
}  // namespace mako
