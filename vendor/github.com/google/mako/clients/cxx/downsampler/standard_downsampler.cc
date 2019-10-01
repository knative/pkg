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
#include "clients/cxx/downsampler/standard_downsampler.h"

#include <stddef.h>

#include <algorithm>
#include <iterator>
#include <random>
#include <unordered_map>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/io/coded_stream.h"
#include "src/google/protobuf/descriptor.h"
#include "src/google/protobuf/repeated_field.h"
#include "clients/cxx/downsampler/metric_set.h"
#include "internal/proto/mako_internal.pb.h"
#include "absl/algorithm/container.h"
#include "absl/base/thread_annotations.h"
#include "absl/memory/memory.h"
#include "absl/random/random.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_join.h"
#include "absl/synchronization/mutex.h"
#include "internal/cxx/load/common/executor.h"
#include "internal/cxx/proto_validation.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace downsampler {

const int kMaxThreads = 8;
const int kMaxErrorStringLength = 1000;
const int kMaxAnnotationsSize = 800000;  // 800kb

namespace {
static const char kNoError[] = "";

int64_t CalculateSampleAnnotationRepeatedFieldSize(
    const google::protobuf::RepeatedPtrField<mako::SampleAnnotation>& annotations,
    int field_number) {
  // See https://developers.google.com/protocol-buffers/docs/encoding#structure
  // for information on how repeated fields are encoded. Also see AddBatch
  // function, where a similar logic is implemented.
  // Per documentation, repeated message is the same as an optional field, but
  // its key can be repeated many times.

  // Key size of this repeated message.
  int64_t key_byte_size =
      google::protobuf::io::CodedOutputStream::VarintSize64((field_number << 3) | 2);
  int64_t total_size = 0;

  for (const mako::SampleAnnotation& annotation : annotations) {
    int64_t message_size = annotation.ByteSizeLong();
    total_size += google::protobuf::io::CodedOutputStream::VarintSize64(message_size) +
                  key_byte_size + message_size;
  }
  return total_size;
}

// Downsamples SampleAnnotations in a SamplePoint in order to fit into
// kMaxAnnotationsSize limit. Could potentially remove all annotations from
// all SamplePoints. Annotations in one SamplePoint are treated atomically, i.e.
// either all annotations from a SamplePoint are kept or removed.
void ProcessAllRecords(
    std::vector<std::unique_ptr<mako::SamplePoint>>* all_records) {
  absl::BitGen bit_generator;
  absl::c_shuffle(*all_records, bit_generator);

  int64_t total_annotations_size = 0;
  if (!all_records->empty()) {
    VLOG(2) << "Parsing annotations from saved SamplePoints";
    int annotation_list_index = all_records->at(0)
                                    ->GetDescriptor()
                                    ->FindFieldByName("sample_annotations_list")
                                    ->number();
    int discarded_annotations = 0;
    for (std::unique_ptr<mako::SamplePoint>& record : *all_records) {
      int64_t current_annotation_size =
          CalculateSampleAnnotationRepeatedFieldSize(
              record->sample_annotations_list(), annotation_list_index);
      if (total_annotations_size + current_annotation_size >
          kMaxAnnotationsSize) {
        // We can't afford this annotation, drop it.
        VLOG(2) << absl::StrCat("Clearing ", current_annotation_size,
                                " bytes of annotations by removing ",
                                record->sample_annotations_list_size(),
                                " messages from sample_annotations_list");
        record->clear_sample_annotations_list();
        current_annotation_size = 0;
      }
      total_annotations_size += current_annotation_size;
    }
    VLOG(2) << absl::StrCat(
        discarded_annotations,
        " annotations were deleted. Space used by annotations: ",
        total_annotations_size);
  }
}

void ProcessAllRecords(
    std::vector<std::unique_ptr<mako::SampleError>>* all_records) {
  for (std::unique_ptr<mako::SampleError>& new_error : *all_records) {
    // Truncate the copy of the sample error as defined in
    // https://github.com/google/mako/blob/master/spec/proto/mako.proto
    if (new_error->error_message().length() > kMaxErrorStringLength) {
      LOG(WARNING) << "Error message from sampler error, truncated to "
                   << kMaxErrorStringLength
                   << " chars. Full error below:\n >>> START\n"
                   << new_error->error_message() << "\n<<< END";
      // Truncate that error message.
      new_error->set_error_message(
          new_error->error_message().substr(kMaxErrorStringLength));
    }
  }
}

std::string DebugStringWithoutLongFields(const DownsamplerInput& input) {
  return absl::StrCat(
      "metric_value_count_max: ", input.metric_value_count_max(),
      "\nsample_error_count_max: ", input.sample_error_count_max(),
      "\nbatch_size_max: ", input.batch_size_max());
}

}  // namespace

