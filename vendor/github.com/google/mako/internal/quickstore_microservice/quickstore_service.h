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
//
// QuickstoreService exposes an RPC interface for running Quickstore
#ifndef INTERNAL_QUICKSTORE_MICROSERVICE_QUICKSTORE_SERVICE_H_
#define INTERNAL_QUICKSTORE_MICROSERVICE_QUICKSTORE_SERVICE_H_

#include <memory>

#include "internal/proto/mako_internal.pb.h"
#include "internal/quickstore_microservice/proto/quickstore.grpc.pb.h"
#include "internal/quickstore_microservice/proto/quickstore.pb.h"
#include "spec/cxx/storage.h"
#include "helpers/cxx/status/statusor.h"
#include "internal/cxx/queue_ifc.h"

namespace mako {
namespace internal {
namespace quickstore_microservice {

class QuickstoreService : public Quickstore::Service {
 public:
  // Exposed for testing
  explicit QuickstoreService(
      mako::internal::QueueInterface<bool>* shutdown_queue,
      std::unique_ptr<mako::Storage> storage)
      : shutdown_queue_(shutdown_queue), storage_(std::move(storage)) {}
  ~QuickstoreService() override {}

  static mako::helpers::StatusOr<std::unique_ptr<QuickstoreService>> Create(
      mako::internal::QueueInterface<bool>* shutdown_queue);

  grpc::Status Store(grpc::ServerContext* context, const StoreInput* request,
                     StoreOutput* response) override;

  grpc::Status ShutdownMicroservice(grpc::ServerContext* context,
                                    const ShutdownInput* request,
                                    ShutdownOutput* response) override;

 private:
  mako::internal::QueueInterface<bool>* shutdown_queue_;
  std::unique_ptr<mako::Storage> storage_;
};

}  // namespace quickstore_microservice
}  // namespace internal
}  // namespace mako
#endif  // INTERNAL_QUICKSTORE_MICROSERVICE_QUICKSTORE_SERVICE_H_
