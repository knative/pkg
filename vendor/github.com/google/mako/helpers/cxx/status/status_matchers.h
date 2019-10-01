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

// Testing utilities for working with ::mako::helpers::Status and
// ::mako::helpers::StatusOr.
//
// Defines the following utilities:
//
//   =================
//   EXPECT_OK(s)
//
//   ASSERT_OK(s)
//   =================
//   Convenience macros for `EXPECT_THAT(s, IsOk())`, where `s` is either
//   a `Status` or a `StatusOr<T>`.
//
//   There are no EXPECT_NOT_OK/ASSERT_NOT_OK macros since they would not
//   provide much value (when they fail, they would just print the OK status
//   which conveys no more information than EXPECT_FALSE(s.ok());
//   If you want to check for particular errors, better alternatives are:
//   EXPECT_THAT(s, StatusIs(expected_error));
//   EXPECT_THAT(s, StatusIs(_, _, HasSubstr("expected error")));
//
//   ===============
//   IsOkAndHolds(m)
//   ===============
//
//   This gMock matcher matches a StatusOr<T> value whose status is OK
//   and whose inner value matches matcher m.  Example:
//
//     using ::testing::MatchesRegex;
//     using ::mako::helpers::IsOkAndHolds;
//     ...
//     StatusOr<string> maybe_name = ...;
//     EXPECT_THAT(maybe_name, IsOkAndHolds(MatchesRegex("John .*")));
//
//   ===============================
//   StatusIs(status_code_matcher,
//            error_message_matcher)
//   ===============================
//
//   This gMock matcher matches a Status or StatusOr<T> or StatusProto value if
//   all of the following are true:
//
//     - the status' error_code() matches status_code_matcher, and
//     - the status' error_message() matches error_message_matcher.
//
//   Example:
//
//     using ::testing::HasSubstr;
//     using ::testing::MatchesRegex;
//     using ::testing::Ne;
//     using ::mako::helpers::StatusIs;
//     using ::testing::_;
//     using ::mako::helpers::StatusOr;
//     StatusOr<string> GetName(int id);
//     ...
//
//     // The status code must be kServerError; the error message can be
//     // anything.
//     EXPECT_THAT(GetName(42), StatusIs(kServerError, _));
//     // The status code can be anything; the error message must match the
//     regex.
//     EXPECT_THAT(GetName(43), StatusIs(_, MatchesRegex("server.*time-out")));
//
//     // The status code
//     // should not be kServerError; the error message can be anything with
//     // "client" in it.
//     EXPECT_CALL(mock_env,
//                 HandleStatus(StatusIs(Ne(kServerError),
//                     HasSubstr("client"))));
//
//   ===============================
//   StatusIs(status_code_matcher)
//   ===============================
//
//   This is a shorthand for
//     StatusIs(status_code_matcher,
//              testing::_)
//   In other words, it's like the two-argument StatusIs(), except that it
//   ignores error message.
//
//   ===============
//   IsOk()
//   ===============
//
//   Matches a mako::helpers::Status or mako::helpers::StatusOr<T> or
//   whose status value is mako:helpers:::Status::OK. Equivalent to
//   'StatusIs(StatusCode::kOk)'. Example:
//     using ::mako::helpers::IsOk;
//     ...
//     StatusOr<string> maybe_name = ...;
//     EXPECT_THAT(maybe_name, IsOk());
//     Status s = ...;
//     EXPECT_THAT(s, IsOk());
#ifndef HELPERS_CXX_STATUS_STATUS_MATCHERS_H_
#define HELPERS_CXX_STATUS_STATUS_MATCHERS_H_

#include <ostream>  // NOLINT
#include <string>
#include <type_traits>

#include "gmock/gmock.h"
#include "helpers/cxx/status/status.h"
#include "helpers/cxx/status/statusor.h"

namespace mako {
namespace helpers {
namespace internal_status {

inline const Status& GetStatus(const Status& status) { return status; }

template <typename T>
inline const Status& GetStatus(const StatusOr<T>& status) {
  return status.status();
}

////////////////////////////////////////////////////////////
// Implementation of IsOkAndHolds().

// Monomorphic implementation of matcher IsOkAndHolds(m).  StatusOrType can be
// either StatusOr<T> or a reference to it.
template <typename StatusOrType>
class IsOkAndHoldsMatcherImpl : public testing::MatcherInterface<StatusOrType> {
 public:
  // Work around google::cloud::StatusOr lacking an element_type member by
  // getting the return type of StatusOrType::ValueOrDie.
  // decltype doesn't evaluate the argument, so casting a nullptr to the correct
  // type and calling ValueOrDie on that result works.
  typedef decltype(
      static_cast<typename std::add_pointer<StatusOrType>::type>(nullptr)
          ->value()) value_type;

  template <typename InnerMatcher>
  explicit IsOkAndHoldsMatcherImpl(InnerMatcher&& inner_matcher)
      : inner_matcher_(testing::SafeMatcherCast<const value_type&>(
            std::forward<InnerMatcher>(inner_matcher))) {}

  void DescribeTo(std::ostream* os) const override {
    *os << "is OK and has a value that ";
    inner_matcher_.DescribeTo(os);
  }

