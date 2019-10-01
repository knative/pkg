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
#include "clients/cxx/analyzers/utest_analyzer.h"

#include <algorithm>
#include <cmath>
#include <memory>
#include <set>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

#include "glog/logging.h"
#include "src/google/protobuf/map.h"
#include "src/google/protobuf/repeated_field.h"
#include "src/google/protobuf/text_format.h"
#include "clients/proto/analyzers/utest_analyzer.pb.h"
#include "spec/proto/mako.pb.h"
#include "absl/container/flat_hash_set.h"
#include "absl/strings/str_cat.h"
#include "clients/cxx/analyzers/util.h"
#include "internal/cxx/filter_utils.h"
#include "internal/cxx/pgmath.h"
#include "spec/proto/mako.pb.h"

namespace mako {
namespace utest_analyzer {

namespace {

// Keys for A/B sample mappings.
constexpr char kSampleAKey[] = "a";
constexpr char kSampleBKey[] = "b";

// SampleIndex members are used for StatsCalculator::sample_ array indexing
enum SampleIndex { kSampleA, kSampleB };

std::pair<DataFilter, DataFilter> ConfigToDataFilters(
    const UTestConfig& config) {
  std::pair<DataFilter, DataFilter> result;
  if (config.has_a_metric_key()) {
    DataFilter& data_filter = result.first;
    data_filter.set_value_key(config.a_metric_key());
    data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  } else {
    result.first = config.a_data_filter();
  }

  if (config.has_b_metric_key()) {
    DataFilter& data_filter = result.second;
    data_filter.set_value_key(config.b_metric_key());
    data_filter.set_data_type(mako::DataFilter::METRIC_SAMPLEPOINTS);
  } else {
    result.second = config.b_data_filter();
  }
  return result;
}

double TiesCorrectionFactor(const std::unordered_map<int, int>& tie_counts,
                            int n) {
  // Based on Nonparametric Statistics for Non-Statisticians: A Step-By-Step
  // Approach By Gregory W. Corder, Dale I. Foreman. pp.100-101
  // c = 1 - sum(t^3 - t)/(n^3 - n)
  // where t is the number of values from a set of ties, and n is the total
  // sample size
  double correction = 0;
  for (auto it = tie_counts.begin(); it != tie_counts.end(); ++it) {
    const double tie_size = it->first;
    const double num_times = it->second;
    correction += num_times * (std::pow(tie_size, 3) - tie_size);
  }
  correction /= std::pow(n, 3) - n;
  return 1.0 - correction;
}

}  // namespace

class StatsCalculator {
 public:
  StatsCalculator(const UTestAnalyzerInput& config,
                  const AnalyzerInput& analyzer_input);

  struct Statistics {
    // Automatically initialize all members to 0
    Statistics()
        : count_a(0),
          count_b(0),
          rank_a(0),
          rank_b(0),
          median_a(std::nan("")),
          median_b(std::nan("")) {}

    // The number of A samples.
    int count_a;

    // The number of B samples.
    int count_b;

    // The sum of the ranks of A samples.
    double rank_a;

    // The sum of the ranks of B samples.
    double rank_b;

    // The median of A samples.
    double median_a;

    // The median of B samples.
    double median_b;

    // Keep track of how many times we encounter a tie among N values.
    // tie_counts[X] is the number of times we saw a tie between X values
    // (whatever that value may have been).
    std::unordered_map<int, int> tie_counts;
  };

  // Return stats for a given absolute shift value.
  StatsCalculator::Statistics GetStatsShiftValue(const std::string& a_metric_key,
                                                 const std::string& b_metric_key,
                                                 double shift_value) const;

  // Return stats for a given relative shift value.
  StatsCalculator::Statistics GetStatsRelativeShiftValue(
      const std::string& a_metric_key, const std::string& b_metric_key,
      double relative_shift_value) const;

 private:
  struct Sample {
    // Run keys for runs we want to include in this sample
    std::unordered_set<std::string> run_key_list;

    // Map from a metric_key to all its associated raw data
    std::unordered_map<std::string, std::vector<double>> sample_data;

