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
#ifndef CLIENTS_CXX_DOWNSAMPLER_STANDARD_DOWNSAMPLER_H_
#define CLIENTS_CXX_DOWNSAMPLER_STANDARD_DOWNSAMPLER_H_

// Downsampler which considers metrics.
//
// This downsampler implementation groups SamplePoints by the metric value set
// they contain. The Mako storage system provides a max number of metrics
// values that are allowed. We give each set an equal share of that max. If a
// set doesn't use its equal share it is distributed to other sets.
//
// For example:
//  MaxMetricsValues = 6
//  1.) SamplePoint('metric1': 10, 'metric2': 1)
//  2.) SamplePoint('metric1': 11, 'metric2': 2)
//  3.) SamplePoint('metric1': 12, 'metric2': 3)
//  4.) SamplePoint('metric3': 10, 'metric4': 4)
//  5.) SamplePoint('metric1': 10)
//  6.) SamplePoint('metric1': 11)
//
//  3 unique metric sets are reported:
//  {'metric1'}
//  {'metric1','metric2'}
//  {'metric3','metric4'}
//
// 10 total metrics values are reported, the max is 6 so some downsampling will
// be required:
//  metric1: 5 (exists in points 1,2,3,5,6 above)
//  metric2: 3 (exists in points 1,2,3 above)
//  metric3: 1 (exists in point 4 above)
//  metric4: 1 (exists in point 4 above)
//
// In this case each unique set is guaranteed 2 metric values to be saved (6/3).
// {'metric1'}:
//   Select point 5 & 6.
// {'metric1','metric2'}
//   Randomly select one of point 1,2 or 3.
// {'metric3','metric4'}
//   Select point 4.
//
// NOTE: In cases when an entire point cannot be added and stay under the max
// number of metric values, we will end up with less than the max number (eg. if
// max metric values is 9 and we only have sample points with two metrics).
//
// A random sampling of SampleErrors are collected, which each sampler getting a
// fair share of the total errors allowed.
//
// In the code below the concept of a slot is introduced. When considering
// SampleErrors, the storage system tells us how many SampleErrors we can save.
// Thus a slot is the same as a single SampleError. When considering
// SamplePoints, the storage systems tells us how many metric values we can
// save. So the slot count for a SamplePoint is how many metric measurements are
// in the sample. Using the example above:
//  * SamplePoints 1-4 take up 2 slots each.
//  * SamplePoints 5,6 take up 1 slot each.
//
// See go/mako-downsampler for design doc for more information.

#include <memory>
#include <random>
#include <string>
#include <utility>

#include "glog/logging.h"
#include "src/google/protobuf/io/coded_stream.h"
#include "clients/cxx/downsampler/metric_set.h"
#include "internal/proto/mako_internal.pb.h"
#include "spec/cxx/downsampler.h"
#include "spec/cxx/fileio.h"
#include "absl/base/attributes.h"
#include "absl/strings/str_cat.h"
#include "internal/cxx/proto_validation.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace downsampler {

// SWIG doesn't support constexpr.
// Using same default as Standard Aggregator
ABSL_CONST_INIT extern const int kMaxThreads;
// TODO(b/136285106) Wonder if this should be part of storage?
ABSL_CONST_INIT extern const int kMaxErrorStringLength;
// Max binary size of all SampleAnnotations in all batches. This should be less
// than maximum sample batch size. The size is in base 10 bytes, e.g.
// 1MB == 1,000,000).
//
// Maximum batch size is defined here:
// https://github.com/google/mako/blob/master/clients/cxx/storage/google3_storage.cc
ABSL_CONST_INIT extern const int kMaxAnnotationsSize;

// Downsampler is an implementation of the Mako Downsampler interface.
//
// This is the standard implementation which parallelizes the file reads across
// multiple threads.
//
// See interface docs for more information.
class Downsampler : public mako::Downsampler {
 public:
  // Default constructor which uses the default maximum number of threads.
  Downsampler() : Downsampler(kMaxThreads) {}

  // Constructor which allows setting the maximum number of threads to be used
  // for processing files in parallel. A value that is not positive indicates no
  // max. If this value is higher than the number of input files, the number of
  // files will be used instead.
  // TODO(b/136282446): Adapt this to use different defaults depending on if
  // we're in a small or large test.
  explicit Downsampler(int max_threads) : max_threads_(max_threads) {
    std::random_device r;
    Reseed(r());
  }

  // Set the FileIO implementation that is used to read samples.
  void SetFileIO(std::unique_ptr<FileIO> fileio) override {
    fileio_ = std::move(fileio);
  }

  // Perform downsampling
  // Returned std::string contains error message, if empty then operation was
  // successful.
  std::string Downsample(const mako::DownsamplerInput& downsampler_input,
                    mako::DownsamplerOutput* downsampler_output) override;

 private:
  // Set the PRNG seed. This is primarily to make the PRNG deterministic for
  // testing purposes
  void Reseed(int prng_seed);

  std::unique_ptr<FileIO> fileio_;
  std::default_random_engine prng_;

  const int max_threads_;

  friend class StandardMetricDownsamplerTest;

  // SWIG doesn't support = delete syntax.
#ifndef SWIG
  // Not copyable.
  Downsampler(const Downsampler&) = delete;
  Downsampler& operator=(const Downsampler&) = delete;
#endif  // SWIG
};

void GetNewRecord(mako::SampleBatch* batch,
                  mako::SamplePoint** new_point);

void GetNewRecord(mako::SampleBatch* batch,
                  mako::SampleError** new_error);

mako::SampleBatch* GetNewBatch(
    const std::string& benchmark_key, const std::string& run_key,
    mako::DownsamplerOutput* downsampler_output, int64_t* batch_size_bytes);

template <typename T>
static std::string AddBatch(const std::string& benchmark_key, const std::string& run_key,
                       const int batch_size_max, const int field_number,
                       T* record, mako::SampleBatch** batch,
                       int64_t* batch_size_bytes,
                       mako::DownsamplerOutput* downsampler_output) {
  mako::internal::StripAuxData(record);
  int64_t record_size_bytes = record->ByteSizeLong();
  // See https://developers.google.com/protocol-buffers/docs/encoding#embedded
  // for documentation on how embedded messages are encoded.
  int64_t record_serialized_size_in_sample_batch =
      record_size_bytes +
      google::protobuf::io::CodedOutputStream::VarintSize64(record_size_bytes) +
      google::protobuf::io::CodedOutputStream::VarintSize64((field_number << 3) | 2);
  if (record_serialized_size_in_sample_batch > batch_size_max) {
    std::string err =
        absl::StrCat("Got single record (", MetricSet(record).ToString(),
                     ", size: ", record_serialized_size_in_sample_batch,
                     ") that is too large to fit in a SampleBatch.",
                     "batch_size_max: ", batch_size_max);
    LOG(ERROR) << err;
    return err;
  }

  if (*batch_size_bytes + record_serialized_size_in_sample_batch >
      batch_size_max) {
    *batch = GetNewBatch(benchmark_key, run_key, downsampler_output,
                         batch_size_bytes);
  }
  T* new_record;
  GetNewRecord(*batch, &new_record);
  *new_record = *record;
  *batch_size_bytes += record_serialized_size_in_sample_batch;
  return "";
}

}  // namespace downsampler
}  // namespace mako

#endif  // CLIENTS_CXX_DOWNSAMPLER_STANDARD_DOWNSAMPLER_H_
