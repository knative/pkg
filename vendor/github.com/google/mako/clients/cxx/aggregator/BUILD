package(default_visibility = ["//visibility:public"])

licenses(["notice"])

cc_library(
    name = "standard_aggregator",
    srcs = ["standard_aggregator.cc"],
    hdrs = [
        "standard_aggregator.h",
        "threadsafe_running_stats.h",
    ],
    deps = [
        "//internal/cxx:pgmath",
        "//internal/cxx:proto_validation",
        "//internal/cxx/load/common:executor",
        "//spec/cxx:aggregator",
        "//spec/cxx:fileio",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/memory",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/synchronization",
        "@com_google_absl//absl/time",
        "@com_google_glog//:glog",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_test(
    name = "standard_aggregator_test",
    size = "small",
    srcs = [
        "standard_aggregator_test.cc",
    ],
    deps = [
        ":standard_aggregator",
        "//clients/cxx/fileio:memory_fileio",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/memory",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/strings:str_format",
        "@com_google_googletest//:gtest_main",
    ],
)