  void DescribeNegationTo(std::ostream* os) const override {
    *os << "isn't OK or has a value that ";
    inner_matcher_.DescribeNegationTo(os);
  }

  bool MatchAndExplain(
      StatusOrType actual_value,
      testing::MatchResultListener* result_listener) const override {
    if (!actual_value.ok()) {
      *result_listener << "which has status " << actual_value.status();
      return false;
    }

    testing::StringMatchResultListener inner_listener;
    const bool matches = inner_matcher_.MatchAndExplain(
        actual_value.value(), &inner_listener);
    const std::string inner_explanation = inner_listener.str();
    if (!inner_explanation.empty()) {
      *result_listener << "which contains value "
                       << testing::PrintToString(actual_value.value())
                       << ", " << inner_explanation;
    }
    return matches;
  }

 private:
  const testing::Matcher<const value_type&> inner_matcher_;
};

// Implements IsOkAndHolds(m) as a polymorphic matcher.
template <typename InnerMatcher>
class IsOkAndHoldsMatcher {
 public:
  explicit IsOkAndHoldsMatcher(InnerMatcher inner_matcher)
      : inner_matcher_(std::move(inner_matcher)) {}

  // Converts this polymorphic matcher to a monomorphic matcher of the
  // given type.  StatusOrType can be either StatusOr<T> or a
  // reference to StatusOr<T>.
  template <typename StatusOrType>
  operator testing::Matcher<StatusOrType>() const {  // NOLINT
    return testing::Matcher<StatusOrType>(
        new IsOkAndHoldsMatcherImpl<StatusOrType>(inner_matcher_));
  }

 private:
  const InnerMatcher inner_matcher_;
};

////////////////////////////////////////////////////////////
// Implementation of StatusIs().

// `StatusCode` is implicitly convertible from `int` and `mako::StatusCode`,
// and explicitly convertible to these types as well.
//
// We need this class because mako::StatusCode (as a scoped enum) is not
// implicitly convertible to int. In order to handle use cases like
// StatusIs(Anyof(mako::StatusCode::kUnknown,
// mako::StatusCode::kCancelled)) which uses polymorphic matchers, we need
// to unify the interfaces into Matcher<StatusCode>.
class StatusCode {
 public:
  StatusCode(int code) : code_(code) {}
  StatusCode(::mako::helpers::StatusCode code)
      : code_(static_cast<int>(code)) {}
  template <typename T>
  StatusCode(T code)  // NOLINT
      : code_(static_cast<int>(code)) {}
  explicit operator int() const { return code_; }
  explicit operator ::mako::helpers::StatusCode() const {
    return static_cast<::mako::helpers::StatusCode>(code_);
  }
  template <typename T>
  explicit operator T() const {
    return static_cast<T>(code_);
  }

  friend inline void PrintTo(const StatusCode& code, std::ostream* os) {
    *os << code.code_;
  }

 private:
  int code_;
};

// Relational operators to handle matchers like Eq, Lt, etc..
inline bool operator==(const StatusCode& lhs, const StatusCode& rhs) {
  return static_cast<int>(lhs) == static_cast<int>(rhs);
}
inline bool operator!=(const StatusCode& lhs, const StatusCode& rhs) {
  return static_cast<int>(lhs) != static_cast<int>(rhs);
}
inline bool operator<(const StatusCode& lhs, const StatusCode& rhs) {
  return static_cast<int>(lhs) < static_cast<int>(rhs);
}
inline bool operator<=(const StatusCode& lhs, const StatusCode& rhs) {
  return static_cast<int>(lhs) <= static_cast<int>(rhs);
}
inline bool operator>(const StatusCode& lhs, const StatusCode& rhs) {
  return static_cast<int>(lhs) > static_cast<int>(rhs);
}
inline bool operator>=(const StatusCode& lhs, const StatusCode& rhs) {
  return static_cast<int>(lhs) >= static_cast<int>(rhs);
}

// StatusIs() is a polymorphic matcher.  This class is the common
// implementation of it shared by all types T where StatusIs() can be
// used as a Matcher<T>.
class StatusIsMatcherCommonImpl {
 public:
  StatusIsMatcherCommonImpl(testing::Matcher<StatusCode> code_matcher,
                            testing::Matcher<const std::string&> message_matcher)
      : code_matcher_(std::move(code_matcher)),
        message_matcher_(std::move(message_matcher)) {}

  void DescribeTo(std::ostream* os) const;

  void DescribeNegationTo(std::ostream* os) const;

  bool MatchAndExplain(const ::mako::helpers::Status& status,
                       testing::MatchResultListener* result_listener) const;

 private:
  const testing::Matcher<StatusCode> code_matcher_;
  const testing::Matcher<const std::string&> message_matcher_;
};

// Monomorphic implementation of matcher StatusIs() for a given type
// T.  T can be Status, StatusOr<>, or a reference to either of them.
template <typename T>
class MonoStatusIsMatcherImpl : public testing::MatcherInterface<T> {
 public:
  explicit MonoStatusIsMatcherImpl(StatusIsMatcherCommonImpl common_impl)
      : common_impl_(std::move(common_impl)) {}

