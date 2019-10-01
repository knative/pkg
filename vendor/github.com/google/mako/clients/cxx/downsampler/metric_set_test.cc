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
#include "clients/cxx/downsampler/metric_set.h"

#include "gtest/gtest.h"

namespace mako {
namespace downsampler {

TEST(MetricSetTest, ConstructorSampleError) {
  mako::SampleError e;
  e.set_sampler_name("Sampler");
  MetricSet ms(&e);
  EXPECT_EQ(ms.key, "Sampler");
  EXPECT_EQ(ms.slot_count, 1);
}

TEST(MetricSetTest, ConstructorSamplePoint) {
  mako::SamplePoint p;
  mako::KeyedValue* kv = p.add_metric_value_list();
  kv->set_value_key("m3");
  kv = p.add_metric_value_list();
  kv->set_value_key("m1");
  kv = p.add_metric_value_list();
  kv->set_value_key("m2");
  MetricSet ms(&p);
  EXPECT_EQ(ms.key, "m1,m2,m3");
  EXPECT_EQ(ms.slot_count, 3);
}

TEST(MetricSetTest, ConstructorSamplePointDuplicateMetrics) {
  mako::SamplePoint p;
  mako::KeyedValue* kv = p.add_metric_value_list();
  kv->set_value_key("m3");
  kv = p.add_metric_value_list();
  kv->set_value_key("m1");
  kv = p.add_metric_value_list();
  kv->set_value_key("m2");
  kv = p.add_metric_value_list();
  kv->set_value_key("m1");
  MetricSet ms(&p);
  EXPECT_EQ(ms.key, "m1,m1,m2,m3");
  EXPECT_EQ(ms.slot_count, 4);
}

TEST(MetricSetTest, MetricSetEquals) {
  mako::SamplePoint p;
  mako::KeyedValue* kv = p.add_metric_value_list();
  kv->set_value_key("m3");
  EXPECT_EQ(MetricSet(&p), MetricSet(&p));
}

TEST(MetricSetTest, MetricSetNotEquals) {
  mako::SamplePoint p1;
  mako::KeyedValue* kv = p1.add_metric_value_list();
  kv->set_value_key("m3");

  mako::SamplePoint p2;
  kv = p2.add_metric_value_list();
  kv->set_value_key("m2");
  EXPECT_NE(MetricSet(&p1), MetricSet(&p2));
}

TEST(MetricSetTest, HashMetricSet) {
  mako::SamplePoint p;
  mako::KeyedValue* kv = p.add_metric_value_list();
  kv->set_value_key("m3");
  EXPECT_EQ(std::hash<std::string>()("m3"), HashMetricSet()(MetricSet(&p)));
}
}  // namespace downsampler
}  // namespace mako