    // Map from a metric_key to which data filter we are using to fetch raw data
    // from each run.
    std::unordered_map<std::string, mako::DataFilter> sample_data_filter;
  };

  // Add relevant data points from the bundle into extracted_data
  void AddBundleData(const RunBundle& data_bundle, const SampleIndex s_index);

  // Return stats for two samples with a given absolute shift value.
  StatsCalculator::Statistics GetStats(const std::vector<double>& sample_a,
                                       const std::vector<double>& sample_b,
                                       double shift_value) const;

  // Sample structs for samples A and B
  Sample sample_[2];
};

StatsCalculator::StatsCalculator(const UTestAnalyzerInput& config,
                                 const AnalyzerInput& analyzer_input) {
  std::string current_run_key =
      analyzer_input.run_to_be_analyzed().run_info().run_key();

  // Record which metric keys we will need for our analysis
  for (const UTestConfig& t_config : config.config_list()) {
    std::pair<DataFilter, DataFilter> filters = ConfigToDataFilters(t_config);
    sample_[kSampleA].sample_data_filter[filters.first.value_key()] =
        filters.first;
    sample_[kSampleA].sample_data[filters.first.value_key()];

    sample_[kSampleB].sample_data_filter[filters.second.value_key()] =
        filters.second;
    sample_[kSampleB].sample_data[filters.second.value_key()];
  }

  if (config.a_sample().include_current_run()) {
    sample_[kSampleA].run_key_list.insert(current_run_key);
    AddBundleData(analyzer_input.run_to_be_analyzed(), kSampleA);
  }
  if (config.b_sample().include_current_run()) {
    sample_[kSampleB].run_key_list.insert(current_run_key);
    AddBundleData(analyzer_input.run_to_be_analyzed(), kSampleB);
  }

  // Extract data from RunInfoQuery A/B-sample results
  if (analyzer_input.historical_run_map_size() > 0) {
    const auto& run_map = analyzer_input.historical_run_map();

    auto it = run_map.find(kSampleAKey);
    if (it != run_map.end()) {
      for (const RunBundle& bundle : it->second.historical_run_list()) {
        sample_[kSampleA].run_key_list.insert(bundle.run_info().run_key());
        AddBundleData(bundle, kSampleA);
      }
    }
    it = run_map.find(kSampleBKey);
    if (it != run_map.end()) {
      for (const RunBundle& bundle : it->second.historical_run_list()) {
        sample_[kSampleB].run_key_list.insert(bundle.run_info().run_key());
        AddBundleData(bundle, kSampleB);
      }
    }
  } else if (analyzer_input.historical_run_list_size() > 0) {
    // TODO(b/132447974): Deprecate after migrating users who run U-Test
    // analyzers by inserting the run bundles in list rather than in map.
    for (const RunInfoQuery& query : config.a_sample().run_query_list()) {
      sample_[kSampleA].run_key_list.insert(query.run_key());
    }
    for (const RunInfoQuery& query : config.b_sample().run_query_list()) {
      sample_[kSampleB].run_key_list.insert(query.run_key());
    }

    for (const RunBundle& bundle : analyzer_input.historical_run_list()) {
      bool aSampleBundle =
          sample_[kSampleA].run_key_list.find(bundle.run_info().run_key()) !=
          sample_[kSampleA].run_key_list.end();
      bool bSampleBundle =
          sample_[kSampleB].run_key_list.find(bundle.run_info().run_key()) !=
          sample_[kSampleB].run_key_list.end();
      if (!aSampleBundle && !bSampleBundle) {
        LOG(WARNING)
            << "A RunInfoQuery was defined without a run key in the "
               "UTestConfig, but only run keys are supported when using the "
               "historical run list. Consider using the historical run map.";
      }
      if (aSampleBundle) {
        AddBundleData(bundle, kSampleA);
      }
      if (bSampleBundle) {
        AddBundleData(bundle, kSampleB);
      }
    }
  }

  // Sort extracted data
  for (int s_index : {kSampleA, kSampleB}) {
    for (auto& data : sample_[s_index].sample_data) {
      std::sort(data.second.begin(), data.second.end());
    }
  }
}

// Adds data from RunBundle to the appropriate samples
void StatsCalculator::AddBundleData(const RunBundle& data_bundle,
                                    const SampleIndex s_index) {
  for (auto it : sample_[s_index].sample_data_filter) {
    const std::string& value_key = it.first;
    const auto& data_filter = it.second;
    std::vector<std::pair<double, double>> results;
    std::string err = mako::internal::ApplyFilter(
        data_bundle.run_info(), data_bundle.batch_list().pointer_begin(),
        data_bundle.batch_list().pointer_end(), data_filter, false, &results);
    if (!err.empty()) {
      LOG(ERROR) << absl::StrCat("Run data extraction failed for run_key(",
                                 data_bundle.run_info().run_key(), "): ", err);
      continue;
    }
    for (const auto& result : results) {
      sample_[s_index].sample_data[value_key].push_back(result.second);
    }
  }
}

StatsCalculator::Statistics StatsCalculator::GetStatsShiftValue(
    const std::string& a_metric_key, const std::string& b_metric_key,
    double shift_value) const {
  // Assumes both samples are sorted in ascending order
  const std::vector<double>& samp_a =
      sample_[kSampleA].sample_data.at(a_metric_key);
  const std::vector<double>& samp_b =
      sample_[kSampleB].sample_data.at(b_metric_key);

  return GetStats(samp_a, samp_b, shift_value);
}

StatsCalculator::Statistics StatsCalculator::GetStatsRelativeShiftValue(
    const std::string& a_metric_key, const std::string& b_metric_key,
    double relative_shift_value) const {
  // Assumes both samples are sorted in ascending order
  const std::vector<double>& samp_a =
      sample_[kSampleA].sample_data.at(a_metric_key);
  const std::vector<double>& samp_b =
      sample_[kSampleB].sample_data.at(b_metric_key);

  mako::internal::RunningStats stats;
  for (const double value : samp_a) {
    stats.Add(value);
  }

  const double shift_value = stats.Mean().value * relative_shift_value;

  return GetStats(samp_a, samp_b, shift_value);
}

StatsCalculator::Statistics StatsCalculator::GetStats(
    const std::vector<double>& sample_a, const std::vector<double>& sample_b,
    double shift_value) const {
  // Calculate the "rank" for both samples
  double rank_a = 0;
  double rank_b = 0;

  int i = 1;

  // Keep track of how many times we encounter a tie among N values.
  // tie_counts[X] is the number of times we saw a tie between X values
  // (whatever that value may have been). We don't actually need the values
  // themselves for our calculations.
  std::unordered_map<int, int> tie_counts;

  // Iterate through both samples
  auto iter_a = sample_a.begin();
  auto end_a = sample_a.end();
  auto iter_b = sample_b.begin();
  auto end_b = sample_b.end();
  while (iter_a != end_a && iter_b != end_b) {
    if (*iter_a + shift_value == *iter_b) {
      // Find all values in both samples with this current value
      double curr = *iter_b;

      iter_a++;
      iter_b++;
      int temp_rank = 2 * i + 1;
      i += 2;

      int a_occurrences = 1;
      int b_occurrences = 1;
      while (iter_a != end_a && *iter_a + shift_value == curr) {
        iter_a++;
        temp_rank += i;
        i++;
        a_occurrences++;
      }
      while (iter_b != end_b && *iter_b == curr) {
        iter_b++;
        temp_rank += i;
        i++;
        b_occurrences++;
      }

      tie_counts[a_occurrences + b_occurrences]++;

      // Compute the average rank for this value
      double val_rank =
          static_cast<double>(temp_rank) / (a_occurrences + b_occurrences);

      // Add the appropriate amount of rank to both samples depending on
      // how many times the current value was present in the sample
      rank_a += val_rank * (a_occurrences);
      rank_b += val_rank * (b_occurrences);

    } else if (*iter_a + shift_value < *iter_b) {
      // Add rank to sample A since it contains a value which is less
      // than the current and any future B sample value
      rank_a += i;
      iter_a++;
      i++;
    } else {
      // Add rank to sample B since it contains a value which is less
      // than the current and any future A sample value
      rank_b += i;
      iter_b++;
      i++;
    }
  }

  // If one sample is larger than the other, we may exit the loop without
  // iterating through all the values. We must compute the remaining rank
  // (total_rank (calculated the nth triangular#) - rank_used (rank_a + rank_b))
  // and add this value to the rank of the longer sample
  int64_t n = sample_a.size() + sample_b.size();
  int64_t total_rank = n * (n + 1) / 2;
  double rem_rank = total_rank - rank_a - rank_b;

  if (iter_a != end_a) {
    rank_a += rem_rank;
  }
  if (iter_b != end_b) {
    rank_b += rem_rank;
  }

  // Iterate through each entire sample set to determine other stats (median).
  mako::internal::RunningStats a_stats;
  mako::internal::RunningStats b_stats;
  for (auto a : sample_a) {
    a_stats.Add(a + shift_value);
  }
  for (auto b : sample_b) {
    b_stats.Add(b);
  }

  // Return relevant data
  StatsCalculator::Statistics stats;
  stats.count_a = sample_a.size();
  stats.count_b = sample_b.size();
  stats.rank_a = rank_a;
  stats.rank_b = rank_b;
  stats.tie_counts = tie_counts;

  mako::internal::RunningStats::Result result;
  result = a_stats.Median();
  if (result.error.empty()) {
    stats.median_a = result.value;
  }
  result = b_stats.Median();
  if (result.error.empty()) {
    stats.median_b = result.value;
  }

  return stats;
}

// Users can choose to consult the map or list for the queries. The
// Analyzer Optimizer prefers run queries from the map so that it knows which
// A/B sample each query belongs to.
bool Analyzer::ConstructHistoricQuery(const AnalyzerHistoricQueryInput& input,
                                      AnalyzerHistoricQueryOutput* output) {
  output->Clear();
  output->set_get_batches(true);
  output->mutable_status()->set_code(Status_Code_SUCCESS);

  const std::string& benchmark_key = input.benchmark_info().benchmark_key();

  // Access mutable map.
  auto& query_map = *output->mutable_run_info_query_map();

  // Add A-sample queries to map and run keys to set.
  for (const auto& query : config_.a_sample().run_query_list()) {
    RunInfoQuery copy = query;
    copy.set_benchmark_key(benchmark_key);
    *query_map[kSampleAKey].add_run_info_query_list() = copy;
    *output->add_run_info_query_list() = copy;
  }

  // Add B-sample queries to map and run keys to set.
  for (const auto& query : config_.b_sample().run_query_list()) {
    RunInfoQuery copy = query;
    copy.set_benchmark_key(benchmark_key);
    *query_map[kSampleBKey].add_run_info_query_list() = copy;
    *output->add_run_info_query_list() = copy;
  }

  return true;
}

bool Analyzer::DoAnalyze(const AnalyzerInput& analyzer_input,
                         AnalyzerOutput* analyzer_output) {
  LOG(INFO) << "START: UTest Analyzer";

  // Save the analyzer configuration.
  google::protobuf::TextFormat::PrintToString(config_,
                                    analyzer_output->mutable_input_config());

  UTestAnalyzerOutput config_out;

  // Validate config - passed to constructor, but we can't return error there
  std::string err = ValidateUTestAnalyzerInput();
  if (!err.empty()) {
    err = absl::StrCat("Bad UTestAnalyzerInput provided to constructor: ", err);
    SetAnalyzerOutputWithError(analyzer_output, &config_out, err);
    LOG(INFO) << "END: UTest Analyzer";
    return false;
  }

  if (!analyzer_input.has_run_to_be_analyzed()) {
    SetAnalyzerOutputWithError(analyzer_output, &config_out,
                               "AnalyzerInput missing run_to_be_analyzed.");
    LOG(INFO) << "END: UTest Analyzer";
    return false;
  }

  if (!analyzer_input.run_to_be_analyzed().has_run_info()) {
    SetAnalyzerOutputWithError(analyzer_output, &config_out,
                               "RunBundle missing run_info.");
    LOG(INFO) << "END: UTest Analyzer";
    return false;
  }

  if (analyzer_input.historical_run_map_size() > 0 &&
      analyzer_input.historical_run_list_size() > 0) {
    SetAnalyzerOutputWithError(analyzer_output, &config_out,
                               "AnalyzerInput run map and run list are both "
                               "nonempty. Only one can be used.");
    LOG(INFO) << "END: UTest Analyzer";
    return false;
  }

  // Compute all the possible stats that we may need in our calculations
  StatsCalculator s_calc(config_, analyzer_input);
  bool overall_regression_found = false;
  std::vector<std::string> failed_config_names;
  int index = 0;

  // Loop through all configs and run the U-Test to determine
  // if a regression is present
  for (auto& config : config_.config_list()) {
    std::string config_name =
        (config.has_config_name() ? config.config_name()
                                  : absl::StrCat("config_list[", index, "]"));
    UTestConfigResult* result = config_out.add_config_result_list();
    std::pair<DataFilter, DataFilter> filters = ConfigToDataFilters(config);
    result->set_a_metric_label(
        mako::analyzer_util::GetHumanFriendlyDataFilterString(
            filters.first,
            analyzer_input.run_to_be_analyzed().benchmark_info()));
    result->set_b_metric_label(
        mako::analyzer_util::GetHumanFriendlyDataFilterString(
            filters.second,
            analyzer_input.run_to_be_analyzed().benchmark_info()));
    std::string err = AnalyzeUTestConfig(config_name, s_calc, config, result);
    if (!err.empty()) {
      SetAnalyzerOutputWithError(analyzer_output, &config_out, err);
      LOG(INFO) << "END: UTest Analyzer";
      return false;
    }
    if (result->regression_found()) {
      failed_config_names.push_back(config_name);
      overall_regression_found = true;
    }
    index++;
  }

  LOG(INFO) << "Overall regression found: "
            << (overall_regression_found ? "Yes" : "No");

  if (overall_regression_found) {
    SetAnalyzerOutputWithRegression(analyzer_output, &config_out,
                                    failed_config_names);
  } else {
    SetAnalyzerOutputPassing(analyzer_output, &config_out);
  }

  LOG(INFO) << "END: UTest Analyzer";
  return true;
}

std::string Analyzer::AnalyzeUTestConfig(const std::string& config_name,
                                    const StatsCalculator& s_calc,
                                    const UTestConfig& config,
                                    UTestConfigResult* result) {
  std::stringstream check_output;
  check_output << "Start u-test analysis for: " << config_name << ".\n";

  const std::string a_value_key = config.has_a_metric_key()
                                 ? config.a_metric_key()
                                 : config.a_data_filter().value_key();
  const std::string b_value_key = config.has_b_metric_key()
                                 ? config.b_metric_key()
                                 : config.b_data_filter().value_key();

  *result->mutable_config() = config;
  if (config.has_config_name()) {
    result->set_config_name(config.config_name());
  }

  // Get the appropriate sample statistics
  StatsCalculator::Statistics unshifted_stats = s_calc.GetStatsShiftValue(
      a_value_key, b_value_key, 0.0 /* shift_value */);
  StatsCalculator::Statistics stats;
  if (config.has_relative_shift_value()) {
    stats = s_calc.GetStatsRelativeShiftValue(a_value_key, b_value_key,
                                              config.relative_shift_value());
  } else if (config.has_shift_value()) {
    stats = s_calc.GetStatsShiftValue(a_value_key, b_value_key,
                                      config.shift_value());
  } else {
    stats = unshifted_stats;
  }

  if (!std::isnan(unshifted_stats.median_a)) {
    result->set_a_median(unshifted_stats.median_a);
  }
  if (!std::isnan(unshifted_stats.median_b)) {
    result->set_b_median(unshifted_stats.median_b);
  }

  // If sample sizes are less than 3, report an error and exit the analyzer
  if (stats.count_a < 3) {
    return absl::StrCat(config_name,
                        "At least 3 points are required in a sample to run "
                        "U-Test but Sample A has ",
                        stats.count_a, " data points");
  } else if (stats.count_b < 3) {
    return absl::StrCat(config_name,
                        "At least 3 points are required in a sample to run "
                        "U-Test but Sample B has ",
                        stats.count_b, " data points");
  }

  check_output << "sample A count: " << stats.count_a << "\n"
               << "sample A rank: " << stats.rank_a << "\n"
               << "sample B count: " << stats.count_b << "\n"
               << "sample B rank: " << stats.rank_b << "\n";

  // If the sample sizes are less than 20, our assumption for u-tests is not
  // met. Log a warning.
  if (stats.count_a < 20 || stats.count_b < 20) {
    LOG(WARNING) << "A UTestSample has less than 20 data points. Z-statistic "
                    "approximation will not be accurate. Results may be "
                    "unreliable.";
  }

  check_output << "shift value: " << config.shift_value() << "\n";

  // Cast to 64 bit int to avoid integer overflow issues
  double n1n2 = static_cast<int64_t>(stats.count_a) * stats.count_b;

  // Compute U-statistics for each sample.
  double u_stat_a = stats.rank_a - (0.5 * stats.count_a * (stats.count_a + 1));
  double u_stat_b = n1n2 - u_stat_a;

  // For a two-sided (aka two-tailed) test (NO_BIAS), it doesn't really matter
  // whether we pick sample A's or sample B's U-statistic, because they're
  // equidistant from the mean (n1n2/2) and being too far in either direction is
  // a regression. But for a one-sided test we have to make sure we pick a
  // "side" that matches up with the rejection region we compare it to.
  // We choose to pick the "bigger" U (the U that would be larger in the case
  // of a regression) in all cases and do a right-tailed test.
  //
  //                          XXXXXXXXXX
  //                      XXXX    ++    XXX
  //                    XXX       |        XX
  //     Little U+---->X+         |          X<------------+Big U
  //                 XX |         |          +X
  //                XX  |         |          |XX
  //               XX   |         |          | XX
  //               X    |         |          |  X
  //               X    |         |          |  X
  //              XX    |         |          |  XX
  //              X     |         |          |  ++     One-tailed
  //             XX     |         |          |  |XX    rejection region
  //            XX      |         |          |  |XXX        +
  //          XXX       |         |          |  |XXXXXXX    |
  //       XXXX         |         |          |  |XXXXXXXXXXX|XX
  // XXXXXXX            |         |          |  |XXXXXXXXXXXvXXXXXXXXX
  // X+-----------------+---------+----------+--+XXXXXXXXXXXXXXXXXXX+XXXXX

  double u_statistic;
  switch (config.direction_bias()) {
    case UTestConfig_DirectionBias_NO_BIAS:
      u_statistic = std::max(u_stat_a, u_stat_b);
      break;
    case UTestConfig_DirectionBias_IGNORE_INCREASE:
      u_statistic = u_stat_a;
      break;
    case UTestConfig_DirectionBias_IGNORE_DECREASE:
      u_statistic = u_stat_b;
      break;
  }

  check_output << "u-statistic: " << u_statistic << "\n";

  // Since we know our samples are relatively large, we know that U is
  // approximately normally distributed. This means we can approximate the
  // Mann-Whitney U distribution with a normal distribution. First we compute
  // the z-statistic in the typical way ((U - mean_U) / stddev_U).

  // Calculate the z-statistic based on the u-statistic
  double z_std_dev = n1n2 * (stats.count_a + stats.count_b + 1) / 12;

  if (stats.tie_counts[stats.count_a + stats.count_b] == 1) {
    // All ranks were tied, meaning all values were the same. This degenerate
    // case is a trivial non-regression.
    result->set_z_statistic(0);
    result->set_p_value(1.0);
    result->set_regression_found(false);
    return "";
  }

  // Introduce a correction factor due to possible the presence of ties
  // (duplicate values in the dataset).
  double correction =
      TiesCorrectionFactor(stats.tie_counts, stats.count_a + stats.count_b);
  CHECK_NE(correction, 0.0)
      << "Internal mako error: utest correction==0.0 should not be "
      << "possible unless all ranks were tied, which should have been detected "
      << "in the block above.";
  z_std_dev *= correction;
  z_std_dev = std::sqrt(z_std_dev);

  // Now that we have the mean (n1*n2)/2 and the standard deviation of U, we
  // can calculate the z-statistic.
  // We subtract 0.5 here as a correction for continuity, to account for the
  // fact that we approximate a discrete distribution (U is calculated from
  // discrete ranks) via a continuous distribution (Z being the Normal/Gaussian
  // distribution).
  double z_statistic = (u_statistic - (0.5 * n1n2) - 0.5) / z_std_dev;

  check_output << "z-statistic: " << z_statistic << "\n";

  // CDF for the z-distribution. We will use this to turn our z-statistic value
  // into an area under the curve to the left of the z-statistic.
  //
  //                           XXXXXXXX
  //                        XXXXXXXXXXXXX
  //                     XXXXXXXXXXXXXXXXXX
  //            CDF    XXXXXXXXXXXXXXXXXXXXX
  //             +    XXXXXXXXXXXXXXXXXXXXXX
  //             |    XXXXXXXXXXXXXXXXXXXXXXX
  //             +----->XXXXXXXXXXXXXXXXXXXXX
  //                 XXXXXXXXXXXXXXXXXXXXXXXX  +z statistic
  //                XXXXXXXXXXXXXXXXXXXXXXXXX <+
  //               XXXXXXXXXXXXXXXXXXXXXXXXXX
  //                XXXXXXXXXXXXXXXXXXXXXXXX|X
  //               XXXXXXXXXXXXXXXXXXXXXXXXX| XX
  //              X XXXXXXXXXXXXXXXXXXXXXXXX|  XX
  //            XXXXXXXXXXXXXXXXXXXXXXXXXXXX|   XXX
  //         XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX|     XXXXX
  //     XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX|         XXXXXXXXXX
  // XXXXX-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX+------------------XXX

  // F(x) = 1/2pi * integral_-inf_to_z(e^((-z^2)/2) dz)
  //      = (erf(x/sqrt(2)) + 1) / 2
  //      = erfc(-x/sqrt(2))/2
  double cdf = std::erfc(-z_statistic / std::sqrt(2)) / 2;

  double p_value = 0;

  // We now easily compute the p value (size of the rejection region).
  if (config.direction_bias() == UTestConfig_DirectionBias_NO_BIAS) {
    p_value = 2 * (cdf > 0.5 ? 1 - cdf : cdf);
  } else {
    p_value = 1 - cdf;
  }

  check_output << "significance level: " << config.significance_level() << "\n";
  check_output << "p-value: " << p_value << "\n";

  // Check if the p-value was statistically significant
  bool regression = p_value < config.significance_level();

  check_output << "regression found: " << (regression ? "Yes" : "No") << "\n";

  // Configure UTestConfigResult
  result->set_z_statistic(z_statistic);
  result->set_p_value(p_value);
  result->set_regression_found(regression);

  check_output << "Completed u-test analysis for: " << config_name << ".\n";
  LOG(INFO) << check_output.str();
  return "";
}

std::string Analyzer::ValidateUTestAnalyzerInput() const {
  // Validate user input protobuf
  std::stringstream err;
  int i = 0;

  if (config_.config_list().empty()) {
    err << "Must include at least one UTestConfig in UTestAnalyzerInput\n";
  }

  if (!config_.has_a_sample()) {
    err << "UTestAnalyzerInput needs to specify a_sample\n";
  } else {
    if (config_.a_sample().run_query_list().empty() &&
        !config_.a_sample().include_current_run()) {
      err << "a_sample must contain at least 1 run key in run_query_list or "
             "must have include_current_run as true.\n";
    }
  }

  if (!config_.has_b_sample()) {
    err << "UTestAnalyzerInput needs to specify b_sample\n";
  } else {
    if (config_.b_sample().run_query_list().empty() &&
        !config_.b_sample().include_current_run()) {
      err << "b_sample must contain at least 1 run key in run_query_list or "
             "must have include_current_run as true.\n";
    }
  }

  for (const UTestConfig& t_config : config_.config_list()) {
    i++;
    if (t_config.has_a_metric_key() && t_config.has_a_data_filter()) {
      err << "UTestConfig[" << i
          << "] has both a_metric_key and a_data_filter provided. These fields "
             "are mutually exclusive\n";
    }
    if (t_config.has_b_metric_key() && t_config.has_b_data_filter()) {
      err << "UTestConfig[" << i
          << "] has both b_metric_key and b_data_filter provided. These fields "
             "are mutually exclusive\n";
    }
    if ((!t_config.has_a_metric_key() || t_config.a_metric_key().empty()) &&
        !t_config.has_a_data_filter()) {
      err << "UTestConfig[" << i
          << "] needs to specify a_metric_key or a_data_filter\n";
    }
    if ((!t_config.has_b_metric_key() || t_config.b_metric_key().empty()) &&
        !t_config.has_b_data_filter()) {
      err << "UTestConfig[" << i
          << "] needs to specify b_metric_key or b_data_filter\n";
    }
    if (!t_config.has_significance_level()) {
      err << "UTestConfig[" << i << "] needs to specify significance_level\n";
    } else if (t_config.significance_level() <= 0 ||
               t_config.significance_level() >= 1) {
      err << "UTestConfig[" << i
          << "] significance_level invalid (must be in range (0, 1))\n";
    }
    if (t_config.has_shift_value() && t_config.has_relative_shift_value()) {
      err << "UTestConfig[" << i
          << "] has both shift_value and relative_shift_value. Can only "
             "specify one.\n";
    }
  }

  return err.str();
}

bool Analyzer::SetAnalyzerOutputPassing(AnalyzerOutput* output,
                                        UTestAnalyzerOutput* custom_output) {
  // custom_output's config_result_list field is set by caller
  *custom_output->mutable_summary() = "okay";

  output->set_regression(false);
  output->set_analyzer_type(analyzer_type());
  output->mutable_status()->set_code(Status_Code_SUCCESS);
  google::protobuf::TextFormat::PrintToString(*custom_output, output->mutable_output());
  return true;
}

bool Analyzer::SetAnalyzerOutputWithError(AnalyzerOutput* output,
                                          UTestAnalyzerOutput* custom_output,
                                          const std::string& err_msg) {
  // custom_output's config_result_list field is set by caller
  *custom_output->mutable_summary() = err_msg;

  LOG(ERROR) << err_msg;
  output->set_regression(false);
  output->set_analyzer_type(analyzer_type());
  output->mutable_status()->set_code(Status_Code_FAIL);
  output->mutable_status()->set_fail_message(err_msg);
  google::protobuf::TextFormat::PrintToString(*custom_output, output->mutable_output());
  return false;
}

bool Analyzer::SetAnalyzerOutputWithRegression(
    AnalyzerOutput* output, UTestAnalyzerOutput* custom_output,
    const std::vector<std::string>& failed_config_names) {
  std::stringstream regression_msg;
  std::string separator;
  regression_msg << failed_config_names.size() << " failed: ";
  for (const std::string& config_name : failed_config_names) {
    regression_msg << separator << config_name;
    separator = ", ";
  }

  // custom_output's config_result_list field is set by caller
  *custom_output->mutable_summary() = regression_msg.str();

  output->set_regression(true);
  output->set_analyzer_type(analyzer_type());
  output->mutable_status()->set_code(Status_Code_SUCCESS);
  google::protobuf::TextFormat::PrintToString(*custom_output, output->mutable_output());
  return true;
}

}  // namespace utest_analyzer
}  // namespace mako
