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

#include <algorithm>
#include <cmath>
#include <type_traits>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/descriptor.h"
#include "src/google/protobuf/repeated_field.h"
#include "gtest/gtest.h"
#include "clients/cxx/fileio/memory_fileio.h"
#include "absl/strings/str_cat.h"
#include "internal/cxx/filter_utils.h"
#include "internal/cxx/pgmath.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace downsampler {

static constexpr int kMetricValueMax = 200;
static constexpr int kSampleErrorCountMax = 10;
static constexpr int kBatchSizeMax = 100000;

class StandardMetricDownsamplerTest : public ::testing::Test {
 protected:
  void SetUp() override {
    d_.SetFileIO(std::unique_ptr<mako::FileIO>(
        new mako::memory_fileio::FileIO()));
  }

  void ReseedDownsampler() { d_.Reseed(99999); }

  Downsampler d_;
};

int CountPointsForMetricKey(
    const std::string& metric_key,
    const google::protobuf::RepeatedPtrField<mako::SampleBatch>& sample_batch_list) {
  int count = 0;
  for (const mako::SampleBatch& sample_batch : sample_batch_list) {
    for (const auto& sample_point : sample_batch.sample_point_list()) {
      for (const auto& metric_value : sample_point.metric_value_list()) {
        if (metric_value.value_key() == metric_key) {
          count++;
        }
      }
    }
  }
  return count;
}

int CountErrorsForSampler(
    const std::string& sampler_name,
    const google::protobuf::RepeatedPtrField<mako::SampleBatch>& sample_batch_list) {
  int count = 0;
  for (const mako::SampleBatch& sample_batch : sample_batch_list) {
    if (sampler_name.empty()) {
      count += sample_batch.sample_error_list_size();
    } else {
      for (const mako::SampleError& sample_error :
           sample_batch.sample_error_list()) {
        if (sample_error.sampler_name() == sampler_name) {
          count++;
        }
      }
    }
  }
  return count;
}

int CountErrors(
    const google::protobuf::RepeatedPtrField<mako::SampleBatch>& sample_batch_list) {
  return CountErrorsForSampler("", sample_batch_list);
}

SampleRecord CreateSampleRecord(
    double input_value, const std::vector<std::pair<std::string, double>>& metrics,
    const std::vector<std::pair<std::string, std::string>>& aux_data = {}) {
  SampleRecord sr;
  SamplePoint* sp = sr.mutable_sample_point();
  sp->set_input_value(input_value);
  for (auto pair : metrics) {
    KeyedValue* kv = sp->add_metric_value_list();
    kv->set_value_key(pair.first);
    kv->set_value(pair.second);
  }
  for (auto kv : aux_data) {
    (*sp->mutable_aux_data())[kv.first] = kv.second;
  }
  return sr;
}

DownsamplerInput CreateDownsamplerInput(
    const std::vector<std::string>& files,
    int sample_error_count_max = kSampleErrorCountMax,
    int metric_value_count_max = kMetricValueMax,
    int batch_size_max = kBatchSizeMax) {
  mako::DownsamplerInput di;

  for (const auto& file : files) {
    mako::SampleFile* sample_file = di.add_sample_file_list();
    sample_file->set_file_path(file);
    sample_file->set_sampler_name(absl::StrCat("Sampler", file));
  }

  // Create RunInfo
  di.mutable_run_info()->set_benchmark_key("benchmark_key");
  di.mutable_run_info()->set_run_key("run_key");
  di.mutable_run_info()->set_timestamp_ms(123456);

  di.set_sample_error_count_max(kSampleErrorCountMax);
  di.set_metric_value_count_max(metric_value_count_max);
  di.set_batch_size_max(batch_size_max);

  return di;
}

void WriteFile(const std::string& file_path,
               const std::vector<mako::SampleRecord>& data) {
  mako::memory_fileio::FileIO fileio;

  ASSERT_TRUE(fileio.Open(file_path, mako::FileIO::AccessMode::kWrite));
  for (const auto& d : data) {
    ASSERT_TRUE(fileio.Write(d));
  }
  ASSERT_TRUE(fileio.Close());
}

TEST_F(StandardMetricDownsamplerTest, MissingFileIO) {
  Downsampler d;
  mako::DownsamplerOutput out;

  std::string err = d.Downsample(CreateDownsamplerInput({}), &out);

  ASSERT_NE("", err);
}

TEST_F(StandardMetricDownsamplerTest, InvalidDownsamplerInput) {
  mako::DownsamplerOutput out;
  mako::DownsamplerInput in = CreateDownsamplerInput({});
  // invalidate the input
  in.clear_run_info();

  std::string err = d_.Downsample(in, &out);

  ASSERT_NE("", err);
}

TEST_F(StandardMetricDownsamplerTest, NoFilesToDownsample) {
  mako::DownsamplerOutput out;

  ASSERT_EQ("", d_.Downsample(CreateDownsamplerInput({}), &out));

  ASSERT_EQ(0, out.sample_batch_list_size());
}

TEST_F(StandardMetricDownsamplerTest, NoSuchFiles) {
  mako::DownsamplerOutput out;

  ASSERT_NE("", d_.Downsample(CreateDownsamplerInput({"NoSuchFile"}), &out));
}

TEST_F(StandardMetricDownsamplerTest, EmptyFile) {
  mako::DownsamplerOutput out;

  mako::memory_fileio::FileIO fileio;
  fileio.Open("file", mako::FileIO::AccessMode::kWrite);
  fileio.Close();

  ASSERT_EQ("", d_.Downsample(CreateDownsamplerInput({"file"}), &out));
  ASSERT_EQ(0, out.sample_batch_list_size());
}

