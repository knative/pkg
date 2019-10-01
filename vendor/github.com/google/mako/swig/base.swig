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

#pragma SWIG nowarn=312,325,451,454,503,362,509

// 312 suppresses warnings about (unnamed) nested classes that SWIG doesn't
// currently support.
// 325 suppresses warnings about named nested classes that SWIG doesn't
// currently support.
// 451 suppresses warnings about setting const char * variable may leak memory.
// 454 suppresses setting global ptr/ref variables may leak memory warning
// 503 suppresses warnings about identifiers that SWIG can't wrap without a
// rename.  For example, an operator< in a class without a rename.
// 362 is similar to 503 but for operator=.
// 509 suppresses warnings about shadowing methods with SWIG-specific ones, or
// when multiple wrapped methods map to the same wrapping method.

%include "typemaps.i"

#if SWIG_VERSION < 0x030000
// Define some C++11 keywords safe to ignore so older SWIG does not choke.
%define constexpr const %enddef
%define final %enddef
%define override %enddef
#endif

// std::string

%include "std_string.i"

namespace std {
  class string;

  %typemap(typecheck) string = char *;
  %typemap(typecheck) const string & = char *;

}

class string;

%typemap(typecheck) string = char *;
%typemap(typecheck) const string & = char *;

%apply std::string { string }
%apply const std::string & { const string & }

%apply const string & {string &};
%apply const string & {string *};

%apply bool * OUTPUT            { bool * OUTPUT2 };
%apply int * OUTPUT             { int * OUTPUT2 };
%apply short * OUTPUT           { short * OUTPUT2 };
%apply long * OUTPUT            { long * OUTPUT2 };
%apply unsigned * OUTPUT        { unsigned * OUTPUT2 };
%apply unsigned short * OUTPUT  { unsigned short * OUTPUT2 };
%apply unsigned long * OUTPUT   { unsigned long * OUTPUT2 };
%apply unsigned char * OUTPUT   { unsigned char * OUTPUT2 };
%apply signed char * OUTPUT     { signed char * OUTPUT2 };
%apply double * OUTPUT          { double * OUTPUT2 };
%apply float * OUTPUT           { float * OUTPUT2 };


%define COPY_TYPEMAPS(oldtype, newtype)
typedef oldtype newtype;
%apply oldtype * OUTPUT { newtype * OUTPUT };
%apply oldtype & OUTPUT { newtype & OUTPUT };
%apply oldtype * INPUT { newtype * INPUT };
%apply oldtype & INPUT { newtype & INPUT };
%apply oldtype * INOUT { newtype * INOUT };
%apply oldtype & INOUT { newtype & INOUT };
%enddef

COPY_TYPEMAPS(signed char, schar);
COPY_TYPEMAPS(unsigned char, uint8);
COPY_TYPEMAPS(short, int16);
COPY_TYPEMAPS(unsigned short, uint16);
COPY_TYPEMAPS(int, int32);
COPY_TYPEMAPS(unsigned int, uint32);
COPY_TYPEMAPS(long long, int64);
COPY_TYPEMAPS(unsigned long long, uint64);

// Since we can't use __SIZE_TYPE__ directly (not defined in SWIG)...
COPY_TYPEMAPS(unsigned long, size_t);
COPY_TYPEMAPS(unsigned int, mode_t);
COPY_TYPEMAPS(long, time_t);
COPY_TYPEMAPS(uint64, Fprint);

#undef COPY_TYPEMAPS

// SWIG macros for explicit API declaration.
// Usage:
//
// %ignoreall
// %unignore SomeName;   // namespace / class / method
// %unignore_method ns::Class::SomeName;  // If you want camel-case someName in Java.
// %include "somelib.h"
// %unignoreall  // mandatory closing "bracket"
%define %ignoreall %ignore ""; %enddef
%define %unignore %rename("%s") %enddef
%define %unignore_method %rename("%(lowercamelcase)s") %enddef
%define %unignoreall %rename("%s") ""; %enddef
