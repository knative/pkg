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
#include "internal/quickstore_microservice/quickstore_service.h"

#include <memory>

#include "src/google/protobuf/repeated_field.h"
#include "clients/cxx/storage/mako_client.h"
#include "helpers/cxx/status/statusor.h"
#include "internal/cxx/queue_ifc.h"
#include "quickstore/cxx/internal/store.h"

namespace mako {
namespace internal {
namespace quickstore_microservice {

namespace {
template <typename T, template<typename> class Container>
std::vector<T> MakeVector(Container<T> proto) {
  return {proto.begin(), proto.end()};
}
}  // namespace

mako::helpers::StatusOr<std::unique_ptr<QuickstoreService>>
QuickstoreService::Create(
    mako::internal::QueueInterface<bool>* shutdown_queue) {
  return {absl::make_unique<QuickstoreService>(
      shutdown_queue, mako::NewMakoClient())};
}

grpc::Status QuickstoreService::Store(grpc::ServerContext* context,
                                      const StoreInput* request,
                                      StoreOutput* response) {
  *response->mutable_quickstore_output() =
      mako::quickstore::internal::SaveWithStorage(
          storage_.get(), request->quickstore_input(),
          MakeVector(request->sample_points()),
          MakeVector(request->sample_errors()),
          MakeVector(request->run_aggregates()),
          MakeVector(request->aggregate_value_keys()),
          MakeVector(request->aggregate_value_types()),
          MakeVector(request->aggregate_value_values()));
  return grpc::Status::OK;
}

grpc::Status QuickstoreService::ShutdownMicroservice(
    grpc::ServerContext* context, const ShutdownInput* request,
    ShutdownOutput* response) {
  shutdown_queue_->put(true);
  return grpc::Status::OK;
}

}  // namespace quickstore_microservice
}  // namespace internal
}  // namespace mako