TEST_F(StandardMetricDownsamplerTest, SmallBatchSizeToForceMultiBatchCreation) {
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});

  // Set a small batch size so we get lots of batches.
  in.set_batch_size_max(3000);

  std::vector<SampleRecord> sample_records;

  for (int i = 0; i < 200; i++) {
    mako::SampleRecord sb = CreateSampleRecord(i, {{"y", i}});
    sb.mutable_sample_error()->set_error_message("An Error message");
    sb.mutable_sample_error()->set_input_value(i);
    sb.mutable_sample_error()->set_sampler_name(absl::StrCat("Sampler", i));
    sample_records.push_back(sb);
    ASSERT_GT(200, sb.ByteSizeLong());
  }

  WriteFile("file1", sample_records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(in, &out));
  ASSERT_GT(out.sample_batch_list_size(), 1);
}

TEST_F(StandardMetricDownsamplerTest, NoDataRequest) {
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});
  in.set_metric_value_count_max(0);
  in.set_sample_error_count_max(0);

  std::vector<SampleRecord> sample_records;

  for (int i = 0; i < 200; i++) {
    mako::SampleRecord sb = CreateSampleRecord(i, {{"y", i}});
    sb.mutable_sample_error()->set_error_message("An Error message");
    sb.mutable_sample_error()->set_input_value(i);
    sb.mutable_sample_error()->set_sampler_name(absl::StrCat("Sampler", i));
    sample_records.push_back(sb);
  }

  WriteFile("file1", sample_records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(in, &out));
  ASSERT_EQ(0, out.sample_batch_list_size());
}

TEST_F(StandardMetricDownsamplerTest, SampleBatchesSortedByInputValue) {
  std::vector<SampleRecord> sample_records;

  for (int input_value : {4, 1, 5, 3, 2}) {
    mako::SampleRecord sb =
        CreateSampleRecord(input_value, {{"y", input_value}});
    sb.mutable_sample_error()->set_error_message("An Error message");
    sb.mutable_sample_error()->set_input_value(input_value);
    sb.mutable_sample_error()->set_sampler_name(
        absl::StrCat("Sampler", input_value));
    sample_records.push_back(sb);
  }

  WriteFile("file1", sample_records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(CreateDownsamplerInput({"file1"}), &out));
  ASSERT_EQ(1, out.sample_batch_list_size());

  std::vector<int> sorted_input_values({1, 2, 3, 4, 5});

  // Check sort order of sample points
  std::vector<int> actual_sample_point_input_values;
  for (const auto& sample_point :
       out.sample_batch_list(0).sample_point_list()) {
    actual_sample_point_input_values.push_back(sample_point.input_value());
  }
  ASSERT_EQ(sorted_input_values, actual_sample_point_input_values);

  // Check sort order of errors
  std::vector<int> actual_sample_error_input_values;
  for (const auto& sample_error :
       out.sample_batch_list(0).sample_error_list()) {
    actual_sample_error_input_values.push_back(sample_error.input_value());
  }
  ASSERT_EQ(sorted_input_values, actual_sample_error_input_values);
}

TEST_F(StandardMetricDownsamplerTest, DuplicateMetricKeysInSampleRecord) {
  std::string duplicate_key = "m1";
  mako::SampleRecord sb = CreateSampleRecord(
      1, {{duplicate_key, 1}, {duplicate_key, 2}, {"m2", 3}});
  WriteFile("file3", {sb});
  mako::DownsamplerOutput out;
  std::string err = d_.Downsample(CreateDownsamplerInput({"file3"}), &out);
  ASSERT_EQ("", err);
}

TEST_F(StandardMetricDownsamplerTest, InvalidSampleRecord) {
  mako::SampleRecord sr;
  WriteFile("file4", {sr});
  mako::DownsamplerOutput out;
  std::string err = d_.Downsample(CreateDownsamplerInput({"file4"}), &out);
  EXPECT_EQ("SampleRecord must contain either sample_point or sample_error.",
            err);
}

TEST_F(StandardMetricDownsamplerTest, MetricSetTooBig) {
  // If we can only save 1 metricValue but a single SamplePoint has > 1 metric
  // inside.
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});
  in.set_metric_value_count_max(1);

  mako::SampleRecord sb =
      CreateSampleRecord(1, {{"y1,y2", 1}, {"y2", 2}, {"y3", 3}});
  // We have more than a single metric packed
  ASSERT_GT(sb.sample_point().metric_value_list_size(), 1);

  WriteFile("file1", {sb});
  mako::DownsamplerOutput out;
  ASSERT_NE("", d_.Downsample(in, &out));

  // But if raise size, then should work.
  mako::DownsamplerOutput out2;
  in.set_metric_value_count_max(10);
  ASSERT_EQ("", d_.Downsample(in, &out2));
}

TEST_F(StandardMetricDownsamplerTest, PointBatchTooBig) {
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});

  // Reset batch size so anything is invalid.
  in.set_batch_size_max(1);

  mako::SampleRecord sb =
      CreateSampleRecord(1, {{"y1", 1}, {"y2", 2}, {"y3", 3}});
  ASSERT_GT(sb.ByteSizeLong(), 1);
  WriteFile("file1", {sb});
  mako::DownsamplerOutput out;
  ASSERT_NE("", d_.Downsample(in, &out));

  // But if raise size, then should work.
  in.set_batch_size_max(sb.ByteSizeLong() * 2);
  mako::DownsamplerOutput out2;
  ASSERT_EQ("", d_.Downsample(in, &out2));
  ASSERT_EQ(1, CountPointsForMetricKey("y1", out2.sample_batch_list()));
  ASSERT_EQ(1, CountPointsForMetricKey("y2", out2.sample_batch_list()));
  ASSERT_EQ(1, CountPointsForMetricKey("y3", out2.sample_batch_list()));
}

