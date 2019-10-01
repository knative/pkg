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
#include "internal/cxx/analyzer_optimizer.h"

#include <memory>

#include "glog/logging.h"
#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "clients/cxx/storage/fake_google3_storage.h"
#include "spec/cxx/analyzer.h"
#include "spec/cxx/mock_analyzer.h"
#include "spec/cxx/mock_storage.h"
#include "spec/cxx/storage.h"
#include "spec/proto/mako.pb.h"
#include "absl/strings/str_cat.h"
#include "testing/cxx/protocol-buffer-matchers.h"

namespace mako {
namespace internal {

using ::mako::EqualsProto;
using ::mako::proto::Partially;
using testing::_;
using testing::DoAll;
using testing::HasSubstr;
using testing::Invoke;
using testing::NiceMock;
using testing::Return;
using testing::SetArgPointee;
using testing::Truly;

// A test fixture is used to setup state.
class AnalyzerOptimizerTest : public ::testing::Test {
 protected:
  // AnalyzerOptimizerTest is a friend of AnalyzerOptimizer
  // for use of AnalyzerOptimizer::QueryRuns().
  std::string QueryRuns(mako::internal::AnalyzerOptimizer* opt,
                        const mako::RunInfoQuery& in_query,
                        mako::RunInfoQueryResponse* out_response) {
    return opt->QueryRuns(in_query, out_response);
  }

  AnalyzerOptimizerTest() {
    // Create a benchmark inside fake storage
    mako::CreationResponse create_resp;
    b_info_.set_benchmark_name("bname");
    b_info_.set_project_name("bname");
    *b_info_.add_owner_list() = "darthvader";
    b_info_.mutable_input_value_info()->set_label("time");
    b_info_.mutable_input_value_info()->set_value_key("t");
    b_info_.set_benchmark_key("bkey");

    // Create a current run inside fake storage
    r_info_.set_benchmark_key(b_info_.benchmark_key());
    r_info_.set_timestamp_ms(1);
    r_info_.set_run_key("current_run_key");

    // Create AnalyzerOptimizer to use inside tests.
    mako::RunBundle run_bundle;
    *run_bundle.mutable_benchmark_info() = b_info_;
    *run_bundle.mutable_run_info() = r_info_;

    cache_.reset(
        new mako::internal::AnalyzerOptimizer(&mock_storage_, run_bundle));
  }

  ~AnalyzerOptimizerTest() override {}

  mako::BenchmarkInfo b_info_;
  mako::RunInfo r_info_;
  NiceMock<MockStorage> mock_storage_;
  std::unique_ptr<mako::internal::AnalyzerOptimizer> cache_;
};

TEST_F(AnalyzerOptimizerTest, AnalyzerNotFound) {
  MockAnalyzer analyzer;
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  ASSERT_NE("", cache_->GetDataForAnalyzer(&analyzer, &warnings, &input));
}

TEST_F(AnalyzerOptimizerTest, CurrentRunInQueryResults) {
  // If the analyzer gets the current run in it's query results verify that
  // a warning message is returned that contains the run key.
  MockAnalyzer analyzer;

  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(false);
  mako::RunInfoQuery* historic_run_info_query =
      query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("current_run_key");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer));

  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(1, analyzers.size());

  // Response contains the current RunInfo key so should see warning.
  mako::RunInfoQueryResponse mock_query_response;
  mock_query_response.mutable_status()->set_code(mako::Status::SUCCESS);
  mako::RunInfo* mock_run_info = mock_query_response.add_run_info_list();
  mock_run_info->set_run_key(r_info_.run_key());
  mock_run_info->set_benchmark_key(b_info_.benchmark_key());

  // Call to QueryRunInfo will return success
  EXPECT_CALL(mock_storage_, QueryRunInfo(_, _))
      .Times(1)
      .WillRepeatedly(
          DoAll(SetArgPointee<1>(mock_query_response), Return(true)));

  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer, &warnings, &input));

  ASSERT_EQ(1, warnings.size());
  EXPECT_THAT(warnings[0], HasSubstr("current_run_key"));
}