// RecordSaver is responsible for holding metadata about a metric set.
//
// A RecordSaver is used to save both SampleErrors and SamplePoints.
//
// The term 'slots' is used below. When holding SampleErrors slots == 1.
// When holding SamplePoints slots = # of metrics inside the sample.
template <typename T>
class RecordSaver {
 public:
  RecordSaver(const MetricSet& metric_set, int prng_seed)
      : parsed_slots_(0), metric_set_(metric_set) {
    prng_.seed(prng_seed);
  }
  // Total number of slots we have saved for this metric key
  int slots() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    return metric_set_.slot_count * saved_.size();
  }

  std::string RemoveRandomSavedRecord() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    if (saved_.empty()) {
      std::string err = absl::StrCat("No records saved for metric set: ",
                                metric_set_.ToString());
      LOG(ERROR) << err;
      return err;
    }

    // Choose random index to remove
    int index_to_delete =
        std::uniform_int_distribution<size_t>{0, saved_.size() - 1}(prng_);
    saved_[index_to_delete] = std::move(saved_.back());
    saved_.pop_back();
    VLOG(1) << "Cleared " << metric_set_.slot_count << " slots from metric set "
            << metric_set_.ToString();
    return kNoError;
  }

  std::string AddSavedRecord(std::unique_ptr<T> item, const MetricSet& metric_set)
      LOCKS_EXCLUDED(mutex_) {
    if (metric_set != metric_set_) {
      std::string err = absl::StrCat("Got: ", metric_set.ToString(),
                                "; expected: ", metric_set_.ToString());
      LOG(ERROR) << err;
      return err;
    }
    absl::MutexLock l(&mutex_);
    saved_.push_back(std::move(item));
    return kNoError;
  }

  int parsed_slots() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    return parsed_slots_;
  }

  int add_parsed_slots() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    parsed_slots_ += metric_set_.slot_count;
    return parsed_slots_;
  }

  int NumberOfRecords() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    return saved_.size();
  }
  // Moves all saved records to the given vector, and resets the state of this
  // Saver
  void MoveSavedRecords(std::vector<std::unique_ptr<T>>* all_records)
      LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    std::move(saved_.begin(), saved_.end(), std::back_inserter(*all_records));
    saved_.clear();
    parsed_slots_ = 0;
  }
  int slots_per_record() { return metric_set_.slot_count; }

 private:
  // Used to synchronize access, since there might potentially be many
  // threads wanting to add or remove records from it at the same time.
  absl::Mutex mutex_;
  // Total number of slots that we have seen so far for this key.
  int parsed_slots_ GUARDED_BY(mutex_);
  // Pseudo-RNG for determining which record to evict
  std::default_random_engine prng_ GUARDED_BY(mutex_);
  // All the saved points (either SamplePoints or SampleErrors).
  std::vector<std::unique_ptr<T>> saved_ GUARDED_BY(mutex_);
  // The metric set for this groups of records.
  const MetricSet metric_set_;
};