TEST_F(StandardMetricDownsamplerTest, ErrorBatchTooBig) {
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});
  std::vector<SampleRecord> sample_records;
  for (int i = 0; i < 100; i++) {
    mako::SampleRecord sb = CreateSampleRecord(1, {});
    sb.clear_sample_point();
    sb.mutable_sample_error()->set_error_message("something");
    sb.mutable_sample_error()->set_input_value(i);
    sb.mutable_sample_error()->set_sampler_name(absl::StrCat("Sampler", i));
    sample_records.push_back(sb);
    ASSERT_GT(sb.ByteSizeLong(), 1);
  }
  WriteFile("file1", sample_records);

  // Reset batch size so anything is invalid.
  in.set_batch_size_max(1);
  mako::DownsamplerOutput out;
  ASSERT_NE("", d_.Downsample(in, &out));

  // But if raise size, then should work.
  in.set_batch_size_max(1024);
  mako::DownsamplerOutput out2;
  ASSERT_EQ("", d_.Downsample(in, &out2));
  ASSERT_GT(CountErrors(out2.sample_batch_list()), 0);
}

TEST_F(StandardMetricDownsamplerTest, ErrorMessagesTruncated) {
  mako::DownsamplerOutput out;
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});

  // Error message should not be truncated.
  mako::SampleRecord small_sample_record =
      CreateSampleRecord(1, {{"y", 1}});
  std::string small_str;
  small_str.resize(kMaxErrorStringLength - 1);
  small_sample_record.mutable_sample_error()->set_error_message(small_str);
  small_sample_record.mutable_sample_error()->set_input_value(1);
  small_sample_record.mutable_sample_error()->set_sampler_name("s1");
  ASSERT_LT(small_str.length(), kMaxErrorStringLength);

  // Error message should not be truncated.
  mako::SampleRecord exact_sample_record =
      CreateSampleRecord(2, {{"y", 1}});
  std::string exact_str;
  exact_str.resize(kMaxErrorStringLength);
  exact_sample_record.mutable_sample_error()->set_error_message(exact_str);
  exact_sample_record.mutable_sample_error()->set_input_value(1);
  exact_sample_record.mutable_sample_error()->set_sampler_name("s1");
  ASSERT_EQ(exact_str.length(), kMaxErrorStringLength);

  // Error message should be truncated.
  mako::SampleRecord big_sample_record = CreateSampleRecord(3, {{"y", 1}});
  std::string big_str;
  big_str.resize(kMaxErrorStringLength + 1);
  big_sample_record.mutable_sample_error()->set_error_message(big_str);
  big_sample_record.mutable_sample_error()->set_input_value(1);
  big_sample_record.mutable_sample_error()->set_sampler_name("s1");
  ASSERT_GT(big_str.length(), kMaxErrorStringLength);

  // These are big, so set our size very large.
  in.set_batch_size_max(big_sample_record.ByteSizeLong() * 3);

  WriteFile("file1",
            {small_sample_record, exact_sample_record, big_sample_record});
  ASSERT_EQ("", d_.Downsample(in, &out));

  for (const mako::SampleBatch& sample_batch : out.sample_batch_list()) {
    ASSERT_GT(sample_batch.sample_error_list_size(), 0);
    for (const mako::SampleError& sample_error :
         sample_batch.sample_error_list()) {
      ASSERT_LE(sample_error.error_message().size(), kMaxErrorStringLength);
    }
  }
}

TEST_F(StandardMetricDownsamplerTest, SingleMetricDownsample) {
  mako::memory_fileio::FileIO fileio;

  // pair = # of SampleRecord to create with points, # of SampleRecords to
  // create with
  // errors
  // If num is > Max then we expect it to truncated, otherwise should not
  // downsample.
  std::vector<std::pair<int, int>> tests = {
      // Only points
      std::make_pair(1, 0),
      std::make_pair(kMetricValueMax - 1, 0),
      std::make_pair(kMetricValueMax, 0),
      std::make_pair(kMetricValueMax + 1, 0),
      std::make_pair(kMetricValueMax * 10, 0),
      // Only errors
      std::make_pair(0, 1),
      std::make_pair(0, kSampleErrorCountMax - 1),
      std::make_pair(0, kSampleErrorCountMax),
      std::make_pair(0, kSampleErrorCountMax + 1),
      std::make_pair(0, kSampleErrorCountMax * 10),
      // Mixed
      std::make_pair(1, 1),
      std::make_pair(kMetricValueMax - 1, kSampleErrorCountMax - 1),
      std::make_pair(kMetricValueMax - 1, kSampleErrorCountMax * 10),
      std::make_pair(kMetricValueMax * 10, kSampleErrorCountMax - 1),
      std::make_pair(kMetricValueMax, kSampleErrorCountMax),
      std::make_pair(kMetricValueMax + 1, kSampleErrorCountMax + 1),
      std::make_pair(kMetricValueMax * 10, kSampleErrorCountMax * 10),
  };

  for (const auto& pair : tests) {
    int number_of_points = pair.first;
    int number_of_errors = pair.second;
    LOG(INFO) << "Current testing with: " << number_of_points << " points and "
              << number_of_errors << " errors.";

    fileio.Clear();
    mako::DownsamplerOutput out;

    std::vector<mako::SampleRecord> data;

    for (int i = 0; i < number_of_points; i++) {
      data.push_back(CreateSampleRecord(i, {{"y", i}}));
    }

    for (int i = 0; i < number_of_errors; i++) {
      SampleRecord sr = CreateSampleRecord(i, {});
      sr.clear_sample_point();
      sr.mutable_sample_error()->set_error_message(absl::StrCat("Error # ", i));
      sr.mutable_sample_error()->set_input_value(i);
      sr.mutable_sample_error()->set_sampler_name("badsampler");
      data.push_back(sr);
    }

    WriteFile("file1", data);

    mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});

    ASSERT_EQ("", d_.Downsample(in, &out));

    if (number_of_points > 0) {
      if (number_of_points > kMetricValueMax) {
        ASSERT_EQ(kMetricValueMax,
                  CountPointsForMetricKey("y", out.sample_batch_list()));
      } else {
        ASSERT_EQ(number_of_points,
                  CountPointsForMetricKey("y", out.sample_batch_list()));
      }
    }

    if (number_of_errors > 0) {
      if (number_of_errors > kSampleErrorCountMax) {
        ASSERT_EQ(kSampleErrorCountMax, CountErrors(out.sample_batch_list()));
      } else {
        ASSERT_EQ(number_of_errors, CountErrors(out.sample_batch_list()));
      }
    }
  }
}

