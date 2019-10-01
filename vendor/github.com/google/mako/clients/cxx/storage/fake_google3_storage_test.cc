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
#include "clients/cxx/storage/fake_google3_storage.h"

#include <set>

#include "gtest/gtest.h"
#include "gmock/gmock.h"
#include "testing/cxx/protocol-buffer-matchers.h"


namespace mako {
namespace fake_google3_storage {
namespace {

using ::mako::EqualsProto;
using ::mako::proto::Partially;
using ::testing::ElementsAre;
using ::mako::RunOrder::BUILD_ID;
using ::mako::RunOrder::TIMESTAMP;

class FakeGoogle3StorageTest : public ::testing::Test {
 protected:
  void TearDown() override { s_.FakeClear(); }
  Storage s_;
};

mako::BenchmarkInfo CreateBenchmarkInfo() {
  mako::BenchmarkInfo benchmark_info;
  benchmark_info.set_benchmark_name("b");
  benchmark_info.set_project_name("p");
  *benchmark_info.add_owner_list() = "*";
  benchmark_info.mutable_input_value_info()->set_label("label");
  benchmark_info.mutable_input_value_info()->set_value_key("k");
  return benchmark_info;
}

mako::RunInfo CreateRunInfo() {
  mako::RunInfo run;
  run.set_benchmark_key("bkey");
  run.set_timestamp_ms(1);
  return run;
}

mako::SampleBatch CreateSampleBatch() {
  mako::SampleBatch batch;
  batch.set_benchmark_key("bkey");
  batch.set_run_key("rkey");
  mako::SamplePoint* point = batch.add_sample_point_list();
  point->set_input_value(0);
  mako::KeyedValue* value = point->add_metric_value_list();
  value->set_value_key("k");
  value->set_value(1.0);
  return batch;
}

TEST_F(FakeGoogle3StorageTest, BadBenchmarkInfo) {
  mako::CreationResponse resp;
  ASSERT_FALSE(s_.CreateBenchmarkInfo(mako::BenchmarkInfo(), &resp));
  ASSERT_EQ(mako::Status::FAIL, resp.status().code());
  ASSERT_NE("", resp.status().fail_message());
}

TEST_F(FakeGoogle3StorageTest, CreateBenchmarkInfo) {
  mako::CreationResponse resp;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(CreateBenchmarkInfo(), &resp));
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());

  mako::CreationResponse resp2;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(CreateBenchmarkInfo(), &resp));
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());

  ASSERT_NE(resp.key(), resp2.key());
}

TEST_F(FakeGoogle3StorageTest, UpdateBenchmarkNoSuchBenchmark) {
  mako::ModificationResponse resp;
  mako::BenchmarkInfo benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_benchmark_key("123");

  ASSERT_FALSE(s_.UpdateBenchmarkInfo(benchmark_info, &resp));
  ASSERT_EQ(mako::Status::FAIL, resp.status().code());
  ASSERT_NE("", resp.status().fail_message());
}

TEST_F(FakeGoogle3StorageTest, UpdateBenchmarkSuccess) {
  // Create a benchmark first.
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(CreateBenchmarkInfo(), &create_resp));

  mako::ModificationResponse resp;
  mako::BenchmarkInfo benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_project_name("A new name");
  benchmark_info.set_benchmark_key(create_resp.key());

  // Need to wait for QueryBenchmarkInfo to validate that update worked.
  ASSERT_TRUE(s_.UpdateBenchmarkInfo(benchmark_info, &resp));
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  ASSERT_EQ(1, resp.count());
}

TEST_F(FakeGoogle3StorageTest, QueryBenchmarkInfoByKey) {
  mako::CreationResponse create_resp;

  // Create a dummy benchmark
  ASSERT_TRUE(s_.CreateBenchmarkInfo(CreateBenchmarkInfo(), &create_resp));

  // Create the benchmark
  mako::BenchmarkInfo bi = CreateBenchmarkInfo();
  bi.set_benchmark_name("the one");
  ASSERT_TRUE(s_.CreateBenchmarkInfo(bi, &create_resp));
  std::string key = create_resp.key();

  // another dummy
  ASSERT_TRUE(s_.CreateBenchmarkInfo(CreateBenchmarkInfo(), &create_resp));

  mako::BenchmarkInfoQueryResponse resp;
  mako::BenchmarkInfoQuery q;
  q.set_benchmark_key(key);
  q.set_benchmark_name("ignore_me");
  ASSERT_TRUE(s_.QueryBenchmarkInfo(q, &resp));
  ASSERT_EQ(1, resp.benchmark_info_list_size());
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  ASSERT_EQ(key, resp.benchmark_info_list(0).benchmark_key());
  ASSERT_EQ("the one", resp.benchmark_info_list(0).benchmark_name());
}