// RecordManager provides mapping from keys->Record (via RecordSavers) and
// metadata about Records.
template <typename T>
class RecordManager {
 public:
  RecordManager(const std::string& name, int64_t max_slots, bool replace,
                int prng_seed)
      : name_(name), max_slot_count_(max_slots), replace_(replace) {
    prng_.seed(prng_seed);
  }

  // Sum of all slots used by RecordSavers
  int slot_count() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    return slot_count_;
  }

  // How many slots we have parsed for type <T>
  int parsed_slots() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    return parsed_slots_;
  }

  // Verify that we have met the slot count restrictions.
  std::string VerifyLimits() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    if (slot_count_ > max_slot_count_) {
      std::string err = absl::StrCat(
          name_,
          " downsampler failed to reduce record count; have: ", slot_count_,
          " max is: ", max_slot_count_);
      LOG(ERROR) << err;
      return err;
    }
    VLOG(1) << "== RecordManager(" << name_ << ") =="
            << " max slots: " << max_slot_count_
            << " total slots consumed: " << slot_count_
            << " total slots parsed: " << parsed_slots_ << " --- ";
    for (const auto& pair : key_to_record_saver_) {
      VLOG(1) << pair.first.ToString()
              << " slots parsed: " << pair.second->parsed_slots()
              << " records saved: " << pair.second->NumberOfRecords();
    }
    VLOG(1) << " --- ";
    return kNoError;
  }

  // Return a vector of all Records saved.
  // Note the unique_ptr's returned. This function should be called only once
  // as it transfers ownership of all Records to the caller.
  std::vector<std::unique_ptr<T>> AllRecords() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    std::vector<std::unique_ptr<T>> all_records;
    all_records.reserve(record_count_);
    for (const auto& pair : key_to_record_saver_) {
      pair.second->MoveSavedRecords(&all_records);
    }
    return all_records;
  }

  std::string HandleRecord(std::unique_ptr<T> new_record) LOCKS_EXCLUDED(mutex_) {
    // If we don't want to save any samples
    if (max_slot_count_ == 0) {
      return kNoError;
    }

    MetricSet metric_set(new_record.get());
    VLOG(2) << "== " << name_ << " (" << metric_set.key << ")";
    if (metric_set.slot_count > max_slot_count_) {
      std::string err = absl::StrCat(
          "Attempting to add a metric to set of size: ", metric_set.slot_count,
          " but max size is: ", max_slot_count_);
      LOG(ERROR) << err;
      return err;
    }

    RecordSaver<T>* saver = GetOrCreate(metric_set);

    int count = CountOfSlotConsumers();
    CHECK(count > 0) << "Should have created RecordSaver for metric "
                     << "set: " << metric_set.ToString();
    int fair_share_slots = max_slot_count_ / count;

    absl::MutexLock l(&mutex_);
    parsed_slots_ += metric_set.slot_count;
    int saver_parsed_slots = saver->add_parsed_slots();
    if (slot_count_ + metric_set.slot_count > max_slot_count_ &&
        saver->slots() >= fair_share_slots) {
      if (!replace_) {
        return kNoError;
      }
      VLOG(2) << "Over quota (slots used by this key: " << saver->slots()
              << ")";
      CHECK(saver_parsed_slots > 0) << "Should have positive number of "
                                    << "parsed slots.";
      if (!GetRandomChoice(static_cast<double>(saver->slots()) /
                           saver_parsed_slots)) {
        VLOG(2) << "Record discarded";
        return kNoError;
      }
    }

    while (slot_count_ + metric_set.slot_count > max_slot_count_) {
      VLOG(2) << "Clearing more records";
      std::string err = RemoveRandomSavedRecord(metric_set);
      if (!err.empty()) {
        LOG(ERROR) << err;
        return err;
      }
    }
    std::string err = saver->AddSavedRecord(std::move(new_record), metric_set);
    if (!err.empty()) {
      LOG(ERROR) << err;
      return err;
    }
    VLOG(2) << metric_set.key << ": " << saver->NumberOfRecords()
            << " records saved, " << saver->slots() << " slots consumed, "
            << saver->parsed_slots() << " slots parsed";
    slot_count_ += metric_set.slot_count;
    ++record_count_;
    return kNoError;
  }

 private:
  // pick a RecordSaver to remove slots from.
  std::string RemoveRandomSavedRecord(const MetricSet& set_being_added)
      EXCLUSIVE_LOCKS_REQUIRED(mutex_) {
    if (key_to_record_saver_.empty()) {
      std::string err = absl::StrCat("RecordManager(", name_,
                                ") asked to choose largest RecordSaver but do "
                                "not have any to remove");
      LOG(ERROR) << err;
      return err;
    }

    // first pick the RecordSaver to remove a record from. we pick the one with
    // the most slots, with tie breaking in favor of the RecordSaver
    // corresponding to the metric set currently being added
    RecordSaver<T>* biggest_saver = nullptr;

    float biggest_slot_count = 0;
    for (const auto& pair : key_to_record_saver_) {
      const MetricSet& metric_set = pair.first;
      RecordSaver<T>* saver = pair.second.get();

      float this_slot_count = saver->slots();

      // ties go to the metric set being added
      if (metric_set == set_being_added) this_slot_count += 0.5;

      if (this_slot_count > biggest_slot_count) {
        biggest_slot_count = this_slot_count;
        biggest_saver = saver;
      }
    }

    CHECK(biggest_saver != nullptr)
        << "Error: No RecordSavers have a >0 slot count?";

    std::string err = biggest_saver->RemoveRandomSavedRecord();
    if (!err.empty()) {
      LOG(ERROR) << err;
      return err;
    }
    slot_count_ -= biggest_saver->slots_per_record();
    --record_count_;
    return kNoError;
  }

  // How many unique RecordSavers we have seen so far
  int CountOfSlotConsumers() LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    return key_to_record_saver_.size();
  }

  bool GetRandomChoice(double probability) EXCLUSIVE_LOCKS_REQUIRED(mutex_) {
    CHECK(probability >= 0.0 || probability <= 1.0)
        << "Probability needs to be between 0 and 1: " << probability;
    return std::uniform_real_distribution<double>{0, 1}(prng_) < probability;
  }

  RecordSaver<T>* GetOrCreate(const MetricSet& metric_set)
      LOCKS_EXCLUDED(mutex_) {
    absl::MutexLock l(&mutex_);
    if (key_to_record_saver_.count(metric_set) == 0) {
      key_to_record_saver_[metric_set] =
          absl::make_unique<RecordSaver<T>>(metric_set, prng_());
    }
    return key_to_record_saver_[metric_set].get();
  }

  absl::Mutex mutex_;
  std::unordered_map<MetricSet, std::unique_ptr<RecordSaver<T>>, HashMetricSet>
      key_to_record_saver_ GUARDED_BY(mutex_);
  std::default_random_engine prng_ GUARDED_BY(mutex_);
  int slot_count_ GUARDED_BY(mutex_) = 0;
  int parsed_slots_ GUARDED_BY(mutex_) = 0;
  int record_count_ GUARDED_BY(mutex_) = 0;

  const std::string name_;
  const int64_t max_slot_count_;
  const bool replace_;
};