struct SamplerInstructions {
  SamplerInstructions(const std::string& sampler_name, int file_count,
                      int metric_count, int expected_metric_count,
                      int error_count, int expected_error_count)
      : sampler_name(sampler_name),
        file_count(file_count),
        metric_count(metric_count),
        expected_metric_count(expected_metric_count),
        error_count(error_count),
        expected_error_count(expected_error_count) {}
  std::string sampler_name;
  // The above totals are written to *each* of these files.
  int file_count;
  int metric_count;
  int expected_metric_count;
  int error_count;
  int expected_error_count;
};

struct MultiSamplerTest {
  MultiSamplerTest(const std::string& test_name,
                   const std::vector<SamplerInstructions>& sampler_instructions)
      : test_name(test_name), sampler_instructions(sampler_instructions) {}
  std::string test_name;
  std::vector<SamplerInstructions> sampler_instructions;
};

TEST_F(StandardMetricDownsamplerTest, MultiSamplerDownsample) {
  mako::memory_fileio::FileIO fileio;

  // Sampler name mapped to how many points we'd like that sampler to create.
  std::vector<MultiSamplerTest> tests;

  tests.push_back(
      MultiSamplerTest("two-samplers-with-few-points",
                       {
                           SamplerInstructions("Sampler1",
                                               1,   // file_count
                                               1,   // metric count
                                               1,   // expected_metric_count
                                               0,   // error_count
                                               0),  // expected_error_count
                           SamplerInstructions("Sampler2",
                                               1,   // file_count
                                               1,   // metric count
                                               1,   // expected_metric_count
                                               0,   // error_count
                                               0),  // expected_error_count
                       }));
  tests.push_back(MultiSamplerTest(
      "no-data", {
                     SamplerInstructions("Sampler1",
                                         1,   // file_count
                                         0,   // metric count
                                         0,   // expected_metric_count
                                         0,   // error_count
                                         0),  // expected_error_count
                     SamplerInstructions("Sampler2",
                                         1,   // file_count
                                         0,   // metric count
                                         0,   // expected_metric_count
                                         0,   // error_count
                                         0),  // expected_error_count
                 }));
  tests.push_back(MultiSamplerTest(
      "two-samplers-with-max-points",
      {
          SamplerInstructions("Sampler1",
                              1,                    // file_count
                              kMetricValueMax,      // metric count
                              kMetricValueMax / 2,  // expected_metric_count
                              0,                    // error_count
                              0),                   // expected_error_count
          SamplerInstructions("Sampler2",
                              1,                    // file_count
                              kMetricValueMax,      // metric count
                              kMetricValueMax / 2,  // expected_metric_count
                              0,                    // error_count
                              0),                   // expected_error_count
      }));
  tests.push_back(MultiSamplerTest(
      "two-samplers-with-zero-and-max-points",
      {
          SamplerInstructions("Sampler1",
                              1,   // file_count
                              0,   // metric count
                              0,   // expected_metric_count
                              0,   // error_count
                              0),  // expected_error_count
          SamplerInstructions("Sampler2",
                              1,                // file_count
                              kMetricValueMax,  // metric count
                              kMetricValueMax,  // expected_metric_count
                              0,                // error_count
                              0),               // expected_error_count
      }));
  tests.push_back(MultiSamplerTest(
      "two-samplers-with-few-and-over-max-points",
      {
          SamplerInstructions("Sampler1",
                              1,   // file_count
                              2,   // metric count
                              2,   // expected_metric_count
                              0,   // error_count
                              0),  // expected_error_count
          SamplerInstructions("Sampler2",
                              1,                    // file_count
                              2 * kMetricValueMax,  // metric count
                              kMetricValueMax - 2,  // expected_metric_count
                              0,                    // error_count
                              0),                   // expected_error_count
      }));

  tests.push_back(MultiSamplerTest(
      "two-samplers-with-few-errors-and-over-max-points",
      {
          SamplerInstructions("Sampler1",
                              1,   // file_count
                              2,   // metric count
                              2,   // expected_metric_count
                              2,   // error_count
                              2),  // expected_error_count
          SamplerInstructions("Sampler2",
                              1,                    // file_count
                              2 * kMetricValueMax,  // metric count
                              kMetricValueMax - 2,  // expected_metric_count
                              0,                    // error_count
                              0),                   // expected_error_count
      }));

  tests.push_back(MultiSamplerTest(
      "two-samplers-with-few-errors-and-over-max-errors",
      {
          SamplerInstructions("Sampler1",
                              1,   // file_count
                              2,   // metric count
                              2,   // expected_metric_count
                              2,   // error_count
                              2),  // expected_error_count
          SamplerInstructions(
              "Sampler2",
              1,                          // file_count
              0,                          // metric count
              0,                          // expected_metric_count
              kSampleErrorCountMax * 2,   // error_count
              kSampleErrorCountMax - 2),  // expected_error_count
      }));

  tests.push_back(MultiSamplerTest(
      "two-samplers-with-over-on-everything",
      {
          SamplerInstructions(
              "Sampler1",
              1,                          // file_count
              kMetricValueMax * 2,        // metric count
              kMetricValueMax / 2,        // expected_metric_count
              kSampleErrorCountMax * 2,   // error_count
              kSampleErrorCountMax / 2),  // expected_error_count
          SamplerInstructions(
              "Sampler2",
              1,                          // file_count
              kMetricValueMax * 2,        // metric count
              kMetricValueMax / 2,        // expected_metric_count
              kSampleErrorCountMax * 2,   // error_count
              kSampleErrorCountMax / 2),  // expected_error_count
      }));

  tests.push_back(MultiSamplerTest(
      "single-sampler-with-multiple-files",
      {
          SamplerInstructions("Sampler1",
                              2,                    // file_count
                              kMetricValueMax / 2,  // metric count
                              kMetricValueMax,      // expected_metric_count
                              kSampleErrorCountMax / 2,  // error_count
                              kSampleErrorCountMax),     // expected_error_count
      }));

  tests.push_back(MultiSamplerTest(
      "two-samplers-with-multiple-files",
      {
          SamplerInstructions("Sampler1",
                              2,       // file_count
                              2,       // metric count
                              2 * 2,   // expected_metric_count
                              2,       // error_count
                              2 * 2),  // expected_error_count
          SamplerInstructions(
              "Sampler2",
              2,                              // file_count
              kMetricValueMax,                // metric count
              kMetricValueMax - 2 * 2,        // expected_metric_count
              kSampleErrorCountMax,           // error_count
              kSampleErrorCountMax - 2 * 2),  // expected_error_count
      }));

  int expected = (kMetricValueMax - (kMetricValueMax / 20)) / 2;
  tests.push_back(MultiSamplerTest(
      "two-samplers-over-max-points-one-few-points",
      {
          SamplerInstructions("Sampler1",
                              1,                    // file_count
                              kMetricValueMax * 5,  // metric count
                              expected,             // expected_metric_count
                              0,                    // error_count
                              0),                   // expected error_count
          SamplerInstructions("Sampler2",
                              1,                    // file_count
                              kMetricValueMax * 5,  // metric count
                              expected,             // expected_metric_count
                              0,                    // error_count
                              0),                   // expected error_count
          SamplerInstructions("Sampler3",
                              1,                     // file_count
                              kMetricValueMax / 20,  // metric count
                              kMetricValueMax / 20,  // expected_metric_count
                              0,                     // error_count
                              0),                    // expected error_count
      }));

  for (const MultiSamplerTest& sampler_test : tests) {
    mako::DownsamplerOutput out;
    std::vector<std::string> files;
    LOG(INFO) << "=======================================";
    LOG(INFO) << "Test name : " << sampler_test.test_name;

    for (const SamplerInstructions& si : sampler_test.sampler_instructions) {
      std::vector<mako::SampleRecord> data;
      LOG(INFO) << " Sampler name: " << si.sampler_name;
      LOG(INFO) << " File Count: " << si.file_count;
      LOG(INFO) << " Metric count: " << si.metric_count;
      LOG(INFO) << " Expected metric count: " << si.expected_metric_count;
      LOG(INFO) << " Error count: " << si.error_count;
      LOG(INFO) << " Expected error count: " << si.expected_error_count;

      for (int i = 0; i < si.metric_count; i++) {
        // Use the sampler name as the metric name as well, to keep them
        // separate
        data.push_back(CreateSampleRecord(i, {{si.sampler_name, i}}));
      }

      for (int i = 0; i < si.error_count; i++) {
        mako::SampleRecord sr = CreateSampleRecord(i, {});
        sr.clear_sample_point();
        sr.mutable_sample_error()->set_error_message(std::to_string(i));
        sr.mutable_sample_error()->set_input_value(i);
        sr.mutable_sample_error()->set_sampler_name(si.sampler_name);
        data.push_back(sr);
      }
      for (int i = 0; i < si.file_count; ++i) {
        // Use sampler name as file name
        std::string file_name = absl::StrCat(si.sampler_name, "_", i);
        WriteFile(file_name, data);
        files.push_back(file_name);
      }
    }

    ASSERT_EQ("", d_.Downsample(CreateDownsamplerInput(files), &out));

    // NOTE: We used the name of samplers as files name as well as metric names.
    for (const SamplerInstructions& si : sampler_test.sampler_instructions) {
      ASSERT_NEAR(
          si.expected_metric_count,
          CountPointsForMetricKey(si.sampler_name, out.sample_batch_list()), 1)
          << out.DebugString();
      ASSERT_NEAR(
          si.expected_error_count,
          CountErrorsForSampler(si.sampler_name, out.sample_batch_list()), 1)
          << out.DebugString();
    }
  }
  LOG(INFO) << "=======================================";
}

