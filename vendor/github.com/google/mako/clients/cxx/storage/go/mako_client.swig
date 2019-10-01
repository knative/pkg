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
//
// SWIG wrapping of the Mako client.
//
// This must not be used directly, but rather through
// //testing/performance/mako/clients/go/storage:mako, which provides a
// Go-friendly API and documentation.
%include "swig/base.swig"
%include "swig/go/protofunc.swig"
%include "swig/go/strings.swig"
%import(module="github.com/google/mako/clients/cxx/storage/go/g3storage_wrap") "clients/cxx/storage/go/google3_storage.swig"

%{
#include "clients/cxx/storage/mako_client_internal.h"
%}

%newobject mako::internal::NewMakoClient;

%ignoreall

%unignore mako;
%unignore mako::internal;
%unignore mako::internal::NewMakoClient;

%apply std::string* OUTPUT {std::string *error};

%include "clients/cxx/storage/mako_client_internal.h"
