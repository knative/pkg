// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
#include "internal/cxx/storage_client/url.h"

#include "gmock/gmock.h"
#include "gtest/gtest.h"
#include "absl/strings/str_cat.h"
#include "absl/types/optional.h"
#include "helpers/cxx/status/status.h"
#include "helpers/cxx/status/status_matchers.h"

namespace mako {
namespace internal {
namespace {

using ::mako::helpers::IsOkAndHolds;
using ::mako::helpers::StatusIs;
using ::testing::Eq;
using ::testing::Property;
using ::testing::StrEq;

TEST(UrlTest, ParsesScheme) {
  EXPECT_THAT(Url::Parse("http://google.com"),
              IsOkAndHolds(Property(&Url::Scheme, Eq("http"))));
}

TEST(UrlTest, ParsesSchemeOther) {
  EXPECT_THAT(Url::Parse("httwhat://google.com"),
              IsOkAndHolds(Property(&Url::Scheme, Eq("httwhat"))));
}

TEST(UrlTest, ImplicitSchemeIsHttps) {
  EXPECT_THAT(Url::Parse("google.com"),
              IsOkAndHolds(Property(&Url::Scheme, Eq("https"))));
}

TEST(UrlTest, ParsesHost) {
  EXPECT_THAT(Url::Parse("http://google.com"),
              IsOkAndHolds(Property(&Url::Host, Eq("google.com"))));
}

TEST(UrlTest, HostWithDash) {
  EXPECT_THAT(Url::Parse("http://go-ogle.com"),
              IsOkAndHolds(Property(&Url::Host, Eq("go-ogle.com"))));
}

TEST(UrlTest, SingleCharHostPlusDomain) {
  EXPECT_THAT(Url::Parse("http://g.com"),
              IsOkAndHolds(Property(&Url::Host, Eq("g.com"))));
}

TEST(UrlTest, SingleCharHost) {
  EXPECT_THAT(Url::Parse("http://g"),
              IsOkAndHolds(Property(&Url::Host, Eq("g"))));
}

TEST(UrlTest, SingleCharHostNoScheme) {
  EXPECT_THAT(Url::Parse("g"), IsOkAndHolds(Property(&Url::Host, Eq("g"))));
}

TEST(UrlTest, ParsesMoreHost) {
  EXPECT_THAT(Url::Parse("http://more.google.com"),
              IsOkAndHolds(Property(&Url::Host, Eq("more.google.com"))));
}

TEST(UrlTest, ParsesPort) {
  EXPECT_THAT(Url::Parse("http://google.com:8080"),
              IsOkAndHolds(Property(&Url::Port, Eq(8080))));
}

TEST(UrlTest, ParsesPortWithPath) {
  EXPECT_THAT(Url::Parse("http://google.com:8080/somepath"),
              IsOkAndHolds(AllOf(Property(&Url::Port, Eq(8080)),
                                 Property(&Url::Path, Eq("/somepath")))));
}

TEST(UrlTest, ImplicitHttpPort) {
  EXPECT_THAT(Url::Parse("http://google.com"),
              IsOkAndHolds(Property(&Url::Port, Eq(80))));
}

TEST(UrlTest, ImplicitHttpsPort) {
  EXPECT_THAT(Url::Parse("https://google.com"),
              IsOkAndHolds(Property(&Url::Port, Eq(443))));
}

TEST(UrlTest, NoImplicitPort) {
  EXPECT_THAT(Url::Parse("httwhat://google.com"),
              IsOkAndHolds(Property(&Url::Port, Eq(absl::nullopt))));
}

TEST(UrlTest, ParsesPath) {
  EXPECT_THAT(Url::Parse("http://google.com/search"),
              IsOkAndHolds(Property(&Url::Path, Eq("/search"))));
}

TEST(UrlTest, ParsesPathTrailingSlash) {
  EXPECT_THAT(Url::Parse("http://google.com/search/"),
              IsOkAndHolds(Property(&Url::Path, Eq("/search/"))));
}

TEST(UrlTest, ParsesMorePath) {
  EXPECT_THAT(Url::Parse("http://google.com/search/more"),
              IsOkAndHolds(Property(&Url::Path, Eq("/search/more"))));
}

TEST(UrlTest, ParsesMorePathTrailingSlash) {
  EXPECT_THAT(Url::Parse("http://google.com/search/more/"),
              IsOkAndHolds(Property(&Url::Path, Eq("/search/more/"))));
}

TEST(UrlTest, ParsesHostWithSlashPath) {
  EXPECT_THAT(Url::Parse("http://google.com/"),
              IsOkAndHolds(Property(&Url::Path, Eq("/"))));
}

TEST(UrlTest, ParsesNoPathDefaultToSlash) {
  EXPECT_THAT(Url::Parse("http://google.com"),
              IsOkAndHolds(Property(&Url::Path, Eq("/"))));
}

TEST(UrlTest, ParsesQuery) {
  EXPECT_THAT(Url::Parse("http://google.com/?option=1"),
              IsOkAndHolds(Property(&Url::Query, Eq("option=1"))));
}

TEST(UrlTest, ParsesMoreQuery) {
  constexpr absl::string_view q =
      "option%5B1%5D%3D2&option%5B2%5D%3D%22test%20spaces%22";
  EXPECT_THAT(Url::Parse(absl::StrCat("http://google.com/?", q)),
              IsOkAndHolds(Property(&Url::Query, Eq(q))));
}

TEST(UrlTest, ParsesQueryWithNoTrailingSlash) {
  EXPECT_THAT(Url::Parse("http://google.com?option=1"),
              IsOkAndHolds(Property(&Url::Query, Eq("option=1"))));
}

TEST(UrlTest, ParsesPathWithQuery) {
  EXPECT_THAT(Url::Parse("http://google.com/search?option=1"),
              IsOkAndHolds(Property(&Url::Path, Eq("/search"))));
  EXPECT_THAT(Url::Parse("http://google.com/search?option=1"),
              IsOkAndHolds(Property(&Url::Query, Eq("option=1"))));
}

TEST(UrlTest, ToString) {
  constexpr absl::string_view url = "http://google.com/search?option=1";
  EXPECT_THAT(Url::Parse(url), IsOkAndHolds(Property(&Url::ToString, Eq(url))));
}

TEST(UrlTest, ToStringWithPort) {
  constexpr absl::string_view url = "http://google.com:8080/search?option=1";
  EXPECT_THAT(Url::Parse(url), IsOkAndHolds(Property(&Url::ToString, Eq(url))));
}

TEST(UrlTest, ToStringWithDefaultPortHttp) {
  constexpr absl::string_view url = "http://google.com:80/search?option=1";
  constexpr absl::string_view want_url = "http://google.com/search?option=1";
  EXPECT_THAT(Url::Parse(url),
              IsOkAndHolds(Property(&Url::ToString, Eq(want_url))));
}

TEST(UrlTest, ToStringWithDefaultPortHttps) {
  constexpr absl::string_view url = "https://google.com:443/search?option=1";
  constexpr absl::string_view want_url = "https://google.com/search?option=1";
  EXPECT_THAT(Url::Parse(url),
              IsOkAndHolds(Property(&Url::ToString, Eq(want_url))));
}

TEST(UrlTest, ToStringImplicitHost) {
  EXPECT_THAT(
      Url::Parse("google.com/search"),
      IsOkAndHolds(Property(&Url::ToString, Eq("https://google.com/search"))));
}

TEST(UrlTest, ToStringAddsDefaultPathSlash) {
  EXPECT_THAT(Url::Parse("http://google.com"),
              IsOkAndHolds(Property(&Url::ToString, Eq("http://google.com/"))));
}

TEST(UrlTest, ParsesIpv6) {
  EXPECT_THAT(Url::Parse("http://[::1]:333/somepath"),
              IsOkAndHolds(Property(&Url::Host, Eq("[::1]"))));
}

TEST(UrlTest, ParsesIpv6WithPortAndPath) {
  EXPECT_THAT(Url::Parse("http://[::1]:333/somepath"),
              IsOkAndHolds(AllOf(Property(&Url::Host, Eq("[::1]")),
                                 Property(&Url::Port, Eq(333)),
                                 Property(&Url::Path, Eq("/somepath")))));
}

TEST(UrlTest, BadUrlNoScheme) {
  EXPECT_THAT(Url::Parse("://google.com"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
}

TEST(UrlTest, BadUrlNoHost) {
  EXPECT_THAT(Url::Parse("http://"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
  EXPECT_THAT(Url::Parse(""), StatusIs(helpers::StatusCode::kInvalidArgument));
}

TEST(UrlTest, BadUrlUnmatchedIpv6) {
  EXPECT_THAT(Url::Parse("http://[::1"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
}

TEST(UrlTest, BadUrlUnrecognizedHost) {
  EXPECT_THAT(Url::Parse("http:///"),  // NOTYPO
              StatusIs(helpers::StatusCode::kInvalidArgument));
  EXPECT_THAT(Url::Parse("/"), StatusIs(helpers::StatusCode::kInvalidArgument));

  EXPECT_THAT(Url::Parse("http://?"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
  EXPECT_THAT(Url::Parse("?"), StatusIs(helpers::StatusCode::kInvalidArgument));

  EXPECT_THAT(Url::Parse("http://:"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
  EXPECT_THAT(Url::Parse(":"), StatusIs(helpers::StatusCode::kInvalidArgument));
}

TEST(UrlTest, BadUrlNoPort) {
  EXPECT_THAT(Url::Parse("http://google.com:"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
  EXPECT_THAT(Url::Parse("http://google.com:/"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
}

TEST(UrlTest, BadUrlBadPort) {
  EXPECT_THAT(Url::Parse("http://google.com:port"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
  EXPECT_THAT(Url::Parse("http://google.com:-1"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
  EXPECT_THAT(Url::Parse("http://google.com:"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
  EXPECT_THAT(Url::Parse("http://google.com:9a3"),
              StatusIs(helpers::StatusCode::kInvalidArgument));
}

TEST(UrlTest, WithPath) {
  helpers::StatusOr<Url> url = Url::Parse("http://google.com");
  ASSERT_OK(url);
  EXPECT_THAT(url.value().WithPath("somepath"),
              AllOf(Property(&Url::Host, Eq("google.com")),
                    Property(&Url::Path, Eq("/somepath"))));
}

TEST(UrlTest, WithTrailingPath) {
  helpers::StatusOr<Url> url = Url::Parse("http://google.com");
  ASSERT_OK(url);
  EXPECT_THAT(url.value().WithPath("somepath/"),
              AllOf(Property(&Url::Host, Eq("google.com")),
                    Property(&Url::Path, Eq("/somepath/"))));
}

TEST(UrlTest, WithMorePath) {
  helpers::StatusOr<Url> url = Url::Parse("http://google.com");
  ASSERT_OK(url);
  EXPECT_THAT(url.value().WithPath("some/path"),
              AllOf(Property(&Url::Host, Eq("google.com")),
                    Property(&Url::Path, Eq("/some/path"))));
}

TEST(UrlTest, WithPathOverwrites) {
  helpers::StatusOr<Url> url = Url::Parse("http://google.com/oldpath");
  ASSERT_OK(url);
  EXPECT_THAT(url.value().WithPath("newpath"),
              AllOf(Property(&Url::Host, Eq("google.com")),
                    Property(&Url::Path, Eq("/newpath"))));
}

}  // namespace
}  // namespace internal
}  // namespace mako