std::string ProcessFile(std::unique_ptr<mako::FileIO> fio,
                   const std::string& file_path,
                   RecordManager<mako::SamplePoint>* sample_manager,
                   RecordManager<mako::SampleError>* error_manager) {
  VLOG(1) << "Reading: " << file_path;
  if (!fio->Open(file_path, mako::FileIO::AccessMode::kRead)) {
    LOG(ERROR) << fio->Error();
    return fio->Error();
  }
  mako::SampleRecord sample_record;
  while (fio->Read(&sample_record)) {
    if (!sample_record.has_sample_point() &&
        !sample_record.has_sample_error()) {
      std::string err =
          "SampleRecord must contain either sample_point or sample_error.";
      LOG(ERROR) << err;
      return err;
    }
    if (sample_record.has_sample_point()) {
      std::string err = sample_manager->HandleRecord(
          absl::WrapUnique(sample_record.release_sample_point()));
      if (!err.empty()) {
        LOG(ERROR) << err;
        return err;
      }
    }
    if (sample_record.has_sample_error()) {
      std::string err = error_manager->HandleRecord(
          absl::WrapUnique(sample_record.release_sample_error()));
      if (!err.empty()) {
        LOG(ERROR) << err;
        return err;
      }
    }
    sample_record.Clear();
  }

  if (!fio->ReadEOF()) {
    std::string err = fio->Error();
    LOG(ERROR) << err;
    return err;
  }
  return kNoError;
}

