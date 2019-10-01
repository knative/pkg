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
#include "clients/cxx/aggregator/standard_aggregator.h"

#include <array>

#include "glog/logging.h"
#include "src/google/protobuf/repeated_field.h"
#include "absl/memory/memory.h"
#include "absl/strings/str_cat.h"
#include "absl/synchronization/mutex.h"
#include "absl/time/clock.h"
#include "absl/time/time.h"
#include "internal/cxx/load/common/executor.h"
#include "internal/cxx/pgmath.h"
#include "internal/cxx/proto_validation.h"

namespace mako {
namespace aggregator {

namespace {
constexpr char kNoError[] = "";

static constexpr std::array<int, 8> kDefaultPercentileMilliRanks = {
    {1000, 2000, 5000, 10000, 90000, 95000, 98000, 99000}};
}  // namespace

ThreadsafeRunningStats* Aggregator::GetOrCreateRunningStats(
    const std::string& value_key,
    std::map<std::string, std::unique_ptr<ThreadsafeRunningStats>>* stats_map) {
  absl::MutexLock l(&mutex_);
  if (stats_map->count(value_key) == 0) {
    (*stats_map)[value_key] =
        absl::make_unique<ThreadsafeRunningStats>(max_sample_size_);
  }
  return stats_map->at(value_key).get();
}

// Return bool if the passed SamplePoint's input_value falls within any of the
// ignore ranges.
bool Aggregator::Ignored(const std::list<mako::Range>& sorted_ignore_list,
                         const mako::SamplePoint& sample_point) {
  for (const auto& range : sorted_ignore_list) {
    // Because ignore list is sorted by start time we can abort search.
    if (range.start() > sample_point.input_value()) {
      return false;
    } else if (range.start() <= sample_point.input_value() &&
               sample_point.input_value() <= range.end()) {
      return true;
    }
  }
  return false;
}

// Calculate aggregates based on AggregatorInput(). Place results in
// AggergatorOutput or return error string.
std::string Aggregator::Aggregate(const mako::AggregatorInput& aggregator_input,
                             mako::AggregatorOutput* aggregator_output) {

  if (!fileio_) {
    return "Must pass a FileIO instance with Aggregator.SetFileIO() first.";
  }
  std::list<mako::Range> sorted_ignore_list;
  std::string err = Init(aggregator_input, &sorted_ignore_list);
  if (!err.empty()) {
    LOG(ERROR) << err;
    return err;
  }

  SampleCounts sample_counts;
  std::map<std::string, std::unique_ptr<ThreadsafeRunningStats>> stats_map;
  err = ProcessFiles(aggregator_input, sorted_ignore_list, &sample_counts,
                     &stats_map);
  if (!err.empty()) {
    LOG(ERROR) << err;
    return err;
  }

  return Complete(aggregator_input, sample_counts, stats_map,
                  aggregator_output);
}

std::string Aggregator::ProcessFiles(
    const mako::AggregatorInput& aggregator_input,
    const std::list<mako::Range>& sorted_ignore_list,
    SampleCounts* sample_counts,
    std::map<std::string, std::unique_ptr<ThreadsafeRunningStats>>* stats_map) {
  std::string err;

  int num_threads = aggregator_input.sample_file_list_size();
  if (max_threads_ > 0 && max_threads_ < num_threads) {
    num_threads = max_threads_;
  }
  LOG(INFO) << "Creating thread pool with " << num_threads << " threads.";
  mako::internal::Executor file_processor(num_threads);
  absl::Mutex m;
  absl::Duration total_fileio_read_time;
  for (const mako::SampleFile& sample_file :
       aggregator_input.sample_file_list()) {
    file_processor.Schedule([&err, &sample_file, &m, &sorted_ignore_list,
                             sample_counts, stats_map, &total_fileio_read_time,
                             this]() {
      SampleCounts file_sample_counts;
      std::unique_ptr<mako::FileIO> fio = fileio_->MakeInstance();
      absl::Duration fileio_read_time;
      std::string error =
          ProcessFile(sorted_ignore_list, sample_file.file_path(), fio.get(),
                      &file_sample_counts, stats_map, &fileio_read_time);
      bool successful_close = fio->Close();
      VLOG(1) << "Spent " << fileio_read_time << " reading points from "
                << sample_file.file_path();
      absl::MutexLock l(&m);
      total_fileio_read_time += fileio_read_time;
      sample_counts->ignored += file_sample_counts.ignored;
      sample_counts->usable += file_sample_counts.usable;
      sample_counts->error += file_sample_counts.error;
      if (!successful_close && !fio->Error().empty()) {
        absl::StrAppend(&err, "\n", fio->Error());
      }
      if (!error.empty()) {
        absl::StrAppend(&err, "\n", error);
      }
    });
  }
  file_processor.Wait();
  LOG(INFO) << "Spent " << total_fileio_read_time << " across " << num_threads
            << " threads reading points. ";
  return err;
}

std::string Aggregator::ProcessFile(
    const std::list<mako::Range>& sorted_ignore_list,
    const std::string& file_path, mako::FileIO* fio, SampleCounts* sample_counts,
    std::map<std::string, std::unique_ptr<ThreadsafeRunningStats>>* stats_map,
    absl::Duration* fileio_read_time) {
  std::map<std::string, std::vector<double>> buffers;
  std::string err;
  VLOG(1) << "Processing file: " << file_path;
  if (!fio->Open(file_path, mako::FileIO::AccessMode::kRead)) {
    return absl::StrCat("Could not open file at path: ", file_path,
                        " Error message: ", fio->Error());
  }
  mako::SampleRecord sample_record;
  while (true) {
    sample_record.Clear();
    absl::Time start = absl::Now();
    if (!fio->Read(&sample_record)) {
      if (!fio->ReadEOF()) {
        return absl::StrCat("Error attempting to read from file: ", file_path,
                            ". Error message: ", fio->Error());
      }
      break;
    }
    *fileio_read_time += absl::Now() - start;
    err = ProcessRecord(sorted_ignore_list, sample_record, &buffers,
                        sample_counts, stats_map);
    if (!err.empty()) {
      return err;
    }
  }
  VLOG(1) << "Done processing file: " << file_path;

  for (const auto& pair : buffers) {
    std::string error = ProcessBuffer(pair.first, pair.second, stats_map);
    if (!error.empty()) {
      absl::StrAppend(&err, "\n", error);
    }
  }
  return err;
}

std::string Aggregator::ProcessRecord(
    const std::list<mako::Range>& sorted_ignore_list,
    const mako::SampleRecord& sample_record,
    std::map<std::string, std::vector<double>>* buffers, SampleCounts* sample_counts,
    std::map<std::string, std::unique_ptr<ThreadsafeRunningStats>>* stats_map) {
  std::string err;
  if (sample_record.has_sample_point()) {
    if (Ignored(sorted_ignore_list, sample_record.sample_point())) {
      ++sample_counts->ignored;
    } else {
      ++sample_counts->usable;
      const auto& sample_point = sample_record.sample_point();
      for (const auto& k : sample_point.metric_value_list()) {
        err = AppendToBuffer(k.value_key(), k.value(), buffers, stats_map);
        if (!err.empty()) {
          return err;
        }
      }
      if (per_sample_point_cb_) {
        err = per_sample_point_cb_(sample_point, buffers, stats_map);
        if (!err.empty()) {
          return err;
        }
      }
    }
  }
  if (sample_record.has_sample_error()) {
    ++sample_counts->error;
  }
  return kNoError;
}

std::string Aggregator::AppendToBuffer(
    const std::string& value_key, const double value,
    std::map<std::string, std::vector<double>>* buffers,
    std::map<std::string, std::unique_ptr<ThreadsafeRunningStats>>* stats_map) {
  std::vector<double>& buffer = (*buffers)[value_key];
  buffer.push_back(value);
  if (buffer.size() > static_cast<std::size_t>(buffer_size_)) {
    std::string err = ProcessBuffer(value_key, buffer, stats_map);
    buffer.clear();
    if (!err.empty()) {
      return err;
    }
  }
  return kNoError;
}

std::string Aggregator::ProcessBuffer(
    const std::string& value_key, const std::vector<double>& buffer,
    std::map<std::string, std::unique_ptr<ThreadsafeRunningStats>>* stats_map) {
  return GetOrCreateRunningStats(value_key, stats_map)->AddVector(buffer);
}

std::string Aggregator::Init(const mako::AggregatorInput& aggregator_input,
                        std::list<mako::Range>* sorted_ignore_list) {
  std::string err = mako::internal::ValidateAggregatorInput(aggregator_input);
  if (!err.empty()) {
    LOG(ERROR) << err;
    return err;
  }

  // Sort ignore ranges.
  for (const mako::LabeledRange& lr :
       aggregator_input.run_info().ignore_range_list()) {
    sorted_ignore_list->push_back(lr.range());
  }
  sorted_ignore_list->sort(
      [](const mako::Range& a, const mako::Range& b) {
        return a.start() <= b.start();
      });
  return kNoError;
}

std::string Aggregator::Complete(
    const mako::AggregatorInput& aggregator_input,
    const SampleCounts& sample_counts,
    const std::map<std::string, std::unique_ptr<ThreadsafeRunningStats>>& stats_map,
    mako::AggregatorOutput* output) {
  // Set the percentil_milli_rank on aggregator_output. If not set on benchmark
  // then use defaults.
  if (aggregator_input.benchmark_info().percentile_milli_rank_list_size() > 0) {
    *output->mutable_aggregate()->mutable_percentile_milli_rank_list() =
        aggregator_input.benchmark_info().percentile_milli_rank_list();
  } else {
    *output->mutable_aggregate()->mutable_percentile_milli_rank_list() = {
        kDefaultPercentileMilliRanks.begin(),
        kDefaultPercentileMilliRanks.end()};
  }

  auto ragg = output->mutable_aggregate()->mutable_run_aggregate();
  ragg->set_error_sample_count(sample_counts.error);
  ragg->set_ignore_sample_count(sample_counts.ignored);
  ragg->set_usable_sample_count(sample_counts.usable);

  // Foreach metric key, create a metric aggregate
  mako::internal::RunningStats::Result result;
  absl::MutexLock l(&mutex_);
  for (auto& kv : stats_map) {
    auto magg = output->mutable_aggregate()->add_metric_aggregate_list();
    magg->set_metric_key(kv.first);
    result = kv.second->Mean();
    if (!result.error.empty()) return result.error;
    magg->set_mean(result.value);
    result = kv.second->Stddev();
    if (!result.error.empty()) return result.error;
    magg->set_standard_deviation(result.value);
    result = kv.second->Mad();
    if (!result.error.empty()) return result.error;
    magg->set_median_absolute_deviation(result.value);
    result = kv.second->Min();
    if (!result.error.empty()) return result.error;
    magg->set_min(result.value);
    result = kv.second->Max();
    if (!result.error.empty()) return result.error;
    magg->set_max(result.value);
    result = kv.second->Median();
    if (!result.error.empty()) return result.error;
    magg->set_median(result.value);
    result = kv.second->Count();
    if (!result.error.empty()) return result.error;
    magg->set_count(result.value);
    for (double pmr : output->aggregate().percentile_milli_rank_list()) {
      result = kv.second->Percentile(pmr / 100000.0);
      if (!result.error.empty()) return result.error;
      magg->add_percentile_list(result.value);
    }
  }
  return kNoError;
}
}  // namespace aggregator
}  // namespace mako