  void DescribeTo(std::ostream* os) const override {
    common_impl_.DescribeTo(os);
  }

  void DescribeNegationTo(std::ostream* os) const override {
    common_impl_.DescribeNegationTo(os);
  }

  bool MatchAndExplain(
      T actual_value,
      testing::MatchResultListener* result_listener) const override {
    return common_impl_.MatchAndExplain(GetStatus(actual_value),
                                        result_listener);
  }

 private:
  StatusIsMatcherCommonImpl common_impl_;
};

// Implements StatusIs() as a polymorphic matcher.
class StatusIsMatcher {
 public:
  template <typename StatusCodeMatcher, typename StatusMessageMatcher>
  StatusIsMatcher(StatusCodeMatcher&& code_matcher,
                  StatusMessageMatcher&& message_matcher)
      : common_impl_(testing::MatcherCast<StatusCode>(
                         std::forward<StatusCodeMatcher>(code_matcher)),
                     testing::MatcherCast<const std::string&>(
                         std::forward<StatusMessageMatcher>(message_matcher))) {
  }

  // Converts this polymorphic matcher to a monomorphic matcher of the
  // given type.  T can be StatusOr<>, Status, or a reference to
  // either of them.
  template <typename T>
  operator testing::Matcher<T>() const {  // NOLINT
    return testing::Matcher<T>(new MonoStatusIsMatcherImpl<T>(common_impl_));
  }

 private:
  const StatusIsMatcherCommonImpl common_impl_;
};

// Monomorphic implementation of matcher IsOk() for a given type T.
// T can be Status, StatusOr<>, or a reference to either of them.
template <typename T>
class MonoIsOkMatcherImpl : public testing::MatcherInterface<T> {
 public:
  void DescribeTo(std::ostream* os) const override { *os << "is OK"; }
  void DescribeNegationTo(std::ostream* os) const override {
    *os << "is not OK";
  }
  bool MatchAndExplain(T actual_value,
                       testing::MatchResultListener*) const override {
    return GetStatus(actual_value).ok();
  }
};

// Implements IsOk() as a polymorphic matcher.
class IsOkMatcher {
 public:
  template <typename T>
  operator testing::Matcher<T>() const {  // NOLINT
    return testing::Matcher<T>(new MonoIsOkMatcherImpl<T>());
  }
};

}  // namespace internal_status

// Macros for testing the results of functions that return
// mako::helpers::Status or mako::helpers::StatusOr<T> (for any type T).
#ifndef EXPECT_OK
#define EXPECT_OK(expression) \
  EXPECT_THAT(expression, ::mako::helpers::IsOk())
#endif  // EXPECT_OK
#ifndef ASSERT_OK
#define ASSERT_OK(expression) \
  ASSERT_THAT(expression, ::mako::helpers::IsOk())
#endif  // ASSERT_OK

// Returns a gMock matcher that matches a StatusOr<> whose status is
// OK and whose value matches the inner matcher.
template <typename InnerMatcher>
internal_status::IsOkAndHoldsMatcher<typename std::decay<InnerMatcher>::type>
IsOkAndHolds(InnerMatcher&& inner_matcher) {
  return internal_status::IsOkAndHoldsMatcher<
      typename std::decay<InnerMatcher>::type>(
      std::forward<InnerMatcher>(inner_matcher));
}

// The one and two-arg StatusIs methods may infer the expected ErrorSpace from
// the StatusCodeMatcher argument. If you call StatusIs(e) or StatusIs(e, msg)
// and the argument `e` is:
// - an enum type,
// - which is associated with a custom ErrorSpace `S`,
// - and is not "OK" (i.e. 0),
// then the matcher will match a Status or StatusOr<> whose error space is `S`.
//
// Otherwise, the expected error space is the canonical error space.

// Returns a gMock matcher that matches a Status or StatusOr<> whose error space
// is the inferred error space (see above), whose status code matches
// code_matcher, and whose error message matches message_matcher.
template <typename StatusCodeMatcher, typename StatusMessageMatcher>
internal_status::StatusIsMatcher StatusIs(
    StatusCodeMatcher&& code_matcher, StatusMessageMatcher&& message_matcher) {
  return internal_status::StatusIsMatcher(
      std::forward<StatusCodeMatcher>(code_matcher),
      std::forward<StatusMessageMatcher>(message_matcher));
}

// Returns a gMock matcher that matches a Status or StatusOr<> whose error space
// is the inferred error space (see above), and whose status code matches
// code_matcher.
template <typename StatusCodeMatcher>
internal_status::StatusIsMatcher StatusIs(StatusCodeMatcher&& code_matcher) {
  return StatusIs(std::forward<StatusCodeMatcher>(code_matcher), testing::_);
}

// Returns a gMock matcher that matches a Status or StatusOr<> which is OK.
inline internal_status::IsOkMatcher IsOk() {
  return internal_status::IsOkMatcher();
}

}  // namespace helpers
}  // namespace mako

#endif  // HELPERS_CXX_STATUS_STATUS_MATCHERS_H_
