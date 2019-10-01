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

// This file contains SWIG bindings to wrap C++ functions with
// protocol buffer arguments, for use in Go.
//
// The main goal is to decouple C++ and Go protocol buffer usage,
// allowing Go code to continue to use Go protocol buffers
// instead of SWIG wrapped protocol buffers.
//
// It achieves this goal by serializing/deserializing the protocol
// buffers when communicating with C++ code.
// Serialization may impose a slight performance penalty; however,
// this is always faster than RPC as no transmission occurs.
//
// The following macros are supported.
// CMsgType refers to a C++ protocol buffer, fully qualified with namespace,
// GoMsgPkg refers to a BUILD target of the Go protocol buffer,
// GoMsgType refers to a Go protocol buffer type: packageName.TypeName.
//
// PROTO_INPUT(CMsgType, GoMsgPkg, GoMsgType)
// PROTO_OUTPUT(CMsgType, GoMsgPkg, GoMsgType)
// PROTO_IN_OUT(CMsgType, GoMsgPkg, GoMsgType)
// PROTO_RETURN(CMsgType, GoMsgPkg, GoMsgType)
// PROTO_ENUM(CMsgType, GoMsgPkg, GoMsgType)
//
// NOTES:
// - Only one of PROTO_INPUT, PROTO_OUTPUT, PROTO_IN_OUT may be used for a
//   given proto type. You may combine any one of those with PROTO_RETURN.
//   PROTO_INPUT can be combined with PROTO_OUTPUT/PROTO_IN_OUT as long as the
//   input parameter is passed by value or by a const pointer (since strictly
//   speaking the parameters are not of the same type in this case).
// - For getting/setting fields of a struct, use PROTO_RETURN.
// - For all of the above, the Go interface will accept/return a pointer to
//   GoMsgType.
// - The mako_go_wrap_cc target that uses this file will have to have the following
//   dependency: "@com_github_golang_protobuf//proto:go_default_library".
//
// Example:
//   .h file:
//     const ns::MyProto1& SomeFunc(const ns::MyProto2& myInput,
//                                  ns::MyProto3* mutate,
//                                  ns::MyProto4* myOutput);
//   .swig file:
//     PROTO_RETURN(ns::MyProto1,
//                  pb "github.com/google/mako/path/to/my/proto/target_go_proto",
//                  pb.MyProto1)
//     PROTO_INPUT(ns::MyProto2,
//                 pb "github.com/google/mako/path/to/my/proto/target_go_proto",
//                 pb.MyProto2)
//     PROTO_IN_OUT(ns::MyProto3,
//                  pb "github.com/google/mako/path/to/my/proto/target_go_proto",
//                  pb.MyProto3)
//     PROTO_OUTPUT(ns::MyProto4,
//                  pb "github.com/google/mako/path/to/my/proto/target_go_proto",
//                  pb.MyProto4)
//     PROTO_ENUM(ns::MyProto5_MyEnum,
//                pb "github.com/google/mako/path/to/my/proto/target_go_proto",
//                pb.MyProto5_MyEnum)
//   Generated .go file:
//     func SomeFunc(myInput *pb.MyProto2,
//                   mutate *pb.MyProto3,
//                   myOutput *pb.MyProto4) *pb.MyProto1
//     (and a pb.MyProto5_MyEnum can be passed in/out or returned)


%fragment("ProtofuncToGoString", "header", fragment="AllocateString") %{
namespace {

_gostring_ ProtofuncToGoString(google::protobuf::Message* pb) {
  if (pb == nullptr) {
    return Swig_AllocateString(nullptr, 0);
  }
  std::string proto_string = " " + pb->SerializePartialAsString();
  return Swig_AllocateString(proto_string.data(), proto_string.length());
}

_gostring_ ProtofuncToGoString(const google::protobuf::Message& pb) {
  std::string proto_string = " " + pb.SerializePartialAsString();
  return Swig_AllocateString(proto_string.data(), proto_string.length());
}

}  // namespace
%}


%define PROTO_RETURN(CMsgType, GoMsgPkg, GoMsgType)

%go_import(GoMsgPkg, "github.com/golang/protobuf/proto")

%typemap(gotype) PROTO_RETURN_TYPE_ALL "*"#GoMsgType
%typemap(imtype) PROTO_RETURN_TYPE* "*string"
%typemap(imtype) PROTO_RETURN_TYPE_NON_PTR "string"

