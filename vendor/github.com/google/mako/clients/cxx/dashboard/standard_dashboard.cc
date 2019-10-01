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
#include "clients/cxx/dashboard/standard_dashboard.h"

#include <sstream>

#include "absl/base/casts.h"
#include "absl/strings/match.h"
#include "absl/strings/str_cat.h"
#include "absl/strings/str_join.h"
#include "absl/strings/str_replace.h"
#include "spec/proto/mako.pb.h"

namespace {

constexpr absl::string_view kHostUrl = "https://mako.dev";
constexpr char kNoError[] = "";

void AddQueryStr(const std::string& k, const std::string& v,
                 std::vector<std::string>* query_parameters) {
  std::string escaped = absl::StrReplaceAll(v, {{" ", "+"}, {"=", "%3D"}});
  query_parameters->push_back(absl::StrCat(k, "=", escaped));
}

void AddQueryInt(const std::string& k, int v,
                 std::vector<std::string>* query_parameters) {
  query_parameters->push_back(absl::StrCat(k, "=", v));
}

std::string AddDataFilter(const mako::DataFilter& df, int idx,
                     std::vector<std::string>* query_parameters) {
  std::string idxS = "";
  if (idx >= 0) {
    idxS = std::to_string(idx);
  }
  switch (df.data_type()) {
    case mako::DataFilter::METRIC_AGGREGATE_COUNT:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()), "count",
                  query_parameters);
      break;
    case mako::DataFilter::METRIC_AGGREGATE_MIN:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()), "min",
                  query_parameters);
      break;
    case mako::DataFilter::METRIC_AGGREGATE_MAX:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()), "max",
                  query_parameters);
      break;
    case mako::DataFilter::METRIC_AGGREGATE_MEAN:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()), "mean",
                  query_parameters);
      break;
    case mako::DataFilter::METRIC_AGGREGATE_MEDIAN:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()), "median",
                  query_parameters);
      break;
    case mako::DataFilter::METRIC_AGGREGATE_STDDEV:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()), "stddev",
                  query_parameters);
      break;
    case mako::DataFilter::METRIC_AGGREGATE_MAD:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()), "mad",
                  query_parameters);
      break;
    case mako::DataFilter::METRIC_AGGREGATE_PERCENTILE:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()),
                  absl::StrCat("p", df.percentile_milli_rank()),
                  query_parameters);
      break;
    case mako::DataFilter::CUSTOM_AGGREGATE:
      AddQueryStr(absl::StrCat(idxS, "~", df.value_key()), "1",
                  query_parameters);
      break;
    case mako::DataFilter::BENCHMARK_SCORE:
      if (idx >= 0) {
        AddQueryStr(absl::StrCat(idxS, "s"), "1", query_parameters);
      } else {
        AddQueryStr(absl::StrCat(idxS, "benchmark-score"), "1",
                    query_parameters);
      }
      break;
    case mako::DataFilter::ERROR_COUNT:
      if (idx >= 0) {
        AddQueryStr(absl::StrCat(idxS, "e"), "1", query_parameters);
      } else {
        AddQueryStr(absl::StrCat(idxS, "error-count"), "1", query_parameters);
      }
      break;
    default:
      return "Dashboard.AggregateChart unknown DataFilter.data_type";
  }
  return kNoError;
}

std::string AssembleUrl(std::string base,
                        const std::vector<std::string>& query_parameters) {
  if (query_parameters.empty()) {
    return base;
  }
  return absl::StrCat(base, "?", absl::StrJoin(query_parameters, "&"));
}

}  // namespace

