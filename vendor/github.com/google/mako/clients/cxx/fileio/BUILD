package(default_visibility = ["//visibility:public"])

licenses(["notice"])

cc_library(
    name = "memory_fileio",
    srcs = ["memory_fileio.cc"],
    hdrs = ["memory_fileio.h"],
    deps = [
        "//spec/cxx:fileio",
        "@com_google_absl//absl/base:core_headers",
        "@com_google_absl//absl/container:flat_hash_map",
        "@com_google_absl//absl/memory",
        "@com_google_absl//absl/strings",
        "@com_google_absl//absl/synchronization",
        "@com_google_glog//:glog",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_test(
    name = "memory_fileio_test",
    size = "small",
    srcs = ["memory_fileio_test.cc"],
    deps = [
        ":memory_fileio",
        "//spec/proto:mako_cc_proto",
        "@com_google_googletest//:gtest_main",
    ],
)
