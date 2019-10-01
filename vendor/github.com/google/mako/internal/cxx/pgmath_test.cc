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
#include "internal/cxx/pgmath.h"

#include <algorithm>
#include <string>
#include <vector>

#include "glog/logging.h"
#include "benchmark/benchmark.h"
#include "gtest/gtest.h"
#include "absl/strings/str_join.h"

namespace mako {
namespace internal {

TEST(PgmathTest, MinMaxMeanStddevCountSum) {
  struct Case {
    std::string name;
    std::vector<double> values;
    double want_min;
    double want_max;
    double want_mean;
    double want_stddev;
    double want_count;
    double want_sum;
    bool want_error;
  };
  std::vector<Case> cases = {
    {
      "basic-unsorted",
      {1000.0, 400.0, 450.0, 500.0, 550.0, 600.0, 0.0},
      0.0,
      1000.0,
      500.0,
      273.861279,
      7.0,
      3500.0,
      false,
    },
    {
      "pos-and-neg",
      {-100, 0, 100},
      -100.0,
      100.0,
      0.0,
      81.649658,
      3.0,
      0.0,
      false,
    },
    {
      "no-values",
      {},
      0.0,
      0.0,
      0.0,
      0.0,
      0.0,
      0.0,
      true,
    },
    {
      "one-value",
      {999.0},
      999.0,
      999.0,
      999.0,
      0.0,
      1.0,
      999.0,
      false,
    },
  };
  for (const Case& c : cases) {
    LOG(INFO) << "Case: " << c.name;
    RunningStats stats;
    stats.AddVector(c.values);
    auto got_min = stats.Min();
    if (c.want_error) {
      ASSERT_FALSE(got_min.error.empty());
    } else {
      ASSERT_TRUE(got_min.error.empty());
      ASSERT_NEAR(got_min.value, c.want_min, 0.000001);
    }
    auto got_max = stats.Max();
    if (c.want_error) {
      ASSERT_FALSE(got_max.error.empty());
    } else {
      ASSERT_TRUE(got_max.error.empty());
      ASSERT_NEAR(got_max.value, c.want_max, 0.000001);
    }
    auto got_mean = stats.Mean();
    if (c.want_error) {
      ASSERT_FALSE(got_mean.error.empty());
    } else {
      ASSERT_TRUE(got_mean.error.empty());
      ASSERT_NEAR(got_mean.value, c.want_mean, 0.000001);
    }
    auto got_stddev = stats.Stddev();
    if (c.want_error) {
      ASSERT_FALSE(got_stddev.error.empty());
    } else {
      ASSERT_TRUE(got_stddev.error.empty());
      ASSERT_NEAR(got_stddev.value, c.want_stddev, 0.000001);
    }
    auto got_count = stats.Count();
    if (c.want_error) {
      ASSERT_FALSE(got_count.error.empty());
    } else {
      ASSERT_TRUE(got_count.error.empty());
      ASSERT_NEAR(got_count.value, c.want_count, 0.000001);
    }
    auto got_sum = stats.Sum();
    if (c.want_error) {
      ASSERT_FALSE(got_sum.error.empty());
    } else {
      ASSERT_TRUE(got_sum.error.empty());
      ASSERT_NEAR(got_sum.value, c.want_sum, 0.000001);
    }
  }
}

TEST(PgmathTest, Percentiles) {
  struct Case {
    std::string name;
    std::vector<double> values;
    std::vector<double> pcts;
    std::vector<double> wants;
    bool want_error;
  };
  std::vector<Case> cases = {
    {
      "3-value-no-interpolation",
      {0.0, 50.0, 100.0},
      {0.0, 0.5, 1.0},
      {0.0, 50.0, 100.0},
      false,
    },
    {
      "2-value-with-interpolation",
      {0.0, 100.0},
      {0.0, 0.5, 1.0},
      {0.0, 50.0, 100.0},
      false,
    },
    {
      "1-value",
      {10},
      {0.0, 0.5, 1.0},
      {10.0, 10.0, 10.0},
      false,
    },
    {
      "pos-and-neg",
      {-100, 0.0, 100.0},
      {0.0, 0.5, 1.0},
      {-100.0, 0.0, 100.0},
      false,
    },
    {
      "no-value",
      {},
      {0.0},
      {0.0},
      true,
    },
    {
      "bad-percent",
      {1.0, 2.0},
      {2.0},
      {0.0},
      true,
    },
  };
  for (const Case& c : cases) {
    LOG(INFO) << "Case: " << c.name;
    ASSERT_EQ(c.pcts.size(), c.wants.size());
    RunningStats stats;
    stats.AddVector(c.values);
    for (auto pcts_iter = c.pcts.begin(), wants_iter = c.wants.begin();
         pcts_iter != c.pcts.end(); pcts_iter++, wants_iter++) {
      auto got = stats.Percentile(*pcts_iter);
      if (c.want_error) {
        ASSERT_FALSE(got.error.empty());
      } else {
        ASSERT_TRUE(got.error.empty());
        ASSERT_NEAR(got.value, *wants_iter, 0.000001);
      }
    }
  }
}

TEST(PgmathTest, Median) {
  struct Case {
    std::string name;
    std::vector<double> values;
    double want;
    bool want_error;
  };
  std::vector<Case> cases = {
    {
      "1-value",
      {1.0},
      1.0,
      false,
    },
    {
      "2-values",
      {0, 1.0},
      0.5,
      false,
    },
    {
      "3-values",
      {0, 2.2, 10.0},
      2.2,
      false,
    },
    {
      "no-data",
      {},
      999.0,
      true,
    },
  };
  for (const Case& c : cases) {
    LOG(INFO) << "Case: " << c.name;
    RunningStats stats;
    stats.AddVector(c.values);
    auto got = stats.Median();
    if (c.want_error) {
      ASSERT_FALSE(got.error.empty());
    } else {
      ASSERT_TRUE(got.error.empty());
      ASSERT_NEAR(got.value, c.want, 0.000001);
    }
  }
}

TEST(PgmathTest, Mad) {
  struct Case {
    std::string name;
    std::vector<double> values;
    double want;
    bool want_error;
  };
  std::vector<Case> cases = {
    {
      "basic-unordered",
      {9, 1, 1, 2, 2, 4, 6},
      1.0,
      false,
    },
    {
      "pos-and-neg",
      {-1, 0, 1},
      1.0,
      false,
    },
    {
      "1-value",
      {1.0},
      0,
      false,
    },
    {
      "2-value",
      {1.0, 2.0},  // residuals 0.5,0.5
      0.5,
      false,
    },
    {
      "3-values",
      {0, 2.0, 10.0},  // residuals: 2,0,8
      2.0,
      false,
    },
    {
      "4-values",
      {0, 4.0, 6.0, 100.0},  // residuals: 5,1,1,95
      3.0,
      false,
    },
    {
      "no-data",
      {},
      999.0,
      true,
    },
  };
  for (const Case& c : cases) {
    LOG(INFO) << "Case: " << c.name;
    RunningStats stats;
    stats.AddVector(c.values);
    auto got = stats.Mad();
    if (c.want_error) {
      ASSERT_FALSE(got.error.empty());
    } else {
      ASSERT_TRUE(got.error.empty());
      ASSERT_NEAR(got.value, c.want, 0.000001);
    }
  }
}

TEST(PgmathTest, SampleRestrictions) {
  struct Case {
    std::string name;
    std::vector<double> values;
    int max_sample_size;
    int want_sample_size;
  };
  std::vector<Case> cases = {
    {
      "not-saving-samples",
      {1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
      0,
      0,
    },
    {
      "samples-lt-population",
      {1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
      5,
      5,
    },
    {
      "samples-eq-population",
      {1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
      10,
      10,
    },
    {
      "samples-gt-population",
      {1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
      20,
      10,
    },
  };
  for (const Case& c : cases) {
    LOG(INFO) << "Case: " << c.name;
    Random random;
    RunningStats::Config config;
    config.max_sample_size = c.max_sample_size;
    config.random = &random;
    RunningStats stats(config);
    stats.AddVector(c.values);
    std::vector<double> values = stats.sample();
    std::sort(values.begin(), values.end());
    // log for manual verification of reservoir sampling
    LOG(INFO) << "sorted sample: " << absl::StrJoin(values, ", ");
    ASSERT_EQ(values.size(), c.want_sample_size);
  }
}

static void BM_Add(benchmark::State& state) {
  RunningStats stats(RunningStats::Config{});
  for (auto _ : state) {
    CHECK_EQ("", stats.Add(1.0f));
  }
}
BENCHMARK(BM_Add);

static void BM_AddSampled(benchmark::State& state) {
  mako::internal::Random rand;
  RunningStats stats(RunningStats::Config(1, &rand));
  for (auto _ : state) {
    CHECK_EQ("", stats.Add(1.0f));
  }
}
BENCHMARK(BM_AddSampled);

static void BM_AddVector(benchmark::State& state) {
  RunningStats stats(RunningStats::Config{});
  std::vector<double> vals(state.range(0));
  for (int i = 0; i < state.range(0); ++i) vals[i] = 0.3*i;

  for (auto _ : state) {
    CHECK_EQ("", stats.AddVector(vals));
  }
}
BENCHMARK(BM_AddVector)->Range(1, 100*1000);

}  // namespace internal
}  // namespace mako
