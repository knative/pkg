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
//
// TODO(b/126415270) We'll hold onto 3x - (downsampler removed points) memory
// during this process. If this becomes a problem we could take a non-const
// vector to the data and delete as we write to disk.
#include "quickstore/cxx/internal/store.h"

#include <functional>
#include <list>
#include <map>
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/map.h"
#include "clients/cxx/aggregator/standard_aggregator.h"
#include "clients/cxx/analyzers/threshold_analyzer.h"
#include "clients/cxx/analyzers/utest_analyzer.h"
#include "clients/cxx/analyzers/window_deviation.h"
#include "clients/cxx/downsampler/standard_downsampler.h"
#include "clients/cxx/fileio/memory_fileio.h"
#include "clients/cxx/storage/mako_client.h" // NOLINT
#include "clients/proto/analyzers/threshold_analyzer.pb.h"
#include "clients/proto/analyzers/utest_analyzer.pb.h"
#include "clients/proto/analyzers/window_deviation.pb.h"
#include "spec/cxx/analyzer.h"
#include "absl/base/const_init.h"
#include "absl/memory/memory.h"
#include "absl/random/random.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/string_view.h"
#include "absl/synchronization/mutex.h"
#include "absl/time/clock.h"
#include "absl/time/time.h"
#include "internal/cxx/load/common/run_analyzers.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace quickstore {
namespace internal {
namespace {

using ::mako::quickstore::QuickstoreInput;
using ::mako::quickstore::QuickstoreOutput;

constexpr char kNoError[] = "";

QuickstoreOutput Fail(const std::string& msg) {
  QuickstoreOutput output;
  output.set_summary_output(msg);
  output.set_status(QuickstoreOutput::ERROR);
  return output;
}

std::string JoinPath(absl::string_view base, absl::string_view path) {
  return absl::StrCat(base, base.back() == '/' ? "" : "/", path);
}
}  // namespace

QuickstoreOutput InternalQuickstore::Save() {
  std::string err;

  // * A global count in the path protects from multiple threads
  //   calling Quickstore concurrently.
  // * A random integer is used to make collisions between different processes
  //   extremely unlikely.
  static absl::Mutex count_mutex(absl::kConstInit);
  static absl::BitGen* gen = new absl::BitGen;
  static int count = 0;
  const std::string& par_dir = input_.has_temp_dir() ? input_.temp_dir() : "/tmp";
  {
    absl::MutexLock lock(&count_mutex);
    tmp_dir_ = JoinPath(par_dir, absl::StrCat("quickstore.", count++, ".",
                                              absl::Uniform<uint64_t>(*gen)));
  }

  if (!input_.has_benchmark_key() || input_.benchmark_key().empty()) {
    err = "Must provide non-empty benchmark_key";
    LOG(ERROR) << err;
    return Fail(err);
  }

  std::list<std::function<std::string()>> steps;
  steps.push_back([this] { return QueryBenchmarkInfo(); });
  steps.push_back([this] { return CreateAndUpdateRunInfo(); });
  steps.push_back([this] { return WriteSampleFile(); });
  steps.push_back([this] { return Aggregate(); });
  steps.push_back([this] { return UpdateMetricAggregates(); });
  steps.push_back([this] { return UpdateRunAggregates(); });
  steps.push_back([this] { return Downsample(); });
  steps.push_back([this] { return Analyze(); });
  steps.push_back([this] { return UpdateRunInfoTags(); });
  steps.push_back([this] { return WriteToStorage(); });

  // Run all steps
  for (const std::function<std::string()>& func : steps) {
    std::string err = func();
    if (!err.empty()) {
      return Fail(err);
    }
  }

  return Complete();
}

std::string InternalQuickstore::UpdateRunAggregates() {
  for (const auto& kv : run_aggregates_) {
    if ("~ignore_sample_count" == kv.value_key()) {
      run_info_.mutable_aggregate()
          ->mutable_run_aggregate()
          ->set_ignore_sample_count(kv.value());
    } else if ("~usable_sample_count" == kv.value_key()) {
      run_info_.mutable_aggregate()
          ->mutable_run_aggregate()
          ->set_usable_sample_count(kv.value());
    } else if ("~error_sample_count" == kv.value_key()) {
      run_info_.mutable_aggregate()
          ->mutable_run_aggregate()
          ->set_error_sample_count(kv.value());
    } else if ("~benchmark_score" == kv.value_key()) {
      run_info_.mutable_aggregate()
          ->mutable_run_aggregate()
          ->set_benchmark_score(kv.value());
    } else {
      // Don't need to merge here b/c no way users could have set any custom
      // aggregates.
      *run_info_.mutable_aggregate()
           ->mutable_run_aggregate()
           ->add_custom_aggregate_list() = kv;
    }
  }
  return kNoError;
}

std::string InternalQuickstore::QueryBenchmarkInfo() {
  std::string err;
  mako::BenchmarkInfoQuery q;
  mako::BenchmarkInfoQueryResponse r;
  q.set_benchmark_key(input_.benchmark_key());
  if (!storage_->QueryBenchmarkInfo(q, &r)) {
    err = absl::StrCat("Error in BenchmarkInfoQuery: ",
                       r.status().fail_message());
    LOG(ERROR) << err;
    return err;
  }
  if (r.benchmark_info_list_size() != 1) {
    err = absl::StrCat("Got ", r.benchmark_info_list_size(),
                       " BenchmarkInfo results from query: ", q.DebugString(),
                       "; want 1. Check your benchmark_key.");
    LOG(ERROR) << err;
    return err;
  }
  benchmark_info_ = r.benchmark_info_list(0);
  run_info_.set_benchmark_key(benchmark_info_.benchmark_key());
  return kNoError;
}

std::string InternalQuickstore::CreateAndUpdateRunInfo() {
  std::string err;
  mako::CreationResponse r;
  // Required for creation.
  if (input_.has_timestamp_ms()) {
    run_info_.set_timestamp_ms(input_.timestamp_ms());
  } else {
    run_info_.set_timestamp_ms(
        static_cast<double>(::absl::ToUnixMillis(::absl::Now())));
  }
  if (input_.has_build_id()) {
    run_info_.set_build_id(input_.build_id());
  }
  if (!storage_->CreateRunInfo(run_info_, &r)) {
    err =
        absl::StrCat("Error in creating RunInfo: ", r.status().fail_message());
    LOG(ERROR) << err;
    return err;
  }

  run_info_.set_run_key(r.key());

  if (input_.has_duration_time_ms()) {
    run_info_.set_duration_time_ms(input_.duration_time_ms());
  }
  for (const auto& tag : input_.tags()) {
    *run_info_.add_tags() = tag;
  }

  if (input_.has_hover_text()) {
    run_info_.set_hover_text(input_.hover_text());
  }
  if (input_.has_description()) {
    run_info_.set_description(input_.description());
  }
  for (const auto& a : input_.annotation_list()) {
    *run_info_.add_annotation_list() = a;
  }
  for (const auto& h : input_.hyperlink_list()) {
    *run_info_.add_hyperlink_list() = h;
  }
  for (const auto& d : input_.aux_data()) {
    *run_info_.add_aux_data() = d;
  }
  for (const auto& i : input_.ignore_range_list()) {
    *run_info_.add_ignore_range_list() = i;
  }
  return kNoError;
}

std::string InternalQuickstore::UpdateMetricAggregates() {
  std::string err;
  if (aggregate_value_keys_.size() != aggregate_types_.size() ||
      aggregate_types_.size() != aggregate_values_.size()) {
    err = absl::StrCat(
        "MetricAggregates must be same size. aggregate_value_keys size: ",
        aggregate_value_keys_.size(),
        "aggregate_types size: ", aggregate_types_.size(),
        "aggregate_values size: ", aggregate_values_.size());
    LOG(ERROR) << err;
    return err;
  }
  if (!run_info_.has_aggregate()) {
    err = "RunInfo missing aggregate";
    LOG(ERROR) << err;
    return err;
  }

  std::vector<std::string> percentile_strings;
  for (const auto& p : run_info_.aggregate().percentile_milli_rank_list()) {
    percentile_strings.push_back(absl::StrCat("p", p));
  }

  // key is value_key.
  std::map<std::string, mako::MetricAggregate> aggs;

  auto i1 = aggregate_value_keys_.begin(), i2 = aggregate_types_.begin();
  auto i3 = aggregate_values_.begin();
  for (; i1 != aggregate_value_keys_.end(); ++i1, ++i2, ++i3) {
    std::string key = *i1;
    std::string type = *i2;
    double value = *i3;

    // Fill each new MetricAggregate with 0's for all percentiles.
    if (!aggs.count(key)) {
      aggs[key].mutable_percentile_list()->Resize(percentile_strings.size(), 0);
    }

    aggs[key].set_metric_key(key);

    if (type == "min") {
      aggs[key].set_min(value);
    } else if (type == "max") {
      aggs[key].set_max(value);
    } else if (type == "mean") {
      aggs[key].set_mean(value);
    } else if (type == "median") {
      aggs[key].set_median(value);
    } else if (type == "standard_deviation") {
      aggs[key].set_standard_deviation(value);
    } else if (type == "median_absolute_deviation") {
      aggs[key].set_median_absolute_deviation(value);
    } else if (type == "count") {
      aggs[key].set_count(value);
    } else if (!type.empty() && type[0] == 'p') {
      auto begin_it = percentile_strings.begin();
      auto end_it = percentile_strings.end();
      auto found_it = std::find(begin_it, end_it, type);
      if (found_it == end_it) {
        err = absl::StrCat("Invalid percentile: ", type);
        LOG(ERROR) << err;
        return err;
      }
      aggs[key].set_percentile_list(std::distance(begin_it, found_it), value);
    } else {
      err = absl::StrCat("Invalid MetricAggregate: metric value key: ", key,
                         " type: ", type, " value: ", value);
      LOG(ERROR) << err;
      return err;
    }
  }

  // Merge MetricAggregates from map into RunInfo
  for (int i = 0; i < run_info_.aggregate().metric_aggregate_list_size(); i++) {
    std::string k = run_info_.aggregate().metric_aggregate_list(i).metric_key();
    if (aggs.count(k)) {
      *run_info_.mutable_aggregate()->mutable_metric_aggregate_list(i) =
          aggs[k];
      aggs.erase(k);
    }
  }

  // Add MetricAggregates that weren't added above
  for (const auto& pair : aggs) {
    *run_info_.mutable_aggregate()->add_metric_aggregate_list() = pair.second;
  }

  return kNoError;
}

std::string InternalQuickstore::WriteSampleFile() {
  std::string err;
  std::string file_path = JoinPath(tmp_dir_, "sample_file");
  sample_file_.set_file_path(file_path);
  sample_file_.set_sampler_name("quickstore");
  if (!fileio_->Open(file_path, mako::FileIO::AccessMode::kWrite)) {
    fileio_->Close();
    err = absl::StrCat("Could not open path: ", file_path,
                       " for writing. Error: ", fileio_->Error());
    LOG(ERROR) << err;
    return err;
  }
  for (const auto& point : points_) {
    mako::SampleRecord sample_record;
    *sample_record.mutable_sample_point() = point;
    if (!fileio_->Write(sample_record)) {
      fileio_->Close();
      err = absl::StrCat("Could not write point to path: ", file_path,
                         ". Error: ", fileio_->Error());
      LOG(ERROR) << err;
      return err;
    }
  }
  for (const auto& error : errors_) {
    mako::SampleRecord sample_record;
    *sample_record.mutable_sample_error() = error;
    if (!sample_record.sample_error().has_sampler_name()) {
      sample_record.mutable_sample_error()->set_sampler_name("quickstore");
    }
    if (!fileio_->Write(sample_record)) {
      fileio_->Close();
      err = absl::StrCat("Could not write error to path: ", file_path,
                         ". Error: ", fileio_->Error());
      LOG(ERROR) << err;
      return err;
    }
  }

  // Close to flush the buffer.
  if (!fileio_->Close()) {
    LOG(WARNING) << absl::StrCat("Could not close path: ", file_path,
                                 ". Error: ", fileio_->Error());
  }

  return kNoError;
}

std::string InternalQuickstore::Aggregate() {
  aggregator_->SetFileIO(fileio_->MakeInstance());
  mako::AggregatorOutput output;
  mako::AggregatorInput input;
  *input.mutable_run_info() = run_info_;
  *input.mutable_benchmark_info() = benchmark_info_;
  *input.add_sample_file_list() = sample_file_;
  std::string err = aggregator_->Aggregate(input, &output);
  if (!err.empty()) {
    err = absl::StrCat("Aggregator error:", err);
    LOG(ERROR) << err;
    return err;
  }

  *run_info_.mutable_aggregate() = output.aggregate();
  return kNoError;
}

std::string InternalQuickstore::Downsample() {
  std::string err;
  downsampler_->SetFileIO(fileio_->MakeInstance());
  mako::DownsamplerOutput output;
  mako::DownsamplerInput input;
  *input.mutable_run_info() = run_info_;
  *input.add_sample_file_list() = sample_file_;
  int max;

  err = storage_->GetMetricValueCountMax(&max);
  if (!err.empty()) {
    err = absl::StrCat("GetMetricValueCountMax error: ", err);
    LOG(ERROR) << err;
    return err;
  }
  input.set_metric_value_count_max(max);

  err = storage_->GetSampleErrorCountMax(&max);
  if (!err.empty()) {
    err = absl::StrCat("GetSampleCountMax error: ", err);
    LOG(ERROR) << err;
    return err;
  }
  input.set_sample_error_count_max(max);

  err = storage_->GetBatchSizeMax(&max);
  if (!err.empty()) {
    err = absl::StrCat("GetBatchSizeMax error: ", err);
    LOG(ERROR) << err;
    return err;
  }
  input.set_batch_size_max(max);

  err = downsampler_->Downsample(input, &output);
  if (!err.empty()) {
    err = absl::StrCat("Downsample error: ", err);
    LOG(ERROR) << err;
    return err;
  }
  for (const auto& sample_batch : output.sample_batch_list()) {
    sample_batches_.push_back(sample_batch);
  }

  return kNoError;
}

std::string InternalQuickstore::Analyze() {
  std::string err;
  std::vector<std::unique_ptr<mako::Analyzer>> analyzers;
  for (const mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput&
           threshold_input : input_.threshold_inputs()) {
    analyzers.push_back(
        absl::make_unique<mako::threshold_analyzer::Analyzer>(
            threshold_input));
  }
  for (const mako::window_deviation::WindowDeviationInput& wda_input :
       input_.wda_inputs()) {
    analyzers.push_back(
        absl::make_unique<mako::window_deviation::Analyzer>(wda_input));
  }
  for (const mako::utest_analyzer::UTestAnalyzerInput& utest_input :
       input_.utest_inputs()) {
    analyzers.push_back(
        absl::make_unique<mako::utest_analyzer::Analyzer>(utest_input));
  }
  std::vector<mako::Analyzer*> analyzer_ptrs;
  analyzer_ptrs.reserve(analyzers.size());
  for (auto& p : analyzers) {
    analyzer_ptrs.push_back(p.get());
  }
  // NOTE: Not using the run_analyzers library b/c it assumes that the Run has
  // already been created. We want to pass sample batches directly in.
  // The library could be modified to take sample batches in if this becomes a
  // problem.
  err = mako::internal::RunAnalyzers(
      benchmark_info_, run_info_, sample_batches_,
      /*attach_e_divisive_regressions_to_changepoints=*/true, storage_,
      &dashboard_, analyzer_ptrs, run_info_.mutable_test_output());
  if (!err.empty()) {
    err = absl::StrCat("Analyzer error: ", err);
    LOG(ERROR) << err;
    return err;
  }
  return kNoError;
}

std::string InternalQuickstore::WriteToStorage() {
  std::string err;
  // Write SampleBatches
  for (const auto& batch : sample_batches_) {
    mako::CreationResponse resp;
    if (!storage_->CreateSampleBatch(batch, &resp)) {
      err = absl::StrCat("Failed to write SampleBatch. Error: ",
                         resp.status().fail_message());
      LOG(ERROR) << err;
      return err;
    }
    // Record them in RunInfo.
    *run_info_.add_batch_key_list() = resp.key();
  }
  // Update RunInfo
  mako::ModificationResponse r;
  if (!storage_->UpdateRunInfo(run_info_, &r)) {
    err = absl::StrCat("Error updating RunInfo: ", r.status().fail_message());
    LOG(ERROR) << err;
    return err;
  }

  // Cannot perform this update inside UpdateRunInfoTags b/c we don't have the
  // key yet.
  DashboardRunChartInput runChartConfig;
  runChartConfig.set_run_key(run_info_.run_key());
  err = dashboard_.RunChart(
      runChartConfig,
      run_info_.mutable_test_output()->mutable_run_chart_link());
  if (!err.empty()) {
    std::string err = absl::StrCat("Dashboard error: ", err);
    LOG(ERROR) << err;
    return err;
  }
  return kNoError;
}

std::string InternalQuickstore::UpdateRunInfoTags() {
  if (run_info_.test_output().test_status() == mako::TestOutput::PASS &&
      input_.has_analysis_pass()) {
    for (const auto& t : input_.analysis_pass().tags()) {
      *run_info_.add_tags() = t;
    }
  }
  if (run_info_.test_output().test_status() ==
          mako::TestOutput::ANALYSIS_FAIL &&
      input_.has_analysis_fail()) {
    for (const auto& t : input_.analysis_fail().tags()) {
      *run_info_.add_tags() = t;
    }
  }
  return kNoError;
}

QuickstoreOutput InternalQuickstore::Complete() {
  QuickstoreOutput out;
  if (run_info_.test_output().test_status() == mako::TestOutput::PASS) {
    out.set_status(QuickstoreOutput::SUCCESS);
  } else if (run_info_.test_output().test_status() ==
             mako::TestOutput::ANALYSIS_FAIL) {
    out.set_status(QuickstoreOutput::ANALYSIS_FAIL);
  } else {
    out.set_status(QuickstoreOutput::ERROR);
  }
  for (const auto& ao : run_info_.test_output().analyzer_output_list()) {
    *out.add_analyzer_output_list() = ao;
  }

  if (input_.delete_sample_files()) {
    if (fileio_->Delete(sample_file_.file_path())) {
      LOG(INFO) << "Sample file deleted";
    } else {
      LOG(WARNING) << "WARNING: Could not delete sample file: "
                   << sample_file_.file_path()
                   << " Error: " << fileio_->Error();
    }
  } else {
    out.add_generated_sample_files(sample_file_.file_path());
  }

  out.set_summary_output(run_info_.test_output().summary_output());
  out.set_run_chart_link(run_info_.test_output().run_chart_link());
  out.set_run_key(run_info_.run_key());

  return out;
}

QuickstoreOutput Save(const QuickstoreInput& input,
                      const std::vector<mako::SamplePoint>& points,
                      const std::vector<mako::SampleError>& errors,
                      const std::vector<mako::KeyedValue>& run_aggregates,
                      const std::vector<std::string>& aggregate_value_keys,
                      const std::vector<std::string>& aggregate_types,
                      const std::vector<double>& aggregate_values) {
  auto s = mako::NewMakoClient();
  return SaveWithStorage(s.get(), input, points, errors, run_aggregates,
                         aggregate_value_keys, aggregate_types,
                         aggregate_values);
}

QuickstoreOutput SaveWithStorage(
    mako::Storage* storage, const QuickstoreInput& input,
    const std::vector<mako::SamplePoint>& points,
    const std::vector<mako::SampleError>& errors,
    const std::vector<mako::KeyedValue>& run_aggregates,
    const std::vector<std::string>& aggregate_value_keys,
    const std::vector<std::string>& aggregate_types,
    const std::vector<double>& aggregate_values) {
  InternalQuickstore quick(
      storage,
          absl::make_unique<mako::memory_fileio::FileIO>(),
      absl::make_unique<mako::aggregator::Aggregator>(),
      absl::make_unique<mako::downsampler::Downsampler>(), input, points,
      errors, run_aggregates, aggregate_value_keys, aggregate_types,
      aggregate_values);
  return quick.Save();
}

}  // namespace internal
}  // namespace quickstore
}  // namespace mako