TEST_F(FakeGoogle3StorageTest, QueryBenchmarkInfoByNameAndOrProject) {
  mako::CreationResponse create_resp;

  // Create a dummy benchmark
  ASSERT_TRUE(s_.CreateBenchmarkInfo(CreateBenchmarkInfo(), &create_resp));

  // create a benchmark with a name/project/owner
  mako::BenchmarkInfo bi1 = CreateBenchmarkInfo();
  bi1.set_benchmark_name("testname1");
  bi1.set_project_name("projname1");
  *bi1.add_owner_list() = "owner1@";
  ASSERT_TRUE(s_.CreateBenchmarkInfo(bi1, &create_resp));

  // create another benchmark with a name/project/owner
  mako::BenchmarkInfo bi2 = CreateBenchmarkInfo();
  bi2.set_benchmark_name("testname2");
  bi2.set_project_name("projname2");
  *bi2.add_owner_list() = "owner2@";
  ASSERT_TRUE(s_.CreateBenchmarkInfo(bi2, &create_resp));

  // see if we can query by benchmark name with empty std::string project name/owner
  mako::BenchmarkInfoQueryResponse resp1;
  mako::BenchmarkInfoQuery q1;
  q1.set_benchmark_name("testname1");
  q1.set_project_name("");
  q1.set_owner("");
  ASSERT_TRUE(s_.QueryBenchmarkInfo(q1, &resp1));
  ASSERT_EQ(mako::Status::SUCCESS, resp1.status().code());
  EXPECT_THAT(resp1.benchmark_info_list(), ElementsAre(Partially(EqualsProto(
    "benchmark_name: 'testname1' project_name: 'projname1'")))
  );

  // see if we can query by project name with empty std::string benchmark name/owner
  mako::BenchmarkInfoQueryResponse resp2;
  mako::BenchmarkInfoQuery q2;
  q2.set_benchmark_name("");
  q2.set_project_name("projname2");
  q2.set_owner("");
  ASSERT_TRUE(s_.QueryBenchmarkInfo(q2, &resp2));
  ASSERT_EQ(mako::Status::SUCCESS, resp2.status().code());
  EXPECT_THAT(resp2.benchmark_info_list(), ElementsAre(Partially(EqualsProto(
      "benchmark_name: 'testname2' project_name: 'projname2'")))
  );

  // see if we can query by owner with empty std::string benchmark name/project name
  mako::BenchmarkInfoQueryResponse resp3;
  mako::BenchmarkInfoQuery q3;
  q3.set_benchmark_name("");
  q3.set_project_name("");
  q3.set_owner("owner1@");
  ASSERT_TRUE(s_.QueryBenchmarkInfo(q3, &resp3));
  ASSERT_EQ(mako::Status::SUCCESS, resp3.status().code());
  EXPECT_THAT(resp3.benchmark_info_list(), ElementsAre(Partially(EqualsProto(
    "benchmark_name: 'testname1' project_name: 'projname1'")))
  );
}

TEST_F(FakeGoogle3StorageTest, QueryBenchmarkInfoTwoFoundByOwner) {
  std::string matching_owner = "superman@";
  std::string non_matching_owner = "r2@";
  mako::BenchmarkInfo benchmark_info;

  // Create two projects with matching owner
  benchmark_info = CreateBenchmarkInfo();
  *benchmark_info.add_owner_list() = matching_owner;
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));
  std::string key1 = create_resp.key();
  benchmark_info = CreateBenchmarkInfo();
  *benchmark_info.add_owner_list() = matching_owner;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));
  std::string key2 = create_resp.key();

  // Create 3rd project w/ other owner
  benchmark_info = CreateBenchmarkInfo();
  *benchmark_info.add_owner_list() = non_matching_owner;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));

  mako::BenchmarkInfoQueryResponse resp;
  mako::BenchmarkInfoQuery query;
  query.set_owner(matching_owner);
  ASSERT_TRUE(s_.QueryBenchmarkInfo(query, &resp));
  ASSERT_EQ(2, resp.benchmark_info_list_size());
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  if (resp.benchmark_info_list(0).benchmark_key() == key1) {
    ASSERT_EQ(key2, resp.benchmark_info_list(1).benchmark_key());
  } else {
    ASSERT_EQ(key2, resp.benchmark_info_list(0).benchmark_key());
    ASSERT_EQ(key1, resp.benchmark_info_list(1).benchmark_key());
  }
}