std::string ProcessFiles(const mako::DownsamplerInput& downsampler_input,
                    mako::FileIO* fileio, int max_threads,
                    RecordManager<mako::SamplePoint>* sample_manager,
                    RecordManager<mako::SampleError>* error_manager) {
  int num_threads = downsampler_input.sample_file_list_size();
  if (max_threads > 0 && max_threads < num_threads) {
    num_threads = max_threads;
  }
  LOG(INFO) << "Creating thread pool with " << num_threads << " threads.";
  mako::internal::Executor file_processor(num_threads);
  absl::Mutex m;
  std::vector<std::string> errors;
  for (const mako::SampleFile& sample_file :
       downsampler_input.sample_file_list()) {
    file_processor.Schedule(
        [&sample_file, &m, &errors, sample_manager, error_manager, fileio]() {
          std::string error =
              ProcessFile(fileio->MakeInstance(), sample_file.file_path(),
                          sample_manager, error_manager);
          if (!error.empty()) {
            absl::MutexLock l(&m);
            errors.push_back(error);
          }
        });
  }
  file_processor.Wait();
  if (!errors.empty()) {
    std::string error = absl::StrJoin(errors, "\n");
    LOG(ERROR) << error;
    return error;
  }
  return kNoError;
}

void Downsampler::Reseed(int prng_seed) { prng_.seed(prng_seed); }

template <typename T>
std::string AddBatches(const std::string& benchmark_key, const std::string& run_key,
                  const int batch_size_max, const int field_number,
                  RecordManager<T>* manager, mako::SampleBatch** batch,
                  int64_t* batch_size_bytes,
                  mako::DownsamplerOutput* downsampler_output) {
  std::vector<std::unique_ptr<T>> all_records = manager->AllRecords();
  // Apply processing to all records
  ProcessAllRecords(&all_records);
  // Sort records by ascending input_value.
  std::sort(all_records.begin(), all_records.end(),
            [](const std::unique_ptr<T>& a, const std::unique_ptr<T>& b) {
              return a->input_value() < b->input_value();
            });
  for (auto& record : all_records) {
    std::string err =
        AddBatch(benchmark_key, run_key, batch_size_max, field_number,
                 record.get(), batch, batch_size_bytes, downsampler_output);
    if (!err.empty()) {
      LOG(ERROR) << err;
      return err;
    }
  }
  return kNoError;
}

