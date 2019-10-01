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

#ifndef CLIENTS_CXX_STORAGE_MAKO_CLIENT_H_
#define CLIENTS_CXX_STORAGE_MAKO_CLIENT_H_

#include <memory>

#include "clients/cxx/storage/base_storage_client.h"
#include "absl/strings/string_view.h"

namespace mako {

// Create a new Mako storage client connected to https://mako.dev.
//
// See https://github.com/google/mako/blob/master/spec/cxx/storage.h for more
// information about interface.
//
// See https://github.com/google/mako/blob/master/spec/proto/mako.proto for
// information about the protobuf structures used below.
//
// Calling this storage client's methods isn't needed when using the normal flow
// of Mako. To create a Mako run from performance metrics use Quickstore
// instead.
//
// Call this class's methods only when you know what you are doing and want to
// directly access Mako storage programmatically.
//
// The Application Default Credentials protocol is used for authentication.
//
// Application Default Credentials:
// (https://developers.google.com/identity/protocols/application-default-credentials)
// for authentication and communicate with the Mako server over HTTP . If
// necessary, use --mako_auth_ca_cert=<path> to specify a path to a CA certs
// bundle file to use for SSL.
//
// The returned BaseStorageClient is thread-safe.
std::unique_ptr<BaseStorageClient> NewMakoClient();

// Intended for use when you need a Mako Client connected to an arbitrary
// server, typically used for testing.
std::unique_ptr<BaseStorageClient> NewMakoClient(absl::string_view hostname);

}  // namespace mako

#endif  // CLIENTS_CXX_STORAGE_MAKO_CLIENT_H_