TEST_F(AnalyzerOptimizerTest, SingleAnalyzerWithSampleBatches) {
  MockAnalyzer analyzer;

  // Add the analyzer, which wants a single query performed with batches.
  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(true);
  mako::RunInfoQuery* historic_run_info_query =
      query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("current_run_key");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer));

  // Should return us the single analyzer
  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(1, analyzers.size());
  EXPECT_EQ(&analyzer, analyzers[0]);

  // mock call to RunQuery will return this result
  // These are the storage results from historicy_run_info_query defined above.
  mako::RunInfoQueryResponse mock_query_response;
  mock_query_response.mutable_status()->set_code(mako::Status::SUCCESS);
  mako::RunInfo* mock_run_info = mock_query_response.add_run_info_list();
  mock_run_info->set_run_key("past_run_key");
  mock_run_info->set_benchmark_key(b_info_.benchmark_key());

  // Call to QueryRunInfo will return a result with 1 RunInfo.
  EXPECT_CALL(mock_storage_,
              QueryRunInfo(
                  // Verify that the query we told the analyzer to run above was
                  // actually
                  // run.
                  Truly([historic_run_info_query](
                      const mako::RunInfoQuery actual_query) {
                    bool actual = historic_run_info_query->run_key() ==
                                  actual_query.run_key();
                    if (!actual) {
                      LOG(ERROR) << "Expected run key: "
                                 << historic_run_info_query->run_key()
                                 << " Actual: " << actual_query.run_key();
                    }
                    return actual;
                  }),
                  _))
      .Times(1)
      .WillRepeatedly(
          DoAll(SetArgPointee<1>(mock_query_response), Return(true)));

  // mock call to QuerySampleBatch will return this result.
  mako::SampleBatchQueryResponse mock_batch_response;
  mock_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  mako::SampleBatch* mock_batch =
      mock_batch_response.add_sample_batch_list();
  mock_batch->set_batch_key("past_batch_key");

  EXPECT_CALL(mock_storage_,
              QuerySampleBatch(
                  // Verify the run returned from the query was copied to the
                  // SampleBatch
                  // query as well as the benchmark key was copied to the query.
                  Truly([mock_run_info,
                         this](const mako::SampleBatchQuery actual_query) {
                    bool actual =
                        mock_run_info->run_key() == actual_query.run_key() &&
                        b_info_.benchmark_key() == actual_query.benchmark_key();
                    if (!actual) {
                      LOG(ERROR)
                          << "Expected run key: " << mock_run_info->run_key()
                          << " actual: " << actual_query.run_key();
                      LOG(ERROR) << "Expected benchmark key: "
                                 << b_info_.benchmark_key()
                                 << " actual: " << actual_query.benchmark_key();
                    }
                    return actual;
                  }),
                  _))
      .Times(1)
      .WillRepeatedly(
          DoAll(SetArgPointee<1>(mock_batch_response), Return(true)));

  // Get data for analyzer.
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer, &warnings, &input));

  // Verify correct information was packed into AnalyzerInput.
  //
  // The current RunBundle contains the data passed into constructor (RunInfo
  // and BenchmarkInfo).
  ASSERT_TRUE(input.has_run_to_be_analyzed());
  ASSERT_TRUE(input.run_to_be_analyzed().has_run_info());
  EXPECT_EQ(r_info_.run_key(), input.run_to_be_analyzed().run_info().run_key());
  ASSERT_TRUE(input.run_to_be_analyzed().has_benchmark_info());
  EXPECT_EQ(b_info_.benchmark_key(),
            input.run_to_be_analyzed().benchmark_info().benchmark_key());

  // Verify historic RunBundles
  ASSERT_EQ(1, input.historical_run_list_size());
  mako::RunBundle historic_bundle = input.historical_run_list(0);
  ASSERT_TRUE(historic_bundle.has_run_info());
  EXPECT_EQ("past_run_key", historic_bundle.run_info().run_key());
  ASSERT_TRUE(historic_bundle.has_benchmark_info());
  EXPECT_EQ(b_info_.benchmark_key(),
            historic_bundle.benchmark_info().benchmark_key());
  ASSERT_EQ(1, historic_bundle.batch_list_size());
  EXPECT_EQ("past_batch_key", historic_bundle.batch_list(0).batch_key());
}

TEST_F(AnalyzerOptimizerTest, SingleAnalyzerWithNoSampleBatches) {
  MockAnalyzer analyzer;

  // Add the analyzer, which wants a single query performed with out batches.
  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(false);
  mako::RunInfoQuery* historic_run_info_query =
      query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("current_run_key");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer));

  // Should return us the single analyzer
  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(1, analyzers.size());
  EXPECT_EQ(&analyzer, analyzers[0]);

  // mock call to RunQuery will return this result
  // These are the storage results from historicy_run_info_query defined above.
  mako::RunInfoQueryResponse mock_query_response;
  mock_query_response.mutable_status()->set_code(mako::Status::SUCCESS);
  mako::RunInfo* mock_run_info = mock_query_response.add_run_info_list();
  mock_run_info->set_run_key("past_run_key");

  // Call to QueryRunInfo will return a result with 1 RunInfo.
  EXPECT_CALL(mock_storage_,
              QueryRunInfo(
                  // Verify that the query we told the analyzer to run above was
                  // actually
                  // run.
                  Truly([historic_run_info_query](
                      const mako::RunInfoQuery actual_query) {
                    bool actual = historic_run_info_query->run_key() ==
                                  actual_query.run_key();
                    if (!actual) {
                      LOG(ERROR) << "Expected run key: "
                                 << historic_run_info_query->run_key()
                                 << " Actual: " << actual_query.run_key();
                    }
                    return actual;
                  }),
                  _))
      .Times(1)
      .WillRepeatedly(
          DoAll(SetArgPointee<1>(mock_query_response), Return(true)));

  // We didn't request SampleBatches so we should never call this function.
  EXPECT_CALL(mock_storage_, QuerySampleBatch(_, _)).Times(0);

  // Get data for analyzer.
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer, &warnings, &input));

  // Verify correct information was packed into AnalyzerInput.
  //
  // The current RunBundle contains the data passed into constructor (RunInfo
  // and BenchmarkInfo).
  ASSERT_TRUE(input.has_run_to_be_analyzed());
  ASSERT_TRUE(input.run_to_be_analyzed().has_run_info());
  EXPECT_EQ(r_info_.run_key(), input.run_to_be_analyzed().run_info().run_key());
  ASSERT_TRUE(input.run_to_be_analyzed().has_benchmark_info());
  EXPECT_EQ(b_info_.benchmark_key(),
            input.run_to_be_analyzed().benchmark_info().benchmark_key());

  // Verify historic RunBundles
  ASSERT_EQ(1, input.historical_run_list_size());
  mako::RunBundle historic_bundle = input.historical_run_list(0);
  ASSERT_TRUE(historic_bundle.has_run_info());
  EXPECT_EQ("past_run_key", historic_bundle.run_info().run_key());
  ASSERT_TRUE(historic_bundle.has_benchmark_info());
  ASSERT_EQ(0, historic_bundle.batch_list_size());
}

TEST_F(AnalyzerOptimizerTest, RunInfoQueryFails) {
  MockAnalyzer analyzer;

  // Add the analyzer
  mako::AnalyzerHistoricQueryOutput query_output;
  mako::RunInfoQuery* historic_run_info_query =
      query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("current_run_key");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer));

  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));

  // Mock call to QueryRunInfo will return a failure with the specified error
  // message.
  std::string query_run_info_error = "boom";
  mako::RunInfoQueryResponse mock_query_response;
  mock_query_response.mutable_status()->set_code(mako::Status::FAIL);
  mock_query_response.mutable_status()->set_fail_message(query_run_info_error);

  // Call to QueryRunInfo will return an error
  EXPECT_CALL(mock_storage_, QueryRunInfo(_, _))
      .Times(1)
      .WillRepeatedly(
          DoAll(SetArgPointee<1>(mock_query_response), Return(false)));

  // Get data for analyzer.
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  EXPECT_THAT(cache_->GetDataForAnalyzer(&analyzer, &warnings, &input),
              HasSubstr(query_run_info_error));
}