std::string Complete(const mako::DownsamplerInput& downsampler_input,
                mako::DownsamplerOutput* downsampler_output,
                RecordManager<mako::SamplePoint>* sample_manager,
                RecordManager<mako::SampleError>* error_manager) {
  std::string err = sample_manager->VerifyLimits();
  if (!err.empty()) {
    LOG(ERROR) << err;
    return err;
  }

  err = error_manager->VerifyLimits();
  if (!err.empty()) {
    LOG(ERROR) << err;
    return err;
  }

  LOG(INFO) << "\n"
            << "Downsampling Results:\n"
            << "  Downsampler: MetricDownsampler\n"
            << "  Sample Files: " << downsampler_input.sample_file_list_size()
            << "\n"
            << "  Total Metrics: " << sample_manager->parsed_slots() << "\n"
            << "  Total Errors: " << error_manager->parsed_slots() << "\n"
            << "  Downsampled Metrics: " << sample_manager->slot_count() << "\n"
            << "  Downsampled Errors: " << error_manager->slot_count() << "\n";

  LOG(INFO) << "Creating SampleBatches";
  std::string benchmark_key = downsampler_input.run_info().benchmark_key();
  std::string run_key = downsampler_input.run_info().run_key();
  if (sample_manager->slot_count() + error_manager->slot_count() > 0) {
    int64_t batch_size_bytes;
    mako::SampleBatch* sample_batch = GetNewBatch(
        benchmark_key, run_key, downsampler_output, &batch_size_bytes);
    auto descriptor = sample_batch->GetDescriptor();

    std::string err = AddBatches(
        benchmark_key, run_key, downsampler_input.batch_size_max(),
        descriptor->FindFieldByName("sample_point_list")->number(),
        sample_manager, &sample_batch, &batch_size_bytes, downsampler_output);
    if (!err.empty()) {
      LOG(ERROR) << err;
      return err;
    }

    err = AddBatches(benchmark_key, run_key, downsampler_input.batch_size_max(),
                     descriptor->FindFieldByName("sample_error_list")->number(),
                     error_manager, &sample_batch, &batch_size_bytes,
                     downsampler_output);
    if (!err.empty()) {
      LOG(ERROR) << err;
      return err;
    }
  }

  LOG(INFO) << downsampler_output->sample_batch_list_size()
            << " SampleBatches created.";
  if (VLOG_IS_ON(1)) {
    VLOG(1) << "SampleBatch sizes: ";
    for (const mako::SampleBatch& sample_batch :
         downsampler_output->sample_batch_list()) {
      VLOG(1) << sample_batch.ByteSize();
    }
  }
  return err;
}

std::string Downsampler::Downsample(
    const mako::DownsamplerInput& downsampler_input,
    mako::DownsamplerOutput* downsampler_output) {

  LOG(INFO) << "Downsampler.Downsample("
            << DebugStringWithoutLongFields(downsampler_input) << ")";
  std::string err;

  if (!fileio_) {
    err = "FileIO has not been set";
    LOG(ERROR) << err;
    return err;
  }

  err = mako::internal::ValidateDownsamplerInput(downsampler_input);
  if (!err.empty()) {
    LOG(ERROR) << err;
    return err;
  }

  LOG(INFO) << "Downsampling start, file count: "
            << downsampler_input.sample_file_list_size();

  RecordManager<mako::SamplePoint> sample_manager(
      "SamplePointManager", downsampler_input.metric_value_count_max(), true,
      prng_());
  RecordManager<mako::SampleError> error_manager(
      "SampleErrorManager", downsampler_input.sample_error_count_max(), false,
      prng_());

  err = ProcessFiles(downsampler_input, fileio_.get(), max_threads_,
                     &sample_manager, &error_manager);
  if (!err.empty()) {
    LOG(ERROR) << err;
    return err;
  }

  err = Complete(downsampler_input, downsampler_output, &sample_manager,
                 &error_manager);
  if (!err.empty()) {
    LOG(ERROR) << err;
    return err;
  }
  LOG(INFO) << "Downsampling complete";
  return kNoError;
}

void GetNewRecord(mako::SampleBatch* batch,
                  mako::SamplePoint** new_point) {
  *new_point = batch->add_sample_point_list();
}

void GetNewRecord(mako::SampleBatch* batch,
                  mako::SampleError** new_error) {
  *new_error = batch->add_sample_error_list();
}

mako::SampleBatch* GetNewBatch(
    const std::string& benchmark_key, const std::string& run_key,
    mako::DownsamplerOutput* downsampler_output,
    int64_t* batch_size_bytes) {
  mako::SampleBatch* batch = downsampler_output->add_sample_batch_list();
  batch->set_benchmark_key(benchmark_key);
  batch->set_run_key(run_key);
  *batch_size_bytes = batch->ByteSizeLong();
  return batch;
}

}  // namespace downsampler
}  // namespace mako