TEST_F(FakeGoogle3StorageTest, QueryBenchmarkInfoUsingCursors) {
  std::string matching_owner = "superman@";
  mako::BenchmarkInfo benchmark_info;

  // Create three projects with matching owner
  benchmark_info = CreateBenchmarkInfo();
  *benchmark_info.add_owner_list() = matching_owner;
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));
  std::string key1 = create_resp.key();
  benchmark_info = CreateBenchmarkInfo();
  *benchmark_info.add_owner_list() = matching_owner;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));
  std::string key2 = create_resp.key();
  benchmark_info = CreateBenchmarkInfo();
  *benchmark_info.add_owner_list() = matching_owner;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));
  std::string key3 = create_resp.key();

  std::set<std::string> keys_retrieved;

  mako::BenchmarkInfoQueryResponse resp;
  mako::BenchmarkInfoQuery query;
  query.set_owner(matching_owner);
  query.set_limit(1);
  ASSERT_TRUE(s_.QueryBenchmarkInfo(query, &resp));
  ASSERT_EQ(1, resp.benchmark_info_list_size());
  ASSERT_NE("", resp.cursor());
  keys_retrieved.insert(resp.benchmark_info_list(0).benchmark_key());

  query.set_cursor(resp.cursor());
  resp.Clear();
  ASSERT_TRUE(s_.QueryBenchmarkInfo(query, &resp));
  ASSERT_EQ(1, resp.benchmark_info_list_size());
  ASSERT_NE("", resp.cursor());
  keys_retrieved.insert(resp.benchmark_info_list(0).benchmark_key());

  query.set_cursor(resp.cursor());
  resp.Clear();
  ASSERT_TRUE(s_.QueryBenchmarkInfo(query, &resp));
  ASSERT_EQ(1, resp.benchmark_info_list_size());
  keys_retrieved.insert(resp.benchmark_info_list(0).benchmark_key());
  // Cursor is empty, no more results
  ASSERT_EQ("", resp.cursor());

  // Make sure all expected keys were returned
  ASSERT_TRUE(keys_retrieved.count(key1));
  ASSERT_TRUE(keys_retrieved.count(key2));
  ASSERT_TRUE(keys_retrieved.count(key3));
}

TEST_F(FakeGoogle3StorageTest, QueryBenchmarkInfoTwoFoundByProject) {
  std::string matching_project_name = "A good project";
  std::string non_matching_project_name = "A bad project";
  mako::BenchmarkInfo benchmark_info;

  // Create two projects with good project name
  benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_project_name(matching_project_name);
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));
  std::string key1 = create_resp.key();
  benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_project_name(matching_project_name);
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));
  std::string key2 = create_resp.key();

  // Create 3rd project w/ bad project name
  benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_project_name(non_matching_project_name);
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));

  mako::BenchmarkInfoQueryResponse resp;
  mako::BenchmarkInfoQuery query;
  query.set_project_name(matching_project_name);
  ASSERT_TRUE(s_.QueryBenchmarkInfo(query, &resp));
  ASSERT_EQ(2, resp.benchmark_info_list_size());
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  if (resp.benchmark_info_list(0).benchmark_key() == key1) {
    ASSERT_EQ(key2, resp.benchmark_info_list(1).benchmark_key());
  } else {
    ASSERT_EQ(key2, resp.benchmark_info_list(0).benchmark_key());
    ASSERT_EQ(key1, resp.benchmark_info_list(1).benchmark_key());
  }
}

TEST_F(FakeGoogle3StorageTest, QueryBenchmarkInfoForUpdatedBenchmark) {
  std::string project_name_1 = "Project1";
  std::string project_name_2 = "Project2";
  mako::BenchmarkInfo benchmark_info;

  // Create with project name 1
  benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_project_name(project_name_1);
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));
  benchmark_info.set_benchmark_key(create_resp.key());

  // Update it to project name 2
  benchmark_info.set_project_name(project_name_2);
  mako::ModificationResponse resp;
  ASSERT_TRUE(s_.UpdateBenchmarkInfo(benchmark_info, &resp));

  // Query by project name 2
  mako::BenchmarkInfoQueryResponse query_resp;
  mako::BenchmarkInfoQuery query;
  query.set_project_name(project_name_2);
  ASSERT_TRUE(s_.QueryBenchmarkInfo(query, &query_resp));
  ASSERT_EQ(1, query_resp.benchmark_info_list_size());
  ASSERT_EQ(benchmark_info.benchmark_key(),
            query_resp.benchmark_info_list(0).benchmark_key());
}

TEST_F(FakeGoogle3StorageTest, DeleteBenchmarkInfo) {
  std::string benchmark_name = "BenchmarkName";
  mako::BenchmarkInfo benchmark_info;

  // Create
  benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_benchmark_name(benchmark_name);
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateBenchmarkInfo(benchmark_info, &create_resp));

  // Delete
  mako::ModificationResponse resp;
  mako::BenchmarkInfoQuery query;
  query.set_benchmark_name(benchmark_name);
  ASSERT_TRUE(s_.DeleteBenchmarkInfo(query, &resp));
  ASSERT_EQ(1, resp.count());
}