// extract inputs of points containing a specific metric from the batches
std::vector<double> GetInputsWithMetric(
    const std::string& metric_name,
    const google::protobuf::RepeatedPtrField<mako::SampleBatch>& batches) {
  std::vector<mako::SamplePoint> point_list;

  DataFilter data_filter;
  data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  data_filter.set_value_key(metric_name);
  bool no_sort_data = false;
  std::vector<std::pair<double, double>> results;

  auto err_str = mako::internal::ApplyFilter(
      mako::RunInfo{}, batches.pointer_begin(), batches.pointer_end(),
      data_filter, no_sort_data, &results);

  EXPECT_EQ("", err_str);

  std::vector<double> inputs(results.size());
  std::transform(results.begin(), results.end(), inputs.begin(),
                 [](const std::pair<double, double> p) { return p.first; });

  return inputs;
}

// computes the expected standard deviation for a discrete uniform distribution
// in the range [a,b]
double expected_std_dev_uniform(int a, int b) {
  return std::sqrt((std::pow(b - a + 1, 2) - 1) / 12);
}

TEST_F(StandardMetricDownsamplerTest, DoubleMetricDownsampledDistribution) {
  mako::memory_fileio::FileIO().Clear();

  // since the downsampler is stochastic, there's a chance a degenerate
  // downsampling could produce a failing test. We make the downsampling
  // deterministic by hardcoding the seed
  ReseedDownsampler();

  mako::DownsamplerOutput out;
  std::vector<mako::SampleRecord> data;

  // uniformly distribute both metrics' input values over the
  // range [0,kMetricValueMax*2)
  const int num_points = kMetricValueMax * 2;
  for (int i = 0; i < num_points; ++i) {
    data.push_back(CreateSampleRecord(i, {{"m1", i}}));
  }
  for (int i = 0; i < num_points; ++i) {
    data.push_back(CreateSampleRecord(i, {{"m2", i}}));
  }

  WriteFile("file1", data);
  ASSERT_EQ("", d_.Downsample(CreateDownsamplerInput({"file1"}), &out));

  // note, we use (numPointsPerMetric-1) in the below two cases because our
  // discrete range is [0, numPointsPerMetric-1]
  double expected_mean = (num_points - 1) / 2.0;
  double expected_std_dev = expected_std_dev_uniform(0, num_points - 1);

  // check that downsampled m1 and m2 results' input values are distributed
  // uniformly along the same range as before
  // if we're within 10% of expected mean and standard deviation, we know
  // the downsampled results are "pretty good"
  mako::internal::RunningStats stats;
  stats.AddVector(GetInputsWithMetric("m1", out.sample_batch_list()));
  EXPECT_NEAR(expected_mean, stats.Mean().value, expected_mean * 0.1);
  EXPECT_NEAR(expected_std_dev, stats.Stddev().value, expected_std_dev * 0.1);

  // now check that downsampled m1 results are distributed evenly
  stats = mako::internal::RunningStats();
  stats.AddVector(GetInputsWithMetric("m2", out.sample_batch_list()));
  EXPECT_NEAR(expected_mean, stats.Mean().value, expected_mean * 0.1);
  EXPECT_NEAR(expected_std_dev, stats.Stddev().value, expected_std_dev * 0.1);
}

