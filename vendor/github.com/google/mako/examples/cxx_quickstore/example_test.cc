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
#include "glog/logging.h"
#include "src/google/protobuf/text_format.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/proto/analyzers/threshold_analyzer.pb.h"
#include "absl/flags/parse.h"
#include "clients/cxx/analyzers/threshold_analyzer.h"
#include "helpers/cxx/status/canonical_errors.h"
#include "helpers/cxx/status/status_matchers.h"
#include "helpers/cxx/status/statusor.h"
#include "examples/cxx_quickstore/perf_data.pb.h"
#include "quickstore/cxx/quickstore.h"
#include "quickstore/quickstore.pb.h"
#include "spec/proto/mako.pb.h"

namespace {

constexpr char kBenchmarkKey[] = "6500986839367680";

mako::helpers::StatusOr<cxx_quickstore::PerfData> ReadData() {
  const std::string perf_data_string = R"proto(
    counters: {
      throughput: 4312.84
      branch_miss_percentage: 1.048
      page_faults: 42383
    }
    samples: { timestamp: 1316713628128 write_latency: 258 cpu_load: 0.13 }
    samples: { timestamp: 1316713628386 read_latency: 737 cpu_load: 0.19 }
    samples: { timestamp: 1316713629123 write_latency: 1256 cpu_load: 0.28 }
    samples: { timestamp: 1316713870379 read_latency: 455 cpu_load: 0.34 }
    samples: { timestamp: 1316713870834 write_latency: 279 cpu_load: 0.38 }
    samples: { timestamp: 1316713871834 read_latency: 383 cpu_load: 0.51 }
    metadata: {
      git_hash: "43bf39d0a2c36583aed1c346a2e5b958ab379718"
      timestamp_ms: 1316713628128
      run_duration_ms: 2438
      tags: "environment=continuous_integration"
      tags: "branch=master"
    }
  )proto";
  cxx_quickstore::PerfData perf_data;
  if (google::protobuf::TextFormat::ParseFromString(perf_data_string, &perf_data)) {
  return perf_data;
  }
  return mako::helpers::InvalidArgumentError(
      "Failed parsing PerfData proto.");
}

TEST(MakoExample, PerformanceTest) {
  // STEP 1: Collect some performance data. Here we read some data from a
  // serialized format.
  //
  // Read more about the Mako data format at
  // http://github.com/google/mako/blob/master/docs/GUIDE.md#preparing-your-performance-test-data.
  auto status_or_data = ReadData();
  ASSERT_OK(status_or_data);
  cxx_quickstore::PerfData data = std::move(status_or_data).value();

  // STEP 2: Configure run metadata in QuickstoreInput.
  //
  // Read about the run metadata you can set in QuickstoreInput at
  // http://github.com/google/mako/blob/master/docs/GUIDE.md#run-metadata.
  mako::quickstore::QuickstoreInput quickstore_input;
  quickstore_input.set_benchmark_key(kBenchmarkKey);
  quickstore_input.set_duration_time_ms(data.metadata().run_duration_ms());
  quickstore_input.set_timestamp_ms(data.metadata().timestamp_ms());
  quickstore_input.set_hover_text(data.metadata().git_hash());
  quickstore_input.mutable_tags()->CopyFrom(data.metadata().tags());

  // STEP 3: Configure an Analyzer
  //
  // Read more about analyzers at
  // https://github.com/google/mako/blob/master/docs/ANALYZERS.md
  mako::analyzers::threshold_analyzer::ThresholdAnalyzerInput*
      analyzer_input = quickstore_input.add_threshold_inputs();

  // Threshold on a metric aggregate (median of WriteLatency)
  mako::analyzers::threshold_analyzer::ThresholdConfig* config =
      analyzer_input->add_configs();
  config->set_config_name("writes_lt_900");
  config->set_max(900);
  mako::DataFilter* data_filter = config->mutable_data_filter();
  data_filter->set_data_type(mako::DataFilter::METRIC_AGGREGATE_MEDIAN);
  data_filter->set_value_key("wl");

  // Threshold on a custom aggregate (Throughput).
  config = analyzer_input->add_configs();
  config->set_config_name("throughput_gt_400");
  config->set_min(4000);
  data_filter = config->mutable_data_filter();
  data_filter->set_data_type(mako::DataFilter::CUSTOM_AGGREGATE);
  data_filter->set_value_key("tp");

  // STEP 4: Create a Quickstore instance which reports to the Mako server
  //
  // Read about setting up authentication at
  // http://github.com/google/mako/blob/master/docs/GUIDE.md#setting-up-authentication
  mako::quickstore::Quickstore quickstore(quickstore_input);

  // STEP 5: Feed your sample point data to the Mako Quickstore client.
  for (const auto& sample : data.samples()) {
    std::map<std::string, double> metrics = {{"cpu", sample.cpu_load()}};
    if (sample.has_read_latency()) {
      metrics["rl"] = sample.read_latency();
    }
    if (sample.has_write_latency()) {
      metrics["wl"] = sample.write_latency();
    }
    quickstore.AddSamplePoint(sample.timestamp(), metrics);
  }

  // STEP 6: Feed your custom aggregate data to the Mako Quickstore client.
  quickstore.AddRunAggregate("tp", data.counters().throughput());
  quickstore.AddRunAggregate("bm", data.counters().branch_miss_percentage());
  quickstore.AddRunAggregate("pf", data.counters().page_faults());

  // STEP 7: Call Store() to instruct Mako to process the data and upload it to
  // http://mako.dev.
  mako::quickstore::QuickstoreOutput output = quickstore.Store();
  switch (output.status()) {
    case mako::quickstore::QuickstoreOutput::SUCCESS:
      LOG(INFO) << " Done! Run can be found at: " << output.run_chart_link();
      break;
    case mako::quickstore::QuickstoreOutput::ERROR:
      FAIL() << "Quickstore Store error: " << output.summary_output();
      break;
    case mako::quickstore::QuickstoreOutput::ANALYSIS_FAIL:
      FAIL() << "Quickstore Analysis Failure: " << output.summary_output()
                 << "\nRun can be found at: " << output.run_chart_link();
      break;
  }
}

}  // namespace
