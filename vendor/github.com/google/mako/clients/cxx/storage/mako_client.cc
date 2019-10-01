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

#include "clients/cxx/storage/mako_client.h"

#include "clients/cxx/storage/google3_storage.h"
#include "absl/flags/flag.h"
#include "absl/strings/str_cat.h"
#include "helpers/cxx/status/statusor.h"
#include "internal/cxx/storage_client/google_oauth_fetcher.h"
#include "internal/cxx/storage_client/http_transport.h"

ABSL_FLAG(bool, mako_auth, true,
          "Whether to attempt to generate an OAuth token and use it to "
          "authenticate with the Mako server. If false, requests are sent "
          "without authentication.");

ABSL_FLAG(std::string, mako_auth_ca_cert, "",
          "The location of the server SSL certificate file to check the "
          "server certificate against. "
);

namespace mako {

namespace {

// The hostname of the Mako server.
constexpr absl::string_view kMakoHostname = "mako.dev";
// The Appspot hostname of the Mako AppEngine app.
constexpr absl::string_view kMakoAppengineHostname = "makoperf.appspot.com";

std::string MaybeGetAppEngineHostname(absl::string_view hostname) {
  // Some usages requires the actual, direct base AppEngine app URL. The
  // dashboard will take care of recognizing this appspot URL and changing it
  // back to mako.dev for the user. We need to check for both with scheme and
  // without scheme because we don't know what the user will pass in.
  if (hostname == kMakoHostname ||
      hostname == absl::StrCat("https://", kMakoHostname)) {
    return std::string(kMakoAppengineHostname);
  }
  return std::string(hostname);
}

}  // namespace

std::unique_ptr<BaseStorageClient> NewMakoClient() {
  return NewMakoClient(kMakoHostname);
}

std::unique_ptr<BaseStorageClient> NewMakoClient(
    absl::string_view hostname) {
  std::string overridden_hostname = google3_storage::ApplyHostnameFlagOverrides(
      MaybeGetAppEngineHostname(hostname));
  if (!absl::GetFlag(FLAGS_mako_auth)) {
    LOG(INFO) << "Creating a Mako storage client for host " << hostname
              << " with no authentication";
    return absl::make_unique<BaseStorageClient>(
        absl::make_unique<internal::HttpTransport>(
            overridden_hostname,
            /*token_provider=*/nullptr,
            absl::GetFlag(FLAGS_mako_auth_ca_cert)));
  }

  LOG(INFO) << "Creating a Mako storage client for host " << hostname
            << " authenticating with Application Default Credentials.";
  return absl::make_unique<BaseStorageClient>(
      absl::make_unique<internal::HttpTransport>(
          hostname, absl::make_unique<internal::GoogleOAuthFetcher>(),
          absl::GetFlag(FLAGS_mako_auth_ca_cert)));
}

}  // namespace mako