TEST_F(StandardMetricDownsamplerTest, DoubleMetricDownsampledEvenly) {
  // This test makes sure we're not "off by 1" when downsampling a
  // greater-than-max number of two different metrics. This happens when a
  // metric goes in and both metrics are consuming max/2 (their fair share of)
  // slots. When choosing which metric from which to evict we should pick the
  // incoming metric so that, once that incoming metric is inserted, we haven't
  // just unbalanced the shares.
  mako::memory_fileio::FileIO().Clear();

  // fill up our list of samples to be "full" -- both "m1" and "m2" are at their
  // fair share
  std::vector<mako::SampleRecord> recordsFull;
  for (auto metric : {"m1", "m2"}) {
    for (int i = 0; i < kMetricValueMax / 2; ++i) {
      recordsFull.push_back(CreateSampleRecord(i, {{metric, i}}));
    }
  }

  // In the case of a tie, a given implementation might be "off-by-1" by always
  // picking the "first seen" metric (in this case "m1") or by always picking
  // the "last seen" metric (in this case "m2"). We can test for both.
  for (auto metric : {"m1", "m2"}) {
    LOG(INFO) << "Testing for even downsampling after adding " << metric;
    mako::DownsamplerOutput out;

    auto records = recordsFull;
    int i = recordsFull.size();
    records.push_back(CreateSampleRecord(i, {{metric, i}}));

    std::string fileName = absl::StrCat("file", metric);
    WriteFile(fileName, records);
    ASSERT_EQ("", d_.Downsample(CreateDownsamplerInput({fileName}), &out));

    int num_m1 = GetInputsWithMetric("m1", out.sample_batch_list()).size();
    int num_m2 = GetInputsWithMetric("m2", out.sample_batch_list()).size();
    ASSERT_EQ(0, kMetricValueMax % 2) << "Make kMetricValueMax even please!";
    ASSERT_EQ(num_m1, num_m2);
  }
}

TEST_F(StandardMetricDownsamplerTest, SampleBatchSizeTest) {
  mako::KeyedValue kv;
  kv.set_value(1);         // 9 bytes:key 1;field 8
  kv.set_value_key("m1");  // 4 bytes:key 1;length 1;field 2
  constexpr int expected_kv_size = 13;
  ASSERT_EQ(expected_kv_size, kv.ByteSizeLong());

  mako::SamplePoint p;
  p.set_input_value(1);             // 9 bytes:key 1;field 8
  *p.add_metric_value_list() = kv;  // 15 bytes:key 1;length 1;field 13
  constexpr int expected_point_size = 24;
  ASSERT_EQ(expected_point_size, p.ByteSizeLong());
  (*p.mutable_aux_data())["key"] = "value";
  ASSERT_LT(expected_point_size, p.ByteSizeLong());

  mako::DownsamplerOutput out;
  mako::SampleBatch* batch = out.add_sample_batch_list();
  mako::SampleBatch* expected_batch = batch;
  constexpr int expected_empty_sample_batch_size = 0;
  ASSERT_EQ(expected_empty_sample_batch_size, batch->ByteSizeLong());

  int64_t calculated_batch_size = 0;
  auto field = batch->GetDescriptor()->FindFieldByName("sample_point_list");
  // key 1; length 1; field expected_point_size
  int expected_batch_size = expected_point_size + 2;
  int num_points = 1000 / expected_batch_size;
  for (int i = 0; i < num_points; ++i) {
    std::string err = AddBatch("benchmark", "run", 1000, field->index(), &p, &batch,
                          &calculated_batch_size, &out);
    ASSERT_EQ("", err);
    ASSERT_EQ(expected_batch, batch) << "New batch created unexpectedly.";
    ASSERT_EQ(expected_batch_size * (i + 1), calculated_batch_size);
    ASSERT_EQ(expected_batch_size * (i + 1), batch->ByteSizeLong());
  }
  std::string err = AddBatch("benchmark", "run", 1000, field->index(), &p, &batch,
                        &calculated_batch_size, &out);
  ASSERT_EQ("", err);
  ASSERT_NE(expected_batch, batch) << "New batch should have been created.";
}