%typemap(out, fragment="ProtofuncToGoString") PROTO_RETURN_TYPE* {
  $result = (_gostring_*) malloc(sizeof(_gostring_));
  _gostring_ tmp = ProtofuncToGoString($1);
  memcpy($result, &tmp, sizeof(_gostring_));
}

%typemap(out, fragment="ProtofuncToGoString") PROTO_RETURN_TYPE_NON_PTR {
  $result = ProtofuncToGoString($1);
}

// Support %newobject to dispose of original return value.
%typemap(newfree) PROTO_RETURN_TYPE* {
  delete $1;
}

%typemap(goout, fragment="CopyString") PROTO_RETURN_TYPE* %{
  {
    defer Swig_free((uintptr)(unsafe.Pointer($1)))
    s := swigCopyString(*$1)
    if s != "" {
      $result = new(GoMsgType)
      if err := proto.Unmarshal([]byte(s[1:]), $result); err != nil {
        // We accept partial messages here.
        if _, ok := err.(*proto.RequiredNotSetError); !ok {
          panic(err)
        }
      }
    }
  }
%}

%typemap(goout, fragment="CopyString") PROTO_RETURN_TYPE_NON_PTR %{
  {
    s := swigCopyString($1)
    if s != "" {
      $result = new(GoMsgType)
      if err := proto.Unmarshal([]byte(s[1:]), $result); err != nil {
        // We accept partial messages here.
        if _, ok := err.(*proto.RequiredNotSetError); !ok {
          panic(err)
        }
      }
    }
  }
%}

%typemap(goin) PROTO_RETURN_TYPE %{
  {
    var err error
    var b []byte
    if b, err = proto.Marshal($input); err != nil {
      // We accept partial messages here.
      if _, ok := err.(*proto.RequiredNotSetError); !ok {
        panic(err)
      }
    }
    $result = string(b)
  }
%}

%typemap(in) PROTO_RETURN_TYPE %{
  $1.ParsePartialFromArray($input.p, $input.n);
%}

%apply PROTO_RETURN_TYPE { CMsgType }
%apply PROTO_RETURN_TYPE* { const CMsgType*, CMsgType* }
%apply PROTO_RETURN_TYPE_NON_PTR { CMsgType, const CMsgType&, CMsgType& }
%apply PROTO_RETURN_TYPE_ALL {
    CMsgType, const CMsgType*, CMsgType*, const CMsgType&, CMsgType& }

%enddef // end PROTO_RETURN


%define PROTO_INPUT(CMsgType, GoMsgPkg, GoMsgType)

%go_import(GoMsgPkg, "github.com/golang/protobuf/proto")

%typemap(gotype) PROTO_INPUT_TYPE_ALL "*"#GoMsgType
%typemap(imtype) PROTO_INPUT_TYPE_ALL "string"

%typemap(goin) PROTO_INPUT_TYPE_ALL %{
  {
    var b []byte
    var err error
    if b, err = proto.Marshal($input); err != nil {
      // We accept partial messages here.
      if _, ok := err.(*proto.RequiredNotSetError); !ok {
        panic(err)
      }
    }
    $result = string(b)
  }
%}

%typemap(in) PROTO_INPUT_TYPE %{
  $1.ParsePartialFromArray((char *)$input.p, $input.n);
%}

%typemap(in) PROTO_INPUT_TYPE_PTR (CMsgType t) %{
  t.ParsePartialFromArray((char *)$input.p, $input.n);
  $1 = &t;
%}

// explicitly set to empty so const pointers don't get the typemap for normal
// pointers that may be defined in PROTO_OUTPUT/IN_OUT
%typemap(argout) PROTO_INPUT_TYPE_CONST_PTR ""
%typemap(goargout) PROTO_INPUT_TYPE_CONST_PTR ""

%apply PROTO_INPUT_TYPE { CMsgType }
%apply PROTO_INPUT_TYPE_PTR {
    const CMsgType*, CMsgType*, const CMsgType&, CMsgType& }
%apply PROTO_INPUT_TYPE_CONST_PTR {
    const CMsgType*, const CMsgType& }
%apply PROTO_INPUT_TYPE_ALL {
    CMsgType, const CMsgType*, CMsgType*, const CMsgType&, CMsgType& }

%enddef // end PROTO_INPUT