TEST_F(AnalyzerOptimizerTest, SampleBatchQueryFails) {
  MockAnalyzer analyzer;

  // Add the analyzer, which wants a single query performed with batches.
  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(true);
  mako::RunInfoQuery* historic_run_info_query =
      query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("current_run_key");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer));

  // Should return us the single analyzer
  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(1, analyzers.size());
  EXPECT_EQ(&analyzer, analyzers[0]);

  // mock call to RunQuery will return this result
  // These are the storage results from historicy_run_info_query defined above.
  mako::RunInfoQueryResponse mock_query_response;
  mock_query_response.mutable_status()->set_code(mako::Status::SUCCESS);
  mako::RunInfo* mock_run_info = mock_query_response.add_run_info_list();
  mock_run_info->set_run_key("past_run_key");

  // Call to QueryRunInfo will return a result with 1 RunInfo.
  EXPECT_CALL(mock_storage_, QueryRunInfo(_, _))
      .Times(1)
      .WillRepeatedly(
          DoAll(SetArgPointee<1>(mock_query_response), Return(true)));

  // mock call to QuerySampleBatch will return this result.
  std::string sample_batch_error = "something went boom";
  mako::SampleBatchQueryResponse mock_batch_response;
  mock_batch_response.mutable_status()->set_code(mako::Status::FAIL);
  mock_batch_response.mutable_status()->set_fail_message(sample_batch_error);

  // Return an error here.
  EXPECT_CALL(mock_storage_, QuerySampleBatch(_, _))
      .Times(1)
      .WillRepeatedly(
          DoAll(SetArgPointee<1>(mock_batch_response), Return(false)));

  // Get data for analyzer.
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  EXPECT_THAT(cache_->GetDataForAnalyzer(&analyzer, &warnings, &input),
              HasSubstr(sample_batch_error));
}

TEST_F(AnalyzerOptimizerTest, MultipleAnalyzers) {
  // analyzer1 asks for runs: 'run1' with sample batches
  MockAnalyzer analyzer1;
  // analyzer2 asks for runs: 'run1' and 'run2' with out sample batches.
  // Because the cache should old the results from 'run1' we should only see 2
  // total queries to QueryRunInfo and 1 query to QuerySampleBatches.
  MockAnalyzer analyzer2;

  // Configure analyzer1's queries
  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(true);
  mako::RunInfoQuery* historic_run_info_query =
      query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("run1");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer1));

  // Configure analyzer2's queries
  query_output.Clear();
  query_output.set_get_batches(false);
  historic_run_info_query = query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("run1");
  historic_run_info_query = query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("run2");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer2));

  // Should return us both analyzers
  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(2, analyzers.size());

  // Mock responses from each of the queries above.
  mako::RunInfoQueryResponse run1_response;
  run1_response.mutable_status()->set_code(mako::Status::SUCCESS);
  mako::RunInfo* run1 = run1_response.add_run_info_list();
  run1->set_run_key("run1");

  mako::RunInfoQueryResponse run2_response;
  run2_response.mutable_status()->set_code(mako::Status::SUCCESS);
  mako::RunInfo* run2 = run2_response.add_run_info_list();
  run2->set_run_key("run2");

  // Call to QueryRunInfo will return a result with 1 RunInfo.
  EXPECT_CALL(mock_storage_, QueryRunInfo(_, _))
      // Should only be called 2 times because run1 results will be cached.
      .Times(2)
      .WillRepeatedly(Invoke(
          [run1_response, run2_response](const mako::RunInfoQuery& query,
                                         mako::RunInfoQueryResponse* resp) {
            if (query.run_key() == "run1") {
              *resp = run1_response;
              return true;
            } else if (query.run_key() == "run2") {
              *resp = run2_response;
              return true;
            }
            resp->mutable_status()->set_code(mako::Status::FAIL);
            resp->mutable_status()->set_fail_message(
                absl::StrCat("Bad query: ", query.ShortDebugString()));
            return false;
          }));

  // Only a query with run key 'run1' should prompt a call for SampleBatches
  mako::SampleBatchQueryResponse run1_batch_response;
  run1_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  mako::SampleBatch* mock_batch =
      run1_batch_response.add_sample_batch_list();
  mock_batch->set_batch_key("run_1_batches");

  // There should be only 1 call to storage, looking for run1 batches.
  EXPECT_CALL(mock_storage_,
              QuerySampleBatch(
                  // Verify query is looking for 'run1' batches.
                  Truly([](const mako::SampleBatchQuery actual_query) {
                    bool actual = actual_query.run_key() == "run1";
                    if (!actual) {
                      LOG(ERROR) << "Expected run key: 'run1' actual: "
                                 << actual_query.run_key();
                    }
                    return actual;
                  }),
                  _))
      .Times(1)
      .WillRepeatedly(
          DoAll(SetArgPointee<1>(run1_batch_response), Return(true)));

  // Get and check data for analyzer1.
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer1, &warnings, &input));
  // Should only contain run1 with sample batches
  ASSERT_EQ(1, input.historical_run_list_size());
  mako::RunBundle historic_bundle = input.historical_run_list(0);
  ASSERT_TRUE(historic_bundle.has_run_info());
  EXPECT_EQ("run1", historic_bundle.run_info().run_key());
  ASSERT_TRUE(historic_bundle.has_benchmark_info());
  ASSERT_EQ(1, historic_bundle.batch_list_size());
  EXPECT_EQ("run_1_batches", historic_bundle.batch_list(0).batch_key());

  // Get and check data for analyzer2.
  input.Clear();
  warnings.clear();
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer2, &warnings, &input));
  // Should only contain run1 with sample batches
  ASSERT_EQ(2, input.historical_run_list_size());
  // Queries were 'run1' then 'run2' so results should be in the same order.
  historic_bundle = input.historical_run_list(0);
  EXPECT_EQ("run1", historic_bundle.run_info().run_key());
  EXPECT_EQ(0, historic_bundle.batch_list_size());
  historic_bundle = input.historical_run_list(1);
  EXPECT_EQ("run2", historic_bundle.run_info().run_key());
  EXPECT_EQ(0, historic_bundle.batch_list_size());
}