TEST_F(FakeGoogle3StorageTest, CountBenchmarkInfo) {
  std::string benchmark_name = "BenchmarkName";
  std::string project_name = "Proj";
  std::string project_name2 = "Proj2";
  mako::BenchmarkInfo benchmark_info;
  mako::BenchmarkInfoQuery query;
  mako::BenchmarkInfoQuery delete_query;
  mako::CountResponse resp;
  mako::CreationResponse create_resp;
  mako::ModificationResponse delete_resp;

  // Count is 0
  query.set_project_name(project_name);
  s_.CountBenchmarkInfo(query, &resp);
  ASSERT_EQ(0, resp.count());

  // Create unmatching
  benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_project_name(project_name2);
  s_.CreateBenchmarkInfo(benchmark_info, &create_resp);

  // Count is still 0
  resp = mako::CountResponse{};
  s_.CountBenchmarkInfo(query, &resp);
  ASSERT_EQ(0, resp.count());

  // Create matching
  benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_project_name(project_name);
  s_.CreateBenchmarkInfo(benchmark_info, &create_resp);

  // Count is 1
  resp = mako::CountResponse{};
  s_.CountBenchmarkInfo(query, &resp);
  ASSERT_EQ(1, resp.count());

  // Create matching
  benchmark_info = CreateBenchmarkInfo();
  benchmark_info.set_project_name(project_name);
  s_.CreateBenchmarkInfo(benchmark_info, &create_resp);
  std::string matching_key = create_resp.key();

  // Count is 2
  resp = mako::CountResponse{};
  s_.CountBenchmarkInfo(query, &resp);
  ASSERT_EQ(2, resp.count());

  // Delete unmatching
  delete_query.set_project_name(project_name2);
  s_.DeleteBenchmarkInfo(delete_query, &delete_resp);

  // Count is still 2
  resp = mako::CountResponse{};
  s_.CountBenchmarkInfo(query, &resp);
  ASSERT_EQ(2, resp.count());

  // Delete matching
  delete_query = mako::BenchmarkInfoQuery{};
  delete_query.set_benchmark_key(matching_key);
  s_.DeleteBenchmarkInfo(delete_query, &delete_resp);

  // Count is 1
  resp = mako::CountResponse{};
  s_.CountBenchmarkInfo(query, &resp);
  ASSERT_EQ(1, resp.count());

  // Delete matching
  delete_query = mako::BenchmarkInfoQuery{};
  delete_query.set_project_name(project_name);
  s_.DeleteBenchmarkInfo(delete_query, &delete_resp);

  // Count is 0
  resp = mako::CountResponse{};
  s_.CountBenchmarkInfo(query, &resp);
  ASSERT_EQ(0, resp.count());
}

TEST_F(FakeGoogle3StorageTest, BadRunInfo) {
  mako::CreationResponse resp;
  ASSERT_FALSE(s_.CreateRunInfo(mako::RunInfo(), &resp));
  ASSERT_EQ(mako::Status::FAIL, resp.status().code());
  ASSERT_NE("", resp.status().fail_message());
}

TEST_F(FakeGoogle3StorageTest, CreateRunInfo) {
  mako::CreationResponse resp;
  ASSERT_TRUE(s_.CreateRunInfo(CreateRunInfo(), &resp));
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());

  mako::CreationResponse resp2;
  ASSERT_TRUE(s_.CreateRunInfo(CreateRunInfo(), &resp));
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());

  ASSERT_NE(resp.key(), resp2.key());
}

TEST_F(FakeGoogle3StorageTest, UpdateRunNoSuchRun) {
  mako::ModificationResponse resp;
  mako::RunInfo run_info = CreateRunInfo();
  run_info.set_benchmark_key("123");
  run_info.set_run_key("456");

  ASSERT_FALSE(s_.UpdateRunInfo(run_info, &resp));
  ASSERT_EQ(mako::Status::FAIL, resp.status().code());
  ASSERT_NE("", resp.status().fail_message());
}

TEST_F(FakeGoogle3StorageTest, UpdateRunSuccess) {
  // Create a run first.
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateRunInfo(CreateRunInfo(), &create_resp));

  mako::ModificationResponse resp;
  mako::RunInfo run_info = CreateRunInfo();
  run_info.set_run_key(create_resp.key());

  // Need to wait for QueryRunInfo to validate that update worked.
  ASSERT_TRUE(s_.UpdateRunInfo(run_info, &resp));
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  ASSERT_EQ(1, resp.count());
}

TEST_F(FakeGoogle3StorageTest, QueryRunInfoByKey) {
  mako::CreationResponse create_resp;

  // Create a dummy run
  ASSERT_TRUE(s_.CreateRunInfo(CreateRunInfo(), &create_resp));

  // Create the run
  mako::RunInfo ri = CreateRunInfo();
  ri.set_description("the one");
  ASSERT_TRUE(s_.CreateRunInfo(ri, &create_resp));
  std::string key = create_resp.key();

  // another dummy
  ASSERT_TRUE(s_.CreateRunInfo(CreateRunInfo(), &create_resp));

  mako::RunInfoQueryResponse resp;
  mako::RunInfoQuery q;
  q.set_run_key(key);
  q.set_benchmark_key("ignore_me");
  ASSERT_TRUE(s_.QueryRunInfo(q, &resp));
  ASSERT_EQ(1, resp.run_info_list_size());
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  ASSERT_EQ(key, resp.run_info_list(0).run_key());
  ASSERT_EQ("the one", resp.run_info_list(0).description());
}

