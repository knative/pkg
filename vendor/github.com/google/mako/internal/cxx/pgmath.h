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

// Mako math library.
#ifndef INTERNAL_CXX_PGMATH_H_
#define INTERNAL_CXX_PGMATH_H_

#include <random>
#include <string>
#include <vector>

namespace mako {
namespace internal {

// Random number generator.
// Ideally, each thread should share at most one of these objects,
// as the underlying random number engine may consume significant memory
// (perhaps hundreds of bytes or more).
class Random {
 public:
  // Constructor. Creates the underlying engine.
  Random();

  // Produces a random integer in the interval [a, b].
  int ProduceInt(int a, int b);

 private:
  // mersenne_twister_engine is a random number engine based on Mersenne Twister
  // algorithm. It produces high quality unsigned integer random numbers.
  std::mt19937 engine_;
};

// Efficiently maintains running stats for a numeric population.
//
// Caller can specify the max maintained sample size for estimating percentiles,
// median, and MAD. All other values are exact or precise within the limits of
// the chosen online algorithms.
//
// We may consider an upgrade for online percentiles:
// A Fast Algorithm for Approximate Quantiles in High Speed Data Streams
// Qi Zhang and Wei Wang
// http://citeseerx.ist.psu.edu/viewdoc/download?doi=10.1.1.74.8534&rep=rep1&type=pdf
class RunningStats {
 public:
  // Optional configuration for construction.
  struct Config {
    Config() : Config(-1, nullptr) {}
    Config(int max_size, Random* rand) :
      max_sample_size(max_size),
      random(rand) {}

    // The max sample size maintained for calculating
    // percentiles, median, and MAD. A negative value indicates
    // no max. A value of 0 may be used to skip maintaining
    // a sample of data for when percentiles, median, and MAD
    // are not needed.
    int max_sample_size;

    // If max_sample_size > 0, this must be set as well.
    // Many instances of RunningStats in a single thread should all share
    // the same Random instance.
    Random* random;
  };

  // Returned from some functions to pair a value with a possible error.
  struct Result {
    Result() : value(0.0) {}
    std::string error;
    double value;
  };

  // Default constructor
  RunningStats() : RunningStats(Config()) {}

  // Alternate constructor used to supply optional configuration.
  explicit RunningStats(const Config& config);

  // Adds a value. Any errors will be returned.
  std::string Add(double x);

  // Adds many values. Any errors will be returned.
  std::string AddVector(const std::vector<double>& values);

  // Returns exact count.
  Result Count() const;

  // Returns exact sum.
  Result Sum() const;

  // Returns exact minimum.
  Result Min() const;

  // Returns exact maximum.
  Result Max() const;

  // Returns median.
  Result Mean() const;

  // Returns median.
  Result Median();

  // Returns population variance.
  Result Variance() const;

  // Returns population standard deviation.
  Result Stddev() const;

  // Returns median absolute deviation (MAD).
  Result Mad();

  // Returns given percentile for percent (pct) in range [0.0, 1.0].
  Result Percentile(double pct);

  // Returns the current sample used for percentiles, median, and MAD.
  const std::vector<double>& sample() const {return sample_;}

 private:
  void SortSample();
  std::string CheckCount() const;
  std::string CheckSample() const;

  // Count of values added
  int n_;
  // User supplied config
  Config config_;
  // Sample being maintained, may be capped by max_sample_size.
  std::vector<double> sample_;
  // True if sample is currently sorted. The sample is not maintained sorted,
  // because values may get swapped many times while adding.
  bool sorted_;
  // Exact min seen so far
  double min_;
  // Exact max seen so far
  double max_;
  // Exact sum seen so far
  double sum_;
  // Values for John D. Cook's running stats algorithms based on Knuth's
  // algorithms: http://www.johndcook.com/blog/standard_deviation/
  double old_m_;
  double new_m_;
  double old_s_;
  double new_s_;
};

}  // namespace internal
}  // namespace mako

#endif  // INTERNAL_CXX_PGMATH_H_
