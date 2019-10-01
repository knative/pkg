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
#ifndef CLIENTS_CXX_STORAGE_BASE_STORAGE_CLIENT_H_
#define CLIENTS_CXX_STORAGE_BASE_STORAGE_CLIENT_H_

#include "clients/cxx/storage/google3_storage.h"

namespace mako {

// TODO(b/123650797): Rename google3_storage::Storage to BaseStorageClient, and
// provide a different interface (analogous to mako_client.h) for prod/internal
// clients.
using BaseStorageClient = google3_storage::Storage;

}  // namespace mako

#endif  // CLIENTS_CXX_STORAGE_BASE_STORAGE_CLIENT_H_
