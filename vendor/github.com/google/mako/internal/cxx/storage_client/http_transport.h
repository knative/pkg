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

// HTTP/Oauth2 version of Mako (go/mako) storage communication.
//
#ifndef INTERNAL_CXX_STORAGE_CLIENT_HTTP_TRANSPORT_H_
#define INTERNAL_CXX_STORAGE_CLIENT_HTTP_TRANSPORT_H_

#include <memory>
#include <string>
#include <utility>

#include "glog/logging.h"
#include "src/google/protobuf/message.h"
#include "absl/strings/string_view.h"
#include "absl/time/time.h"
#include "helpers/cxx/status/status.h"
#include "internal/cxx/storage_client/http_client.h"
#include "internal/cxx/storage_client/oauth_token_provider.h"
#include "internal/cxx/storage_client/transport.h"

namespace mako {
namespace internal {

// This transport relies on using HTTP to communicate with the server, instead
// of Google-specific transports.
//
// If an OAuthTokenProvider instance is injected, OAuth2 authentication will be
// passed by setting the token returned by the OAuthTokenProvider in an
// "Authorization: Bearer <token>" request header.
//
// This class is thread-safe.
class HttpTransport : public StorageTransport {
 public:
  // Constructs an HttpTransport that uses the default OAuthTokenProvider
  // (MetadataOAuthFetcher).
  explicit HttpTransport(absl::string_view host);

  // Constructs an HttpTransport that will use the provided OAuthTokenProvider.
  // The OAuthTokenProvider must be thread-safe (go/thread-safe).
  HttpTransport(absl::string_view host,
                std::unique_ptr<OAuthTokenProvider> token_provider)
      : HttpTransport(host, std::move(token_provider), "") {}

  // Constructs an HttpTransport that will use the provided OAuthTokenProvider,
  // and overrides the HTTP Client's CA cert path.
  //
  // The OAuthTokenProvider must be thread-safe (go/thread-safe).
  HttpTransport(absl::string_view host,
                std::unique_ptr<OAuthTokenProvider> token_provider,
                absl::string_view ca_certificate_path)
      : host_(host),
        token_provider_(std::move(token_provider)),
        client_(ca_certificate_path) {}

  helpers::Status Connect() override;

  void set_client_tool_tag(absl::string_view) override;

  void use_local_gae_server(bool use_local_gae_server);

  // Sends a POST request to `path` on the server, serializing the `request`
  // into the HTTP body. The server's response body will be deserialized into
  // `response`.
  helpers::Status Call(absl::string_view path, const google::protobuf::Message& request,
                       absl::Duration timeout,
                       google::protobuf::Message* response) override;

  // TODO(b/73734783): Remove this.
  absl::Duration last_call_server_elapsed_time() const override {
    LOG(FATAL) << "Not implemented";
    return absl::ZeroDuration();
  }

  // Fetches the token provider.
  OAuthTokenProvider* token_provider() { return token_provider_.get(); }

  // The hostname backing this Storage implementation.
  // Returns a URL without the trailing slash.
  std::string GetHostname() override { return host_; }

 private:
  const std::string host_;
  std::string client_tool_tag_ = "unknown";
  const std::unique_ptr<OAuthTokenProvider> token_provider_;
  HttpClient client_;
  bool use_local_gae_server_ = false;
  std::string ca_authority_path_;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_STORAGE_CLIENT_HTTP_TRANSPORT_H_
