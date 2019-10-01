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

#include "internal/cxx/utils/stringutil.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace mako {
namespace internal {
namespace {

using ::testing::IsEmpty;
using ::testing::StrEq;

TEST(StringutilTest, Empty) {
  absl::string_view v;
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), IsEmpty());
}

TEST(StringutilTest, BasicallyEmpty) {
  std::string s = "\n";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), IsEmpty());
}

TEST(StringutilTest, Newline) {
  std::string s = "\nTest";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, NewlineRight) {
  std::string s = "Test\n";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, Tab) {
  std::string s = "\tTest";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, TabRight) {
  std::string s = "Test\t";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, CarriageReturn) {
  std::string s = "\r\nTest";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, CarriageReturnRight) {
  std::string s = "Test\r\n";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, Multiple) {
  std::string s = "\t\tTest";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, MultipleRight) {
  std::string s = "Test\r\n\r\n";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, Both) {
  std::string s = "\nTest\n";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, BothMultiple) {
  std::string s = "\t\tTest\t\t";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Test"));
}

TEST(StringutilTest, NotMiddle) {
  std::string s = "\nTe\nst\n";
  absl::string_view v(s);
  TrimWhitespace(&v);
  EXPECT_THAT(std::string(v), StrEq("Te\nst"));
}

}  // namespace
}  // namespace internal
}  // namespace mako