TEST_F(StandardMetricDownsamplerTest, PointBiggerThanSampleBatchMaxSizeTest) {
  mako::KeyedValue kv;
  kv.set_value(1);         // 9 bytes:key 1;field 8
  kv.set_value_key("m1");  // 4 bytes:key 1;length 1;field 2
  constexpr int expected_kv_size = 13;
  ASSERT_EQ(expected_kv_size, kv.ByteSizeLong());

  mako::SamplePoint p;
  p.set_input_value(1);             // 9 bytes:key 1;field 8
  *p.add_metric_value_list() = kv;  // 15 bytes:key 1;length 1;field 13
  constexpr int expected_point_size = 24;
  ASSERT_EQ(expected_point_size, p.ByteSizeLong());

  mako::DownsamplerOutput out;
  mako::SampleBatch* batch = out.add_sample_batch_list();
  constexpr int expected_empty_sample_batch_size = 0;
  ASSERT_EQ(expected_empty_sample_batch_size, batch->ByteSizeLong());

  int64_t calculated_batch_size = 0;
  auto field = batch->GetDescriptor()->FindFieldByName("sample_point_list");
  std::string err = AddBatch("benchmark", "run", expected_point_size, field->index(),
                        &p, &batch, &calculated_batch_size, &out);
  ASSERT_NE("", err) << "Point should have been too big to put in a batch";
  ASSERT_EQ(0, batch->sample_point_list_size());
}

TEST_F(StandardMetricDownsamplerTest, DownsamplingWorstCaseTest) {
  std::vector<std::string> metrics;
  for (int i = 0; i < 1000; ++i) {
    metrics.emplace_back(absl::StrCat("m", i));
  }
  std::vector<mako::SampleRecord> records;
  for (int i = 0; i < 1000; ++i) {
    SampleRecord sr;
    SamplePoint* sp = sr.mutable_sample_point();
    sp->set_input_value(i);
    for (const auto& metric : metrics) {
      KeyedValue* kv = sp->add_metric_value_list();
      kv->set_value_key(metric);
      kv->set_value(i);
    }
    records.emplace_back(std::move(sr));
  }

  mako::DownsamplerOutput out;

  std::string fileName = "DownsamplingWorstCase";
  WriteFile(fileName, records);
  int sample_error_count_max = 5000;
  int metric_value_count_max = 50000;
  int batch_size_max = 1000000;

  ASSERT_EQ("", d_.Downsample(CreateDownsamplerInput(
                                  {fileName}, sample_error_count_max,
                                  metric_value_count_max, batch_size_max),
                              &out));
  ASSERT_LE(out.sample_batch_list_size(), 5);
  int num_sample_points = 0;
  for (const auto& batch : out.sample_batch_list()) {
    num_sample_points += batch.sample_point_list_size();
  }
  ASSERT_EQ(metric_value_count_max / 1000, num_sample_points);
}

TEST_F(StandardMetricDownsamplerTest,
       DownsamplingDownsamplesAnnotationsInSamplePoints) {
  // Write 10kb std::string
  int annotations_string_size = 10000;
  // String that we will be writing
  std::string annotation_string(annotations_string_size, 'a');
  // To fill 3 times more than max annotations size allow to
  int number_of_annotations = kMaxAnnotationsSize / annotations_string_size * 3;
  int number_of_annotations_in_a_point = 4;
  int number_of_points =
      number_of_annotations / number_of_annotations_in_a_point;

  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});
  // SampleBatch can be up to 2 times larger than the annotations.
  in.set_batch_size_max(kMaxAnnotationsSize * 2);
  // Allow big number of metrics size to fit all points.
  in.set_metric_value_count_max(number_of_annotations * 2);

  std::vector<SampleRecord> sample_records;

  for (int i = 0; i != number_of_points; i++) {
    mako::SampleRecord sb = CreateSampleRecord(i, {{"y", i}});
    mako::SamplePoint* sample_point = sb.mutable_sample_point();
    for (int j = 0; j != number_of_annotations_in_a_point; ++j) {
      sample_point->add_sample_annotations_list()->set_text(annotation_string);
    }
    sample_records.push_back(sb);
  }

  WriteFile("file1", sample_records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(in, &out));
  EXPECT_EQ(out.sample_batch_list_size(), 1);

  // Total size in bytes of all annotations. We don't need exact size with the
  // encoding overhead here to make the test very simple.
  int total_annotations_size = 0;
  for (const mako::SampleBatch& batch : out.sample_batch_list()) {
    for (const mako::SamplePoint& sample_point :
         batch.sample_point_list()) {
      // If we have annotations in this SamplePoint, then check it.
      if (sample_point.sample_annotations_list_size()) {
        // Check that all annotations in one SamplePoint were preserved
        EXPECT_EQ(number_of_annotations_in_a_point,
                  sample_point.sample_annotations_list_size());
        for (const mako::SampleAnnotation& annotation :
             sample_point.sample_annotations_list()) {
          total_annotations_size += annotation.ByteSizeLong();
        }
      }
    }
  }
  EXPECT_GT(total_annotations_size, kMaxAnnotationsSize * 0.8);
  EXPECT_LT(total_annotations_size, kMaxAnnotationsSize);
}

TEST_F(StandardMetricDownsamplerTest,
       DownsamplingRemovesAnnotationsOnOneSamplePointWithLargeAnnotations) {
  // Write 100kb strings
  int annotations_string_size = 100000;
  // String that we will be writing
  std::string annotation_string(annotations_string_size, 'a');
  // We want to exceed the limit
  int annotations_count = 1 + kMaxAnnotationsSize / annotations_string_size;

  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});
  in.set_batch_size_max(kMaxAnnotationsSize * 2);
  in.set_metric_value_count_max(10);

  std::vector<SampleRecord> sample_records;
  mako::SampleRecord sb = CreateSampleRecord(0, {{"y", 0}});
  mako::SamplePoint* sample_point = sb.mutable_sample_point();
  for (int j = 0; j != annotations_count; ++j) {
    sample_point->add_sample_annotations_list()->set_text(annotation_string);
  }
  sample_records.push_back(sb);

  WriteFile("file1", sample_records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(in, &out));
  EXPECT_EQ(out.sample_batch_list_size(), 1);

  for (const mako::SampleBatch& batch : out.sample_batch_list()) {
    for (const mako::SamplePoint& sample_point :
         batch.sample_point_list()) {
      // We should not have any annotations here
      EXPECT_FALSE(sample_point.sample_annotations_list_size());
    }
  }
}