TEST_F(AnalyzerOptimizerTest, ResultsOrderAndDuplicatesRemoved) {
  // Analyzer has 3 queries:
  // - 'query1' returns 'run3' and 'run2'
  // - 'query2' returns 'run1' and 'run2'
  // - 'query3' returns no results
  // We expect the data passed to analyzer to be:
  //  - 'run3', 'run2', 'run1'
  //  - 'batch3', 'batch2', 'batch1'
  MockAnalyzer analyzer;

  // Configure analyzer's queries
  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(true);
  // query1
  mako::RunInfoQuery* historic_run_info_query =
      query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("query1");
  // query2
  historic_run_info_query = query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("query2");
  // query3
  historic_run_info_query = query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("query3");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer));

  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(1, analyzers.size());

  // query1 response
  mako::RunInfoQueryResponse query1_reponse;
  query1_reponse.mutable_status()->set_code(mako::Status::SUCCESS);
  query1_reponse.add_run_info_list()->set_run_key("run3");
  query1_reponse.add_run_info_list()->set_run_key("run2");

  // query2 response
  mako::RunInfoQueryResponse query2_reponse;
  query2_reponse.mutable_status()->set_code(mako::Status::SUCCESS);
  query2_reponse.add_run_info_list()->set_run_key("run1");
  query2_reponse.add_run_info_list()->set_run_key("run2");

  // query3 response
  mako::RunInfoQueryResponse query3_reponse;
  query3_reponse.mutable_status()->set_code(mako::Status::SUCCESS);

  // Call to QueryRunInfo will return a result with 1 RunInfo.
  EXPECT_CALL(mock_storage_, QueryRunInfo(_, _))
      // Should only be called 3 times because run1 results will be cached.
      .Times(3)
      .WillRepeatedly(Invoke([query1_reponse, query2_reponse, query3_reponse](
          const mako::RunInfoQuery& query,
          mako::RunInfoQueryResponse* resp) {
        if (query.run_key() == "query1") {
          *resp = query1_reponse;
          return true;
        } else if (query.run_key() == "query2") {
          *resp = query2_reponse;
          return true;
        } else if (query.run_key() == "query3") {
          *resp = query3_reponse;
          return true;
        }
        resp->mutable_status()->set_code(mako::Status::FAIL);
        resp->mutable_status()->set_fail_message(
            absl::StrCat("Bad query: ", query.ShortDebugString()));
        return false;
      }));

  mako::SampleBatchQueryResponse run1_batch_response;
  run1_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run1_batch_response.add_sample_batch_list()->set_batch_key("run1_batch");

  mako::SampleBatchQueryResponse run2_batch_response;
  run2_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run2_batch_response.add_sample_batch_list()->set_batch_key("run2_batch");
  run2_batch_response.add_sample_batch_list()->set_batch_key("run2a_batch");

  mako::SampleBatchQueryResponse run3_batch_response;
  run3_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run3_batch_response.add_sample_batch_list()->set_batch_key("run3_batch");

  EXPECT_CALL(mock_storage_, QuerySampleBatch(_, _))
      .Times(3)
      .WillRepeatedly(Invoke(
          [run1_batch_response, run2_batch_response, run3_batch_response](
              const mako::SampleBatchQuery& query,
              mako::SampleBatchQueryResponse* resp) {
            if (query.run_key() == "run1") {
              *resp = run1_batch_response;
              return true;
            } else if (query.run_key() == "run2") {
              *resp = run2_batch_response;
              return true;
            } else if (query.run_key() == "run3") {
              *resp = run3_batch_response;
              return true;
            }
            resp->mutable_status()->set_code(mako::Status::FAIL);
            resp->mutable_status()->set_fail_message(
                absl::StrCat("Bad query: ", query.ShortDebugString()));
            return false;
          }));

  // Get and check data for analyzer1.
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer, &warnings, &input));
  // Verify order of runs
  // Run3
  ASSERT_EQ(3, input.historical_run_list_size());
  EXPECT_EQ("run3", input.historical_run_list(0).run_info().run_key());
  ASSERT_EQ(1, input.historical_run_list(0).batch_list_size());
  EXPECT_EQ("run3_batch",
            input.historical_run_list(0).batch_list(0).batch_key());
  // Run2
  EXPECT_EQ("run2", input.historical_run_list(1).run_info().run_key());
  ASSERT_EQ(2, input.historical_run_list(1).batch_list_size());
  EXPECT_EQ("run2_batch",
            input.historical_run_list(1).batch_list(0).batch_key());
  EXPECT_EQ("run2a_batch",
            input.historical_run_list(1).batch_list(1).batch_key());
  // Run1
  EXPECT_EQ("run1", input.historical_run_list(2).run_info().run_key());
  ASSERT_EQ(1, input.historical_run_list(2).batch_list_size());
  EXPECT_EQ("run1_batch",
            input.historical_run_list(2).batch_list(0).batch_key());
}

