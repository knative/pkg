licenses(["notice"])

cc_test(
    name = "example_test",
    srcs = ["example_test.cc"],
    tags = [
        "manual",
        "notap",
    ],
    deps = [
        ":perf_data_cc_proto",
        "//clients/cxx/analyzers:threshold_analyzer",
        "//clients/proto/analyzers:threshold_analyzer_cc_proto",
        "//helpers/cxx/quickstore",
        "//helpers/cxx/status",
        "//helpers/cxx/status:status_matchers",
        "//helpers/cxx/status:statusor",
        "//quickstore:quickstore_cc_proto",
        "//spec/proto:mako_cc_proto",
        "@com_google_absl//absl/flags:parse",
        "@com_google_glog//:glog",
        "@com_google_googletest//:gtest_main",
        "@com_google_protobuf//:protobuf",
    ],
)

proto_library(
    name = "perf_data_proto",
    srcs = ["perf_data.proto"],
)

cc_proto_library(
    name = "perf_data_cc_proto",
    deps = [":perf_data_proto"],
)