TEST_F(
    StandardMetricDownsamplerTest,
    DownsamplingPreservesAnnotationsIfAdditionalMetricForAnnotationsIsAdded) {
  // This unit-test tests a workaround around downsampler downsampling
  // annotations. Here we have 10010 SamplePoints where only 10 points have
  // annotations. The downsampler should enforce maximum of 40 values stored.
  //
  // In usual case, each annotation is saved with a probability of 1/10010.
  // With an additional metric that is present only when we have annotations,
  // all 10 annotations should be saved.
  //
  // We will recommend this workaround for our clients that will experience
  // a significant amount of their annotations getting downsampled.
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});
  in.set_batch_size_max(10000);
  in.set_metric_value_count_max(40);

  std::vector<SampleRecord> records;
  for (int i = 0; i != 10000; ++i) {
    mako::SampleRecord sr = CreateSampleRecord(i, {{"y", i}});
    records.push_back(sr);
  }

  // Write 10 points with annotations, adding an additional metric to preserve
  // the annotations.
  //
  // We have 40 maximum metrics to be saved, 20 should be saved for {"y"}
  // metric, 20 for {"y", "a"} metric. 20 metrics for {"y", "a"} means that 10
  // records will be saved, so all of the annotations should be saved.
  //
  // Without the additional metric, each annotation has 1/10010 chance to be
  // saved.
  for (int i = 0; i != 10; ++i) {
    mako::SampleRecord sr = CreateSampleRecord(i, {{"y", i}, {"a", 0}});
    sr.mutable_sample_point()->add_sample_annotations_list()->set_text("a");
    records.push_back(sr);
  }

  WriteFile("file1", records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(in, &out));
  EXPECT_EQ(out.sample_batch_list_size(), 1);

  int annotations_after_downsampling_count = 0;
  for (const mako::SampleBatch& batch : out.sample_batch_list()) {
    for (const mako::SamplePoint& sample_point :
         batch.sample_point_list()) {
      annotations_after_downsampling_count +=
          sample_point.sample_annotations_list_size();
    }
  }
  EXPECT_EQ(annotations_after_downsampling_count, 10);
}

TEST_F(StandardMetricDownsamplerTest,
       DownsamplingDoesNotDownsampleAnnotationsWhenCloseToLimits) {
  // Maximum size + 10 bytes on encoding (should be more than enough).
  std::string annotation_string(kMaxAnnotationsSize - 10, 'a');
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});
  in.set_batch_size_max(kMaxAnnotationsSize * 2);
  in.set_metric_value_count_max(10);

  std::vector<SampleRecord> sample_records;

  mako::SampleRecord sb = CreateSampleRecord(0, {{"y", 0}});
  mako::SamplePoint* sample_point = sb.mutable_sample_point();
  sample_point->add_sample_annotations_list()->set_text(annotation_string);
  sample_records.push_back(sb);

  WriteFile("file1", sample_records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(in, &out));
  EXPECT_EQ(out.sample_batch_list_size(), 1);

  int total_annotations_count = 0;
  for (const mako::SampleBatch& batch : out.sample_batch_list()) {
    for (const mako::SamplePoint& sample_point :
         batch.sample_point_list()) {
      total_annotations_count += sample_point.sample_annotations_list_size();
    }
  }
  EXPECT_EQ(total_annotations_count, 1);
}

TEST_F(StandardMetricDownsamplerTest,
       DownsamplingDownsamplesAnnotationsWhenStringOfExactlyMaxSize) {
  // Maximum size + encoding should not fit.
  std::string annotation_string(kMaxAnnotationsSize, 'a');
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});
  in.set_batch_size_max(kMaxAnnotationsSize * 2);
  in.set_metric_value_count_max(10);

  std::vector<SampleRecord> sample_records;

  mako::SampleRecord sb = CreateSampleRecord(0, {{"y", 0}});
  mako::SamplePoint* sample_point = sb.mutable_sample_point();
  sample_point->add_sample_annotations_list()->set_text(annotation_string);
  sample_records.push_back(sb);

  WriteFile("file1", sample_records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(in, &out));
  EXPECT_EQ(out.sample_batch_list_size(), 1);

  int total_annotations_count = 0;
  for (const mako::SampleBatch& batch : out.sample_batch_list()) {
    for (const mako::SamplePoint& sample_point :
         batch.sample_point_list()) {
      total_annotations_count += sample_point.sample_annotations_list_size();
    }
  }
  EXPECT_EQ(total_annotations_count, 0);
}

TEST_F(StandardMetricDownsamplerTest, AuxDataIsRemovedTest) {
  mako::DownsamplerInput in = CreateDownsamplerInput({"file1"});

  // Set a small batch size so we get lots of batches.
  in.set_batch_size_max(3000);

  std::vector<SampleRecord> sample_records;

  for (int i = 0; i < 200; i++) {
    mako::SampleRecord sb =
        CreateSampleRecord(i, {{"y", i}}, {{"key1", "value1"}});
    sb.mutable_sample_error()->set_error_message("An Error message");
    sb.mutable_sample_error()->set_input_value(i);
    sb.mutable_sample_error()->set_sampler_name(absl::StrCat("Sampler", i));
    sample_records.push_back(sb);
    ASSERT_GT(200, sb.ByteSizeLong());
    ASSERT_EQ(sb.sample_point().aux_data_size(), 1);
  }

  WriteFile("file1", sample_records);
  mako::DownsamplerOutput out;
  ASSERT_EQ("", d_.Downsample(in, &out));
  ASSERT_GT(out.sample_batch_list_size(), 1);
  for (auto& batch : out.sample_batch_list()) {
    for (auto& point : batch.sample_point_list()) {
      EXPECT_EQ(point.aux_data_size(), 0);
    }
  }
}

}  // namespace downsampler
}  // namespace mako
