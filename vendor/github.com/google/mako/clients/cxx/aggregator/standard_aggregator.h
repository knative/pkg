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
#ifndef CLIENTS_CXX_AGGREGATOR_STANDARD_AGGREGATOR_H_
#define CLIENTS_CXX_AGGREGATOR_STANDARD_AGGREGATOR_H_

#include <functional>
#include <list>
#include <map>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "clients/cxx/aggregator/threadsafe_running_stats.h"
#include "spec/cxx/aggregator.h"
#include "spec/cxx/fileio.h"
#include "absl/synchronization/mutex.h"
#include "absl/time/time.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace aggregator {

// See http://go/mako-aggregator-performance for the experimental data used
// to choose these defaults.
// Buffer size of 10 and 8 threads offer significant performance gains over
// buffer size of 1 and 1 thread, and increasing them further does not lead to
// equivalent increases in performance.
const int kDefaultBufferSize = 10;
const int kDefaultMaxThreads = 8;

// Aggregator is an implementation of the Mako Aggregator interface.
//
// This is implementation parallelizes aggregation across multiple threads.
//
// See interface at https://github.com/google/mako/blob/master/spec/cxx/aggregator.h
class Aggregator : public mako::Aggregator {
 public:
  // Default constructor
  Aggregator() : Aggregator(-1, kDefaultMaxThreads, kDefaultBufferSize) {}

  // Alternate constructor which allows setting per-metric sample size,
  // maximum number of threads, and buffer size
  //   max_sample_size: The max per-metric sample size temporarily
  //     maintained in memory for calculating percentiles and
  //     median. A value of -1 indicates no max. Rarely set this
  //     when you collect more measurements than can fit in
  //     memory of the master process. As an alternative,
  //     increase the memory available.
  //   max_threads: The maximum number of threads to be used for processing
  //     files in parallel. A value that is not positive indicates no max. If
  //     this value is higher than the number of input files, the number of
  //     files will be used instead.
  //     TODO(b/136282446): adapt this to use different defaults depending on
  //     if we're in a small or large test.
  //   buffer_size: The number of values each thread will buffer for each
  //     metric. The buffering is used to reduce contention, since global locks
  //     are necessary in places during aggregation. Without the buffering (or
  //     with buffer_size = 1), we must lock on each point, so a large portion
  //     of the performance gained from parallelizing the FileIO reads is lost.
  //     The amount of extra memory used will be proportional to
  //     max_threads*buffer_size*num_metrics, where num_metrics is the number of
  //     metrics reported on each point (which will vary from test to test).
  //     With the defaults of 8 threads and buffer size of 10, the extra memory
  //     used will be around:
  //       1 metric per point: 640 bytes
  //       10 metrics per point: 6400 bytes
  //       100 metrics per point: 64000 bytes
  //       500 metrics per point: 320000 bytes.
  Aggregator(int max_sample_size, int max_threads, int buffer_size) :
    buffer_size_(buffer_size),
    max_sample_size_(max_sample_size),
    max_threads_(max_threads) {
  }

  // Set the FileIO implementation that is used to read samples.
  void SetFileIO(std::unique_ptr<FileIO> fileio) override {
    fileio_ = std::move(fileio);
  }

  // Compute aggregates
  // Returned std::string contains error message, if empty then operation was
  // successful.
  std::string Aggregate(const mako::AggregatorInput& aggregator_input,
                   mako::AggregatorOutput* aggregator_output) override;

 protected:
  // An internal mechanism to allow simple extensions to this aggregator.
  //
  // The given PerSamplePointCallback will be called for every SamplePoint
  // that the aggregator processes, allowing the called code to collect
  // additional data.
  //
  // The map pointer points to the buffers for the currently running thread.
  // If you wish for additional metrics to be collected, pass it along with the
  // key and value to AppendToBuffer in your callback. This should help limit
  // contention resulting from the necessary synchronization.
  //
  // A getter for the FileIO pointer is provided so the aggregator extensions
  // can make additional passes over the data files if necessary.
  //
  // WARNING: Please do not use without contacting the mako team. We have
  // future plans to make custom aggregator implementations easier.
  // TODO(b/29609023): Move the important pieces of the standard aggregator to
  // mako/helpers/advanced.
  typedef std::function<std::string(
      const mako::SamplePoint&, std::map<std::string, std::vector<double> >*,
      std::map<std::string, std::unique_ptr<ThreadsafeRunningStats> >*)>
      PerSamplePointCallback;
  void SetPerSamplePointCallback(PerSamplePointCallback cb) {
    per_sample_point_cb_ = std::move(cb);
  }
  mako::FileIO* GetFileIO() {
    return fileio_.get();
  }
  std::string AppendToBuffer(
      const std::string& value_key, const double value,
      std::map<std::string, std::vector<double> >* buffers,
      std::map<std::string, std::unique_ptr<ThreadsafeRunningStats> >* stats_map);

 private:
  struct SampleCounts {
    int usable = 0;
    int ignored = 0;
    int error = 0;
  };

  bool Ignored(const std::list<mako::Range>& sorted_ignore_list,
               const mako::SamplePoint& sample_point);

  std::string Init(const mako::AggregatorInput& aggregator_input,
              std::list<mako::Range>* sorted_ignore_list);
  std::string Complete(
      const mako::AggregatorInput& aggregator_input,
      const SampleCounts& sample_counts,
      const std::map<std::string, std::unique_ptr<ThreadsafeRunningStats> >&
          stats_map,
      mako::AggregatorOutput* output);

  std::string ProcessFiles(
      const mako::AggregatorInput& aggregator_input,
      const std::list<mako::Range>& sorted_ignore_list,
      SampleCounts* sample_counts,
      std::map<std::string, std::unique_ptr<ThreadsafeRunningStats> >* stats_map);
  std::string ProcessFile(
      const std::list<mako::Range>& sorted_ignore_list,
      const std::string& file_path, mako::FileIO* fio,
      SampleCounts* sample_counts,
      std::map<std::string, std::unique_ptr<ThreadsafeRunningStats> >* stats_map,
      absl::Duration* fileio_read_time);
  std::string ProcessRecord(
      const std::list<mako::Range>& sorted_ignore_list,
      const mako::SampleRecord& sample_record,
      std::map<std::string, std::vector<double> >* buffers,
      SampleCounts* sample_counts,
      std::map<std::string, std::unique_ptr<ThreadsafeRunningStats> >* stats_map);
  std::string ProcessBuffer(
      const std::string& value_key, const std::vector<double>& buffer,
      std::map<std::string, std::unique_ptr<ThreadsafeRunningStats> >* stats_map);

  // Used to synchronize access to the underlying map.
  ThreadsafeRunningStats* GetOrCreateRunningStats(
      const std::string& value_key,
      std::map<std::string, std::unique_ptr<ThreadsafeRunningStats> >* stats_map);

  std::unique_ptr<mako::FileIO> fileio_;

  const int buffer_size_;
  const int max_sample_size_;
  const int max_threads_;
  PerSamplePointCallback per_sample_point_cb_;
  // Synchronizes calls to GetOrCreateRunningStats, so exactly one RunningStats
  // is created for each metric key.
  absl::Mutex mutex_;
};

}  // namespace aggregator
}  // namespace mako

#endif  // CLIENTS_CXX_AGGREGATOR_STANDARD_AGGREGATOR_H_