TEST_F(FakeGoogle3StorageTest, QueryRunInfoTwoFoundByTag) {
  // Applied and queried by
  std::string matching_tag_1 = "superman@example.com";
  // Applied but not queried by
  std::string matching_tag_2 = "clarkkent@example.com";
  // Applied to non matching run_info
  std::string non_matching_tag = "r2-d2@example.com";
  mako::RunInfo run_info;

  // Create two projects with matching tag
  run_info = CreateRunInfo();
  *run_info.add_tags() = matching_tag_1;
  *run_info.add_tags() = matching_tag_2;
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  std::string key1 = create_resp.key();
  run_info = CreateRunInfo();
  *run_info.add_tags() = matching_tag_1;
  *run_info.add_tags() = matching_tag_2;
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  std::string key2 = create_resp.key();

  // Create 3rd project w/ other tag
  run_info = CreateRunInfo();
  *run_info.add_tags() = non_matching_tag;
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));

  mako::RunInfoQueryResponse resp;
  mako::RunInfoQuery query;
  *query.add_tags() = matching_tag_1;
  ASSERT_TRUE(s_.QueryRunInfo(query, &resp));
  ASSERT_EQ(2, resp.run_info_list_size());
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  if (resp.run_info_list(0).run_key() == key1) {
    ASSERT_EQ(key2, resp.run_info_list(1).run_key());
  } else {
    ASSERT_EQ(key2, resp.run_info_list(0).run_key()) << resp.ShortDebugString();
    ASSERT_EQ(key1, resp.run_info_list(1).run_key());
  }
}

TEST_F(FakeGoogle3StorageTest, QueryRunInfoUsingCursors) {
  std::string matching_tag = "superman@";
  mako::RunInfo run_info;

  // Create three projects with matching tag
  run_info = CreateRunInfo();
  *run_info.add_tags() = matching_tag;
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  std::string key1 = create_resp.key();
  run_info = CreateRunInfo();
  *run_info.add_tags() = matching_tag;
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  std::string key2 = create_resp.key();
  run_info = CreateRunInfo();
  *run_info.add_tags() = matching_tag;
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  std::string key3 = create_resp.key();

  std::set<std::string> keys_retrieved;

  mako::RunInfoQueryResponse resp;
  mako::RunInfoQuery query;
  *query.add_tags() = matching_tag;
  query.set_limit(1);
  ASSERT_TRUE(s_.QueryRunInfo(query, &resp));
  ASSERT_EQ(1, resp.run_info_list_size());
  ASSERT_NE("", resp.cursor());
  keys_retrieved.insert(resp.run_info_list(0).run_key());

  query.set_cursor(resp.cursor());
  resp.Clear();
  ASSERT_TRUE(s_.QueryRunInfo(query, &resp));
  ASSERT_EQ(1, resp.run_info_list_size());
  ASSERT_NE("", resp.cursor());
  keys_retrieved.insert(resp.run_info_list(0).run_key());

  query.set_cursor(resp.cursor());
  resp.Clear();
  ASSERT_TRUE(s_.QueryRunInfo(query, &resp));
  ASSERT_EQ(1, resp.run_info_list_size());
  keys_retrieved.insert(resp.run_info_list(0).run_key());
  // Cursor is empty, no more results
  ASSERT_EQ("", resp.cursor());

  // Make sure all expected keys were returned
  ASSERT_TRUE(keys_retrieved.count(key1));
  ASSERT_TRUE(keys_retrieved.count(key2));
  ASSERT_TRUE(keys_retrieved.count(key3));
}

TEST_F(FakeGoogle3StorageTest, QueryRunInfoResultsOrderedByTimestamp) {
  mako::CreationResponse create_resp;
  mako::RunInfo run_info;

  run_info = CreateRunInfo();
  run_info.set_timestamp_ms(1);
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  run_info.set_run_key(create_resp.key());
  std::string last_run_key = create_resp.key();
  // Update timestamp so should be last.
  mako::ModificationResponse mod_resp;
  run_info.set_timestamp_ms(10);
  ASSERT_TRUE(s_.UpdateRunInfo(run_info, &mod_resp));

  run_info = CreateRunInfo();
  run_info.set_timestamp_ms(5);
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  std::string middle_run_key = create_resp.key();

  run_info = CreateRunInfo();
  run_info.set_timestamp_ms(4);
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  std::string first_run_key = create_resp.key();

  // Query and make sure results are in correct order.
  mako::RunInfoQuery query;
  mako::RunInfoQueryResponse resp;
  query.set_benchmark_key(run_info.benchmark_key());
  ASSERT_TRUE(s_.QueryRunInfo(query, &resp));

  ASSERT_EQ(3, resp.run_info_list_size());
  ASSERT_EQ(last_run_key, resp.run_info_list(0).run_key());
  ASSERT_EQ(middle_run_key, resp.run_info_list(1).run_key());
  ASSERT_EQ(first_run_key, resp.run_info_list(2).run_key());
}

