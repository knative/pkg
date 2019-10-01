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

%include "swig/base.swig"
%{
#include <memory>
#include "absl/strings/string_view.h"
%}

%typemap(gotype) std::string * OUTPUT %{[]string%}

%typemap(in) std::string * OUTPUT (std::string temp) {
  if ($input.len == 0) {
    _swig_gopanic("String output array must contain at least one element.");
  }
  $1 = &temp;
}

%typemap(argout,fragment="AllocateString") std::string * OUTPUT {
  _gostring_* a = (_gostring_*) $input.array;
  a[0] = Swig_AllocateString($1->data(), $1->length());
}

%typemap(goargout,fragment="CopyString") std::string * OUTPUT %{
  $1[0] = swigCopyString($input[0])
%}


%typemap(typecheck) absl::string_view = char *;
%typemap(typecheck) const absl::string_view & = char *;

// const absl::string_view& arguments are provided as Go strings.
%typemap(gotype) const absl::string_view& "string"

%typemap(in) const absl::string_view& (absl::string_view spiece) {
  // Avoid using NULL, so that NULL can mean 'no data'.
  if ($input.p == NULL) {
    spiece = absl::string_view("", 0);
  } else {
    spiece = absl::string_view($input.p, $input.n);
  }
  $1 = &spiece;
}

// absl::string_view arguments are provided as Go strings.
%typemap(gotype) absl::string_view "string"

%typemap(in) absl::string_view {
  // Avoid using NULL, so that NULL can mean 'no data'.
  if ($input.p == NULL) {
    $1 = absl::string_view("", 0);
  } else {
    $1 = absl::string_view($input.p, $input.n);
  }
}

%typemap(out,fragment="AllocateString") absl::string_view %{
  $result = Swig_AllocateString($1.data(), $1.length());
%}

%typemap(goout,fragment="CopyString") absl::string_view %{
  $result = swigCopyString($1)
%}

// Definitions for absl::string_view* output parameter.
// Use "%apply absl::string_view *OUTPUT {absl::string_view* actual_arg_name};",
// and pass is a []string with exactly 1 item.
%typemap(gotype) absl::string_view * OUTPUT %{[]string%}

%typemap(in) absl::string_view * OUTPUT (absl::string_view temp) {
  if ($input.len == 0) {
    _swig_gopanic("absl::string_view output array must contain at least one element.");
  }
  $1 = &temp;
}

%typemap(argout,fragment="AllocateString") absl::string_view * OUTPUT {
  _gostring_* a = (_gostring_*) $input.array;
  a[0] = Swig_AllocateString($1->data(), $1->length());
}

%typemap(goargout,fragment="CopyString") absl::string_view * OUTPUT %{
  $1[0] = swigCopyString($input[0])
%}