%define PROTO_OUTPUT(CMsgType, GoMsgPkg, GoMsgType)

%go_import(GoMsgPkg, "github.com/golang/protobuf/proto")

%typemap(gotype) PROTO_OUTPUT_TYPE* "*"#GoMsgType
%typemap(imtype) PROTO_OUTPUT_TYPE* "*string"

%typemap(in) PROTO_OUTPUT_TYPE* (CMsgType t) {
  $1 = &t;
}

%typemap(argout, fragment="ProtofuncToGoString") PROTO_OUTPUT_TYPE* {
  *$input = ProtofuncToGoString($1);
}

%typemap(goargout, fragment="CopyString") PROTO_OUTPUT_TYPE* %{
  {
    s := swigCopyString(*$input)
    if s != "" {
      if err := proto.Unmarshal([]byte(s[1:]), $1); err != nil {
        // We accept partial messages here.
        if _, ok := err.(*proto.RequiredNotSetError); !ok {
          panic(err)
        }
      }
    }
  }
%}

%typemap(goin) PROTO_OUTPUT_TYPE* %{
  $result = new(string)
%}

%apply PROTO_OUTPUT_TYPE* { CMsgType* }

%enddef // end PROTO_OUTPUT


%define PROTO_IN_OUT(CMsgType, GoMsgPkg, GoMsgType)

%go_import(GoMsgPkg, "github.com/golang/protobuf/proto")

%typemap(gotype) PROTO_IN_OUT_TYPE* "*"#GoMsgType
%typemap(imtype) PROTO_IN_OUT_TYPE* "*string"

// We can't pass the value down as a pointer to a Go string, since
// that would require a pointer to a pointer, which is forbidden by
// the cgo pointer passing rules.  Instead we construct a byte array
// with an 8 byte length followed by the actual data, and type convert
// that to a string pointer.
%typemap(goin) PROTO_IN_OUT_TYPE* %{
  {
    var b []byte
    var err error
    if b, err = proto.Marshal($input); err != nil {
      // We accept partial messages here.
      if _, ok := err.(*proto.RequiredNotSetError); !ok {
        panic(err)
      }
    }
    v := len(b)

    // We need to allocate enough space here for the argout typemap
    // to store a string value.
    c := uintptr(v + 8)
    if c < unsafe.Sizeof("") {
      c = unsafe.Sizeof("")
    }
    b1 := make([]byte, 8, c)
    for i := 0; i < 8; i++ {
      b1[i] = byte(v & 0xff)
      v >>= 8
    }
    b1 = append(b1, b...)
    $result = (*string)(unsafe.Pointer(&b1[0]))
  }
%}

%typemap(in) PROTO_IN_OUT_TYPE* (CMsgType t) %{
  {
    size_t v = 0;
    const char* s = (const char*)$input;
    for (size_t i = 0; i < 8; i++) {
      if (i < sizeof(v))
        v |= (size_t)*s++ << (i*8);
    }
    t.ParsePartialFromArray((const char*)$input + 8, v);
    $1 = &t;
  }
%}

%typemap(argout, fragment="ProtofuncToGoString") PROTO_IN_OUT_TYPE* {
  *$input = ProtofuncToGoString($1);
}

%typemap(goargout, fragment="CopyString") PROTO_IN_OUT_TYPE* %{
  {
    s := swigCopyString(*$input)
    if s != "" {
      if err := proto.Unmarshal([]byte(s[1:]), $1); err != nil {
        // We accept partial messages here.
        if _, ok := err.(*proto.RequiredNotSetError); !ok {
          panic(err)
        }
      }
    }
  }
%}

%apply PROTO_IN_OUT_TYPE* { CMsgType* }

%enddef // end PROTO_IN_OUT


%define PROTO_ENUM(CMsgType, GoMsgPkg, GoMsgType)

%go_import(GoMsgPkg)

// By value.
%typemap(gotype) PROTO_ENUM_TYPE "GoMsgType"
%typemap(imtype) PROTO_ENUM_TYPE "int"

%typemap(goin) PROTO_ENUM_TYPE %{
  $result = int($input)
%}

%typemap(in) PROTO_ENUM_TYPE %{
  $1 = static_cast<CMsgType>($input);
%}

%typemap(out) PROTO_ENUM_TYPE {
  $result = static_cast<intgo>($1);
}

%typemap(goout) PROTO_ENUM_TYPE {
  $result = GoMsgType($1)
}