TEST_F(FakeGoogle3StorageTest, QueryRunInfoForUpdatedRun) {
  double timestamp_ms = 100;
  int64_t build_id = 5000;
  double new_timestamp_ms = 150;
  int64_t new_build_id = 10000;
  mako::RunInfo run_info;
  ASSERT_GT(new_timestamp_ms, timestamp_ms);
  ASSERT_GT(new_build_id, build_id);

  // Create with timestamp_ms
  run_info = CreateRunInfo();
  run_info.set_timestamp_ms(timestamp_ms);
  run_info.set_build_id(build_id);
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateRunInfo(run_info, &create_resp));
  run_info.set_run_key(create_resp.key());

  // Query by timestamp.
  mako::RunInfoQueryResponse query_resp;
  mako::RunInfoQuery query;
  query.set_min_timestamp_ms(timestamp_ms - 1);
  ASSERT_TRUE(s_.QueryRunInfo(query, &query_resp));
  ASSERT_EQ(1, query_resp.run_info_list_size());
  ASSERT_EQ(run_info.run_key(), query_resp.run_info_list(0).run_key());

  // Query by build_id.
  query_resp.Clear();
  query.Clear();
  query.set_min_build_id(build_id - 1);
  query.set_run_order(TIMESTAMP);
  ASSERT_FALSE(s_.QueryRunInfo(query, &query_resp));
  ASSERT_EQ("Attempted to filter query by build_id range without run_order "
            "set to BUILD_ID", query_resp.mutable_status()->fail_message());
  query.set_run_order(BUILD_ID);
  ASSERT_TRUE(s_.QueryRunInfo(query, &query_resp));
  ASSERT_EQ(1, query_resp.run_info_list_size());
  ASSERT_EQ(run_info.run_key(), query_resp.run_info_list(0).run_key());

  // Update it to new_timestamp_ms and new build_id
  run_info.set_timestamp_ms(new_timestamp_ms);
  run_info.set_build_id(new_build_id);
  mako::ModificationResponse resp;
  ASSERT_TRUE(s_.UpdateRunInfo(run_info, &resp));

  // Query by old timestamp
  query_resp.Clear();
  query.Clear();
  query.set_max_timestamp_ms(timestamp_ms + 1);
  ASSERT_TRUE(s_.QueryRunInfo(query, &query_resp));
  ASSERT_EQ(0, query_resp.run_info_list_size());

  // Query by old build_id
  query_resp.Clear();
  query.Clear();
  query.set_run_order(BUILD_ID);
  query.set_max_build_id(build_id + 1);
  ASSERT_TRUE(s_.QueryRunInfo(query, &query_resp));
  ASSERT_EQ(0, query_resp.run_info_list_size());

  // Query by new build_id
  query_resp.Clear();
  query.Clear();
  query.set_run_order(BUILD_ID);
  query.set_max_build_id(new_build_id + 1);
  ASSERT_TRUE(s_.QueryRunInfo(query, &query_resp));
  ASSERT_EQ(1, query_resp.run_info_list_size());
}

TEST_F(FakeGoogle3StorageTest, DeleteRunInfo) {
  // Create
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateRunInfo(CreateRunInfo(), &create_resp));

  // Delete
  mako::ModificationResponse resp;
  mako::RunInfoQuery query;
  query.set_run_key(create_resp.key());
  ASSERT_TRUE(s_.DeleteRunInfo(query, &resp));
  ASSERT_EQ(1, resp.count());
}

TEST_F(FakeGoogle3StorageTest, CountRunInfo) {
  mako::RunInfo run_info;
  mako::RunInfoQuery query;
  mako::RunInfoQuery delete_query;
  mako::CountResponse resp;
  mako::CreationResponse create_resp;
  mako::ModificationResponse delete_resp;

  // create two benchmarks
  mako::CreationResponse benchmark_create_resp;
  s_.CreateBenchmarkInfo(CreateBenchmarkInfo(), &benchmark_create_resp);
  std::string benchmark_key = benchmark_create_resp.key();
  s_.CreateBenchmarkInfo(CreateBenchmarkInfo(), &benchmark_create_resp);
  std::string benchmark_key2 = benchmark_create_resp.key();

  // Count is 0
  query.set_benchmark_key(benchmark_key);
  s_.CountRunInfo(query, &resp);
  ASSERT_EQ(0, resp.count());

  // Create unmatching
  run_info = CreateRunInfo();
  run_info.set_benchmark_key(benchmark_key2);
  s_.CreateRunInfo(run_info, &create_resp);

  // Count is still 0
  resp = mako::CountResponse{};
  s_.CountRunInfo(query, &resp);
  ASSERT_EQ(0, resp.count());

  // Create matching
  run_info = CreateRunInfo();
  run_info.set_benchmark_key(benchmark_key);
  s_.CreateRunInfo(run_info, &create_resp);

  // Count is 1
  resp = mako::CountResponse{};
  s_.CountRunInfo(query, &resp);
  ASSERT_EQ(1, resp.count());

  // Create matching
  run_info = CreateRunInfo();
  run_info.set_benchmark_key(benchmark_key);
  s_.CreateRunInfo(run_info, &create_resp);
  std::string matching_key = create_resp.key();

  // Count is 2
  resp = mako::CountResponse{};
  s_.CountRunInfo(query, &resp);
  ASSERT_EQ(2, resp.count());

  // Delete unmatching
  delete_query.set_benchmark_key(benchmark_key2);
  s_.DeleteRunInfo(delete_query, &delete_resp);

  // Count is still 2
  resp = mako::CountResponse{};
  s_.CountRunInfo(query, &resp);
  ASSERT_EQ(2, resp.count());

  // Delete matching
  delete_query = mako::RunInfoQuery{};
  delete_query.set_run_key(matching_key);
  s_.DeleteRunInfo(delete_query, &delete_resp);

  // Count is 1
  resp = mako::CountResponse{};
  s_.CountRunInfo(query, &resp);
  ASSERT_EQ(1, resp.count());

  // Delete matching
  delete_query = mako::RunInfoQuery{};
  delete_query.set_benchmark_key(benchmark_key);
  s_.DeleteRunInfo(delete_query, &delete_resp);

  // Count is 0
  resp = mako::CountResponse{};
  s_.CountRunInfo(query, &resp);
  ASSERT_EQ(0, resp.count());
}

