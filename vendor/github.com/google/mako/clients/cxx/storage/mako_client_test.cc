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
#include "clients/cxx/storage/mako_client.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/cxx/storage/google3_storage.h"
#include "absl/flags/flag.h"
#include "internal/cxx/storage_client/google_oauth_fetcher.h"
#include "internal/cxx/storage_client/http_transport.h"
#include "internal/cxx/storage_client/oauth_token_provider.h"
#include "internal/cxx/storage_client/transport.h"

ABSL_DECLARE_FLAG(bool, mako_auth);
ABSL_DECLARE_FLAG(std::string, mako_auth_service_account);
ABSL_DECLARE_FLAG(bool, mako_auth_force_adc);
ABSL_DECLARE_FLAG(bool, mako_internal_auth_testuser_ok);

namespace mako {
namespace {

using ::absl::flags_internal::FlagSaver;

OAuthTokenProvider* GetTokenProvider(BaseStorageClient* client) {
  return dynamic_cast<internal::HttpTransport*>(client->transport())
      ->token_provider();
}
TEST(MakoStorageTest, NoAuthNoHostnameProvided) {
  FlagSaver flag_saver;
  absl::SetFlag(&FLAGS_mako_auth, false);
  auto client = NewMakoClient();
  EXPECT_EQ(GetTokenProvider(client.get()), nullptr);
  EXPECT_EQ(client->GetHostname(), "makoperf.appspot.com");
}

TEST(MakoStorageTest, NoAuthHostnameProvided) {
  FlagSaver flag_saver;
  absl::SetFlag(&FLAGS_mako_auth, false);
  const std::string host = "http://example.com";
  auto client = NewMakoClient(host);
  EXPECT_EQ(GetTokenProvider(client.get()), nullptr);
  EXPECT_EQ(client->GetHostname(), host);
}

}  // namespace
}  // namespace mako