%apply PROTO_ENUM_TYPE { CMsgType }

// By pointer.
%typemap(gotype) PROTO_ENUM_TYPE* "*"#GoMsgType
%typemap(imtype) PROTO_ENUM_TYPE* "*int"

%typemap(goin) PROTO_ENUM_TYPE* %{
  if $input != nil {
    res := int(*$input)
    $result = &res
  }
%}

%typemap(in) PROTO_ENUM_TYPE* {
  $1 = reinterpret_cast<CMsgType*>($input);
}

%typemap(out) PROTO_ENUM_TYPE* {
  $result = reinterpret_cast<intgo*>($1);
}

%typemap(goout) PROTO_ENUM_TYPE* {
  if $1 != nil {
    res := GoMsgType(*$1)
    $result = &res
  }
}

%typemap(goargout) PROTO_ENUM_TYPE* %{
  if $input != nil {
    *$1 = GoMsgType(*$input)
  }
%}

%apply PROTO_ENUM_TYPE* { CMsgType*, const CMsgType* }

%enddef // end PROTO_ENUM_INPUT

%inline %{

// A class we use to transfer output strings from C++ to Go.
struct GoProtoSliceSupport {
  std::string s;
};

%}

// Mark the C++ type std::vector<CMsgType> as an input type.
// This macro should be used when a C++ function is taking a vector of
// ProtocolMessages as input.
// Eg.
// void func(const std::vector<ProtoMessage>& msg) {
// }
//
// In the swig file include:
// C_PROTO_VECTOR_INPUT_SLICE(ProtoMessage)
//
// This will generate a go function that takes [][]byte as input.
// The input is a slice of serialized ProtoMessage.
// So you need to write a simple go wrapper to Marshal the ProtoMessage
// and pass it to the generated function.

// Based on the macro above. Maps a Go [][]byte to a std::vector<proto>.
%define C_PROTO_VECTOR_INPUT_SLICE(CMsgType)

// Map C++ input vector of protos to [][]byte on the go end.
%typemap(gotype) C_PROTO_VECTOR_INPUT_SLICE_TYPE "[][]byte"

// Move the data from Go to C++ using a C++ pointer to a GoProtoSliceSupport.
%typemap(imtype) C_PROTO_VECTOR_INPUT_SLICE_TYPE "uintptr"

%typemap(goin, fragment="GoProtoVectorInOut") C_PROTO_VECTOR_INPUT_SLICE_TYPE %{
  {
    p := NewGoProtoSliceSupport()
    defer DeleteGoProtoSliceSupport(p)
    tot := 8
    for _, b := range $input {
      tot += 8 + len(b)
    }
    buf := make([]byte, tot)
    swigPutInt64(buf, len($input))
    off := 8
    for _, b := range $input {
      swigPutInt64(buf[off:], len(b))
      off += 8
      copy(buf[off:], b)
      off += len(b)
    }
    p.SetS(string(buf))
    $result = p.Swigcptr()
  }
%}

%typemap(in, fragment="CProtoVectorInOut")
  C_PROTO_VECTOR_INPUT_SLICE_TYPE (std::vector<CMsgType> t) {
  {
    GoProtoSliceSupport* p = reinterpret_cast<GoProtoSliceSupport*>($input);
    const char* s = p->s.data();
    size_t c = SwigGetInt64(s);
    t.resize(c);
    s += 8;
    for (long i = 0; i < c; i++) {
      size_t slen = SwigGetInt64(s);
      s += 8;
      t[i].ParseFromArray(s, slen);
      s += slen;
    }
  }
  $1 = &t;
}

%apply C_PROTO_VECTOR_INPUT_SLICE_TYPE {const std::vector<CMsgType>& };

%enddef // end C_PROTO_VECTOR_INPUT_SLICE

%fragment("GoProtoVectorInOut", "go_header") %{
func swigPutInt64(buf []byte, v int) {
  for i := 0; i < 8; i++ {
    buf[i] = byte(v & 0xff)
    v >>= 8
  }
}
%}

%fragment("CProtoVectorInOut", "header") %{
static size_t SwigGetInt64(const char *s) {
  size_t v = 0;
  for (size_t i = 0; i < 8; i++) {
    if (i < sizeof(v))
      v |= (size_t)*s++ << (i*8);
  }
  return v;
}
%}