TEST_F(FakeGoogle3StorageTest, BadSampleBatch) {
  mako::CreationResponse resp;
  ASSERT_FALSE(s_.CreateSampleBatch(mako::SampleBatch(), &resp));
  ASSERT_EQ(mako::Status::FAIL, resp.status().code());
  ASSERT_NE("", resp.status().fail_message());
}

TEST_F(FakeGoogle3StorageTest, CreateSampleBatch) {
  mako::CreationResponse resp;
  ASSERT_TRUE(s_.CreateSampleBatch(CreateSampleBatch(), &resp));
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());

  mako::CreationResponse resp2;
  ASSERT_TRUE(s_.CreateSampleBatch(CreateSampleBatch(), &resp));
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());

  ASSERT_NE(resp.key(), resp2.key());
}

TEST_F(FakeGoogle3StorageTest, QuerySampleBatchByKey) {
  mako::CreationResponse create_resp;

  // Create a dummy batch
  ASSERT_TRUE(s_.CreateSampleBatch(CreateSampleBatch(), &create_resp));

  // Create the batch
  mako::SampleBatch sb = CreateSampleBatch();
  sb.set_run_key("the one");
  ASSERT_TRUE(s_.CreateSampleBatch(sb, &create_resp));
  std::string key = create_resp.key();

  // another dummy
  ASSERT_TRUE(s_.CreateSampleBatch(CreateSampleBatch(), &create_resp));

  mako::SampleBatchQueryResponse resp;
  mako::SampleBatchQuery q;
  q.set_batch_key(key);
  q.set_run_key("ignore_me");
  ASSERT_TRUE(s_.QuerySampleBatch(q, &resp));
  ASSERT_EQ(1, resp.sample_batch_list_size());
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  ASSERT_EQ(key, resp.sample_batch_list(0).batch_key());
  ASSERT_EQ("the one", resp.sample_batch_list(0).run_key());
}

TEST_F(FakeGoogle3StorageTest, QuerySampleBatch) {
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateSampleBatch(CreateSampleBatch(), &create_resp));
  ASSERT_TRUE(s_.CreateSampleBatch(CreateSampleBatch(), &create_resp));

  mako::SampleBatchQueryResponse resp;
  mako::SampleBatchQuery query;

  // By key
  query.set_batch_key(create_resp.key());
  ASSERT_TRUE(s_.QuerySampleBatch(query, &resp));
  ASSERT_EQ(1, resp.sample_batch_list_size());
  ASSERT_EQ(mako::Status::SUCCESS, resp.status().code());
  ASSERT_EQ(create_resp.key(), resp.sample_batch_list(0).batch_key());

  // By benchmark
  resp.Clear();
  query.Clear();
  // CreateSampleBatch returns same key each time
  query.set_benchmark_key(CreateSampleBatch().benchmark_key());
  ASSERT_TRUE(s_.QuerySampleBatch(query, &resp));
  ASSERT_EQ(2, resp.sample_batch_list_size());
}

TEST_F(FakeGoogle3StorageTest, DeleteSampleBatch) {
  // Create
  mako::CreationResponse create_resp;
  ASSERT_TRUE(s_.CreateSampleBatch(CreateSampleBatch(), &create_resp));

  // Delete
  mako::ModificationResponse resp;
  mako::SampleBatchQuery query;
  query.set_batch_key(create_resp.key());
  ASSERT_TRUE(s_.DeleteSampleBatch(query, &resp));
  ASSERT_EQ(1, resp.count());
}

TEST_F(FakeGoogle3StorageTest, CheckLimits) {
  int data;
  ASSERT_TRUE(s_.GetMetricValueCountMax(&data).empty());
  ASSERT_GT(data, 0);
  ASSERT_TRUE(s_.GetSampleErrorCountMax(&data).empty());
  ASSERT_GT(data, 0);
  ASSERT_TRUE(s_.GetBatchSizeMax(&data).empty());
  ASSERT_GT(data, 0);
}

TEST_F(FakeGoogle3StorageTest, QueryBenchmarkInfoOverLimits) {
  int max_benchmarks_returned = 2;
  Storage s(100, 100, 100, max_benchmarks_returned, 100, 100);

  for (int i = 0; i < max_benchmarks_returned + 1; i++) {
    mako::CreationResponse create_resp;
    ASSERT_TRUE(s.CreateBenchmarkInfo(CreateBenchmarkInfo(), &create_resp));
  }

  mako::BenchmarkInfoQueryResponse resp;
  mako::BenchmarkInfoQuery query;
  query.set_limit(max_benchmarks_returned + 1);
  ASSERT_TRUE(s.QueryBenchmarkInfo(query, &resp));
  ASSERT_EQ(max_benchmarks_returned, resp.benchmark_info_list_size());
  ASSERT_NE("", resp.cursor());

  // Query again for remaining records
  query.set_cursor(resp.cursor());
  resp.Clear();
  ASSERT_TRUE(s.QueryBenchmarkInfo(query, &resp));
  ASSERT_EQ(1, resp.benchmark_info_list_size()) << resp.ShortDebugString();
  ASSERT_EQ("", resp.cursor());
}