TEST_F(AnalyzerOptimizerTest, TwoSampleMapAndListDefined) {
  MockAnalyzer analyzer;

  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(true);

  // Map defined
  (*query_output.mutable_run_info_query_map())["a"]
      .add_run_info_query_list()
      ->set_run_key("query1");
  (*query_output.mutable_run_info_query_map())["b"]
      .add_run_info_query_list()
      ->set_run_key("query2");

  // List defined. Should be ignored by Analyzer Optimizer.
  query_output.add_run_info_query_list()->set_run_key("query1");
  query_output.add_run_info_query_list()->set_run_key("query2");

  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer));

  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(1, analyzers.size());

  mako::RunInfoQueryResponse query1_response;
  query1_response.mutable_status()->set_code(mako::Status::SUCCESS);
  query1_response.add_run_info_list()->set_run_key("run3");
  query1_response.add_run_info_list()->set_run_key("run1");

  mako::RunInfoQueryResponse query2_response;
  query2_response.mutable_status()->set_code(mako::Status::SUCCESS);
  query2_response.add_run_info_list()->set_run_key("run1");
  query2_response.add_run_info_list()->set_run_key("run2");

  EXPECT_CALL(
      mock_storage_,
      QueryRunInfo(
          Partially(EqualsProto<mako::RunInfoQuery>("run_key: 'query1'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(query1_response), Return(true)));

  EXPECT_CALL(
      mock_storage_,
      QueryRunInfo(
          Partially(EqualsProto<mako::RunInfoQuery>("run_key: 'query2'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(query2_response), Return(true)));

  mako::SampleBatchQueryResponse run1_batch_response;
  run1_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run1_batch_response.add_sample_batch_list()->set_batch_key("run1_batch");

  mako::SampleBatchQueryResponse run2_batch_response;
  run2_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run2_batch_response.add_sample_batch_list()->set_batch_key("run2_batch");
  run2_batch_response.add_sample_batch_list()->set_batch_key("run2a_batch");

  mako::SampleBatchQueryResponse run3_batch_response;
  run3_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run3_batch_response.add_sample_batch_list()->set_batch_key("run3_batch");

  EXPECT_CALL(
      mock_storage_,
      QuerySampleBatch(
          Partially(EqualsProto<mako::SampleBatchQuery>("run_key: 'run1'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(run1_batch_response), Return(true)));
  EXPECT_CALL(
      mock_storage_,
      QuerySampleBatch(
          Partially(EqualsProto<mako::SampleBatchQuery>("run_key: 'run2'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(run2_batch_response), Return(true)));
  EXPECT_CALL(
      mock_storage_,
      QuerySampleBatch(
          Partially(EqualsProto<mako::SampleBatchQuery>("run_key: 'run3'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(run3_batch_response), Return(true)));

  // Check AnalyzerInput.
  mako::AnalyzerInput actual;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer, &warnings, &actual));

  // Check that the historical run list is not populated.
  ASSERT_EQ(0, actual.historical_run_list_size());

  mako::AnalyzerInput expected;

  // Construct expected A samples
  auto& a_samples = (*expected.mutable_historical_run_map())["a"];

  mako::RunBundle* rb = a_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run3");
  (*rb->add_batch_list()).set_batch_key("run3_batch");

  rb = a_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run1");
  (*rb->add_batch_list()).set_batch_key("run1_batch");

  // Construct expected B samples
  auto& b_samples = (*expected.mutable_historical_run_map())["b"];
  rb = b_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run1");
  (*rb->add_batch_list()).set_batch_key("run1_batch");

  rb = b_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run2");
  (*rb->add_batch_list()).set_batch_key("run2_batch");
  (*rb->add_batch_list()).set_batch_key("run2a_batch");

  EXPECT_THAT(actual, Partially(EqualsProto(expected)));
}

TEST_F(AnalyzerOptimizerTest, TwoSampleMapDefined) {
  // Analyzer has A/B samples.
  // A-sample
  // - 'query1' returns 'run3' and 'run1'
  // - 'query2' returns 'run1' and 'run2'
  // B-sample
  // - 'query2'
  // - 'query3' returns no results

  // We expect the data passed to analyzer to be:
  // A-sample
  // - 'run3', 'run1', 'run2'
  // - 'batch3', 'batch1', ('batch2', 'batch2a')
  // B-sample
  // - 'run1', 'run2'
  // - 'batch1', ('batch2', 'batch2a')
  MockAnalyzer analyzer;

  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(true);

  // A-sample has 2 queries.
  (*query_output.mutable_run_info_query_map())["a"]
      .add_run_info_query_list()
      ->set_run_key("query1");
  (*query_output.mutable_run_info_query_map())["a"]
      .add_run_info_query_list()
      ->set_run_key("query2");

  // B-sample has 2 queries. 1 query also exists in A-sample.
  (*query_output.mutable_run_info_query_map())["b"]
      .add_run_info_query_list()
      ->set_run_key("query2");
  (*query_output.mutable_run_info_query_map())["b"]
      .add_run_info_query_list()
      ->set_run_key("query3");

  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer));

  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(1, analyzers.size());

  mako::RunInfoQueryResponse query1_response;
  query1_response.mutable_status()->set_code(mako::Status::SUCCESS);
  query1_response.add_run_info_list()->set_run_key("run3");
  query1_response.add_run_info_list()->set_run_key("run1");

  mako::RunInfoQueryResponse query2_response;
  query2_response.mutable_status()->set_code(mako::Status::SUCCESS);
  query2_response.add_run_info_list()->set_run_key("run1");
  query2_response.add_run_info_list()->set_run_key("run2");

  mako::RunInfoQueryResponse query3_response;
  query3_response.mutable_status()->set_code(mako::Status::SUCCESS);

  EXPECT_CALL(
      mock_storage_,
      QueryRunInfo(
          Partially(EqualsProto<mako::RunInfoQuery>("run_key: 'query1'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(query1_response), Return(true)));

  EXPECT_CALL(
      mock_storage_,
      QueryRunInfo(
          Partially(EqualsProto<mako::RunInfoQuery>("run_key: 'query2'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(query2_response), Return(true)));

  EXPECT_CALL(
      mock_storage_,
      QueryRunInfo(
          Partially(EqualsProto<mako::RunInfoQuery>("run_key: 'query3'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(query3_response), Return(true)));

  mako::SampleBatchQueryResponse run1_batch_response;
  run1_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run1_batch_response.add_sample_batch_list()->set_batch_key("run1_batch");

  mako::SampleBatchQueryResponse run2_batch_response;
  run2_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run2_batch_response.add_sample_batch_list()->set_batch_key("run2_batch");
  run2_batch_response.add_sample_batch_list()->set_batch_key("run2a_batch");

  mako::SampleBatchQueryResponse run3_batch_response;
  run3_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run3_batch_response.add_sample_batch_list()->set_batch_key("run3_batch");

  EXPECT_CALL(
      mock_storage_,
      QuerySampleBatch(
          Partially(EqualsProto<mako::SampleBatchQuery>("run_key: 'run1'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(run1_batch_response), Return(true)));
  EXPECT_CALL(
      mock_storage_,
      QuerySampleBatch(
          Partially(EqualsProto<mako::SampleBatchQuery>("run_key: 'run2'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(run2_batch_response), Return(true)));
  EXPECT_CALL(
      mock_storage_,
      QuerySampleBatch(
          Partially(EqualsProto<mako::SampleBatchQuery>("run_key: 'run3'")),
          _))
      .WillOnce(DoAll(SetArgPointee<1>(run3_batch_response), Return(true)));

  // Check AnalyzerInput.
  mako::AnalyzerInput actual;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer, &warnings, &actual));

  // Check that the historical run list is not populated.
  ASSERT_EQ(0, actual.historical_run_list_size());

  mako::AnalyzerInput expected;

  // Construct expected A samples
  auto& a_samples = (*expected.mutable_historical_run_map())["a"];

  mako::RunBundle* rb = a_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run3");
  (*rb->add_batch_list()).set_batch_key("run3_batch");

  rb = a_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run1");
  (*rb->add_batch_list()).set_batch_key("run1_batch");

  rb = a_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run2");
  (*rb->add_batch_list()).set_batch_key("run2_batch");
  (*rb->add_batch_list()).set_batch_key("run2a_batch");

  // Construct expected B samples
  auto& b_samples = (*expected.mutable_historical_run_map())["b"];
  rb = b_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run1");
  (*rb->add_batch_list()).set_batch_key("run1_batch");

  rb = b_samples.add_historical_run_list();
  (*rb->mutable_run_info()).set_run_key("run2");
  (*rb->add_batch_list()).set_batch_key("run2_batch");
  (*rb->add_batch_list()).set_batch_key("run2a_batch");

  EXPECT_THAT(actual, Partially(EqualsProto(expected)));
}

TEST_F(AnalyzerOptimizerTest, CacheHit) {
  // 2 Analyzers will ask for 2 queries with batches. The queries will ask for
  // the same run key but with different ordering of tags. These queries should
  // be the same from the cache perspective.
  // - 'query1' will ask for 'run1' with tags 'a', 'b'
  // - 'query2' will ask for 'run1' with tags 'b', 'a'
  // We should see the cache only allow 1 query to storage.
  MockAnalyzer analyzer1;
  MockAnalyzer analyzer2;

  // query response
  mako::RunInfoQueryResponse query_reponse;
  query_reponse.mutable_status()->set_code(mako::Status::SUCCESS);
  query_reponse.add_run_info_list()->set_run_key("run1");

  // query batch response
  mako::SampleBatchQueryResponse run1_batch_response;
  run1_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run1_batch_response.add_sample_batch_list()->set_batch_key("run1_batch");

  // Create a AnalyzerOptimizer instance with cache sizes large enough to fit
  // this response. Just set it much larger than we need it to be (10x).
  mako::RunBundle run_bundle;
  *run_bundle.mutable_benchmark_info() = b_info_;
  *run_bundle.mutable_run_info() = r_info_;
  int run_info_cache_size = 10 * query_reponse.ByteSize();
  int batch_cache_size = 10 * run1_batch_response.ByteSize();
  cache_.reset(new mako::internal::AnalyzerOptimizer(
      &mock_storage_, run_bundle, run_info_cache_size, batch_cache_size));

  // Configure analyzer1 queries (tags a,b)
  {
    mako::AnalyzerHistoricQueryOutput query_output;
    query_output.set_get_batches(true);
    mako::RunInfoQuery* historic_run_info_query =
        query_output.add_run_info_query_list();
    historic_run_info_query->set_run_key("query");
    *historic_run_info_query->add_tags() = "a";
    *historic_run_info_query->add_tags() = "b";
    ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer1));
  }

  // Configure analyzer2 queries (tags b,c)
  {
    mako::AnalyzerHistoricQueryOutput query_output;
    query_output.set_get_batches(true);
    mako::RunInfoQuery* historic_run_info_query =
        query_output.add_run_info_query_list();
    historic_run_info_query->set_run_key("query");
    *historic_run_info_query->add_tags() = "b";
    *historic_run_info_query->add_tags() = "a";
    ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer2));
  }

  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(2, analyzers.size());

  // Call to QueryRunInfo will return a result with 1 RunInfo.
  EXPECT_CALL(mock_storage_, QueryRunInfo(_, _))
      // Should be called 1 times because last result is cached.
      .Times(1)
      .WillRepeatedly(
          Invoke([query_reponse](const mako::RunInfoQuery& query,
                                 mako::RunInfoQueryResponse* resp) {
            if (query.run_key() == "query") {
              *resp = query_reponse;
              return true;
            }
            resp->mutable_status()->set_code(mako::Status::FAIL);
            resp->mutable_status()->set_fail_message(
                absl::StrCat("Bad query: ", query.ShortDebugString()));
            return false;
          }));

  EXPECT_CALL(mock_storage_, QuerySampleBatch(_, _))
      // Should be called 1 times because last result is cached.
      .Times(1)
      .WillRepeatedly(Invoke(
          [run1_batch_response](const mako::SampleBatchQuery& query,
                                mako::SampleBatchQueryResponse* resp) {
            if (query.run_key() == "run1") {
              *resp = run1_batch_response;
              return true;
            }
            resp->mutable_status()->set_code(mako::Status::FAIL);
            resp->mutable_status()->set_fail_message(
                absl::StrCat("Bad query: ", query.ShortDebugString()));
            return false;
          }));

  // Get and check data for analyzer1.
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer1, &warnings, &input));
  ASSERT_EQ(1, input.historical_run_list_size());
  EXPECT_EQ("run1", input.historical_run_list(0).run_info().run_key());
  ASSERT_EQ(1, input.historical_run_list(0).batch_list_size());
  EXPECT_EQ("run1_batch",
            input.historical_run_list(0).batch_list(0).batch_key());

  // Get and check data for analyzer2.
  warnings.clear();
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer2, &warnings, &input));
  ASSERT_EQ(1, input.historical_run_list_size());
  EXPECT_EQ("run1", input.historical_run_list(0).run_info().run_key());
  ASSERT_EQ(1, input.historical_run_list(0).batch_list_size());
  EXPECT_EQ("run1_batch",
            input.historical_run_list(0).batch_list(0).batch_key());
}

TEST_F(AnalyzerOptimizerTest, CacheEvicted) {
  // 2 Analyzers will ash for 2 identical queries with batches:
  // - 'query' will ask for 'run1'
  // Usually we would only see 1 calls to storage. But we'll make 'run1' too
  // large to fit in cache. This will force us to make 2 calls to storage.
  MockAnalyzer analyzer1;
  MockAnalyzer analyzer2;

  // query response
  mako::RunInfoQueryResponse query_reponse;
  query_reponse.mutable_status()->set_code(mako::Status::SUCCESS);
  query_reponse.add_run_info_list()->set_run_key("run1");

  // query batch response
  mako::SampleBatchQueryResponse run1_batch_response;
  run1_batch_response.mutable_status()->set_code(mako::Status::SUCCESS);
  run1_batch_response.add_sample_batch_list()->set_batch_key("run1_batch");

  // Create a AnalyzerOptimizer instance with cache sizes too small to fit this
  // response.
  mako::RunBundle run_bundle;
  *run_bundle.mutable_benchmark_info() = b_info_;
  *run_bundle.mutable_run_info() = r_info_;
  int run_info_cache_size = query_reponse.ByteSize() - 1;
  int batch_cache_size = run1_batch_response.ByteSize() - 1;
  cache_.reset(new mako::internal::AnalyzerOptimizer(
      &mock_storage_, run_bundle, run_info_cache_size, batch_cache_size));

  // Configure analyzer's queries
  mako::AnalyzerHistoricQueryOutput query_output;
  query_output.set_get_batches(true);
  // query1
  mako::RunInfoQuery* historic_run_info_query =
      query_output.add_run_info_query_list();
  historic_run_info_query->set_run_key("query");
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer1));
  ASSERT_EQ("", cache_->AddAnalyzer(query_output, &analyzer2));

  std::vector<mako::Analyzer*> analyzers;
  ASSERT_EQ("", cache_->OrderAnalyzers(&analyzers));
  EXPECT_EQ(2, analyzers.size());

  // Call to QueryRunInfo will return a result with 1 RunInfo.
  EXPECT_CALL(mock_storage_, QueryRunInfo(_, _))
      // Should be called 2 times because last result is evicted from cache.
      .Times(2)
      .WillRepeatedly(
          Invoke([query_reponse](const mako::RunInfoQuery& query,
                                 mako::RunInfoQueryResponse* resp) {
            if (query.run_key() == "query") {
              *resp = query_reponse;
              return true;
            }
            resp->mutable_status()->set_code(mako::Status::FAIL);
            resp->mutable_status()->set_fail_message(
                absl::StrCat("Bad query: ", query.ShortDebugString()));
            return false;
          }));

  EXPECT_CALL(mock_storage_, QuerySampleBatch(_, _))
      // Should be called 2 times because last result is evicted from cache.
      .Times(2)
      .WillRepeatedly(Invoke(
          [run1_batch_response](const mako::SampleBatchQuery& query,
                                mako::SampleBatchQueryResponse* resp) {
            if (query.run_key() == "run1") {
              *resp = run1_batch_response;
              return true;
            }
            resp->mutable_status()->set_code(mako::Status::FAIL);
            resp->mutable_status()->set_fail_message(
                absl::StrCat("Bad query: ", query.ShortDebugString()));
            return false;
          }));

  // Get and check data for analyzer1.
  mako::AnalyzerInput input;
  std::vector<std::string> warnings;
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer1, &warnings, &input));
  ASSERT_EQ(1, input.historical_run_list_size());
  EXPECT_EQ("run1", input.historical_run_list(0).run_info().run_key());
  ASSERT_EQ(1, input.historical_run_list(0).batch_list_size());
  EXPECT_EQ("run1_batch",
            input.historical_run_list(0).batch_list(0).batch_key());

  // Get and check data for analyzer2.
  warnings.clear();
  ASSERT_EQ("", cache_->GetDataForAnalyzer(&analyzer2, &warnings, &input));
  ASSERT_EQ(1, input.historical_run_list_size());
  EXPECT_EQ("run1", input.historical_run_list(0).run_info().run_key());
  ASSERT_EQ(1, input.historical_run_list(0).batch_list_size());
  EXPECT_EQ("run1_batch",
            input.historical_run_list(0).batch_list(0).batch_key());
}

TEST_F(AnalyzerOptimizerTest, QueryRuns) {
  // Create fake with server limit max of 2
  mako::fake_google3_storage::Storage fake(10, 10, 10, 10, 2, 10);
  mako::internal::AnalyzerOptimizer opt(&fake, mako::RunBundle());

  // Create 5 runs
  for (int i = 0; i < 5; ++i) {
    mako::RunInfo run;
    run.set_benchmark_key("b");
    run.set_timestamp_ms(i);
    mako::CreationResponse create_response;
    ASSERT_TRUE(fake.CreateRunInfo(run, &create_response));
  }

  // Query without limit should return all 5 results, since not specifying
  // a limit defaults to a limit of 100
  mako::RunInfoQuery query;
  query.set_benchmark_key("b");
  mako::RunInfoQueryResponse response;
  ASSERT_EQ("", QueryRuns(&opt, query, &response));
  ASSERT_EQ(5, response.run_info_list_size());
  ASSERT_EQ(4, response.run_info_list(0).timestamp_ms());
  ASSERT_EQ(3, response.run_info_list(1).timestamp_ms());
  ASSERT_EQ(2, response.run_info_list(2).timestamp_ms());
  ASSERT_EQ(1, response.run_info_list(3).timestamp_ms());
  ASSERT_EQ(0, response.run_info_list(4).timestamp_ms());

  // Query with limit of 1 (no cursor use) should return latest 1 result
  query.set_limit(1);
  ASSERT_EQ("", QueryRuns(&opt, query, &response));
  ASSERT_EQ(1, response.run_info_list_size());
  ASSERT_EQ(4, response.run_info_list(0).timestamp_ms());

  // Query with limit of 3 (forces cursor use) should return latest 3 results
  query.set_limit(3);
  ASSERT_EQ("", QueryRuns(&opt, query, &response));
  ASSERT_EQ(3, response.run_info_list_size());
  ASSERT_EQ(4, response.run_info_list(0).timestamp_ms());
  ASSERT_EQ(3, response.run_info_list(1).timestamp_ms());
  ASSERT_EQ(2, response.run_info_list(2).timestamp_ms());

  // Query with limit of 10 (cursor empty on last result) should return all
  query.set_limit(10);
  ASSERT_EQ("", QueryRuns(&opt, query, &response));
  ASSERT_EQ(5, response.run_info_list_size());
  ASSERT_EQ(4, response.run_info_list(0).timestamp_ms());
  ASSERT_EQ(3, response.run_info_list(1).timestamp_ms());
  ASSERT_EQ(2, response.run_info_list(2).timestamp_ms());
  ASSERT_EQ(1, response.run_info_list(3).timestamp_ms());
  ASSERT_EQ(0, response.run_info_list(4).timestamp_ms());
}

TEST_F(AnalyzerOptimizerTest, QueryRunsNoLimit) {
  // Create fake with server limit max of 150
  mako::fake_google3_storage::Storage fake(10, 10, 10, 10, 150, 10);
  fake.FakeClear();
  mako::internal::AnalyzerOptimizer opt(&fake, mako::RunBundle());
  // Create runs
  for (int i=0; i < 150; ++i) {
    mako::RunInfo run;
    run.set_benchmark_key("b");
    run.set_timestamp_ms(i);
    mako::CreationResponse create_response;
    ASSERT_TRUE(fake.CreateRunInfo(run, &create_response));
  }

  // Query without limit should return 100 results.
  mako::RunInfoQuery query;
  query.set_benchmark_key("b");
  mako::RunInfoQueryResponse response;
  ASSERT_EQ("", QueryRuns(&opt, query, &response));
  ASSERT_EQ(100, response.run_info_list_size());
}

TEST_F(AnalyzerOptimizerTest, QueryRunsLimit0) {
  // Create fake with server limit max of 150
  mako::fake_google3_storage::Storage fake(10, 10, 10, 10, 150, 10);
  fake.FakeClear();
  mako::internal::AnalyzerOptimizer opt(&fake, mako::RunBundle());
  // Create runs
  for (int i=0; i < 150; ++i) {
    mako::RunInfo run;
    run.set_benchmark_key("b");
    run.set_timestamp_ms(i);
    mako::CreationResponse create_response;
    ASSERT_TRUE(fake.CreateRunInfo(run, &create_response));
  }

  // Query without limit should return 100 results.
  mako::RunInfoQuery query;
  query.set_benchmark_key("b");
  query.set_limit(0);
  mako::RunInfoQueryResponse response;
  ASSERT_EQ("", QueryRuns(&opt, query, &response));
  ASSERT_EQ(100, response.run_info_list_size());
}

TEST_F(AnalyzerOptimizerTest, QueryRunsLimit1000) {
  // Create fake with server limit max of 150
  mako::fake_google3_storage::Storage fake(10, 10, 10, 10, 150, 10);
  fake.FakeClear();
  mako::internal::AnalyzerOptimizer opt(&fake, mako::RunBundle());
  // Create runs
  for (int i=0; i < 1100; ++i) {
    mako::RunInfo run;
    run.set_benchmark_key("b");
    run.set_timestamp_ms(i);
    mako::CreationResponse create_response;
    ASSERT_TRUE(fake.CreateRunInfo(run, &create_response));
  }

  // Query without limit should return 100 results.
  mako::RunInfoQuery query;
  query.set_benchmark_key("b");
  query.set_limit(1000);
  mako::RunInfoQueryResponse response;
  ASSERT_EQ("", QueryRuns(&opt, query, &response));
  ASSERT_EQ(1000, response.run_info_list_size());
}

TEST_F(AnalyzerOptimizerTest, QueryRunsInvalidLimit) {
  // Create fake with server limit max of 150
  mako::fake_google3_storage::Storage fake(10, 10, 10, 10, 150, 10);
  fake.FakeClear();
  mako::internal::AnalyzerOptimizer opt(&fake, mako::RunBundle());

  // Create one run
  mako::RunInfo run;
  run.set_benchmark_key("b");
  run.set_timestamp_ms(1);
  mako::CreationResponse create_response;
  ASSERT_TRUE(fake.CreateRunInfo(run, &create_response));

  mako::RunInfoQuery query;
  mako::RunInfoQueryResponse response;
  query.set_benchmark_key("b");
  // Negative limits are invalid
  query.set_limit(-1);
  ASSERT_THAT(QueryRuns(&opt, query, &response),
              HasSubstr("Query limit is not in range [0,1000]"));

  // Limits bigger than 1000 are invalid
  query.set_limit(1001);
  ASSERT_THAT(QueryRuns(&opt, query, &response),
              HasSubstr("Query limit is not in range [0,1000]"));

  // Check that a valid limit still works
  query.set_limit(500);
  ASSERT_EQ("", QueryRuns(&opt, query, &response));
  ASSERT_EQ(1, response.run_info_list_size());
}

}  // namespace internal
}  // namespace mako
