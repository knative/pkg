#Google3 implementation of mako downsampler interface.
package(
    default_visibility = ["//visibility:public"],
)

licenses(["notice"])

cc_library(
    name = "standard_downsampler",
    srcs = ["standard_downsampler.cc"],
    hdrs = ["standard_downsampler.h"],
    deps = [
        ":metric_set",
        "//internal/cxx:proto_validation",
        "//internal/cxx/load/common:executor",
        "//internal/proto:mako_internal_cc_proto",
        "//spec/cxx:downsampler",
        "//spec/cxx:fileio",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/algorithm:container",
        "@com_google_absl//absl/base:core_headers",
        "@com_google_absl//absl/memory",
        "@com_google_absl//absl/random",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/synchronization",
        "@com_google_glog//:glog",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_test(
    name = "standard_downsampler_test",
    size = "small",
    srcs = ["standard_downsampler_test.cc"],
    deps = [
        ":standard_downsampler",
        "//clients/cxx/fileio:memory_fileio",
        "//internal/cxx:filter_utils",
        "//internal/cxx:pgmath",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/strings",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_library(
    name = "metric_set",
    hdrs = ["metric_set.h"],
    visibility = ["//visibility:private"],
    deps = [
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/strings",
    ],
)

cc_test(
    name = "metric_set_test",
    size = "small",
    srcs = ["metric_set_test.cc"],
    deps = [
        ":metric_set",
        "@com_google_googletest//:gtest_main",
    ],
)
