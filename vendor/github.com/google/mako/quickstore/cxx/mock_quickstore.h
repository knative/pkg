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
#ifndef QUICKSTORE_CXX_MOCK_QUICKSTORE_H_
#define QUICKSTORE_CXX_MOCK_QUICKSTORE_H_

#include <map>
#include <string>

#include "gmock/gmock.h"
#include "proto/quickstore/quickstore.pb.h"
#include "quickstore/cxx/quickstore.h"

namespace mako {
namespace quickstore {

class MockQuickstore : public Quickstore {
 public:
  explicit MockQuickstore(const std::string& benchmark_key)
      : Quickstore(benchmark_key) {}
  explicit MockQuickstore(
      const mako::quickstore::QuickstoreInput& input)
      : Quickstore(input) {}

  MOCK_METHOD1(AddSamplePoint, std::string(const mako::SamplePoint& point));
  MOCK_METHOD2(AddSamplePoint,
               std::string(double xval, const std::map<std::string, double>& yvals));

  MOCK_METHOD2(AddError, std::string(double xval, const std::string& error_msg));
  MOCK_METHOD1(AddError, std::string(const mako::SampleError&));

  MOCK_METHOD2(AddRunAggregate, std::string(const std::string& value_key, double value));
  MOCK_METHOD3(AddMetricAggregate,
               std::string(const std::string& value_key, const std::string& aggregate_type,
                      double value));
  MOCK_METHOD0(Store, mako::quickstore::QuickstoreOutput());
};

}  // namespace quickstore
}  // namespace mako

#endif  // QUICKSTORE_CXX_MOCK_QUICKSTORE_H_