namespace mako {
namespace standard_dashboard {

Dashboard::Dashboard(absl::string_view hostname) : hostname_(hostname) {
  // The Mako client replaces "mako.dev" with "makoperf.appspot.com" because:
  // - a container might not recognize the certificate used to sign mako.dev.
  // Swap it back here so that the user gets mako.dev URLs.
  if (hostname_ == "makoperf.appspot.com" ||
      hostname_ == "https://makoperf.appspot.com") {
    hostname_ = "https://mako.dev";
  }
  if (!absl::StartsWith(hostname_, "http://") &&
      !absl::StartsWith(hostname_, "https://")) {
    hostname_ = absl::StrCat("https://", hostname_);
  }
}

Dashboard::Dashboard() : Dashboard(kHostUrl) {}

std::string Dashboard::AggregateChart(
    const mako::DashboardAggregateChartInput& input, std::string* link) const {
  if (!input.has_benchmark_key() || input.benchmark_key().empty()) {
    return "DashboardAggregateChartInput.benchmark_key empty or missing.";
  }
  std::string url = absl::StrCat(hostname_, "/benchmark");

  std::vector<std::string> query_parameters;
  AddQueryStr("benchmark_key", input.benchmark_key(), &query_parameters);
  for (const auto& sel : input.value_selections()) {
    std::string err = AddDataFilter(sel, -1, &query_parameters);
    if (!err.empty()) {
      return err;
    }
  }
  for (const auto& t : input.tags()) {
    AddQueryStr("tag", t, &query_parameters);
  }
  if (input.has_min_timestamp_ms()) {
    AddQueryStr(
        "tmin",
        absl::StrCat(absl::implicit_cast<int64_t>(input.min_timestamp_ms())),
        &query_parameters);
  }
  if (input.has_max_timestamp_ms()) {
    AddQueryStr(
        "tmax",
        absl::StrCat(absl::implicit_cast<int64_t>(input.max_timestamp_ms())),
        &query_parameters);
  }
  if (input.has_max_runs()) {
    AddQueryInt("maxruns", input.max_runs(), &query_parameters);
  }
  if (input.run_order() == mako::RunOrder::BUILD_ID) {
    AddQueryStr("order", "bid", &query_parameters);
  }
  if (input.has_min_build_id()) {
    AddQueryStr("bidmin", absl::StrCat(input.min_build_id()),
                &query_parameters);
  }
  if (input.has_max_build_id()) {
    AddQueryStr("bidmax", absl::StrCat(input.max_build_id()),
                &query_parameters);
  }
  if (input.highlight_series_on_hover()) {
    AddQueryInt("hiser",
                absl::implicit_cast<int>(input.highlight_series_on_hover()),
                &query_parameters);
  }
  *link = AssembleUrl(url, query_parameters);
  return kNoError;
}

std::string Dashboard::RunChart(const mako::DashboardRunChartInput& input,
                           std::string* link) const {
  if (!input.has_run_key() || input.run_key().empty()) {
    return "DashboardRunChartInput.run_key missing.";
  }
  std::string url = absl::StrCat(hostname_, "/run");
  std::vector<std::string> query_parameters;
  AddQueryStr("run_key", input.run_key(), &query_parameters);
  for (const auto& key : input.metric_keys()) {
    AddQueryStr(absl::StrCat("~", key), "1", &query_parameters);
  }
  if (input.highlight_series_on_hover()) {
    AddQueryInt("hiser",
                absl::implicit_cast<int>(input.highlight_series_on_hover()),
                &query_parameters);
  }
  *link = AssembleUrl(url, query_parameters);
  return kNoError;
}

std::string Dashboard::CompareAggregateChart(
    const mako::DashboardCompareAggregateChartInput& input,
    std::string* link) const {
  std::string pre = "DashboardCompareAggregateChartInput.";
  if (input.series_list_size() == 0) {
    return absl::StrCat(pre, "series_list empty.");
  }
  std::string url = absl::StrCat(hostname_, "/cmpagg");
  std::vector<std::string> query_parameters;
  for (int i = 0; i < input.series_list_size(); i++) {
    std::string idxS = std::to_string(i);
    const auto& s = input.series_list().Get(i);
    if (!s.has_series_label() || s.series_label().empty()) {
      return absl::StrCat(pre, "series_list.series_label missing");
    }
    if (!s.has_benchmark_key() || s.benchmark_key().empty()) {
      return absl::StrCat(pre, "series_list.benchmark_key missing");
    }
    if (!s.has_value_selection()) {
      return absl::StrCat(pre, "series_list.value_selection missing");
    }
    AddQueryStr(absl::StrCat(idxS, "n"), s.series_label(), &query_parameters);
    AddQueryStr(absl::StrCat(idxS, "b"), s.benchmark_key(), &query_parameters);
    AddDataFilter(s.value_selection(), i, &query_parameters);
    for (const auto& t : s.tags()) {
      AddQueryStr(absl::StrCat(idxS, "t"), t, &query_parameters);
    }
  }
  if (input.has_min_timestamp_ms()) {
    AddQueryStr(
        "tmin",
        absl::StrCat(absl::implicit_cast<int64_t>(input.min_timestamp_ms())),
        &query_parameters);
  }
  if (input.has_max_timestamp_ms()) {
    AddQueryStr(
        "tmax",
        absl::StrCat(absl::implicit_cast<int64_t>(input.max_timestamp_ms())),
        &query_parameters);
  }
  if (input.has_max_runs()) {
    AddQueryInt("maxruns", input.max_runs(), &query_parameters);
  }
  if (input.highlight_series_on_hover()) {
    AddQueryInt("hiser",
                absl::implicit_cast<int>(input.highlight_series_on_hover()),
                &query_parameters);
  }
  *link = AssembleUrl(url, query_parameters);
  return kNoError;
}

std::string Dashboard::CompareRunChart(
    const mako::DashboardCompareRunChartInput& input, std::string* link) const {
  if (input.run_keys_size() == 0) {
    return "DashboardCompareRunChartInput run_keys empty.";
  }
  std::string url = absl::StrCat(hostname_, "/cmprun");
  std::vector<std::string> query_parameters;
  for (int i = 0; i < input.run_keys_size(); i++) {
    AddQueryStr(absl::StrCat(i, "r"), input.run_keys(i), &query_parameters);
  }
  for (const auto& key : input.metric_keys()) {
    AddQueryStr(absl::StrCat("~", key), "1", &query_parameters);
  }
  if (input.highlight_series_on_hover()) {
    AddQueryInt("hiser",
                absl::implicit_cast<int>(input.highlight_series_on_hover()),
                &query_parameters);
  }
  *link = AssembleUrl(url, query_parameters);
  return kNoError;
}

std::string Dashboard::VisualizeAnalysis(
    const mako::DashboardVisualizeAnalysisInput& input,
    std::string* link) const {
  std::string url = absl::StrCat(hostname_, "/analysis-results");
  std::vector<std::string> query_parameters;
  AddQueryStr("run_key", input.run_key(), &query_parameters);
  *link = AssembleUrl(url, query_parameters);
  if (input.has_analysis_key()) {
    absl::StrAppend(link, "#analysis", input.analysis_key());
  }
  return kNoError;
}

}  // namespace standard_dashboard
}  // namespace mako