TEST_F(FakeGoogle3StorageTest, QueryRunInfoOverLimits) {
  int max_runs_returned = 2;
  Storage s(100, 100, 100, 100, max_runs_returned, 100);

  for (int i = 0; i < max_runs_returned + 1; i++) {
    mako::CreationResponse create_resp;
    ASSERT_TRUE(s.CreateRunInfo(CreateRunInfo(), &create_resp));
  }

  mako::RunInfoQueryResponse resp;
  mako::RunInfoQuery query;
  query.set_limit(max_runs_returned + 1);
  ASSERT_TRUE(s.QueryRunInfo(query, &resp));
  ASSERT_EQ(max_runs_returned, resp.run_info_list_size());
  ASSERT_NE("", resp.cursor());

  // Query again for remaining records
  query.set_cursor(resp.cursor());
  resp.Clear();
  ASSERT_TRUE(s.QueryRunInfo(query, &resp));
  ASSERT_EQ(1, resp.run_info_list_size()) << resp.ShortDebugString();
  ASSERT_EQ("", resp.cursor());
}

TEST_F(FakeGoogle3StorageTest, QuerySampleBatchOverLimits) {
  int max_batches_returned = 2;
  Storage s(100, 100, 100, 100, 100, max_batches_returned);

  for (int i = 0; i < max_batches_returned + 1; i++) {
    mako::CreationResponse create_resp;
    ASSERT_TRUE(s.CreateSampleBatch(CreateSampleBatch(), &create_resp));
  }

  mako::SampleBatchQueryResponse resp;
  mako::SampleBatchQuery query;
  query.set_limit(max_batches_returned + 1);
  ASSERT_TRUE(s.QuerySampleBatch(query, &resp));
  ASSERT_EQ(max_batches_returned, resp.sample_batch_list_size());
  ASSERT_NE("", resp.cursor());

  // Query again for remaining records
  query.set_cursor(resp.cursor());
  resp.Clear();
  ASSERT_TRUE(s.QuerySampleBatch(query, &resp));
  ASSERT_EQ(1, resp.sample_batch_list_size()) << resp.ShortDebugString();
  ASSERT_EQ("", resp.cursor());
}

TEST_F(FakeGoogle3StorageTest, QuerySampleBatchWithLimit0) {
  int max_batches_returned = 2;
  Storage s(100, 100, 100, 100, 100, max_batches_returned);

  for (int i = 0; i < max_batches_returned + 1; i++) {
    mako::CreationResponse create_resp;
    ASSERT_TRUE(s.CreateSampleBatch(CreateSampleBatch(), &create_resp));
  }

  mako::SampleBatchQueryResponse resp;
  mako::SampleBatchQuery query;
  // Queries with limit 0 are interpreted as having no limit
  query.set_limit(0);
  ASSERT_TRUE(s.QuerySampleBatch(query, &resp));
  ASSERT_EQ(max_batches_returned, resp.sample_batch_list_size());
  ASSERT_NE("", resp.cursor());
}

TEST_F(FakeGoogle3StorageTest, QueryRunInfoWithLimit0) {
  int max_runs_returned = 2;
  Storage s(100, 100, 100, 100, max_runs_returned, 100);

  for (int i = 0; i < max_runs_returned + 1; i++) {
    mako::CreationResponse create_resp;
    ASSERT_TRUE(s.CreateRunInfo(CreateRunInfo(), &create_resp));
  }

  mako::RunInfoQueryResponse resp;
  mako::RunInfoQuery query;
  // Queries with limit 0 are interpreted as having no limit.
  query.set_limit(0);
  ASSERT_TRUE(s.QueryRunInfo(query, &resp));
  ASSERT_EQ(max_runs_returned, resp.run_info_list_size());
  ASSERT_NE("", resp.cursor());
}

TEST_F(FakeGoogle3StorageTest, QueryBenchmarkInfoWithLimit0) {
  int max_benchmarks_returned = 2;
  Storage s(100, 100, 100, max_benchmarks_returned, 100, 100);

  for (int i = 0; i < max_benchmarks_returned + 1; i++) {
    mako::CreationResponse create_resp;
    ASSERT_TRUE(s.CreateBenchmarkInfo(CreateBenchmarkInfo(), &create_resp));
  }

  mako::BenchmarkInfoQueryResponse resp;
  mako::BenchmarkInfoQuery query;
  // Queries with limit 0 are interpreted as having no limit.
  query.set_limit(0);
  ASSERT_TRUE(s.QueryBenchmarkInfo(query, &resp));
  ASSERT_EQ(max_benchmarks_returned, resp.benchmark_info_list_size());
  ASSERT_NE("", resp.cursor());
}

}  // namespace
}  // namespace fake_google3_storage
}  // namespace mako
